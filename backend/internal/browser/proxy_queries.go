package browser

func ListProxiesWithFallback(proxyDAO ProxyDAO, fallback []Proxy) []Proxy {
	if proxyDAO != nil {
		if list, err := proxyDAO.List(); err == nil {
			return list
		}
	}
	return append([]Proxy{}, fallback...)
}

func ListProxyGroups(proxyDAO ProxyDAO) []string {
	if proxyDAO != nil {
		if groups, err := proxyDAO.ListGroups(); err == nil {
			return groups
		}
	}
	return nil
}

func ListProxiesByGroupWithFallback(proxyDAO ProxyDAO, groupName string, fallback []Proxy) []Proxy {
	if proxyDAO != nil {
		if list, err := proxyDAO.ListByGroup(groupName); err == nil {
			return list
		}
	}

	var result []Proxy
	for _, item := range fallback {
		if item.GroupName == groupName {
			result = append(result, item)
		}
	}
	return result
}

func LatestProxiesWithFallback(proxyDAO ProxyDAO, fallback []Proxy) []Proxy {
	if proxyDAO != nil {
		if list, err := proxyDAO.List(); err == nil && len(list) > 0 {
			return list
		}
	}
	return fallback
}
