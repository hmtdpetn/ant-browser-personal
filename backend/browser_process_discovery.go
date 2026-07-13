package backend

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type browserUserDataProcess struct {
	PID         int    `json:"pid"`
	DebugPort   int    `json:"debugPort"`
	CommandLine string `json:"commandLine"`
}

type browserRuntimeDetection struct {
	PID        int
	DebugPort  int
	DebugReady bool
}

var errBrowserStartHandledByRecoveredRuntime = errors.New("browser start handled by recovered runtime")

var findBrowserUserDataProcesses = findBrowserUserDataProcessesOS
var terminateBrowserUserDataProcess = terminateBrowserUserDataProcessOS

var remoteDebuggingPortPattern = regexp.MustCompile(`(?i)--remote-debugging-port=(\d+)`)

func parseRemoteDebuggingPort(commandLine string) int {
	matches := remoteDebuggingPortPattern.FindStringSubmatch(commandLine)
	if len(matches) < 2 {
		return 0
	}
	port, err := strconv.Atoi(matches[1])
	if err != nil || port <= 0 {
		return 0
	}
	return port
}

func detectBrowserRuntimeByUserDataDir(userDataDir string) (browserRuntimeDetection, bool) {
	userDataDir = strings.TrimSpace(userDataDir)
	if userDataDir == "" {
		return browserRuntimeDetection{}, false
	}

	if detection, ok := detectBrowserRuntimeByActivePort(userDataDir); ok {
		return detection, true
	}

	processes, err := findBrowserUserDataProcesses(userDataDir)
	if err != nil || len(processes) == 0 {
		return browserRuntimeDetection{}, false
	}

	for _, process := range processes {
		debugPort := process.DebugPort
		if debugPort <= 0 {
			debugPort = parseRemoteDebuggingPort(process.CommandLine)
		}
		if debugPort > 0 {
			if err := probeBrowserDebugPort(debugPort, browserDebugProbeTimeout); err == nil {
				return browserRuntimeDetection{PID: process.PID, DebugPort: debugPort, DebugReady: true}, true
			}
		}
	}

	first := processes[0]
	return browserRuntimeDetection{PID: first.PID, DebugPort: first.DebugPort}, true
}

func detectBrowserRuntimeByActivePort(userDataDir string) (browserRuntimeDetection, bool) {
	userDataDir = strings.TrimSpace(userDataDir)
	if userDataDir == "" {
		return browserRuntimeDetection{}, false
	}

	if debugPort, err := readBrowserDebugPortFile(userDataDir); err == nil && debugPort > 0 {
		if err := probeBrowserDebugPort(debugPort, browserDebugProbeTimeout); err == nil {
			return browserRuntimeDetection{DebugPort: debugPort, DebugReady: true}, true
		}
	}

	return browserRuntimeDetection{}, false
}

func terminateBrowserProcessesByUserDataDir(userDataDir string, timeout time.Duration) (bool, error) {
	processes, err := findBrowserUserDataProcesses(userDataDir)
	if err != nil {
		return false, err
	}
	if len(processes) == 0 {
		return false, nil
	}

	var errs []error
	terminated := false
	for _, process := range processes {
		if process.PID <= 0 || process.PID == os.Getpid() {
			continue
		}
		terminated = true
		if err := terminateBrowserUserDataProcess(process.PID, timeout); err != nil {
			errs = append(errs, fmt.Errorf("pid %d: %w", process.PID, err))
		}
	}
	return terminated, errors.Join(errs...)
}
