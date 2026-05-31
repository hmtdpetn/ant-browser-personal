package proxy

import (
	"ant-chrome/backend/internal/config"
	"os/exec"
	"sync"
)

// SingBoxBridge sing-box 桥接进程
type SingBoxBridge struct {
	NodeKey   string
	Port      int
	Cmd       *exec.Cmd
	Pid       int
	Running   bool
	Stopping  bool
	LastError string
}

// SingBoxManager sing-box 桥接管理器
type SingBoxManager struct {
	Config       *config.Config
	AppRoot      string // 应用根目录，所有相对路径基于此解析
	Bridges      map[string]*SingBoxBridge
	OnBridgeDied func(key string, err error)
	mu           sync.Mutex
}

// NewSingBoxManager 创建 sing-box 管理器
func NewSingBoxManager(cfg *config.Config, appRoot string) *SingBoxManager {
	return &SingBoxManager{
		Config:  cfg,
		AppRoot: appRoot,
		Bridges: make(map[string]*SingBoxBridge),
	}
}
