//go:build windows

package backend

import (
	"path/filepath"
	"time"
	"unsafe"

	"ant-chrome/backend/internal/logger"

	"golang.org/x/sys/windows"
)

const (
	browserBadgeRetryCount    = 20
	browserBadgeRetryInterval = 300 * time.Millisecond
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows      = user32.NewProc("EnumWindows")
	procGetWindowPID     = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible  = user32.NewProc("IsWindowVisible")
	procSendMessage      = user32.NewProc("SendMessageW")
	procLoadImage        = user32.NewProc("LoadImageW")
	procDestroyIcon      = user32.NewProc("DestroyIcon")
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procCreateToolhelp32 = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First   = kernel32.NewProc("Process32FirstW")
	procProcess32Next    = kernel32.NewProc("Process32NextW")
)

func (a *App) applyBrowserProfileWindowBadgeAsync(profileID string, userDataDir string, iconPath string) {
	if profileID == "" || iconPath == "" {
		return
	}
	go func() {
		log := logger.New("Browser")
		for attempt := 0; attempt < browserBadgeRetryCount; attempt++ {
			if a.applyBrowserProfileWindowBadge(profileID, iconPath) {
				return
			}
			time.Sleep(browserBadgeRetryInterval)
		}
		log.Warn("profile taskbar badge window not found", logger.F("profile_id", profileID), logger.F("icon", iconPath))
	}()
}

func (a *App) applyBrowserProfileWindowBadge(profileID string, iconPath string) bool {
	a.browserMgr.Mutex.Lock()
	cmd := a.browserMgr.BrowserProcesses[profileID]
	a.browserMgr.Mutex.Unlock()
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return false
	}

	targetPIDs := collectProcessTree(uint32(cmd.Process.Pid))
	if len(targetPIDs) == 0 {
		targetPIDs = map[uint32]struct{}{uint32(cmd.Process.Pid): {}}
	}
	return setBadgeIconForProcessWindows(targetPIDs, filepath.Clean(iconPath)) > 0
}

func setBadgeIconForProcessWindows(targetPIDs map[uint32]struct{}, iconPath string) int {
	iconPathPtr, err := windows.UTF16PtrFromString(iconPath)
	if err != nil {
		return 0
	}
	largeIcon := loadIconFromFile(iconPathPtr, 32, 32)
	smallIcon := loadIconFromFile(iconPathPtr, 16, 16)
	if largeIcon == 0 && smallIcon == 0 {
		return 0
	}

	type enumState struct {
		pids      map[uint32]struct{}
		largeIcon uintptr
		smallIcon uintptr
		count     int
	}
	state := &enumState{pids: targetPIDs, largeIcon: largeIcon, smallIcon: smallIcon}
	callback := windows.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		state := (*enumState)(unsafe.Pointer(lparam))
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		var pid uint32
		procGetWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if _, ok := state.pids[pid]; !ok {
			return 1
		}
		if state.largeIcon != 0 {
			procSendMessage.Call(hwnd, 0x0080, 1, state.largeIcon)
		}
		if state.smallIcon != 0 {
			procSendMessage.Call(hwnd, 0x0080, 0, state.smallIcon)
			procSendMessage.Call(hwnd, 0x0080, 2, state.smallIcon)
		}
		state.count++
		return 1
	})
	procEnumWindows.Call(callback, uintptr(unsafe.Pointer(state)))
	return state.count
}

func loadIconFromFile(path *uint16, width int32, height int32) uintptr {
	handle, _, _ := procLoadImage.Call(
		0,
		uintptr(unsafe.Pointer(path)),
		1,
		uintptr(width),
		uintptr(height),
		0x00000010,
	)
	return handle
}

func collectProcessTree(rootPID uint32) map[uint32]struct{} {
	out := map[uint32]struct{}{rootPID: {}}
	snapshot, _, _ := procCreateToolhelp32.Call(0x00000002, 0)
	if snapshot == 0 || snapshot == uintptr(windows.InvalidHandle) {
		return out
	}
	defer windows.CloseHandle(windows.Handle(snapshot))

	var entries []processEntry32
	var entry processEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if r, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry))); r == 0 {
		return out
	}
	for {
		entries = append(entries, entry)
		entry.Size = uint32(unsafe.Sizeof(entry))
		if r, _, _ := procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry))); r == 0 {
			break
		}
	}

	changed := true
	for changed {
		changed = false
		for _, entry := range entries {
			if _, ok := out[entry.ParentProcessID]; ok {
				if _, exists := out[entry.ProcessID]; !exists {
					out[entry.ProcessID] = struct{}{}
					changed = true
				}
			}
		}
	}
	return out
}

type processEntry32 struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [260]uint16
}
