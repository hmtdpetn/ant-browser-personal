package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ant-chrome/backend/internal/config"
)

func TestFetchIPHealthInfoReturnsSourceMetadataOnParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	data, err := FetchIPHealthInfo(
		"proxy-1",
		[]config.BrowserProxy{{ProxyId: "proxy-1", ProxyConfig: "direct://"}},
		nil,
		nil,
		&IPHealthConfig{
			URL:    server.URL,
			Source: "json",
			Parser: "json",
		},
	)
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if !strings.Contains(err.Error(), "source=json") {
		t.Fatalf("expected source in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "parser=json") {
		t.Fatalf("expected parser in error, got %v", err)
	}
	if got := mapString(data, "_source"); got != "json" {
		t.Fatalf("source metadata = %q, want json", got)
	}
	if got := mapString(data, "_targetUrl"); got != server.URL {
		t.Fatalf("target url metadata = %q, want %q", got, server.URL)
	}
	if got := mapString(data, "_parser"); got != "json" {
		t.Fatalf("parser metadata = %q, want json", got)
	}
	if got := mapString(data, "_bodySnippet"); got != "not-json" {
		t.Fatalf("body snippet = %q, want not-json", got)
	}
}
