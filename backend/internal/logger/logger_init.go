package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Init 初始化全局日志（简单版本，仅控制台输出）
func Init(ctx context.Context, levelStr string) {
	InitWithConfig(ctx, LoggerConfig{
		Level:       levelStr,
		FileEnabled: false,
		Format:      "text",
	})
}

// InitWithConfig 使用配置初始化全局日志
func InitWithConfig(ctx context.Context, config LoggerConfig) {
	globalMu.Lock()
	defer globalMu.Unlock()

	// 解析日志级别，无效级别使用默认 INFO
	level := ParseLevel(config.Level)
	if config.Level != "" && level == INFO && strings.ToLower(config.Level) != "info" {
		// 无效级别，记录警告（使用 fmt 因为 logger 还未初始化）
		fmt.Printf("[WARN] Invalid log level '%s', using default 'INFO'\n", config.Level)
	}

	// 创建格式化器
	var formatter Formatter
	switch strings.ToLower(config.Format) {
	case "json":
		formatter = NewJSONFormatter()
	default:
		formatter = NewTextFormatter()
	}

	// 创建控制台写入器
	consoleWriter := NewConsoleWriter(formatter)

	logger := &Logger{
		level:         level,
		ctx:           ctx,
		writers:       []Writer{consoleWriter, globalMemoryWriter},
		consoleWriter: consoleWriter,
	}

	// 如果启用文件日志，创建文件写入器
	if config.FileEnabled && config.FilePath != "" {
		fileWriter, rotationManager, err := createFileWriterWithRotation(config, formatter)
		if err != nil {
			// 文件写入器创建失败，回退到仅控制台输出
			fmt.Printf("[WARN] Failed to create file writer: %v, falling back to console only\n", err)
			logger.fileWriteFailed = true
		} else {
			logger.fileWriter = fileWriter
			logger.rotationManager = rotationManager
			logger.writers = append(logger.writers, fileWriter)
		}
	}

	globalLogger = logger
}

// createFileWriterWithRotation 创建带分片功能的文件写入器
func createFileWriterWithRotation(config LoggerConfig, formatter Formatter) (*FileWriter, *RotationManager, error) {
	// 确保目录存在
	dir := filepath.Dir(config.FilePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	// 计算缓冲区大小（KB -> 字节）
	bufferSize := config.BufferSize * 1024
	if bufferSize <= 0 {
		bufferSize = 4 * 1024 // 默认 4KB
	}

	// 计算刷新间隔
	flushInterval := time.Duration(config.FlushIntervalMs) * time.Millisecond
	if flushInterval <= 0 {
		flushInterval = time.Second
	}

	// 异步队列大小
	asyncQueueSize := config.AsyncQueueSize
	if asyncQueueSize <= 0 {
		asyncQueueSize = 1000
	}

	fileConfig := FileWriterConfig{
		FilePath:       config.FilePath,
		BufferSize:     bufferSize,
		FlushInterval:  flushInterval,
		AsyncQueueSize: asyncQueueSize,
	}

	// 使用异步文件写入器
	fileWriter, err := NewAsyncFileWriter(fileConfig, formatter)
	if err != nil {
		return nil, nil, err
	}

	// 创建分片管理器（如果启用）
	var rotationManager *RotationManager
	if config.Rotation.Enabled {
		rotationPolicy := createRotationPolicy(config.Rotation)
		rotationManager = NewRotationManager(RotationManagerConfig{
			BasePath:   config.FilePath,
			MaxBackups: config.Rotation.MaxBackups,
			MaxAge:     config.Rotation.MaxAge,
			Policy:     rotationPolicy,
		})
	}

	return fileWriter, rotationManager, nil
}

// createRotationPolicy 根据配置创建分片策略
func createRotationPolicy(config RotationConfig) RotationPolicy {
	var policies []RotationPolicy

	// 时间分片策略
	if config.TimeInterval != "" {
		var interval TimeInterval
		switch strings.ToLower(config.TimeInterval) {
		case "hourly":
			interval = Hourly
		default:
			interval = Daily
		}
		policies = append(policies, NewTimeRotationPolicy(interval))
	}

	// 大小分片策略
	if config.MaxSizeMB > 0 {
		policies = append(policies, NewSizeRotationPolicyMB(config.MaxSizeMB))
	}

	// 如果有多个策略，使用组合策略
	if len(policies) > 1 {
		return NewCompositeRotationPolicy(policies...)
	} else if len(policies) == 1 {
		return policies[0]
	}

	// 默认按天分片
	return NewTimeRotationPolicy(Daily)
}

// Close 关闭全局日志
func Close() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalLogger == nil {
		return nil
	}

	var lastErr error
	for _, writer := range globalLogger.writers {
		if err := writer.Close(); err != nil {
			lastErr = err
		}
	}

	globalLogger = nil
	return lastErr
}

// New 创建新的日志记录器
func New(component string) *Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalLogger == nil {
		// 如果全局日志未初始化，创建一个默认的
		consoleWriter := NewConsoleWriter(NewTextFormatter())
		return &Logger{
			level:         INFO,
			component:     component,
			writers:       []Writer{consoleWriter},
			consoleWriter: consoleWriter,
		}
	}

	return &Logger{
		level:           globalLogger.level,
		component:       component,
		ctx:             globalLogger.ctx,
		writers:         globalLogger.writers,
		consoleWriter:   globalLogger.consoleWriter,
		fileWriter:      globalLogger.fileWriter,
		rotationManager: globalLogger.rotationManager,
		fileWriteFailed: globalLogger.fileWriteFailed,
	}
}
