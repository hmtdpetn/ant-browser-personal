package backend

import (
	"testing"

	"ant-chrome/backend/internal/automation"
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/launchcode"
)

func newAutomationTargetTestApp(t *testing.T) *App {
	t.Helper()

	app := NewApp(t.TempDir())
	app.config = config.DefaultConfig()
	app.browserMgr = browser.NewManager(app.config, app.appRoot)
	app.launchCodeSvc = launchcode.NewLaunchCodeService(launchcode.NewMemoryLaunchCodeDAO())
	app.browserMgr.CodeProvider = app.launchCodeSvc
	return app
}

func createAutomationTargetProfile(t *testing.T, app *App, input browser.ProfileInput) *browser.Profile {
	t.Helper()

	profile, err := app.browserMgr.Create(input)
	if err != nil {
		t.Fatalf("create profile failed: %v", err)
	}
	if profile == nil {
		t.Fatal("create profile returned nil")
	}
	return profile
}

func TestResolveAutomationScriptTargetUsesExistingProfile(t *testing.T) {
	app := newAutomationTargetTestApp(t)
	first := createAutomationTargetProfile(t, app, browser.ProfileInput{
		ProfileName: "buyer-001",
		Keywords:    []string{"buyer-001"},
	})
	_, err := app.launchCodeSvc.SetCode(first.ProfileId, "BUYER_001")
	if err != nil {
		t.Fatalf("set code failed: %v", err)
	}

	selector, summary, err := app.resolveAutomationScriptTarget(automation.ScriptRecord{
		ID:   "script-existing",
		Name: "使用已有实例",
		TargetConfig: automation.ScriptTargetConfig{
			Mode: "existing",
			Selector: automation.ScriptTargetSelector{
				Code: "buyer_001",
			},
		},
	})
	if err != nil {
		t.Fatalf("resolveAutomationScriptTarget returned error: %v", err)
	}
	if selector["profileId"] != first.ProfileId {
		t.Fatalf("unexpected selector: %+v want profileId=%s", selector, first.ProfileId)
	}
	if summary == "" {
		t.Fatalf("expected target summary to be populated")
	}
}

func TestResolveAutomationScriptTargetPrefersProfileIDWhenStoredCodeIsStale(t *testing.T) {
	app := newAutomationTargetTestApp(t)
	first := createAutomationTargetProfile(t, app, browser.ProfileInput{
		ProfileName: "buyer-001",
	})
	if _, err := app.launchCodeSvc.SetCode(first.ProfileId, "BUYER_001"); err != nil {
		t.Fatalf("set initial code failed: %v", err)
	}
	if _, err := app.launchCodeSvc.SetCode(first.ProfileId, "BUYER_RENAMED"); err != nil {
		t.Fatalf("set updated code failed: %v", err)
	}

	selector, summary, err := app.resolveAutomationScriptTarget(automation.ScriptRecord{
		ID:   "script-existing",
		Name: "使用已有实例",
		TargetConfig: automation.ScriptTargetConfig{
			Mode: "existing",
			Selector: automation.ScriptTargetSelector{
				ProfileID: first.ProfileId,
				Code:      "BUYER_001",
			},
		},
	})
	if err != nil {
		t.Fatalf("resolveAutomationScriptTarget returned error: %v", err)
	}
	if selector["profileId"] != first.ProfileId {
		t.Fatalf("unexpected selector: %+v want profileId=%s", selector, first.ProfileId)
	}
	updatedProfiles := app.browserMgr.List()
	if len(updatedProfiles) == 0 {
		t.Fatalf("expected profiles to be available after resolve")
	}
	expectedSummary := ""
	for _, item := range updatedProfiles {
		if item.ProfileId == first.ProfileId {
			expectedSummary = automationProfileLabel(item)
			break
		}
	}
	if expectedSummary == "" {
		t.Fatalf("expected updated profile summary to be available")
	}
	if summary != expectedSummary {
		t.Fatalf("expected updated target summary %q, got %q", expectedSummary, summary)
	}
}

func TestResolveAutomationScriptTargetCreatesProfileFromTemplate(t *testing.T) {
	app := newAutomationTargetTestApp(t)
	template := createAutomationTargetProfile(t, app, browser.ProfileInput{
		ProfileName: "template-buyer",
		Tags:        []string{"template"},
	})
	_, err := app.launchCodeSvc.SetCode(template.ProfileId, "TPL_001")
	if err != nil {
		t.Fatalf("set code failed: %v", err)
	}

	before := app.browserMgr.List()
	selector, summary, err := app.resolveAutomationScriptTarget(automation.ScriptRecord{
		ID:   "script-create",
		Name: "按模板新建",
		TargetConfig: automation.ScriptTargetConfig{
			Mode: "create",
			TemplateSelector: automation.ScriptTargetSelector{
				Code: "TPL_001",
			},
			CreateNameTemplate: "${templateName}-${scriptName}",
		},
	})
	if err != nil {
		t.Fatalf("resolveAutomationScriptTarget returned error: %v", err)
	}

	after := app.browserMgr.List()
	if len(after) != len(before)+1 {
		t.Fatalf("expected profile count to grow by one: before=%d after=%d", len(before), len(after))
	}

	newProfileID, _ := selector["profileId"].(string)
	if newProfileID == "" || newProfileID == template.ProfileId {
		t.Fatalf("unexpected created selector: %+v", selector)
	}

	var created *browser.Profile
	for i := range after {
		if after[i].ProfileId == newProfileID {
			created = &after[i]
			break
		}
	}
	if created == nil {
		t.Fatalf("created profile not found in list")
	}
	if created.ProfileName != "template-buyer-按模板新建" {
		t.Fatalf("unexpected created profile name: %q", created.ProfileName)
	}
	if summary == "" {
		t.Fatalf("expected create summary to be populated")
	}
}

func TestResolveAutomationScriptTargetRotatesProfiles(t *testing.T) {
	app := newAutomationTargetTestApp(t)
	first := createAutomationTargetProfile(t, app, browser.ProfileInput{
		ProfileName: "buyer-a",
		Tags:        []string{"pool"},
	})
	second := createAutomationTargetProfile(t, app, browser.ProfileInput{
		ProfileName: "buyer-b",
		Tags:        []string{"pool"},
	})

	script := automation.ScriptRecord{
		ID:   "script-rotate",
		Name: "轮询实例",
		TargetConfig: automation.ScriptTargetConfig{
			Mode: "rotate",
			Selector: automation.ScriptTargetSelector{
				Tags: []string{"pool"},
			},
		},
	}

	firstSelector, _, err := app.resolveAutomationScriptTarget(script)
	if err != nil {
		t.Fatalf("first resolve returned error: %v", err)
	}
	secondSelector, _, err := app.resolveAutomationScriptTarget(script)
	if err != nil {
		t.Fatalf("second resolve returned error: %v", err)
	}
	thirdSelector, _, err := app.resolveAutomationScriptTarget(script)
	if err != nil {
		t.Fatalf("third resolve returned error: %v", err)
	}

	if firstSelector["profileId"] != first.ProfileId {
		t.Fatalf("expected first rotation profile %s, got %+v", first.ProfileId, firstSelector)
	}
	if secondSelector["profileId"] != second.ProfileId {
		t.Fatalf("expected second rotation profile %s, got %+v", second.ProfileId, secondSelector)
	}
	if thirdSelector["profileId"] != first.ProfileId {
		t.Fatalf("expected third rotation profile %s, got %+v", first.ProfileId, thirdSelector)
	}
}
