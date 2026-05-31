//go:build !windows
// +build !windows

package automation

import "syscall"

func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
