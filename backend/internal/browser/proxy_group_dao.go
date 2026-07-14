package browser

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProxyGroupDAO manages persistent hierarchical groups for browser proxies.
type ProxyGroupDAO interface {
	List() ([]*ProxyGroup, error)
	GetByID(groupID string) (*ProxyGroup, error)
	Create(input ProxyGroupInput) (*ProxyGroup, error)
	Update(groupID string, input ProxyGroupInput) (*ProxyGroup, error)
	Delete(groupID string) error
	MoveProxies(proxyIDs []string, groupID string) error
	ResolveAssignment(groupID, groupName string) (string, string, error)
}

// SQLiteProxyGroupDAO stores proxy groups in SQLite.
type SQLiteProxyGroupDAO struct {
	db *sql.DB
}

func NewSQLiteProxyGroupDAO(db *sql.DB) *SQLiteProxyGroupDAO {
	return &SQLiteProxyGroupDAO{db: db}
}

func (d *SQLiteProxyGroupDAO) List() ([]*ProxyGroup, error) {
	rows, err := d.db.Query(`
		SELECT group_id, group_name, parent_id, sort_order, created_at, updated_at
		FROM browser_proxy_groups
		ORDER BY sort_order ASC, lower(group_name) ASC, created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list proxy groups: %w", err)
	}
	defer rows.Close()

	var groups []*ProxyGroup
	for rows.Next() {
		group, err := scanProxyGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (d *SQLiteProxyGroupDAO) GetByID(groupID string) (*ProxyGroup, error) {
	row := d.db.QueryRow(`
		SELECT group_id, group_name, parent_id, sort_order, created_at, updated_at
		FROM browser_proxy_groups WHERE group_id = ?`, strings.TrimSpace(groupID))
	group, err := scanProxyGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("proxy group not found: %s", groupID)
	}
	if err != nil {
		return nil, fmt.Errorf("get proxy group: %w", err)
	}
	return group, nil
}

func (d *SQLiteProxyGroupDAO) Create(input ProxyGroupInput) (*ProxyGroup, error) {
	name := strings.TrimSpace(input.GroupName)
	parentID := strings.TrimSpace(input.ParentId)
	if name == "" {
		return nil, errors.New("proxy group name is required")
	}
	if parentID != "" {
		if _, err := d.GetByID(parentID); err != nil {
			return nil, errors.New("parent proxy group does not exist")
		}
	}
	if err := d.ensureUniqueName("", parentID, name); err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	group := &ProxyGroup{
		GroupId: uuid.NewString(), GroupName: name, ParentId: parentID,
		SortOrder: input.SortOrder, CreatedAt: now, UpdatedAt: now,
	}
	_, err := d.db.Exec(`
		INSERT INTO browser_proxy_groups
		(group_id, group_name, parent_id, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		group.GroupId, group.GroupName, group.ParentId, group.SortOrder, group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create proxy group: %w", err)
	}
	return group, nil
}

func (d *SQLiteProxyGroupDAO) Update(groupID string, input ProxyGroupInput) (*ProxyGroup, error) {
	groupID = strings.TrimSpace(groupID)
	name := strings.TrimSpace(input.GroupName)
	parentID := strings.TrimSpace(input.ParentId)
	if name == "" {
		return nil, errors.New("proxy group name is required")
	}
	existing, err := d.GetByID(groupID)
	if err != nil {
		return nil, err
	}
	if parentID != "" {
		if _, err := d.GetByID(parentID); err != nil {
			return nil, errors.New("parent proxy group does not exist")
		}
		if err := d.checkCircularReference(groupID, parentID); err != nil {
			return nil, err
		}
	}
	if err := d.ensureUniqueName(groupID, parentID, name); err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	tx, err := d.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin proxy group update: %w", err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`
		UPDATE browser_proxy_groups
		SET group_name = ?, parent_id = ?, sort_order = ?, updated_at = ?
		WHERE group_id = ?`, name, parentID, input.SortOrder, now, groupID); err != nil {
		return nil, fmt.Errorf("update proxy group: %w", err)
	}
	if _, err = tx.Exec(`UPDATE browser_proxies SET group_name = ? WHERE group_id = ?`, name, groupID); err != nil {
		return nil, fmt.Errorf("sync proxy group name: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit proxy group update: %w", err)
	}
	return &ProxyGroup{
		GroupId: groupID, GroupName: name, ParentId: parentID, SortOrder: input.SortOrder,
		CreatedAt: existing.CreatedAt, UpdatedAt: now,
	}, nil
}

func (d *SQLiteProxyGroupDAO) Delete(groupID string) error {
	group, err := d.GetByID(groupID)
	if err != nil {
		return err
	}
	parentName := ""
	if group.ParentId != "" {
		parent, parentErr := d.GetByID(group.ParentId)
		if parentErr != nil {
			return parentErr
		}
		parentName = parent.GroupName
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("begin proxy group delete: %w", err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE browser_proxy_groups SET parent_id = ?, updated_at = ? WHERE parent_id = ?`, group.ParentId, time.Now().Format(time.RFC3339), groupID); err != nil {
		return fmt.Errorf("move child proxy groups: %w", err)
	}
	if _, err = tx.Exec(`UPDATE browser_proxies SET group_id = ?, group_name = ? WHERE group_id = ?`, group.ParentId, parentName, groupID); err != nil {
		return fmt.Errorf("move grouped proxies: %w", err)
	}
	if _, err = tx.Exec(`DELETE FROM browser_proxy_groups WHERE group_id = ?`, groupID); err != nil {
		return fmt.Errorf("delete proxy group: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit proxy group delete: %w", err)
	}
	return nil
}

func (d *SQLiteProxyGroupDAO) MoveProxies(proxyIDs []string, groupID string) error {
	if len(proxyIDs) == 0 {
		return nil
	}
	groupID = strings.TrimSpace(groupID)
	groupName := ""
	if groupID != "" {
		group, err := d.GetByID(groupID)
		if err != nil {
			return err
		}
		groupName = group.GroupName
	}

	placeholders := make([]string, len(proxyIDs))
	args := make([]any, 0, len(proxyIDs)+2)
	args = append(args, groupID, groupName)
	for i, proxyID := range proxyIDs {
		placeholders[i] = "?"
		args = append(args, strings.TrimSpace(proxyID))
	}
	query := `UPDATE browser_proxies SET group_id = ?, group_name = ? WHERE proxy_id IN (` + strings.Join(placeholders, ",") + `)`
	if _, err := d.db.Exec(query, args...); err != nil {
		return fmt.Errorf("move proxies to group: %w", err)
	}
	return nil
}

func (d *SQLiteProxyGroupDAO) ResolveAssignment(groupID, groupName string) (string, string, error) {
	groupID = strings.TrimSpace(groupID)
	groupName = strings.TrimSpace(groupName)
	if groupID != "" {
		group, err := d.GetByID(groupID)
		if err != nil {
			return "", "", err
		}
		return group.GroupId, group.GroupName, nil
	}
	if groupName == "" {
		return "", "", nil
	}

	var id, name string
	err := d.db.QueryRow(`
		SELECT group_id, group_name FROM browser_proxy_groups
		WHERE parent_id = '' AND lower(group_name) = lower(?)
		ORDER BY created_at ASC LIMIT 1`, groupName).Scan(&id, &name)
	if err == nil {
		return id, name, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("resolve proxy group: %w", err)
	}
	group, err := d.Create(ProxyGroupInput{GroupName: groupName})
	if err != nil {
		return "", "", err
	}
	return group.GroupId, group.GroupName, nil
}

func (d *SQLiteProxyGroupDAO) ensureUniqueName(excludeID, parentID, name string) error {
	var count int
	if err := d.db.QueryRow(`
		SELECT COUNT(1) FROM browser_proxy_groups
		WHERE parent_id = ? AND lower(group_name) = lower(?) AND group_id != ?`,
		parentID, name, excludeID).Scan(&count); err != nil {
		return fmt.Errorf("check proxy group name: %w", err)
	}
	if count > 0 {
		return errors.New("a proxy group with the same name already exists at this level")
	}
	return nil
}

func (d *SQLiteProxyGroupDAO) checkCircularReference(groupID, parentID string) error {
	if groupID == parentID {
		return errors.New("a proxy group cannot be its own parent")
	}
	visited := map[string]bool{}
	currentID := parentID
	for currentID != "" {
		if currentID == groupID {
			return errors.New("a proxy group cannot be moved below its descendant")
		}
		if visited[currentID] {
			return errors.New("proxy group hierarchy contains a cycle")
		}
		visited[currentID] = true
		group, err := d.GetByID(currentID)
		if err != nil {
			return err
		}
		currentID = group.ParentId
	}
	return nil
}

func scanProxyGroup(scanner scanner) (*ProxyGroup, error) {
	var group ProxyGroup
	if err := scanner.Scan(&group.GroupId, &group.GroupName, &group.ParentId, &group.SortOrder, &group.CreatedAt, &group.UpdatedAt); err != nil {
		return nil, err
	}
	return &group, nil
}
