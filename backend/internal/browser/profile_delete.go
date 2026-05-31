package browser

import (
	"ant-chrome/backend/internal/logger"
	"fmt"
)

// Delete 删除配置
func (m *Manager) Delete(profileId string) error {
	log := logger.New("Browser")
	m.InitData()
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if _, exists := m.Profiles[profileId]; !exists {
		log.Error("浏览器配置不存在", logger.F("profile_id", profileId))
		return fmt.Errorf("profile not found")
	}
	delete(m.Profiles, profileId)
	log.Info("浏览器配置删除", logger.F("profile_id", profileId))

	if m.ProfileDAO != nil {
		if err := m.ProfileDAO.Delete(profileId); err != nil {
			log.Error("数据库删除实例失败", logger.F("profile_id", profileId), logger.F("error", err))
			return err
		}
	} else {
		if err := m.SaveProfiles(); err != nil {
			return err
		}
	}

	if m.CodeProvider != nil {
		_ = m.CodeProvider.Remove(profileId)
	}
	return nil
}
