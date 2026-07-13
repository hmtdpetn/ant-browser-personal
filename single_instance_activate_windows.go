//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	swShow    = 5
	swRestore = 9
)

var (
	user32Activate               = windows.NewLazySystemDLL("user32.dll")
	procAllowSetForegroundWindow = user32Activate.NewProc("AllowSetForegroundWindow")
	procAttachThreadInput        = user32Activate.NewProc("AttachThreadInput")
	procEnumWindows              = user32Activate.NewProc("EnumWindows")
	procGetForegroundWindow      = user32Activate.NewProc("GetForegroundWindow")
	procGetWindow                = user32Activate.NewProc("GetWindow")
	procGetWindowTextLengthW     = user32Activate.NewProc("GetWindowTextLengthW")
	procGetWindowThreadProcessID = user32Activate.NewProc("GetWindowThreadProcessId")
	procIsIconic                 = user32Activate.NewProc("IsIconic")
	procIsWindowVisible          = user32Activate.NewProc("IsWindowVisible")
	procShowWindow               = user32Activate.NewProc("ShowWindow")
	procSetForegroundWindow      = user32Activate.NewProc("SetForegroundWindow")
	procSetWindowPos             = user32Activate.NewProc("SetWindowPos")
)

const (
	gwOwner       = 4
	hwndTopmost   = ^uintptr(0)
	hwndNotopmost = ^uintptr(1)
	swpNoMove     = 0x0002
	swpNoSize     = 0x0001
	swpShowWindow = 0x0040
)

func grantExistingSingleInstanceForeground(pid int) {
	if pid > 0 {
		procAllowSetForegroundWindow.Call(uintptr(pid))
	}
}

func activateExistingSingleInstanceWindow(pid int) {
	if pid <= 0 {
		return
	}
	grantExistingSingleInstanceForeground(pid)
	if hwnd := findTopLevelWindowByPID(uint32(pid)); hwnd != 0 {
		procShowWindow.Call(hwnd, swRestore)
		procShowWindow.Call(hwnd, swShow)
		if ok, _, _ := procSetForegroundWindow.Call(hwnd); ok == 0 {
			attachForegroundThreadInput(hwnd, func() {
				procSetForegroundWindow.Call(hwnd)
			})
		}
		if ok, _, _ := procSetForegroundWindow.Call(hwnd); ok == 0 {
			procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
			procSetWindowPos.Call(hwnd, hwndNotopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpShowWindow)
			procSetForegroundWindow.Call(hwnd)
		}
	}
}

func attachForegroundThreadInput(hwnd uintptr, fn func()) {
	foreground, _, _ := procGetForegroundWindow.Call()
	if foreground == 0 || hwnd == 0 {
		fn()
		return
	}

	var foregroundPID uint32
	foregroundThread, _, _ := procGetWindowThreadProcessID.Call(foreground, uintptr(unsafe.Pointer(&foregroundPID)))
	var targetPID uint32
	targetThread, _, _ := procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&targetPID)))
	if foregroundThread == 0 || targetThread == 0 || foregroundThread == targetThread {
		fn()
		return
	}

	procAttachThreadInput.Call(foregroundThread, targetThread, 1)
	defer procAttachThreadInput.Call(foregroundThread, targetThread, 0)
	fn()
}

func findTopLevelWindowByPID(pid uint32) uintptr {
	type candidateWindow struct {
		hwnd      uintptr
		visible   bool
		titled    bool
		owned     bool
		minimized bool
	}
	var candidates []candidateWindow
	callback := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var windowPID uint32
		procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID == pid {
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			titleLength, _, _ := procGetWindowTextLengthW.Call(hwnd)
			owner, _, _ := procGetWindow.Call(hwnd, gwOwner)
			minimized, _, _ := procIsIconic.Call(hwnd)
			candidates = append(candidates, candidateWindow{
				hwnd:      hwnd,
				visible:   visible != 0,
				titled:    titleLength > 0,
				owned:     owner != 0,
				minimized: minimized != 0,
			})
		}
		return 1
	})
	procEnumWindows.Call(callback, 0)
	if len(candidates) == 0 {
		return 0
	}
	for _, candidate := range candidates {
		if candidate.visible && candidate.titled && !candidate.owned {
			return candidate.hwnd
		}
	}
	for _, candidate := range candidates {
		if candidate.titled && !candidate.owned {
			return candidate.hwnd
		}
	}
	for _, candidate := range candidates {
		if candidate.visible && !candidate.owned {
			return candidate.hwnd
		}
	}
	for _, candidate := range candidates {
		if candidate.minimized && !candidate.owned {
			return candidate.hwnd
		}
	}
	return candidates[0].hwnd
}
