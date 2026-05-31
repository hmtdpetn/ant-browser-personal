package backend

import (
	"bytes"
	"net/http"
	"regexp"
	"testing"
)

func TestAutomationDemoLaunchCodeFormat(t *testing.T) {
	code := automationDemoLaunchCode()
	if matched := regexp.MustCompile(`^DEMO_[A-Z0-9]{6}$`).MatchString(code); !matched {
		t.Fatalf("expected demo launch code to match DEMO_[A-Z0-9]{6}, got %q", code)
	}
}

func TestNewAutomationDemoPayloadUsesRequestedCode(t *testing.T) {
	app := &App{}
	payload := app.newAutomationDemoPayload(http.MethodPost, automationDemoProfilesPath, http.StatusCreated, map[string]interface{}{
		"ok":        true,
		"profileId": "profile-1",
	}, automationDemoResultOptions{
		RequestedCode: "DEMO_ABC123",
	})

	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", payload["ok"])
	}
	if payload["launchCode"] != "DEMO_ABC123" {
		t.Fatalf("expected launchCode to fall back to requested code, got %#v", payload["launchCode"])
	}
	if payload["profileId"] != "profile-1" {
		t.Fatalf("expected profileId to be propagated, got %#v", payload["profileId"])
	}
}

func TestBuildAutomationDemoCreateRequestUsesOptions(t *testing.T) {
	requestedCode, payload := buildAutomationDemoCreateRequest(automationDemoCreateOptions{
		ProfileName:          "我的演示实例",
		LaunchCode:           " demo_custom ",
		StartURL:             " https://example.com/order ",
		LaunchArgs:           []string{" --lang=en-US ", "", "--window-size=1280,800"},
		SkipDefaultStartURLs: true,
		AutoLaunch:           true,
	})

	if requestedCode != "DEMO_CUSTOM" {
		t.Fatalf("expected requested code to be normalized, got %q", requestedCode)
	}

	profile, ok := payload["profile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected profile payload, got %#v", payload["profile"])
	}
	if profile["profileName"] != "我的演示实例" {
		t.Fatalf("expected custom profile name, got %#v", profile["profileName"])
	}
	launchArgs, ok := profile["launchArgs"].([]string)
	if !ok {
		t.Fatalf("expected launchArgs to be []string, got %#v", profile["launchArgs"])
	}
	if len(launchArgs) != 2 || launchArgs[0] != "--lang=en-US" || launchArgs[1] != "--window-size=1280,800" {
		t.Fatalf("expected launchArgs to be normalized, got %#v", launchArgs)
	}

	start, ok := payload["start"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected start payload, got %#v", payload["start"])
	}
	startURLs, ok := start["startUrls"].([]string)
	if !ok || len(startURLs) != 1 || startURLs[0] != "https://example.com/order" {
		t.Fatalf("expected startUrls to be normalized, got %#v", start["startUrls"])
	}
	if start["skipDefaultStartUrls"] != true {
		t.Fatalf("expected skipDefaultStartUrls=true, got %#v", start["skipDefaultStartUrls"])
	}
}

func TestDecodeAutomationDemoBodyFallsBackToRawText(t *testing.T) {
	payload, err := decodeAutomationDemoBody(bytes.NewBufferString("plain-text-response"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if payload["rawBody"] != "plain-text-response" {
		t.Fatalf("expected rawBody fallback, got %#v", payload["rawBody"])
	}
}
