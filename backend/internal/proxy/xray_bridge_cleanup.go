package proxy

import (
	"ant-chrome/backend/internal/logger"
	"time"
)

func (m *XrayManager) cleanupLoop() {
	ticker := time.NewTicker(xrayBridgeCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.recycleIdleBridges()
		case <-m.stopCh:
			return
		}
	}
}

func (m *XrayManager) recycleIdleBridges() {
	now := time.Now()
	var stale []*XrayBridge

	m.mu.Lock()
	for key, bridge := range m.Bridges {
		if bridge == nil {
			delete(m.Bridges, key)
			continue
		}
		if bridge.RefCount > 0 {
			continue
		}
		if now.Sub(bridge.LastUsedAt) < xrayBridgeIdleTTL {
			continue
		}

		bridge.Stopping = true
		stale = append(stale, bridge)
		delete(m.Bridges, key)
	}
	m.mu.Unlock()

	if len(stale) == 0 {
		return
	}

	log := logger.New("Xray")
	for _, bridge := range stale {
		log.Info("回收空闲桥接进程", logger.F("key", bridge.NodeKey), logger.F("pid", bridge.Pid))
		m.stopBridgeProcess(bridge)
	}
}

func (m *XrayManager) stopBridgeProcess(bridge *XrayBridge) {
	if bridge == nil || bridge.Cmd == nil || bridge.Cmd.Process == nil {
		return
	}
	_ = bridge.Cmd.Process.Kill()
}
