package backend

import (
	"fmt"
	"os/exec"
	stdruntime "runtime"
	"strings"
	"time"
)

func (a *App) stopBrowserProcess(cmd *exec.Cmd) error {
	return a.stopProcessCmd(cmd)
}

func (a *App) stopProcessCmd(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if stdruntime.GOOS == "windows" {
		pid := cmd.Process.Pid
		if pid > 0 {
			softKillCmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid), "/T")
			hideWindow(softKillCmd)
			if err := softKillCmd.Run(); err == nil {
				if waitProcessExitWindows(pid, 3*time.Second) {
					return nil
				}
				forceKillCmd := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", pid), "/T")
				hideWindow(forceKillCmd)
				if forceErr := forceKillCmd.Run(); forceErr == nil {
					_ = waitProcessExitWindows(pid, 2*time.Second)
					return nil
				}
			}
		}
	}

	err := cmd.Process.Kill()
	if err == nil || isProcessAlreadyFinished(err) {
		return nil
	}
	return err
}

func isProcessAlreadyFinished(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}
	if strings.Contains(message, "process already finished") {
		return true
	}
	if strings.Contains(message, "not found") {
		return true
	}
	if strings.Contains(message, "no process") {
		return true
	}
	if strings.Contains(message, "不存在") {
		return true
	}
	return false
}

func waitProcessExitWindows(pid int, timeout time.Duration) bool {
	if pid <= 0 {
		return true
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		alive, err := isProcessAliveWindows(pid)
		if err == nil && !alive {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	alive, err := isProcessAliveWindows(pid)
	if err != nil {
		return false
	}
	return !alive
}

func isProcessAliveWindows(pid int) (bool, error) {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return false, nil
	}
	if strings.HasPrefix(strings.ToUpper(line), "INFO:") {
		return false, nil
	}
	token := fmt.Sprintf("\",\"%d\",", pid)
	return strings.Contains(line, token), nil
}
