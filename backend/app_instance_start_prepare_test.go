package backend

import (
	"ant-chrome/backend/internal/browser"
	"testing"
)

func TestRunningProfileFingerprintExpectedArgsIgnoreExtraLaunchArgs(t *testing.T) {
	app := NewApp(t.TempDir())
	app.browserMgr = browser.NewManager(nil, app.appRoot)
	profile := &browser.Profile{
		ProfileId: "profile-running",
		Running:   true,
		LastLaunchArgs: []string{
			"--fingerprint=123",
			"--lang=zh-CN",
		},
	}

	expectedArgs := app.fingerprintCheckExpectedArgsForRunningProfile(profile, []string{"--timezone=Asia/Tokyo"})
	actual := buildBrowserFingerprintExpected(expectedArgs)
	if actual.Language != "zh-CN" {
		t.Fatalf("language = %q, want zh-CN", actual.Language)
	}
	if actual.Timezone != "" {
		t.Fatalf("timezone = %q, want no extra launch arg in running profile expected args", actual.Timezone)
	}
}
