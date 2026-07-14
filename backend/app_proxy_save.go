package backend

import (
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/logger"
	"ant-chrome/backend/internal/proxy"
)

func (a *App) SaveBrowserProxies(proxies []BrowserProxy) error {
	log := logger.New("Browser")
	normalized := proxy.NormalizeBrowserProxies(proxies, generateUUID)

	if a.browserMgr.ProxyGroupDAO != nil {
		for i := range normalized {
			groupID, groupName, err := a.browserMgr.ProxyGroupDAO.ResolveAssignment(normalized[i].GroupId, normalized[i].GroupName)
			if err != nil {
				log.Error("Resolve proxy group failed", logger.F("proxy_id", normalized[i].ProxyId), logger.F("error", err))
				return err
			}
			normalized[i].GroupId = groupID
			normalized[i].GroupName = groupName
		}
	}

	a.config.Browser.Proxies = normalized

	if a.browserMgr.ProxyDAO != nil {
		if err := a.browserMgr.ProxyDAO.DeleteAll(); err != nil {
			log.Error("清空代理表失败", logger.F("error", err))
			return err
		}
		for _, item := range normalized {
			if err := a.browserMgr.ProxyDAO.Upsert(item); err != nil {
				log.Error("代理保存失败", logger.F("proxy_id", item.ProxyId), logger.F("error", err))
				return err
			}
		}
		log.Info("代理列表已保存到数据库", logger.F("count", len(normalized)))
		a.reconcileProfileProxyBindings()
		return nil
	}

	if err := config.SaveProxies(a.resolveAppPath("proxies.yaml"), normalized); err != nil {
		log.Error("代理列表保存失败", logger.F("error", err))
		return err
	}
	a.reconcileProfileProxyBindings()
	return nil
}
