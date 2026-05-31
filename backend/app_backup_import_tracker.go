package backend

import (
	"ant-chrome/backend/internal/backup"
	"strings"
)

type backupMergeStats struct {
	Imported  int
	Skipped   int
	Conflicts int
}

type backupImportIssue struct {
	ComponentID   string `json:"componentId"`
	ComponentName string `json:"componentName"`
	Error         string `json:"error"`
}

type backupImportTracker struct {
	componentEntries   map[string]backup.ManifestEntry
	componentUniverse  map[string]struct{}
	failedComponentIDs map[string]struct{}
	issues             []backupImportIssue
}

func newBackupImportTracker(componentEntries map[string]backup.ManifestEntry) *backupImportTracker {
	universe := make(map[string]struct{}, len(componentEntries))
	for id := range componentEntries {
		universe[id] = struct{}{}
	}

	return &backupImportTracker{
		componentEntries:   componentEntries,
		componentUniverse:  universe,
		failedComponentIDs: make(map[string]struct{}),
		issues:             make([]backupImportIssue, 0),
	}
}

func (t *backupImportTracker) RecordIssue(componentID, componentName string, err error) {
	if t == nil || err == nil {
		return
	}

	componentID = strings.TrimSpace(componentID)
	componentName = strings.TrimSpace(componentName)
	if componentID != "" {
		t.componentUniverse[componentID] = struct{}{}
		t.failedComponentIDs[componentID] = struct{}{}
		if componentName == "" {
			if entry, ok := t.componentEntries[componentID]; ok {
				componentName = backupResolveManifestComponentName(entry)
			}
		}
	}
	if componentName == "" {
		componentName = "未知模块"
	}

	t.issues = append(t.issues, backupImportIssue{
		ComponentID:   componentID,
		ComponentName: componentName,
		Error:         err.Error(),
	})
}

func (t *backupImportTracker) Summary() (totalComponents, successCount, failedCount int, partial bool) {
	if t == nil {
		return 0, 0, 0, false
	}

	totalComponents = len(t.componentUniverse)
	failedCount = len(t.failedComponentIDs)
	successCount = totalComponents - failedCount
	if successCount < 0 {
		successCount = 0
	}
	partial = failedCount > 0
	return totalComponents, successCount, failedCount, partial
}

func (t *backupImportTracker) FailedComponents() []map[string]string {
	if t == nil {
		return nil
	}

	failedComponents := make([]map[string]string, 0, len(t.issues))
	for _, item := range t.issues {
		failedComponents = append(failedComponents, map[string]string{
			"componentId":   item.ComponentID,
			"componentName": item.ComponentName,
			"error":         item.Error,
		})
	}
	return failedComponents
}
