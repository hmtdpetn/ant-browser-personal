package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ant-chrome/backend/internal/logger"
	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DownloadAndExtractCore 执行异步下载解压并在过程中发送事件
func (m *Manager) DownloadAndExtractCore(ctx context.Context, coreName string, targetUrl string, proxyConfig string) {
	log := logger.New("Browser")
	t := time.Now()

	sendEvent := func(phase string, progress int, msg string) {
		runtime.EventsEmit(ctx, "download:progress", DownloadProgress{
			Phase:    phase,
			Progress: progress,
			Message:  msg,
		})
	}

	sendEvent("downloading", 0, "开始解析地址并创建下载请求: "+targetUrl)

	// 1. 检查名称重复
	coreName = strings.TrimSpace(coreName)
	for _, c := range m.ListCores() {
		if strings.EqualFold(c.CoreName, coreName) || filepath.Base(c.CorePath) == coreName {
			sendEvent("error", 0, "名称已存在，请换一个名称")
			return
		}
	}

	// 确保外层 chrome/ 目录存在
	chromeDir := m.ResolveRelativePath("chrome")
	if err := os.MkdirAll(chromeDir, 0755); err != nil {
		sendEvent("error", 0, "创建 chrome 目录失败")
		return
	}

	targetDir := filepath.Join(chromeDir, coreName)
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		sendEvent("error", 0, "同名文件夹已存在: "+coreName)
		return
	}
	// 2. 准备 HttpClient（优先从 Windows 注册表读取真实系统代理，而非仅靠环境变量）
	transport := &http.Transport{}
	if proxyConfig == "__system__" {
		// http.ProxyFromEnvironment 只读环境变量，而 Clash 的全局代理写在 Windows 注册表里
		// 必须直接读取注册表才能拿到正确的代理地址
		if sysProxy, rErr := readSystemProxy(); rErr == nil && sysProxy != "" {
			if proxyURL, pErr := url.Parse(sysProxy); pErr == nil {
				transport.Proxy = http.ProxyURL(proxyURL)
				sendEvent("downloading", 0, "已从系统注册表读取代理: "+sysProxy)
			} else {
				// 解析失败则回退到环境变量
				transport.Proxy = http.ProxyFromEnvironment
			}
		} else {
			// 没有系统代理配置或读取失败，尝试环境变量兜底
			transport.Proxy = http.ProxyFromEnvironment
			sendEvent("downloading", 0, "系统注册表无代理配置，使用环境变量兜底")
		}
	} else if proxyConfig != "" && proxyConfig != "direct://" && proxyConfig != "__direct__" {
		if proxyURL, pErr := url.Parse(proxyConfig); pErr == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			sendEvent("error", 0, "代理地址解析失败: "+pErr.Error())
			return
		}
	}

	client := &http.Client{
		Timeout:   0, // 取消全局超时，依靠 context 和分片连接维持
		Transport: transport,
	}

	tempFile, err := os.CreateTemp(chromeDir, "download_*.zip")
	if err != nil {
		sendEvent("error", 0, "创建临时文件失败: "+err.Error())
		return
	}
	tempFilePath := tempFile.Name()
	defer func() {
		tempFile.Close()
		os.Remove(tempFilePath) // 清理临时文件
	}()

	sendEvent("downloading", 0, "开始分析下载链接(检测多线程支持)...")

	err = doConcurrentDownload(ctx, client, targetUrl, tempFile, sendEvent)
	if err != nil {
		sendEvent("error", 0, "下载失败: "+err.Error())
		return
	}

	tempFile.Close() // 解压前先关闭写句柄
	sendEvent("extracting", 0, "下载完成，正在准备解压文件...")
	log.Info("内核下载完成", logger.F("url", targetUrl), logger.F("temp", tempFilePath), logger.F("cost", time.Since(t).String()))

	// 3. 执行解压，并剥离顶层文件夹
	if err := extractZipAndStripRoot(tempFilePath, targetDir, func(p int, msg string) {
		sendEvent("extracting", p, msg)
	}); err != nil {
		os.RemoveAll(targetDir) // 删除不完整的解压文件
		sendEvent("error", 0, "解压失败: "+err.Error())
		return
	}

	// 4. 将新内核配置入库
	corePath := filepath.Join("chrome", coreName)
	if m.ValidateCorePath(corePath).Valid {
		newCore := CoreInput{
			CoreId:    uuid.NewString(), // 使用固定的 UUID 或生成新的
			CoreName:  coreName,
			CorePath:  corePath,
			IsDefault: len(m.ListCores()) == 0, // 如果没有其他内核，这设为默认
		}
		if err := m.SaveCore(newCore); err != nil {
			sendEvent("error", 0, "保存配置入库失败: "+err.Error())
			return
		}
		sendEvent("done", 100, "内核下载与配置成功！")
		log.Info("内核下载配置入库成功", logger.F("core_name", coreName))
	} else {
		os.RemoveAll(targetDir) // 删除不正确的解压内容
		sendEvent("error", 0, fmt.Sprintf("解压后未找到浏览器可执行文件（候选：%s），请检查压缩包内容！", strings.Join(CoreExecutableCandidates(), ", ")))
	}
}
