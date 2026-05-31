package automation

import (
	"context"
	"fmt"
)

func (m *Manager) SelfCheck(ctx context.Context) (RuntimeCheckResult, error) {
	state := m.CurrentState()
	if !state.Ready {
		return RuntimeCheckResult{}, fmt.Errorf("自动化运行时尚未就绪")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := m.verifyNodeWithPlaywright(ctx, state.NodePath, state.RuntimeDir)
	if err != nil {
		return RuntimeCheckResult{}, fmt.Errorf("自动化运行时自检失败: %w", err)
	}
	result.NodeSource = state.NodeSource
	return result, nil
}
