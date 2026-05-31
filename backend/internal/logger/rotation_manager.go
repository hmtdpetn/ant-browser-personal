package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotationManagerConfig 分片管理器配置
type RotationManagerConfig struct {
	BasePath   string         // 基础日志文件路径
	MaxBackups int            // 最大保留文件数
	MaxAge     int            // 最大保留天数
	Policy     RotationPolicy // 分片策略
}

// RotationManager 日志分片管理器
// 负责执行分片操作和清理历史文件
type RotationManager struct {
	config     RotationManagerConfig
	mu         sync.Mutex
	currentSeq int // 当前序号
}

// NewRotationManager 创建分片管理器
func NewRotationManager(config RotationManagerConfig) *RotationManager {
	if config.MaxBackups <= 0 {
		config.MaxBackups = 5
	}
	return &RotationManager{
		config:     config,
		currentSeq: 0,
	}
}

// ShouldRotate 检查是否需要分片
func (m *RotationManager) ShouldRotate(fileInfo os.FileInfo, entry *LogEntry) bool {
	if m.config.Policy == nil {
		return false
	}
	return m.config.Policy.ShouldRotate(fileInfo, entry)
}

// Rotate 执行分片操作
// 返回新的日志文件路径
func (m *RotationManager) Rotate(currentFile *os.File) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if currentFile == nil {
		return "", fmt.Errorf("current file is nil")
	}

	// 获取当前文件信息
	basePath := m.config.BasePath
	timestamp := time.Now()

	// 生成分片文件名
	rotatedName := m.generateRotatedFileName(basePath, timestamp)

	// 关闭当前文件
	if err := currentFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close current file: %w", err)
	}

	// 重命名当前文件为分片文件
	if err := os.Rename(basePath, rotatedName); err != nil {
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	// 清理历史文件
	if err := m.cleanupOldFiles(); err != nil {
		// 清理失败不影响主流程，只记录错误
		fmt.Fprintf(os.Stderr, "failed to cleanup old files: %v\n", err)
	}

	return rotatedName, nil
}

// generateRotatedFileName 生成分片文件名
// 格式: {basename}.{timestamp}[.{sequence}].log
func (m *RotationManager) generateRotatedFileName(basePath string, timestamp time.Time) string {
	ext := filepath.Ext(basePath)
	nameWithoutExt := strings.TrimSuffix(basePath, ext)
	if ext == "" {
		ext = ".log"
	}

	dateStr := timestamp.Format("2006-01-02")

	// 检查是否已存在同日期的文件，确定序号
	seq := m.findNextSequence(nameWithoutExt, dateStr, ext)

	if seq > 0 {
		// 格式: app.2024-01-15.1.log
		return fmt.Sprintf("%s.%s.%d%s", nameWithoutExt, dateStr, seq, ext)
	}
	// 格式: app.2024-01-15.log
	return fmt.Sprintf("%s.%s%s", nameWithoutExt, dateStr, ext)
}

// findNextSequence 查找下一个可用序号
func (m *RotationManager) findNextSequence(nameWithoutExt, dateStr, ext string) int {
	dir := filepath.Dir(nameWithoutExt)
	if dir == "" {
		dir = "."
	}
	baseName := filepath.Base(nameWithoutExt)

	// 查找已存在的同日期文件
	pattern := fmt.Sprintf("%s.%s*%s", baseName, dateStr, ext)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(matches) == 0 {
		return 0
	}

	// 找到最大序号
	maxSeq := 0
	seqPattern := regexp.MustCompile(fmt.Sprintf(`%s\.%s(?:\.(\d+))?%s$`,
		regexp.QuoteMeta(baseName),
		regexp.QuoteMeta(dateStr),
		regexp.QuoteMeta(ext)))

	for _, match := range matches {
		fileName := filepath.Base(match)
		if submatches := seqPattern.FindStringSubmatch(fileName); submatches != nil {
			if len(submatches) > 1 && submatches[1] != "" {
				var seq int
				fmt.Sscanf(submatches[1], "%d", &seq)
				if seq > maxSeq {
					maxSeq = seq
				}
			} else {
				// 无序号的文件存在，下一个从1开始
				if maxSeq == 0 {
					maxSeq = 0
				}
			}
		}
	}

	return maxSeq + 1
}

// cleanupOldFiles 清理历史文件
func (m *RotationManager) cleanupOldFiles() error {
	files, err := m.listRotatedFiles()
	if err != nil {
		return err
	}

	// 按修改时间排序（最新的在前）
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	// 删除超出数量限制的文件
	if len(files) > m.config.MaxBackups {
		for _, f := range files[m.config.MaxBackups:] {
			if err := os.Remove(f.Path); err != nil {
				return fmt.Errorf("failed to remove old file %s: %w", f.Path, err)
			}
		}
	}

	// 删除超出时间限制的文件
	if m.config.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -m.config.MaxAge)
		for _, f := range files {
			if f.ModTime.Before(cutoff) {
				if err := os.Remove(f.Path); err != nil {
					return fmt.Errorf("failed to remove old file %s: %w", f.Path, err)
				}
			}
		}
	}

	return nil
}

// rotatedFileInfo 分片文件信息
type rotatedFileInfo struct {
	Path    string
	ModTime time.Time
}

// listRotatedFiles 列出所有分片文件
func (m *RotationManager) listRotatedFiles() ([]rotatedFileInfo, error) {
	basePath := m.config.BasePath
	dir := filepath.Dir(basePath)
	if dir == "" {
		dir = "."
	}

	ext := filepath.Ext(basePath)
	nameWithoutExt := filepath.Base(strings.TrimSuffix(basePath, ext))
	if ext == "" {
		ext = ".log"
	}

	// 匹配模式: app.YYYY-MM-DD*.log
	pattern := fmt.Sprintf("%s.[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]*%s", nameWithoutExt, ext)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	var files []rotatedFileInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		files = append(files, rotatedFileInfo{
			Path:    match,
			ModTime: info.ModTime(),
		})
	}

	return files, nil
}

// GetRotatedFileCount 获取当前分片文件数量
func (m *RotationManager) GetRotatedFileCount() (int, error) {
	files, err := m.listRotatedFiles()
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

// GetConfig 获取配置
func (m *RotationManager) GetConfig() RotationManagerConfig {
	return m.config
}
