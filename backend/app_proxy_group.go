package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/logger"
	"fmt"
)

type BrowserProxyGroup = browser.ProxyGroup
type BrowserProxyGroupInput = browser.ProxyGroupInput
type BrowserProxyGroupWithCount = browser.ProxyGroupWithCount

// BrowserProxyGroupList returns all persistent proxy groups with direct counts.
func (a *App) BrowserProxyGroupList() []BrowserProxyGroupWithCount {
	log := logger.New("ProxyGroup")
	if a.browserMgr == nil || a.browserMgr.ProxyGroupDAO == nil {
		log.Error("ProxyGroupDAO is not initialized")
		return []BrowserProxyGroupWithCount{}
	}
	groups, err := a.browserMgr.ProxyGroupDAO.List()
	if err != nil {
		log.Error("List proxy groups failed", logger.F("error", err))
		return []BrowserProxyGroupWithCount{}
	}
	counts := map[string]int{}
	if a.browserMgr.ProxyDAO != nil {
		proxies, listErr := a.browserMgr.ProxyDAO.List()
		if listErr != nil {
			log.Error("Count grouped proxies failed", logger.F("error", listErr))
		} else {
			for _, item := range proxies {
				if item.GroupId != "" {
					counts[item.GroupId]++
				}
			}
		}
	}
	result := make([]BrowserProxyGroupWithCount, 0, len(groups))
	for _, group := range groups {
		result = append(result, BrowserProxyGroupWithCount{
			ProxyGroup: *group,
			ProxyCount: counts[group.GroupId],
		})
	}
	return result
}

func (a *App) BrowserProxyGroupCreate(input BrowserProxyGroupInput) (*BrowserProxyGroup, error) {
	if a.browserMgr == nil || a.browserMgr.ProxyGroupDAO == nil {
		return nil, fmt.Errorf("ProxyGroupDAO is not initialized")
	}
	return a.browserMgr.ProxyGroupDAO.Create(input)
}

func (a *App) BrowserProxyGroupUpdate(groupID string, input BrowserProxyGroupInput) (*BrowserProxyGroup, error) {
	if a.browserMgr == nil || a.browserMgr.ProxyGroupDAO == nil {
		return nil, fmt.Errorf("ProxyGroupDAO is not initialized")
	}
	return a.browserMgr.ProxyGroupDAO.Update(groupID, input)
}

func (a *App) BrowserProxyGroupDelete(groupID string) error {
	if a.browserMgr == nil || a.browserMgr.ProxyGroupDAO == nil {
		return fmt.Errorf("ProxyGroupDAO is not initialized")
	}
	return a.browserMgr.ProxyGroupDAO.Delete(groupID)
}

func (a *App) BrowserProxyMoveToGroup(proxyIDs []string, groupID string) error {
	if a.browserMgr == nil || a.browserMgr.ProxyGroupDAO == nil {
		return fmt.Errorf("ProxyGroupDAO is not initialized")
	}
	return a.browserMgr.ProxyGroupDAO.MoveProxies(proxyIDs, groupID)
}
