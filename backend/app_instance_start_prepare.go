package backend

import (
	"ant-chrome/backend/internal/browser"
	"ant-chrome/backend/internal/logger"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type browserStartInput struct {
	ProfileID            string
	ExtraLaunchArgs      []string
	StartURLs            []string
	SkipDefaultStartURLs bool
	PreferVisibleWindow  bool
	ForceDirectProxy     bool
	TemporaryProxyID     string
	TemporaryProxyConfig string
}

type browserStartPlan struct {
	profile               *BrowserProfile
	chromeBinaryPath      string
	userDataDir           string
	args                  []string
	effectiveProxy        string
	acquiredXrayBridgeKey string
	releaseXrayBridge     bool
	assignedDebugPort     int
	startReadyTimeout     time.Duration
	startStableWindow     time.Duration
	maxStartAttempts      int
	totalReadyTimeout     time.Duration
}

func newBrowserStartInput(profileID string, extraLaunchArgs []string, startURLs []string, skipDefaultStartURLs bool, preferVisibleWindow bool, forceDirectProxy bool, proxyID string, proxyConfig string) browserStartInput {
	normalizedExtraLaunchArgs := normalizeNonEmptyStrings(extraLaunchArgs)
	if preferVisibleWindow {
		normalizedExtraLaunchArgs = ensureNewWindowLaunchArg(normalizedExtraLaunchArgs)
	}

	return browserStartInput{
		ProfileID:            profileID,
		ExtraLaunchArgs:      normalizedExtraLaunchArgs,
		StartURLs:            normalizeNonEmptyStrings(startURLs),
		SkipDefaultStartURLs: skipDefaultStartURLs,
		PreferVisibleWindow:  preferVisibleWindow,
		ForceDirectProxy:     forceDirectProxy,
		TemporaryProxyID:     strings.TrimSpace(proxyID),
		TemporaryProxyConfig: strings.TrimSpace(proxyConfig),
	}
}

func (input browserStartInput) hasTemporaryProxy() bool {
	return strings.TrimSpace(input.TemporaryProxyID) != "" || strings.TrimSpace(input.TemporaryProxyConfig) != ""
}

func (plan *browserStartPlan) releaseBridgeIfNeeded(a *App) {
	if plan == nil || a == nil {
		return
	}
	if plan.releaseXrayBridge && plan.acquiredXrayBridgeKey != "" && a.xrayMgr != nil {
		a.xrayMgr.ReleaseBridge(plan.acquiredXrayBridgeKey)
	}
}

func (a *App) resolveBrowserStartProfile(input browserStartInput) (*BrowserProfile, bool, error) {
	log := logger.New("Browser")

	profile, exists := a.browserMgr.Profiles[input.ProfileID]
	if !exists {
		err := fmt.Errorf("实例启动失败：未找到实例配置（ID=%s）。请刷新列表后重试。", input.ProfileID)
		log.Error("实例不存在", logger.F("profile_id", input.ProfileID), logger.F("reason", err.Error()))
		return nil, false, err
	}

	if !profile.Running {
		return profile, false, nil
	}

	if !isBrowserProfileLive(profile, a.browserMgr.BrowserProcesses[input.ProfileID]) {
		log.Info("检测到实例运行状态已失效，准备重新启动",
			logger.F("profile_id", input.ProfileID),
			logger.F("pid", profile.Pid),
			logger.F("debug_port", profile.DebugPort),
		)
		a.markProfileStoppedLocked(input.ProfileID, profile)
		return profile, false, nil
	}

	if input.PreferVisibleWindow {
		if err := a.openBrowserWindowForRunningProfile(profile, input.ExtraLaunchArgs, input.StartURLs); err != nil {
			startErr := fmt.Errorf("实例已在运行，但窗口唤起失败：%w", err)
			log.Error("运行中实例窗口唤起失败",
				logger.F("profile_id", input.ProfileID),
				logger.F("debug_port", profile.DebugPort),
				logger.F("error", err.Error()),
				logger.F("reason", startErr.Error()),
			)
			profile.LastError = startErr.Error()
			return profile, true, startErr
		}
	}

	if a.launchServer != nil && profile.DebugReady {
		a.launchServer.SetActiveProfile(profile)
	}
	a.emitBrowserInstanceStarted(profile, true)
	return profile, true, nil
}

func (a *App) prepareBrowserStartPlan(input browserStartInput, profile *BrowserProfile) (*browserStartPlan, error) {
	sanitizedProfileLaunchArgs, sanitizedExtraLaunchArgs, chromeBinaryPath, userDataDir, err := a.prepareBrowserLaunchContext(input, profile)
	if err != nil {
		return nil, err
	}

	effectiveProxy, acquiredXrayBridgeKey, releaseXrayBridge, err := a.resolveBrowserStartProxy(input, profile)
	if err != nil {
		return nil, err
	}

	startReadyTimeout, startStableWindow := a.browserStartTimingSettings()
	maxStartAttempts := browserStartAttemptCount()
	totalReadyTimeout := time.Duration(maxStartAttempts) * startReadyTimeout

	assignedDebugPort, err := nextAvailablePort()
	if err != nil {
		startErr := fmt.Errorf("实例启动失败：本地调试端口分配失败。原因：%v。请关闭占用端口的程序后重试。", err)
		logger.New("Browser").Error("调试端口分配失败",
			logger.F("profile_id", input.ProfileID),
			logger.F("error", err.Error()),
			logger.F("reason", startErr.Error()),
		)
		profile.LastError = startErr.Error()
		return nil, startErr
	}

	return &browserStartPlan{
		profile:               profile,
		chromeBinaryPath:      chromeBinaryPath,
		userDataDir:           userDataDir,
		args:                  buildBrowserLaunchArgs(profile, userDataDir, assignedDebugPort, effectiveProxy, sanitizedProfileLaunchArgs, sanitizedExtraLaunchArgs, input.StartURLs, a.browserDefaultStartURLs(), input.SkipDefaultStartURLs, browserRestoreLastSession(a.config)),
		effectiveProxy:        effectiveProxy,
		acquiredXrayBridgeKey: acquiredXrayBridgeKey,
		releaseXrayBridge:     releaseXrayBridge,
		assignedDebugPort:     assignedDebugPort,
		startReadyTimeout:     startReadyTimeout,
		startStableWindow:     startStableWindow,
		maxStartAttempts:      maxStartAttempts,
		totalReadyTimeout:     totalReadyTimeout,
	}, nil
}

func (a *App) prepareBrowserLaunchContext(input browserStartInput, profile *BrowserProfile) ([]string, []string, string, string, error) {
	log := logger.New("Browser")

	sanitizedProfileLaunchArgs, managedProfileArgs := sanitizeManagedLaunchArgs(profile.LaunchArgs)
	sanitizedExtraLaunchArgs, managedExtraArgs := sanitizeManagedLaunchArgs(input.ExtraLaunchArgs)
	logManagedLaunchArgOverrides(log, input.ProfileID, "profile.launchArgs", managedProfileArgs)
	logManagedLaunchArgOverrides(log, input.ProfileID, "start.extraLaunchArgs", managedExtraArgs)

	proxyChanged := a.browserMgr.ApplyDefaults(profile)
	if proxyChanged {
		_ = a.browserMgr.SaveProfiles()
	}

	chromeBinaryPath, err := a.browserMgr.ResolveChromeBinary(profile)
	if err != nil {
		startErr := fmt.Errorf("实例启动失败：%w", err)
		log.Error("内核路径解析失败",
			logger.F("profile_id", input.ProfileID),
			logger.F("error", err.Error()),
			logger.F("reason", startErr.Error()),
		)
		profile.LastError = startErr.Error()
		return nil, nil, "", "", startErr
	}

	userDataDir := a.browserMgr.ResolveUserDataDir(profile)
	if err := os.MkdirAll(userDataDir, 0o755); err != nil {
		startErr := fmt.Errorf("实例启动失败：无法创建用户数据目录 %s。原因：%w。请检查目录权限或路径配置。", userDataDir, err)
		log.Error("用户数据目录创建失败",
			logger.F("profile_id", input.ProfileID),
			logger.F("dir", userDataDir),
			logger.F("error", err.Error()),
			logger.F("reason", startErr.Error()),
		)
		profile.LastError = startErr.Error()
		return nil, nil, "", "", startErr
	}

	if err := browser.EnsureDefaultBookmarks(userDataDir, a.BookmarkList()); err != nil {
		log.Error("默认书签写入失败", logger.F("error", err.Error()))
	}

	if !browserRestoreLastSession(a.config) {
		if err := browser.ClearSessionRestoreData(userDataDir); err != nil {
			sessionDir := filepath.Join(userDataDir, "Default", "Sessions")
			startErr := fmt.Errorf("实例启动失败：无法清理上次会话缓存 %s。原因：%w。请关闭占用该目录的浏览器进程后重试。", sessionDir, err)
			log.Error("会话恢复缓存清理失败",
				logger.F("profile_id", input.ProfileID),
				logger.F("dir", sessionDir),
				logger.F("error", err.Error()),
				logger.F("reason", startErr.Error()),
			)
			profile.LastError = startErr.Error()
			return nil, nil, "", "", startErr
		}
	}

	return sanitizedProfileLaunchArgs, sanitizedExtraLaunchArgs, chromeBinaryPath, userDataDir, nil
}

func buildBrowserLaunchArgs(profile *BrowserProfile, userDataDir string, debugPort int, effectiveProxy string, sanitizedProfileLaunchArgs []string, sanitizedExtraLaunchArgs []string, startURLs []string, defaultStartURLs []string, skipDefaultStartURLs bool, restoreLastSession bool) []string {
	args := []string{
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		fmt.Sprintf("--remote-debugging-port=%d", debugPort),
		"--disable-session-crashed-bubble",
	}

	hasFingerprint := false
	for _, arg := range profile.FingerprintArgs {
		if strings.HasPrefix(arg, "--fingerprint=") {
			hasFingerprint = true
			break
		}
	}
	if !hasFingerprint {
		seed := 0
		for _, char := range profile.ProfileId {
			seed = (seed << 5) - seed + int(char)
		}
		if seed < 0 {
			seed = -seed
		}
		args = append(args, fmt.Sprintf("--fingerprint=%d", seed))
	}

	if effectiveProxy == "direct://" {
		args = append(args, "--proxy-server=direct://")
	} else if effectiveProxy != "" {
		args = append(args, fmt.Sprintf("--proxy-server=%s", effectiveProxy))
	}

	args = append(args, profile.FingerprintArgs...)
	args = append(args, sanitizedProfileLaunchArgs...)
	args = append(args, sanitizedExtraLaunchArgs...)
	return appendLaunchTargets(args, startURLs, defaultStartURLs, skipDefaultStartURLs, restoreLastSession)
}
