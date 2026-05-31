package browser

import (
	"ant-chrome/backend/internal/logger"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Copy 复制实例配置（除指纹参数外全部复制，指纹使用默认值生成新种子）
func (m *Manager) Copy(profileId string, newName string) (*Profile, error) {
	log := logger.New("Browser")
	m.InitData()
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if m.Config.App.MaxProfileLimit > 0 && len(m.Profiles) >= m.Config.App.MaxProfileLimit {
		log.Error("复制实例失败: 达到数量上限", logger.F("limit", m.Config.App.MaxProfileLimit))
		return nil, newProfileLimitExceededError(m.Config.App.MaxProfileLimit, "复制实例")
	}

	src, exists := m.Profiles[profileId]
	if !exists {
		log.Error("源实例不存在", logger.F("profile_id", profileId))
		return nil, fmt.Errorf("profile not found")
	}

	now := time.Now().Format(time.RFC3339)
	newId := uuid.NewString()

	profileName := strings.TrimSpace(newName)
	if profileName == "" {
		profileName = src.ProfileName + " (副本)"
	}

	profile := &Profile{
		ProfileId:          newId,
		ProfileName:        profileName,
		UserDataDir:        newId,
		CoreId:             normalizeProfileCoreID(src.CoreId),
		FingerprintArgs:    append([]string{}, m.Config.Browser.DefaultFingerprintArgs...),
		ProxyId:            src.ProxyId,
		ProxyConfig:        src.ProxyConfig,
		ProxyBindSourceID:  src.ProxyBindSourceID,
		ProxyBindSourceURL: src.ProxyBindSourceURL,
		ProxyBindName:      src.ProxyBindName,
		ProxyBindUpdatedAt: src.ProxyBindUpdatedAt,
		LaunchArgs:         append([]string{}, src.LaunchArgs...),
		Tags:               append([]string{}, src.Tags...),
		Keywords:           append([]string{}, src.Keywords...),
		GroupId:            src.GroupId,
		Running:            false,
		DebugPort:          0,
		Pid:                0,
		LastError:          "",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	m.Profiles[newId] = profile
	log.Info("实例复制成功", logger.F("src_id", profileId), logger.F("new_id", newId), logger.F("new_name", profileName))

	if err := m.SaveProfiles(); err != nil {
		return nil, err
	}

	m.ensureProfileLaunchCode(profile)
	return profile, nil
}
