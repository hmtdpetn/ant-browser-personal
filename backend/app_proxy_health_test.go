package backend

import (
	"errors"
	"testing"
)

func TestBuildProxyIPHealthResultPreservesErrorSourceMetadata(t *testing.T) {
	result := buildProxyIPHealthResult("proxy-1", map[string]interface{}{
		"_source":    "trace",
		"_targetUrl": "https://example.invalid/trace",
		"_parser":    "cloudflare_trace",
	}, errors.New("request failed"))

	if result.Source != "trace" {
		t.Fatalf("source = %q, want trace", result.Source)
	}
	if result.Error != "request failed" {
		t.Fatalf("error = %q, want request failed", result.Error)
	}
	rawError, _ := result.RawData["error"].(string)
	if rawError != "request failed" {
		t.Fatalf("raw error = %q, want request failed", rawError)
	}
	if got, _ := result.RawData["_targetUrl"].(string); got != "https://example.invalid/trace" {
		t.Fatalf("target url = %q, want trace url", got)
	}
}
