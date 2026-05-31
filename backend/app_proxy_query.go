package backend

import (
	"ant-chrome/backend/internal/proxy"
)

func (a *App) BrowserProxyList() []BrowserProxy {
	if a.browserMgr.ProxyDAO != nil {
		if list, err := a.browserMgr.ProxyDAO.List(); err == nil {
			return list
		}
	}
	return append([]BrowserProxy{}, a.config.Browser.Proxies...)
}

// BrowserProxyListGroups 获取所有代理分组名称
func (a *App) BrowserProxyListGroups() []string {
	if a.browserMgr.ProxyDAO != nil {
		if groups, err := a.browserMgr.ProxyDAO.ListGroups(); err == nil {
			return groups
		}
	}
	return nil
}

// BrowserProxyListByGroup 按分组名称查询代理
func (a *App) BrowserProxyListByGroup(groupName string) []BrowserProxy {
	if a.browserMgr.ProxyDAO != nil {
		if list, err := a.browserMgr.ProxyDAO.ListByGroup(groupName); err == nil {
			return list
		}
	}

	var result []BrowserProxy
	for _, item := range a.config.Browser.Proxies {
		if item.GroupName == groupName {
			result = append(result, item)
		}
	}
	return result
}

// ValidateProxyConfig 验证代理配置是否支持
func (a *App) ValidateProxyConfig(proxyConfig string, proxyId string) ProxyValidationResult {
	proxies := a.getLatestProxies()
	supported, errorMsg := proxy.ValidateProxyConfig(proxyConfig, proxies, proxyId)
	return ProxyValidationResult{
		Supported: supported,
		ErrorMsg:  errorMsg,
	}
}

// TestProxyConnectivity 测试代理连通性
func (a *App) TestProxyConnectivity(proxyId string, proxyConfig string) ProxyTestResult {
	proxies := a.getLatestProxies()
	result := proxy.TestConnectivity(proxyId, proxyConfig, proxies, nil)
	return ProxyTestResult{ProxyId: result.ProxyId, Ok: result.Ok, LatencyMs: result.LatencyMs, Error: result.Error}
}

// TestProxyRealConnectivity 通过真实 HTTP 请求测试代理连通性（Wails 绑定）
// 参考 Clash URLTest 策略：多 URL fallback + 复用桥接 + TCP ping 降级
func (a *App) TestProxyRealConnectivity(proxyId string) ProxyTestResult {
	proxies := a.getLatestProxies()
	result := proxy.SpeedTest(proxyId, proxies, a.xrayMgr, a.singboxMgr, nil)
	return ProxyTestResult{ProxyId: result.ProxyId, Ok: result.Ok, LatencyMs: result.LatencyMs, Error: result.Error}
}

// getLatestProxies 获取最新的代理列表，优先从数据库读取
func (a *App) getLatestProxies() []BrowserProxy {
	if a.browserMgr.ProxyDAO != nil {
		if list, err := a.browserMgr.ProxyDAO.List(); err == nil && len(list) > 0 {
			return list
		}
	}
	return a.config.Browser.Proxies
}
