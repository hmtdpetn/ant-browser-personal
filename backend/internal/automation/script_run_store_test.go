package automation

import (
	"path/filepath"
	"testing"
)

func TestScriptRunStoreSaveAndList(t *testing.T) {
	store := NewScriptRunStore(filepath.Join(t.TempDir(), "data", "automation", "runs"))

	first, err := store.Save(ScriptRunRecord{
		ID:         "run-1",
		ScriptID:   "script-1",
		ScriptName: "脚本 1",
		Status:     "success",
		Summary:    "ok",
		StartedAt:  "2026-04-02T09:00:00Z",
		FinishedAt: "2026-04-02T09:00:01Z",
		DurationMs: 1000,
	})
	if err != nil {
		t.Fatalf("Save first returned error: %v", err)
	}
	if first.ID != "run-1" {
		t.Fatalf("expected run id run-1, got %q", first.ID)
	}

	if _, err := store.Save(ScriptRunRecord{
		ID:         "run-2",
		ScriptID:   "script-2",
		ScriptName: "脚本 2",
		Status:     "failed",
		Summary:    "bad",
		StartedAt:  "2026-04-02T10:00:00Z",
		FinishedAt: "2026-04-02T10:00:02Z",
		DurationMs: 2000,
	}); err != nil {
		t.Fatalf("Save second returned error: %v", err)
	}

	items, err := store.List(10)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two runs, got %d", len(items))
	}
	if items[0].ID != "run-2" {
		t.Fatalf("expected latest run first, got %q", items[0].ID)
	}
}
