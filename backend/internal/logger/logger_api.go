package logger

import (
	"fmt"
)

// SetLevel 动态设置日志级别（并发安全）
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetLevelString 通过字符串动态设置日志级别
func (l *Logger) SetLevelString(levelStr string) {
	l.SetLevel(ParseLevel(levelStr))
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetGlobalLevel 设置全局日志级别
func SetGlobalLevel(level Level) {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalLogger != nil {
		globalLogger.mu.Lock()
		globalLogger.level = level
		globalLogger.mu.Unlock()
	}
}

// SetGlobalLevelString 通过字符串设置全局日志级别
func SetGlobalLevelString(levelStr string) {
	SetGlobalLevel(ParseLevel(levelStr))
}

// Debug 记录调试日志
func (l *Logger) Debug(msg string, fields ...Field) {
	l.mu.RLock()
	level := l.level
	l.mu.RUnlock()

	if level <= DEBUG {
		l.log(DEBUG, msg, fields...)
	}
}

// Info 记录信息日志
func (l *Logger) Info(msg string, fields ...Field) {
	l.mu.RLock()
	level := l.level
	l.mu.RUnlock()

	if level <= INFO {
		l.log(INFO, msg, fields...)
	}
}

// Warn 记录警告日志
func (l *Logger) Warn(msg string, fields ...Field) {
	l.mu.RLock()
	level := l.level
	l.mu.RUnlock()

	if level <= WARN {
		l.log(WARN, msg, fields...)
	}
}

// Error 记录错误日志
func (l *Logger) Error(msg string, fields ...Field) {
	l.mu.RLock()
	level := l.level
	l.mu.RUnlock()

	if level <= ERROR {
		l.log(ERROR, msg, fields...)
	}
}

// log 内部日志记录方法
func (l *Logger) log(level Level, msg string, fields ...Field) {
	// 创建日志条目
	entry := NewLogEntry(level, l.component, msg)

	// 添加字段
	if len(fields) > 0 {
		fieldMap := make(map[string]interface{}, len(fields))
		for _, field := range fields {
			fieldMap[field.Key] = field.Value
		}
		entry.WithFields(fieldMap)
	}

	// 写入所有写入器
	l.writeEntry(entry)
}

// writeEntry 写入日志条目到所有写入器
func (l *Logger) writeEntry(entry *LogEntry) {
	l.mu.RLock()
	writers := l.writers
	fileWriter := l.fileWriter
	consoleWriter := l.consoleWriter
	fileWriteFailed := l.fileWriteFailed
	l.mu.RUnlock()

	// 如果文件写入已失败，只写入控制台
	if fileWriteFailed {
		if consoleWriter != nil {
			_ = consoleWriter.Write(entry)
		}
		return
	}

	// 写入所有写入器
	for _, writer := range writers {
		if err := writer.Write(entry); err != nil {
			// 如果是文件写入器失败，标记并回退到控制台
			if writer == fileWriter {
				l.handleFileWriteError(entry, err)
			}
		}
	}
}

// handleFileWriteError 处理文件写入错误
func (l *Logger) handleFileWriteError(entry *LogEntry, err error) {
	l.mu.Lock()
	if !l.fileWriteFailed {
		l.fileWriteFailed = true
		// 记录错误到控制台
		fmt.Printf("[ERROR] File write failed: %v, falling back to console only\n", err)
	}
	l.mu.Unlock()
}

// LogEntry 直接写入日志条目（用于拦截器等高级用法）
func (l *Logger) LogEntry(entry *LogEntry) {
	l.mu.RLock()
	level := l.level
	l.mu.RUnlock()

	// 检查日志级别
	if entry.Level < level {
		return
	}

	l.writeEntry(entry)
}

// WithComponent 创建带有组件名的新日志记录器
func (l *Logger) WithComponent(component string) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &Logger{
		level:           l.level,
		component:       component,
		ctx:             l.ctx,
		writers:         l.writers,
		consoleWriter:   l.consoleWriter,
		fileWriter:      l.fileWriter,
		rotationManager: l.rotationManager,
		fileWriteFailed: l.fileWriteFailed,
	}
}

// Flush 刷新所有写入器的缓冲区
func (l *Logger) Flush() error {
	l.mu.RLock()
	fileWriter := l.fileWriter
	l.mu.RUnlock()

	if fileWriter != nil {
		return fileWriter.Flush()
	}
	return nil
}

// GetRotationManager 获取分片管理器
func (l *Logger) GetRotationManager() *RotationManager {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.rotationManager
}

// F 创建字段的便捷函数
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Fs 创建多个字段的便捷函数
func Fs(keyValues ...interface{}) []Field {
	fields := make([]Field, 0, len(keyValues)/2)
	for i := 0; i < len(keyValues)-1; i += 2 {
		if key, ok := keyValues[i].(string); ok {
			fields = append(fields, Field{Key: key, Value: keyValues[i+1]})
		}
	}
	return fields
}

// IsFileEnabled 检查文件日志是否启用
func (l *Logger) IsFileEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.fileWriter != nil && !l.fileWriteFailed
}

// GetWriters 获取所有写入器（用于测试）
func (l *Logger) GetWriters() []Writer {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.writers
}

// ShouldLog 检查指定级别是否应该被记录
func (l *Logger) ShouldLog(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}
