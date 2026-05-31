package browser

import (
	"fmt"
	"strings"
)

type resolvedProfileProxyInput struct {
	ProxyId            string
	ProxyConfig        string
	SelectedProxy      Proxy
	HasSelectedProxy   bool
	FallbackToDirect   bool
	UsedConfigFallback bool
}

// resolveProfileProxyInput 统一处理实例输入中的代理参数。
// 规则：
// 1. proxyId 命中代理池 => 使用代理池配置；
// 2. proxyId 未命中且提供 proxyConfig => 改为自定义代理（清空 proxyId）；
// 3. proxyId/proxyConfig 都为空 => 回退直连；
// 4. proxyId 未命中且 proxyConfig 为空 => 直接报错，避免静默回退。
func (m *Manager) resolveProfileProxyInput(proxyIdInput string, proxyConfigInput string) (resolvedProfileProxyInput, error) {
	proxyId := strings.TrimSpace(proxyIdInput)
	proxyConfig := strings.TrimSpace(proxyConfigInput)

	if proxyId != "" {
		if proxyItem, ok := m.GetProxyByID(proxyId); ok {
			return resolvedProfileProxyInput{
				ProxyId:          strings.TrimSpace(proxyItem.ProxyId),
				ProxyConfig:      strings.TrimSpace(proxyItem.ProxyConfig),
				SelectedProxy:    proxyItem,
				HasSelectedProxy: true,
			}, nil
		}
		if proxyConfig != "" {
			return resolvedProfileProxyInput{
				ProxyId:            "",
				ProxyConfig:        proxyConfig,
				UsedConfigFallback: true,
			}, nil
		}
		return resolvedProfileProxyInput{}, fmt.Errorf("代理ID不存在（proxy id not found: %s），且未提供 proxyConfig", proxyId)
	}

	if proxyConfig != "" {
		return resolvedProfileProxyInput{
			ProxyId:     "",
			ProxyConfig: proxyConfig,
		}, nil
	}

	return resolvedProfileProxyInput{
		ProxyId:          "",
		ProxyConfig:      "",
		FallbackToDirect: true,
	}, nil
}
