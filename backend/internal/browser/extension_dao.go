package browser

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ExtensionDAO interface {
	List() ([]Extension, error)
	ListEnabled() ([]Extension, error)
	ListByIDs(extensionIDs []string) ([]Extension, error)
	Get(extensionID string) (Extension, error)
	Upsert(extension Extension) error
	SetEnabled(extensionID string, enabled bool) error
	Delete(extensionID string) error
	GetProfileSettings(profileID string) (ProfileExtensionSettings, error)
	SetProfileSettings(profileID string, extensionIDs []string, configured bool) (ProfileExtensionSettings, error)
	DeleteProfileSettings(profileID string) error
}

type SQLiteExtensionDAO struct {
	db *sql.DB
}

func NewSQLiteExtensionDAO(db *sql.DB) *SQLiteExtensionDAO {
	return &SQLiteExtensionDAO{db: db}
}

func (d *SQLiteExtensionDAO) List() ([]Extension, error) {
	return d.listWhere("", nil)
}

func (d *SQLiteExtensionDAO) ListEnabled() ([]Extension, error) {
	return d.listWhere("WHERE enabled = ?", []any{1})
}

func (d *SQLiteExtensionDAO) ListByIDs(extensionIDs []string) ([]Extension, error) {
	ids := normalizeExtensionIDs(extensionIDs)
	if len(ids) == 0 {
		return []Extension{}, nil
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return d.listWhere("WHERE enabled = 1 AND extension_id IN ("+strings.Join(placeholders, ",")+")", args)
}

func (d *SQLiteExtensionDAO) Get(extensionID string) (Extension, error) {
	row := d.db.QueryRow(`
		SELECT extension_id, name, version, description, icon_data_url, manifest_json, source_url, install_dir, enabled, installed_at, updated_at
		FROM browser_extensions WHERE extension_id = ?`, strings.TrimSpace(extensionID))
	return scanExtension(row)
}

func (d *SQLiteExtensionDAO) Upsert(extension Extension) error {
	now := time.Now().Format(time.RFC3339)
	if strings.TrimSpace(extension.InstalledAt) == "" {
		extension.InstalledAt = now
	}
	extension.UpdatedAt = now
	_, err := d.db.Exec(`
		INSERT INTO browser_extensions (extension_id, name, version, description, icon_data_url, manifest_json, source_url, install_dir, enabled, installed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(extension_id) DO UPDATE SET
		  name = excluded.name,
		  version = excluded.version,
		  description = excluded.description,
		  icon_data_url = excluded.icon_data_url,
		  manifest_json = excluded.manifest_json,
		  source_url = excluded.source_url,
		  install_dir = excluded.install_dir,
		  enabled = excluded.enabled,
		  updated_at = excluded.updated_at`,
		extension.ExtensionID,
		extension.Name,
		extension.Version,
		extension.Description,
		extension.IconDataURL,
		extension.ManifestJSON,
		extension.SourceURL,
		extension.InstallDir,
		boolToInt(extension.Enabled),
		extension.InstalledAt,
		extension.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("保存插件失败: %w", err)
	}
	return nil
}

func (d *SQLiteExtensionDAO) SetEnabled(extensionID string, enabled bool) error {
	result, err := d.db.Exec(
		`UPDATE browser_extensions SET enabled = ?, updated_at = ? WHERE extension_id = ?`,
		boolToInt(enabled), time.Now().Format(time.RFC3339), strings.TrimSpace(extensionID),
	)
	if err != nil {
		return fmt.Errorf("更新插件状态失败: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *SQLiteExtensionDAO) Delete(extensionID string) error {
	_, err := d.db.Exec(`DELETE FROM browser_extensions WHERE extension_id = ?`, strings.TrimSpace(extensionID))
	if err != nil {
		return fmt.Errorf("删除插件失败: %w", err)
	}
	_, _ = d.db.Exec(`DELETE FROM browser_profile_extensions WHERE extension_id = ?`, strings.TrimSpace(extensionID))
	return nil
}

func (d *SQLiteExtensionDAO) GetProfileSettings(profileID string) (ProfileExtensionSettings, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return ProfileExtensionSettings{}, fmt.Errorf("实例 ID 不能为空")
	}
	settings := ProfileExtensionSettings{ProfileID: profileID}
	var configured int
	row := d.db.QueryRow(`SELECT configured, updated_at FROM browser_profile_extension_settings WHERE profile_id = ?`, profileID)
	if err := row.Scan(&configured, &settings.UpdatedAt); err != nil && err != sql.ErrNoRows {
		return ProfileExtensionSettings{}, err
	} else if err == nil {
		settings.Configured = configured != 0
	}

	rows, err := d.db.Query(`SELECT extension_id FROM browser_profile_extensions WHERE profile_id = ? AND enabled = 1 ORDER BY created_at ASC`, profileID)
	if err != nil {
		return ProfileExtensionSettings{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var extensionID string
		if err := rows.Scan(&extensionID); err != nil {
			return ProfileExtensionSettings{}, err
		}
		settings.ExtensionIDs = append(settings.ExtensionIDs, extensionID)
	}
	return settings, rows.Err()
}

func (d *SQLiteExtensionDAO) SetProfileSettings(profileID string, extensionIDs []string, configured bool) (ProfileExtensionSettings, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return ProfileExtensionSettings{}, fmt.Errorf("实例 ID 不能为空")
	}
	ids := normalizeExtensionIDs(extensionIDs)
	now := time.Now().Format(time.RFC3339)
	tx, err := d.db.Begin()
	if err != nil {
		return ProfileExtensionSettings{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO browser_profile_extension_settings (profile_id, configured, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(profile_id) DO UPDATE SET configured = excluded.configured, updated_at = excluded.updated_at`, profileID, boolToInt(configured), now); err != nil {
		return ProfileExtensionSettings{}, err
	}
	if _, err := tx.Exec(`DELETE FROM browser_profile_extensions WHERE profile_id = ?`, profileID); err != nil {
		return ProfileExtensionSettings{}, err
	}
	for _, extensionID := range ids {
		if _, err := tx.Exec(`INSERT INTO browser_profile_extensions (profile_id, extension_id, enabled, created_at, updated_at) VALUES (?, ?, 1, ?, ?)`, profileID, extensionID, now, now); err != nil {
			return ProfileExtensionSettings{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return ProfileExtensionSettings{}, err
	}
	return d.GetProfileSettings(profileID)
}

func (d *SQLiteExtensionDAO) DeleteProfileSettings(profileID string) error {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM browser_profile_extensions WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM browser_profile_extension_settings WHERE profile_id = ?`, profileID); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *SQLiteExtensionDAO) listWhere(where string, args []any) ([]Extension, error) {
	query := `
		SELECT extension_id, name, version, description, icon_data_url, manifest_json, source_url, install_dir, enabled, installed_at, updated_at
		FROM browser_extensions ` + where + ` ORDER BY installed_at DESC, name ASC`
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询插件列表失败: %w", err)
	}
	defer rows.Close()

	items := []Extension{}
	for rows.Next() {
		extension, err := scanExtension(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, extension)
	}
	return items, rows.Err()
}

type extensionScanner interface {
	Scan(dest ...any) error
}

func scanExtension(scanner extensionScanner) (Extension, error) {
	var extension Extension
	var enabled int
	if err := scanner.Scan(
		&extension.ExtensionID,
		&extension.Name,
		&extension.Version,
		&extension.Description,
		&extension.IconDataURL,
		&extension.ManifestJSON,
		&extension.SourceURL,
		&extension.InstallDir,
		&enabled,
		&extension.InstalledAt,
		&extension.UpdatedAt,
	); err != nil {
		return Extension{}, err
	}
	extension.Enabled = enabled != 0
	return extension, nil
}

func normalizeExtensionIDs(extensionIDs []string) []string {
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(extensionIDs))
	for _, extensionID := range extensionIDs {
		id := strings.TrimSpace(extensionID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}
