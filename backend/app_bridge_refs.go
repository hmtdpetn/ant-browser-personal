package backend

import "strings"

func (a *App) bindProfileXrayBridge(profileId string, bridgeKey string) {
	profileId = strings.TrimSpace(profileId)
	bridgeKey = strings.TrimSpace(bridgeKey)
	if profileId == "" || bridgeKey == "" {
		return
	}

	a.bridgeMu.Lock()
	a.xrayBridgeRefs[profileId] = bridgeKey
	a.bridgeMu.Unlock()
}

func (a *App) releaseProfileXrayBridge(profileId string) {
	profileId = strings.TrimSpace(profileId)
	if profileId == "" {
		return
	}

	a.bridgeMu.Lock()
	bridgeKey := a.xrayBridgeRefs[profileId]
	delete(a.xrayBridgeRefs, profileId)
	a.bridgeMu.Unlock()

	if bridgeKey != "" && a.xrayMgr != nil {
		a.xrayMgr.ReleaseBridge(bridgeKey)
	}
}

func (a *App) clearProfileXrayBridges() {
	a.bridgeMu.Lock()
	a.xrayBridgeRefs = make(map[string]string)
	a.bridgeMu.Unlock()
}
