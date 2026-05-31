package automation

import (
	"context"
	"fmt"
	"strings"
)

type runtimeInstallWorkspace struct {
	TempRoot   string
	StagingDir string
}

func (w runtimeInstallWorkspace) cleanup() {
	_ = removeRuntimeInstallWorkspace(w.StagingDir)
}

func ensureRuntimeInstallContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (m *Manager) beginRuntimeInstall(flagAlreadySet bool) bool {
	if flagAlreadySet {
		return true
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.installing {
		return false
	}
	m.installing = true
	m.lastError = ""
	return true
}

func (m *Manager) finishRuntimeInstall() {
	m.mu.Lock()
	m.installing = false
	m.mu.Unlock()
}

func validateRuntimeVersion(runtimeVersion string) error {
	if strings.TrimSpace(runtimeVersion) == "" {
		return fmt.Errorf("automation runtime version is empty")
	}
	return nil
}

func (m *Manager) clearRuntimeInstallError() {
	m.mu.Lock()
	m.lastError = ""
	m.mu.Unlock()
}
