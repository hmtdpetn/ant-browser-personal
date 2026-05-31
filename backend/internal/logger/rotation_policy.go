package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// TimeInterval 时间分片间隔类型
type TimeInterval string

const (
	// Daily 每天分片
	Daily TimeInterval = "daily"
	// Hourly 每小时分片
	Hourly TimeInterval = "hourly"
)

// TimeRotationPolicy 按时间分片策略
// 支持按天或按小时分片
type TimeRotationPolicy struct {
	interval   TimeInterval
	lastRotate time.Time
	mu         sync.RWMutex
}

// NewTimeRotationPolicy 创建时间分片策略
func NewTimeRotationPolicy(interval TimeInterval) *TimeRotationPolicy {
	return &TimeRotationPolicy{
		interval:   interval,
		lastRotate: time.Time{}, // 零值，首次检查时会初始化
	}
}

// ShouldRotate 判断是否应该触发时间分片
func (p *TimeRotationPolicy) ShouldRotate(fileInfo os.FileInfo, entry *LogEntry) bool {
	if fileInfo == nil || entry == nil {
		return false
	}

	p.mu.RLock()
	lastRotate := p.lastRotate
	p.mu.RUnlock()

	entryTime := entry.Timestamp
	if entryTime.IsZero() {
		entryTime = time.Now()
	}

	// 首次检查，使用文件修改时间作为基准
	if lastRotate.IsZero() {
		p.mu.Lock()
		p.lastRotate = fileInfo.ModTime()
		p.mu.Unlock()
		lastRotate = fileInfo.ModTime()
	}

	switch p.interval {
	case Daily:
		// 检查是否跨天
		return !sameDay(lastRotate, entryTime)
	case Hourly:
		// 检查是否跨小时
		return !sameHour(lastRotate, entryTime)
	default:
		// 默认按天
		return !sameDay(lastRotate, entryTime)
	}
}

// GetRotatedFileName 获取分片后的文件名
func (p *TimeRotationPolicy) GetRotatedFileName(baseName string, timestamp time.Time) string {
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	if ext == "" {
		ext = ".log"
	}

	switch p.interval {
	case Hourly:
		// 格式: app.2024-01-15-14.log
		return fmt.Sprintf("%s.%s%s", nameWithoutExt, timestamp.Format("2006-01-02-15"), ext)
	default:
		// 格式: app.2024-01-15.log
		return fmt.Sprintf("%s.%s%s", nameWithoutExt, timestamp.Format("2006-01-02"), ext)
	}
}

// UpdateLastRotate 更新最后分片时间
func (p *TimeRotationPolicy) UpdateLastRotate(t time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastRotate = t
}

// sameDay 判断两个时间是否在同一天
func sameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// sameHour 判断两个时间是否在同一小时
func sameHour(t1, t2 time.Time) bool {
	return sameDay(t1, t2) && t1.Hour() == t2.Hour()
}

// SizeRotationPolicy 按大小分片策略
// 当文件大小超过指定阈值时触发分片
type SizeRotationPolicy struct {
	maxSize  int64 // 最大文件大小（字节）
	sequence int   // 当前序号（同一天内多次分片）
	mu       sync.RWMutex
}

// NewSizeRotationPolicy 创建大小分片策略
// maxSizeBytes: 最大文件大小（字节）
func NewSizeRotationPolicy(maxSizeBytes int64) *SizeRotationPolicy {
	return &SizeRotationPolicy{
		maxSize:  maxSizeBytes,
		sequence: 0,
	}
}

// NewSizeRotationPolicyMB 创建大小分片策略（MB为单位）
// maxSizeMB: 最大文件大小（MB）
func NewSizeRotationPolicyMB(maxSizeMB int) *SizeRotationPolicy {
	return NewSizeRotationPolicy(int64(maxSizeMB) * 1024 * 1024)
}

// ShouldRotate 判断是否应该触发大小分片
func (p *SizeRotationPolicy) ShouldRotate(fileInfo os.FileInfo, entry *LogEntry) bool {
	if fileInfo == nil {
		return false
	}
	return fileInfo.Size() >= p.maxSize
}

// GetRotatedFileName 获取分片后的文件名
func (p *SizeRotationPolicy) GetRotatedFileName(baseName string, timestamp time.Time) string {
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	if ext == "" {
		ext = ".log"
	}

	p.mu.Lock()
	p.sequence++
	seq := p.sequence
	p.mu.Unlock()

	// 格式: app.2024-01-15.1.log
	return fmt.Sprintf("%s.%s.%d%s", nameWithoutExt, timestamp.Format("2006-01-02"), seq, ext)
}

// ResetSequence 重置序号（通常在日期变化时调用）
func (p *SizeRotationPolicy) ResetSequence() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sequence = 0
}

// GetMaxSize 获取最大文件大小
func (p *SizeRotationPolicy) GetMaxSize() int64 {
	return p.maxSize
}

// CompositeRotationPolicy 组合分片策略
// 任一子策略满足条件即触发分片
type CompositeRotationPolicy struct {
	policies []RotationPolicy
	mu       sync.RWMutex
}

// NewCompositeRotationPolicy 创建组合分片策略
func NewCompositeRotationPolicy(policies ...RotationPolicy) *CompositeRotationPolicy {
	return &CompositeRotationPolicy{
		policies: policies,
	}
}

// ShouldRotate 判断是否应该触发分片
// 任一子策略返回 true 即触发
func (p *CompositeRotationPolicy) ShouldRotate(fileInfo os.FileInfo, entry *LogEntry) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, policy := range p.policies {
		if policy.ShouldRotate(fileInfo, entry) {
			return true
		}
	}
	return false
}

// GetRotatedFileName 获取分片后的文件名
// 使用第一个策略的命名规则
func (p *CompositeRotationPolicy) GetRotatedFileName(baseName string, timestamp time.Time) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.policies) > 0 {
		return p.policies[0].GetRotatedFileName(baseName, timestamp)
	}

	// 默认命名
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	if ext == "" {
		ext = ".log"
	}
	return fmt.Sprintf("%s.%s%s", nameWithoutExt, timestamp.Format("2006-01-02"), ext)
}

// AddPolicy 添加子策略
func (p *CompositeRotationPolicy) AddPolicy(policy RotationPolicy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.policies = append(p.policies, policy)
}

// GetPolicies 获取所有子策略
func (p *CompositeRotationPolicy) GetPolicies() []RotationPolicy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]RotationPolicy, len(p.policies))
	copy(result, p.policies)
	return result
}
