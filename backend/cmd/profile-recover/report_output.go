package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func writeReport(report *recoveryReport, now time.Time) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report is nil")
	}

	reportDir := filepath.Join(report.UserDataRoot, "recovery-reports")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", err
	}
	reportPath := filepath.Join(reportDir, fmt.Sprintf("profile-recover-%s.json", now.Format("20060102-150405")))
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		return "", err
	}
	return reportPath, nil
}

func printSummary(report *recoveryReport) {
	fmt.Printf("AppRoot: %s\n", report.AppRoot)
	fmt.Printf("DBPath: %s\n", report.DBPath)
	fmt.Printf("UserDataRoot: %s\n", report.UserDataRoot)
	mode := "preview"
	if report.Apply {
		mode = "apply"
	}
	fmt.Printf("Mode: %s\n", mode)
	if report.SelectedCore.CoreID != "" || report.SelectedCore.CoreName != "" {
		fmt.Printf("SelectedCore: %s (%s)\n", report.SelectedCore.CoreName, report.SelectedCore.CoreID)
	}
	if report.BackupDir != "" {
		fmt.Printf("BackupDir: %s\n", report.BackupDir)
	}
	if report.ReportPath != "" {
		fmt.Printf("Report: %s\n", report.ReportPath)
	}
	fmt.Printf("Scanned=%d Candidates=%d Existing=%d Restored=%d RepairCopies=%d Skipped=%d Warnings=%d\n",
		report.Summary.Scanned,
		report.Summary.Candidates,
		report.Summary.Existing,
		report.Summary.Restored,
		report.Summary.RepairCopies,
		report.Summary.Skipped,
		report.Summary.Warnings,
	)
	for _, entry := range report.Entries {
		fmt.Printf("- [%s] %s", entry.Action, entry.DirName)
		if entry.RestoredProfileName != "" {
			fmt.Printf(" -> %s", entry.RestoredProfileName)
		}
		if entry.ExistingProfileName != "" {
			fmt.Printf(" -> %s", entry.ExistingProfileName)
		}
		if entry.Reason != "" {
			fmt.Printf(" (%s)", entry.Reason)
		}
		fmt.Println()
	}
	if len(report.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}
}
