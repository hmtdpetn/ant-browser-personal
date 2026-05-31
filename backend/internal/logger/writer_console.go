package logger

import (
	"fmt"
	"os"
	"sync"
)

// ConsoleWriter 控制台写入器
// 将日志输出到标准输出
type ConsoleWriter struct {
	formatter Formatter
	mu        sync.Mutex
}

// NewConsoleWriter 创建新的控制台写入器
func NewConsoleWriter(formatter Formatter) *ConsoleWriter {
	if formatter == nil {
		formatter = NewTextFormatter()
	}
	return &ConsoleWriter{
		formatter: formatter,
	}
}

// Write 写入日志条目到控制台
func (w *ConsoleWriter) Write(entry *LogEntry) error {
	if entry == nil {
		return nil
	}

	data, err := w.formatter.Format(entry)
	if err != nil {
		return fmt.Errorf("failed to format log entry: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	_, err = os.Stdout.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to console: %w", err)
	}

	return nil
}

// Close 关闭控制台写入器（无操作）
func (w *ConsoleWriter) Close() error {
	return nil
}
