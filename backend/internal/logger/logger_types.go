package logger

import (
	"context"
	"strings"
	"sync"
)

// Level 日志级别
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String 返回日志级别的字符串表示
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 解析日志级别字符串
func ParseLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// Field 结构化日志字段
type Field struct {
	Key   string
	Value interface{}
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level           string
	FileEnabled     bool
	FilePath        string
	Format          string // "text" or "json"
	BufferSize      int    // 缓冲区大小（KB）
	AsyncQueueSize  int    // 异步队列大小
	FlushIntervalMs int    // 刷新间隔（毫秒）

	// 分片配置
	Rotation RotationConfig
}

// RotationConfig 日志分片配置
type RotationConfig struct {
	Enabled      bool
	MaxSizeMB    int    // 单文件最大大小（MB）
	MaxAge       int    // 保留天数
	MaxBackups   int    // 保留文件数
	TimeInterval string // 时间间隔: "daily", "hourly"
}

// Logger 日志记录器
type Logger struct {
	level     Level
	component string
	ctx       context.Context

	// 写入器
	writers       []Writer
	consoleWriter Writer
	fileWriter    *FileWriter

	// 分片管理器
	rotationManager *RotationManager

	// 并发安全
	mu sync.RWMutex

	// 文件写入失败标志
	fileWriteFailed bool
}

// 全局日志实例
var (
	globalLogger *Logger
	globalMu     sync.RWMutex
)

// DefaultLoggerConfig 返回默认日志配置
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:           "info",
		FileEnabled:     false,
		FilePath:        "data/logs/app.log",
		Format:          "text",
		BufferSize:      4, // 4KB
		AsyncQueueSize:  1000,
		FlushIntervalMs: 1000, // 1秒
		Rotation: RotationConfig{
			Enabled:      false,
			MaxSizeMB:    100,
			MaxAge:       7,
			MaxBackups:   5,
			TimeInterval: "daily",
		},
	}
}
