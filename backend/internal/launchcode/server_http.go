package launchcode

import "net/http"

func (s *LaunchServer) buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/automation/scripts", s.handleAutomationScripts)
	mux.HandleFunc("/api/automation/scripts/", s.handleAutomationScriptByID)
	mux.HandleFunc("/api/automation/scripts/run", s.handleAutomationScriptRun)
	mux.HandleFunc("/api/automation/scripts/runs", s.handleAutomationScriptRuns)
	mux.HandleFunc("/api/automation/hooks/", s.handleAutomationPublicHook)
	mux.HandleFunc("/api/groups/move-profiles", s.handleMoveProfilesToGroup)
	mux.HandleFunc("/api/groups", s.handleGroups)
	mux.HandleFunc("/api/groups/", s.handleGroupByID)
	mux.HandleFunc("/api/proxy-groups/move-proxies", s.handleMoveProxiesToGroup)
	mux.HandleFunc("/api/proxy-groups", s.handleProxyGroups)
	mux.HandleFunc("/api/proxy-groups/", s.handleProxyGroupByID)
	mux.HandleFunc("/api/profiles", s.handleProfiles)
	mux.HandleFunc("/api/profiles/", s.handleProfileByID)
	mux.HandleFunc("/api/runtime/active", s.handleRuntimeActive)
	mux.HandleFunc("/api/runtime/session", s.handleRuntimeSession)
	mux.HandleFunc("/api/runtime/status", s.handleRuntimeStatus)
	mux.HandleFunc("/api/runtime/stop", s.handleRuntimeStop)
	mux.HandleFunc("/api/proxy-gateway/switch", s.handleProxyGateway)
	mux.HandleFunc("/api/proxy-gateway/status", s.handleProxyGateway)
	mux.HandleFunc("/api/proxy-gateway/routing", s.handleProxyGatewayRouting)
	mux.HandleFunc("/api/launch", s.handleLaunchWithBody)
	mux.HandleFunc("/api/launch/logs", s.handleLaunchLogs)
	mux.HandleFunc("/api/launch/", s.handleLaunch)
	mux.HandleFunc("/", s.handleCDPProxy)
	return mux
}

func (s *LaunchServer) buildHandler(includeLocalhost bool) http.Handler {
	var handler http.Handler = s.buildMux()
	handler = s.apiAuthMiddleware(handler)
	if includeLocalhost {
		handler = s.localhostMiddleware(handler)
	}
	return handler
}

// handleHealth GET /api/health
func (s *LaunchServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"ok":    false,
			"error": "method not allowed",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// NewTestHandler 返回不含 localhost 限制的 handler，仅供测试使用
func NewTestHandler(s *LaunchServer) http.Handler {
	return s.buildHandler(false)
}
