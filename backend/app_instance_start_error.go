package backend

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type browserStartupExitError struct {
	exitErr    error
	stderrTail string
}

func (e *browserStartupExitError) Error() string {
	detail := e.Detail()
	if detail == "" && e.exitErr != nil {
		detail = strings.TrimSpace(e.exitErr.Error())
	}
	if detail == "" {
		return "browser process exited before ready"
	}
	return fmt.Sprintf("browser process exited before ready: %s", detail)
}

func (e *browserStartupExitError) Detail() string {
	lines := strings.Split(strings.TrimSpace(e.stderrTail), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return ""
}

func newBrowserStartupExitError(result browserProcessExitResult) error {
	return &browserStartupExitError{
		exitErr:    result.Err,
		stderrTail: result.StderrTail,
	}
}

func describeChromeProcessStartError(chromeBinaryPath string, err error) string {
	raw := strings.TrimSpace(err.Error())
	lower := strings.ToLower(raw)

	switch {
	case strings.Contains(lower, "access is denied"),
		strings.Contains(lower, "permission denied"),
		strings.Contains(raw, "拒绝访问"):
		return fmt.Sprintf("实例启动失败：系统拒绝启动浏览器进程。可执行文件：%s。请检查文件权限、杀毒软件拦截，或尝试以管理员身份运行。", chromeBinaryPath)
	case strings.Contains(lower, "not a valid win32 application"),
		strings.Contains(raw, "不是有效的 win32 应用程序"),
		strings.Contains(raw, "不是有效的 Win32 应用程序"),
		strings.Contains(raw, "bad exe format"),
		strings.Contains(lower, "exec format error"),
		strings.Contains(lower, "cannot execute binary file"):
		return fmt.Sprintf("实例启动失败：当前浏览器内核与系统/架构不兼容。可执行文件：%s。请确认 Linux 环境使用的是对应架构的 Chrome 内核，而不是 Windows 可执行文件。", chromeBinaryPath)
	case strings.Contains(raw, "系统找不到指定的文件"),
		strings.Contains(lower, "file not found"),
		strings.Contains(lower, "no such file"),
		strings.Contains(lower, "cannot find the file"):
		return fmt.Sprintf("实例启动失败：浏览器可执行文件不存在。可执行文件：%s。请检查内核路径是否正确，或重新下载内核。", chromeBinaryPath)
	case strings.Contains(raw, "目录名称无效"),
		strings.Contains(lower, "directory name is invalid"):
		return fmt.Sprintf("实例启动失败：浏览器工作目录无效。当前目录：%s。请检查内核路径配置是否正确。", chromeBinaryPath)
	default:
		return fmt.Sprintf("实例启动失败：浏览器进程拉起失败。可执行文件：%s。原因：%s。请检查内核文件是否完整、启动参数是否正确，或是否被安全软件拦截。", chromeBinaryPath, raw)
	}
}

func describeBrowserReadyTimeout(debugPort int, timeout time.Duration) string {
	if debugPort <= 0 {
		return fmt.Sprintf("实例启动失败：浏览器进程已拉起，但在 %s 内未完成就绪，且未获取到调试端口。请检查内核文件是否完整、启动参数是否正确，或是否被安全软件拦截。", timeout.Round(time.Second))
	}
	return fmt.Sprintf("实例启动失败：浏览器进程已拉起，但在 %s 内未完成就绪，调试端口 %d 未就绪。请检查内核文件是否完整、启动参数是否正确，或是否被安全软件拦截。", timeout.Round(time.Second), debugPort)
}

func describeBrowserReadyFailure(chromeBinaryPath string, debugPort int, timeout time.Duration, err error) string {
	var exitErr *browserStartupExitError
	if errors.As(err, &exitErr) {
		detail := exitErr.Detail()
		if detail == "" && exitErr.exitErr != nil {
			detail = strings.TrimSpace(exitErr.exitErr.Error())
		}
		if detail != "" {
			return fmt.Sprintf("实例启动失败：浏览器进程在完成就绪前退出。可执行文件：%s。原因：%s。请检查内核文件是否完整、启动参数是否正确，或是否被安全软件拦截。", chromeBinaryPath, detail)
		}
		return fmt.Sprintf("实例启动失败：浏览器进程在完成就绪前退出。可执行文件：%s。请检查内核文件是否完整、启动参数是否正确，或是否被安全软件拦截。", chromeBinaryPath)
	}
	return describeBrowserReadyTimeout(debugPort, timeout)
}
