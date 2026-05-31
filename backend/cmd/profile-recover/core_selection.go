package main

import (
	"ant-chrome/backend/internal/apppath"
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func normalizeRepairStrategy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "none":
		return "none"
	case "risky":
		return "risky"
	default:
		return ""
	}
}

func selectCore(appRoot string, cfg *config.Config, dbPath string, apply bool) (selectedCore, []string, error) {
	var warnings []string

	if info, err := os.Stat(dbPath); err == nil && !info.IsDir() {
		db, err := openQueryDB(dbPath)
		if err != nil {
			return selectedCore{}, warnings, fmt.Errorf("open database for core selection: %w", err)
		}
		defer db.Close()

		cores, err := browser.NewSQLiteCoreDAO(db).List()
		if err == nil && len(cores) > 0 {
			picked := pickCoreFromList(appRoot, cores, "database")
			return picked, warnings, nil
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("load cores from database failed, fallback to config: %v", err))
		}
	}

	if len(cfg.Browser.Cores) > 0 {
		picked := pickCoreFromConfig(appRoot, cfg.Browser.Cores, "config")
		return picked, warnings, nil
	}

	if apply {
		warnings = append(warnings, "no browser core was found; restored profiles will be created with empty core_id")
	}

	return selectedCore{
		CoreID:   "",
		CoreName: "",
		CorePath: "",
		Source:   "none",
	}, warnings, nil
}

func pickCoreFromList(appRoot string, cores []browser.Core, source string) selectedCore {
	for _, core := range cores {
		if core.IsDefault {
			return buildSelectedCore(appRoot, core.CoreId, core.CoreName, core.CorePath, source)
		}
	}
	first := cores[0]
	return buildSelectedCore(appRoot, first.CoreId, first.CoreName, first.CorePath, source)
}

func pickCoreFromConfig(appRoot string, cores []config.BrowserCore, source string) selectedCore {
	for _, core := range cores {
		if core.IsDefault {
			return buildSelectedCore(appRoot, core.CoreId, core.CoreName, core.CorePath, source)
		}
	}
	first := cores[0]
	return buildSelectedCore(appRoot, first.CoreId, first.CoreName, first.CorePath, source)
}

func buildSelectedCore(appRoot, coreID, coreName, corePath, source string) selectedCore {
	coreAbsPath := apppath.Resolve(appRoot, corePath)
	return selectedCore{
		CoreID:     strings.TrimSpace(coreID),
		CoreName:   strings.TrimSpace(coreName),
		CorePath:   strings.TrimSpace(corePath),
		BinaryPath: filepath.Join(coreAbsPath, "chrome.exe"),
		Source:     source,
	}
}
