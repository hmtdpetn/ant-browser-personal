//go:build !windows
// +build !windows

package automation

import (
	"os/exec"
	"syscall"
)

func hideWindow(cmd *exec.Cmd) {
}

func prepareTaskCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
