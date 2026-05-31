//go:build windows
// +build windows

package automation

import (
	"os/exec"
	"syscall"
)

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func prepareTaskCommand(cmd *exec.Cmd) {
	hideWindow(cmd)
}
