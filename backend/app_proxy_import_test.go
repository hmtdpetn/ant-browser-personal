package backend

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testClashSubscriptionYAML = `
proxies:
  - name: test-node
    type: http
    server: example.com
    port: 8080
`

func TestBrowserProxyFetchClashByURLFallbackAfterHTTPStatus(t *testing.T) {
	var seenUserAgents []string
	var seenAccept string
	var seenCacheControl string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenUserAgents = append(seenUserAgents, r.Header.Get("User-Agent"))
		if len(seenUserAgents) == 1 {
			seenAccept = r.Header.Get("Accept")
			seenCacheControl = r.Header.Get("Cache-Control")
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		fmt.Fprint(w, testClashSubscriptionYAML)
	}))
	defer server.Close()

	result, err := (&App{}).BrowserProxyFetchClashByURL(server.URL + "/sub?token=test-token")
	if err != nil {
		t.Fatalf("BrowserProxyFetchClashByURL returned error: %v", err)
	}
	if got := result["proxyCount"]; got != 1 {
		t.Fatalf("proxyCount = %v, want 1", got)
	}
	if len(seenUserAgents) != 2 {
		t.Fatalf("request count = %d, want 2", len(seenUserAgents))
	}
	if seenUserAgents[0] != clashSubscriptionUserAgents[0] {
		t.Fatalf("first User-Agent = %q, want %q", seenUserAgents[0], clashSubscriptionUserAgents[0])
	}
	if seenUserAgents[1] != clashSubscriptionUserAgents[1] {
		t.Fatalf("second User-Agent = %q, want %q", seenUserAgents[1], clashSubscriptionUserAgents[1])
	}
	if seenAccept != "application/yaml,text/yaml,text/plain,*/*" {
		t.Fatalf("Accept = %q", seenAccept)
	}
	if seenCacheControl != "no-cache" {
		t.Fatalf("Cache-Control = %q", seenCacheControl)
	}
}

func TestBrowserProxyFetchClashByURLFallbackAfterHTMLContent(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body>client not supported</body></html>")
			return
		}
		fmt.Fprint(w, testClashSubscriptionYAML)
	}))
	defer server.Close()

	result, err := (&App{}).BrowserProxyFetchClashByURL(server.URL)
	if err != nil {
		t.Fatalf("BrowserProxyFetchClashByURL returned error: %v", err)
	}
	if got := result["proxyCount"]; got != 1 {
		t.Fatalf("proxyCount = %v, want 1", got)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
}

func TestBrowserProxyFetchClashByURLAllFallbackErrorsHideURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	rawURL := server.URL + "/sub/path?token=secret-token"
	_, err := (&App{}).BrowserProxyFetchClashByURL(rawURL)
	if err == nil {
		t.Fatal("BrowserProxyFetchClashByURL returned nil error, want failure")
	}
	errText := err.Error()
	for _, forbidden := range []string{rawURL, "secret-token", "token=", "/sub/path"} {
		if strings.Contains(errText, forbidden) {
			t.Fatalf("error %q leaked %q", errText, forbidden)
		}
	}
}

func TestDirectAnyTLSURIStaysDisabledForPersonalClashFlow(t *testing.T) {
	if _, ok := proxyURIToClashNode("anytls://password@example.com:443?sni=example.com", 0); ok {
		t.Fatal("direct anytls URI must not bypass the personal Clash YAML parser")
	}
}

func TestBrowserProxyFetchClashByURLWithOptionsUsesSelectedUserAgentOnly(t *testing.T) {
	const selectedUserAgent = "MySubscriptionClient/1.0"
	var seenUserAgents []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenUserAgents = append(seenUserAgents, r.Header.Get("User-Agent"))
		fmt.Fprint(w, testClashSubscriptionYAML)
	}))
	defer server.Close()

	result, err := (&App{}).BrowserProxyFetchClashByURLWithOptions(server.URL, ClashSubscriptionFetchOptions{
		UserAgent:       selectedUserAgent,
		FallbackEnabled: false,
	})
	if err != nil {
		t.Fatalf("BrowserProxyFetchClashByURLWithOptions returned error: %v", err)
	}
	if len(seenUserAgents) != 1 || seenUserAgents[0] != selectedUserAgent {
		t.Fatalf("seen User-Agents = %#v, want only %q", seenUserAgents, selectedUserAgent)
	}
	if got := result["usedUserAgent"]; got != selectedUserAgent {
		t.Fatalf("usedUserAgent = %v, want %q", got, selectedUserAgent)
	}
	if got := result["fallbackUsed"]; got != false {
		t.Fatalf("fallbackUsed = %v, want false", got)
	}
}

func TestBrowserProxyFetchClashByURLWithOptionsFallsBackAfterSelectedUserAgent(t *testing.T) {
	const selectedUserAgent = "MySubscriptionClient/2.0"
	var seenUserAgents []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenUserAgents = append(seenUserAgents, r.Header.Get("User-Agent"))
		if len(seenUserAgents) == 1 {
			http.Error(w, "unsupported client", http.StatusForbidden)
			return
		}
		fmt.Fprint(w, testClashSubscriptionYAML)
	}))
	defer server.Close()

	result, err := (&App{}).BrowserProxyFetchClashByURLWithOptions(server.URL, ClashSubscriptionFetchOptions{
		UserAgent:       selectedUserAgent,
		FallbackEnabled: true,
	})
	if err != nil {
		t.Fatalf("BrowserProxyFetchClashByURLWithOptions returned error: %v", err)
	}
	if len(seenUserAgents) != 2 {
		t.Fatalf("request count = %d, want 2", len(seenUserAgents))
	}
	if seenUserAgents[0] != selectedUserAgent || seenUserAgents[1] != clashSubscriptionUserAgents[0] {
		t.Fatalf("seen User-Agents = %#v", seenUserAgents)
	}
	if got := result["fallbackUsed"]; got != true {
		t.Fatalf("fallbackUsed = %v, want true", got)
	}
}

func TestBrowserProxyFetchClashByURLWithOptionsRejectsUnsafeUserAgent(t *testing.T) {
	for _, userAgent := range []string{
		"safe-prefix\r\nX-Injected: yes",
		strings.Repeat("x", maxClashSubscriptionUserAgentBytes+1),
	} {
		_, err := (&App{}).BrowserProxyFetchClashByURLWithOptions("https://example.com/sub", ClashSubscriptionFetchOptions{
			UserAgent:       userAgent,
			FallbackEnabled: false,
		})
		if err == nil {
			t.Fatalf("unsafe User-Agent %q was accepted", userAgent)
		}
	}
}

func TestBrowserProxySubscriptionUserAgentsIncludesFlClashPresets(t *testing.T) {
	options := (&App{}).BrowserProxySubscriptionUserAgents()
	var foundClashVerge bool
	var foundClashForWindows bool
	for _, option := range options {
		if option.UserAgent == "clash-verge/v2.4.2" && strings.Contains(option.Source, "FlClash") {
			foundClashVerge = true
		}
		if option.UserAgent == "ClashforWindows/0.19.23" && strings.Contains(option.Source, "FlClash") {
			foundClashForWindows = true
		}
	}
	if !foundClashVerge || !foundClashForWindows {
		t.Fatalf("FlClash presets missing: %#v", options)
	}
}
