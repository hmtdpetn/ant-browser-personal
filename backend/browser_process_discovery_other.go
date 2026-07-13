//go:build !windows
// +build !windows

package backend

import (
	"fmt"
	"os"
	"time"
)

func findBrowserUserDataProcessesOS(userDataDir string) ([]browserUserDataProcess, error) {
	return nil, nil
}

func terminateBrowserUserDataProcessOS(pid int, timeout time.Duration) error {
	if pid <= 0 {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := process.Kill(); err != nil {
		return err
	}
	return fmt.Errorf("process termination fallback does not wait for pid %d", pid)
}
