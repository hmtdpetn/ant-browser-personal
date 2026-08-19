package launchcode

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/gateway"
)

type proxyGatewayAPIFake struct {
	profile *browser.Profile
	status  gateway.Status
	routing gateway.RoutingConfig
	force   bool
}

func (f *proxyGatewayAPIFake) SwitchProxy(profileID, proxyID, proxyConfig string, force bool) (ProxyGatewaySwitchResult, error) {
	f.force = force
	f.status.ProfileID = profileID
	f.status.CurrentRouteID = proxyID
	f.status.ProxyURL = "socks5://127.0.0.1:19001"
	f.status.Mode = f.routing.Mode
	return ProxyGatewaySwitchResult{Profile: f.profile, Gateway: f.status, AppliedLive: true}, nil
}

func (f *proxyGatewayAPIFake) GetRouting(_ string) (gateway.RoutingConfig, error) {
	return f.routing, nil
}

func (f *proxyGatewayAPIFake) SaveRouting(profileID string, routing gateway.RoutingConfig, force bool) (gateway.Status, error) {
	f.routing = routing
	f.force = force
	f.status.ProfileID = profileID
	f.status.Mode = routing.Mode
	return f.status, nil
}

func (f *proxyGatewayAPIFake) Status(profileID string) (gateway.Status, error) {
	f.status.ProfileID = profileID
	return f.status, nil
}

func proxyGatewayAPIRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func TestProxyGatewayHTTPAPI(t *testing.T) {
	fake := &proxyGatewayAPIFake{
		profile: &browser.Profile{ProfileId: "profile-1", ProfileName: "Profile 1", Running: true},
		routing: gateway.RoutingConfig{Mode: gateway.ModeProxy},
	}
	server := NewLaunchServer(nil, nil, nil, 0)
	server.SetProxyGatewayController(fake)
	handler := NewTestHandler(server)

	status := proxyGatewayAPIRequest(t, handler, http.MethodGet, "/api/proxy-gateway/status?profileId=profile-1", "")
	if status.Code != http.StatusOK {
		t.Fatalf("status code = %d: %s", status.Code, status.Body.String())
	}
	var statusPayload struct {
		OK      bool           `json:"ok"`
		Gateway gateway.Status `json:"gateway"`
	}
	if err := json.NewDecoder(status.Body).Decode(&statusPayload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !statusPayload.OK || statusPayload.Gateway.ProfileID != "profile-1" {
		t.Fatalf("unexpected status payload: %+v", statusPayload)
	}

	switchRes := proxyGatewayAPIRequest(t, handler, http.MethodPost, "/api/proxy-gateway/switch", `{"selector":{"profileId":"profile-1"},"proxyId":"proxy-new","force":true}`)
	if switchRes.Code != http.StatusOK || !fake.force {
		t.Fatalf("switch status = %d force=%v: %s", switchRes.Code, fake.force, switchRes.Body.String())
	}

	routingRes := proxyGatewayAPIRequest(t, handler, http.MethodPut, "/api/proxy-gateway/routing", `{"profileId":"profile-1","routing":{"mode":"rule","rules":[{"id":"r1","matchType":"domain","pattern":"example.com","action":"direct"}]},"force":true}`)
	if routingRes.Code != http.StatusOK || fake.routing.Mode != gateway.ModeRule || !fake.force {
		t.Fatalf("routing status = %d routing=%+v force=%v: %s", routingRes.Code, fake.routing, fake.force, routingRes.Body.String())
	}

	getRouting := proxyGatewayAPIRequest(t, handler, http.MethodGet, "/api/proxy-gateway/routing?profileId=profile-1", "")
	if getRouting.Code != http.StatusOK {
		t.Fatalf("get routing status = %d: %s", getRouting.Code, getRouting.Body.String())
	}

	invalid := proxyGatewayAPIRequest(t, handler, http.MethodPut, "/api/proxy-gateway/routing", `{"profileId":"profile-1","routing":{"mode":"rule","rules":[{"matchType":"ip_cidr","pattern":"bad","action":"direct"}]}}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid routing status = %d, want %d: %s", invalid.Code, http.StatusBadRequest, invalid.Body.String())
	}
}

func TestProxyGatewayHTTPAPIRequiresController(t *testing.T) {
	server := NewLaunchServer(nil, nil, nil, 0)
	res := proxyGatewayAPIRequest(t, NewTestHandler(server), http.MethodGet, "/api/proxy-gateway/status?profileId=profile-1", "")
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status without controller = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}
