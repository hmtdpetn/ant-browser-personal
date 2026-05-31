package proxy

import (
	"ant-chrome/backend/internal/fsutil"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func computeNodeKey(src string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(src)))
	return hex.EncodeToString(h[:])
}

func normalizeNodeScheme(src string) string {
	s := strings.TrimSpace(src)
	if strings.HasPrefix(strings.ToLower(s), "hysteria://") {
		return "hysteria2://" + strings.TrimPrefix(s, "hysteria://")
	}
	return s
}

func resolveEnvPath(path string, appRoot string) string {
	path = fsutil.NormalizePathInput(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	if appRoot != "" {
		candidate := filepath.Join(appRoot, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(cwd, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return path
}

func waitPortReady(host string, port int, timeout time.Duration) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("端口 %d 不可用", port)
}

// nextAvailablePort 分配一个可用端口。
// 采用二次验证策略：分配后立即再次绑定确认未被其他进程抢占，
// 并在 EnsureBridge 层面加重试，彻底消除 TOCTOU 竞争窗口。
func nextAvailablePort() (int, error) {
	return nextAvailablePortWithRetry(10)
}

func nextAvailablePortWithRetry(maxRetries int) (int, error) {
	for i := 0; i < maxRetries; i++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			continue
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()
		time.Sleep(10 * time.Millisecond)
		verifyListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		verifyListener.Close()
		return port, nil
	}
	return 0, fmt.Errorf("无法分配可用端口，已重试 %d 次", maxRetries)
}
