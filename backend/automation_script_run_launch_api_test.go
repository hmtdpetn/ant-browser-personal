package backend

import (
	"reflect"
	"testing"
)

func TestParseDualInstanceRuntimeParamsBackfillsDefaultStartURLs(t *testing.T) {
	browsers, timeoutMs, err := parseDualInstanceRuntimeParams(`{"browsers":[{"code":"buyer_001"},{"code":"buyer_002"}]}`)
	if err != nil {
		t.Fatalf("parseDualInstanceRuntimeParams returned error: %v", err)
	}
	if timeoutMs != dualInstanceRuntimeDefaultTimeoutMs {
		t.Fatalf("unexpected timeoutMs: got %d want %d", timeoutMs, dualInstanceRuntimeDefaultTimeoutMs)
	}
	if len(browsers) != 2 {
		t.Fatalf("unexpected browser count: got %d want 2", len(browsers))
	}

	if !reflect.DeepEqual(browsers[0].StartURLs, []string{"https://finance.sina.com.cn/"}) {
		t.Fatalf("unexpected browser[0] startUrls: %+v", browsers[0].StartURLs)
	}
	if !reflect.DeepEqual(browsers[1].StartURLs, []string{"https://map.baidu.com/"}) {
		t.Fatalf("unexpected browser[1] startUrls: %+v", browsers[1].StartURLs)
	}
}

func TestParseDualInstanceRuntimeParamsKeepsProvidedStartURLs(t *testing.T) {
	browsers, _, err := parseDualInstanceRuntimeParams(`{"browsers":[{"code":"buyer_001","startUrls":["https://example.com"]}]}`)
	if err != nil {
		t.Fatalf("parseDualInstanceRuntimeParams returned error: %v", err)
	}
	if len(browsers) != 1 {
		t.Fatalf("unexpected browser count: got %d want 1", len(browsers))
	}
	if !reflect.DeepEqual(browsers[0].StartURLs, []string{"https://example.com"}) {
		t.Fatalf("unexpected startUrls: %+v", browsers[0].StartURLs)
	}
}

func TestParseDualInstanceRuntimeParamsUsesDefaultStartURLsForFallbackCodes(t *testing.T) {
	browsers, _, err := parseDualInstanceRuntimeParams(`{}`)
	if err != nil {
		t.Fatalf("parseDualInstanceRuntimeParams returned error: %v", err)
	}
	if len(browsers) != 2 {
		t.Fatalf("unexpected browser count: got %d want 2", len(browsers))
	}

	if browsers[0].Code != "BUYER_001" || browsers[1].Code != "BUYER_002" {
		t.Fatalf("unexpected fallback codes: %+v", browsers)
	}
	if !reflect.DeepEqual(browsers[0].StartURLs, []string{"https://finance.sina.com.cn/"}) {
		t.Fatalf("unexpected browser[0] startUrls: %+v", browsers[0].StartURLs)
	}
	if !reflect.DeepEqual(browsers[1].StartURLs, []string{"https://map.baidu.com/"}) {
		t.Fatalf("unexpected browser[1] startUrls: %+v", browsers[1].StartURLs)
	}
}
