package logger

// MultiWriter 多写入器
// 同时写入多个目标
type MultiWriter struct {
	writers []Writer
}

// NewMultiWriter 创建多写入器
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write 写入日志到所有写入器
func (w *MultiWriter) Write(entry *LogEntry) error {
	var lastErr error
	for _, writer := range w.writers {
		if err := writer.Write(entry); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Close 关闭所有写入器
func (w *MultiWriter) Close() error {
	var lastErr error
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// AddWriter 添加写入器
func (w *MultiWriter) AddWriter(writer Writer) {
	w.writers = append(w.writers, writer)
}
