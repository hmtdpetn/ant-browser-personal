package browser

import (
	"ant-chrome/backend/internal/logger"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Create 创建配置
func (m *Manager) Create(input ProfileInput) (*Profile, error) {
	log := logger.New("Browser")
	m.InitData()
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if m.Config.App.MaxProfileLimit > 0 && len(m.Profiles) >= m.Config.App.MaxProfileLimit {
		return nil, fmt.Errorf("实例数量已达上限 (%d个)，无法创建新的实例。请兑换额度后重试！", m.Config.App.MaxProfileLimit)
	}

	now := time.Now().Format(time.RFC3339)
	profileId := uuid.NewString()
	userDataDir := strings.TrimSpace(input.UserDataDir)
	if userDataDir == "" {
		userDataDir = profileId
	}
	resolvedProxy, err := m.resolveProfileProxyInput(input.ProxyId, input.ProxyConfig)
	if err != nil {
		log.Error("代理绑定失败", logger.F("profile_id", profileId), logger.F("proxy_id", strings.TrimSpace(input.ProxyId)), logger.F("error", err.Error()))
		return nil, err
	}
	coreId := normalizeProfileCoreID(input.CoreId)
	if coreId == "" {
		if defaultCore, ok := m.GetDefaultCore(); ok {
			coreId = defaultCore.CoreId
		}
	}
	profile := &Profile{
		ProfileId:       profileId,
		ProfileName:     input.ProfileName,
		UserDataDir:     userDataDir,
		CoreId:          coreId,
		FingerprintArgs: input.FingerprintArgs,
		ProxyId:         resolvedProxy.ProxyId,
		ProxyConfig:     resolvedProxy.ProxyConfig,
		LaunchArgs:      input.LaunchArgs,
		Tags:            input.Tags,
		Keywords:        append([]string{}, input.Keywords...),
		GroupId:         strings.TrimSpace(input.GroupId),
		Running:         false,
		DebugPort:       0,
		Pid:             0,
		LastError:       "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if resolvedProxy.HasSelectedProxy {
		_ = BindProfileToProxy(profile, resolvedProxy.SelectedProxy, true)
	} else if resolvedProxy.FallbackToDirect {
		_ = m.bindProfileToDirectProxy(profile)
	}
	if resolvedProxy.UsedConfigFallback {
		log.Warn("代理ID未命中，已改为使用输入的代理配置",
			logger.F("profile_id", profileId),
			logger.F("proxy_id", strings.TrimSpace(input.ProxyId)),
		)
	}
	m.Profiles[profileId] = profile
	log.Info("浏览器配置创建", logger.F("profile_id", profileId), logger.F("profile_name", input.ProfileName))
	if err := m.SaveProfiles(); err != nil {
		return nil, err
	}
	m.ensureProfileLaunchCode(profile)
	return profile, nil
}

func (m *Manager) ensureProfileLaunchCode(profile *Profile) {
	if m.CodeProvider == nil || profile == nil {
		return
	}
	if code, err := m.CodeProvider.EnsureCode(profile.ProfileId); err == nil {
		profile.LaunchCode = code
	}
}

func newProfileLimitExceededError(limit int, action string) error {
	return fmt.Errorf("实例数量已达上限 (%d个)，无法%s。请兑换额度后重试！", limit, action)
}

func buildProfileGroupID(value string) string {
	return strings.TrimSpace(value)
}
