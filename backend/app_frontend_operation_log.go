package backend

import "ant-chrome/backend/internal/logger"

// FrontendOperationLog records frontend-triggered Wails operation results into the app log.
func (a *App) FrontendOperationLog(level string, method string, success bool, durationMs int64, message string) {
	log := logger.New("Frontend")
	fields := []logger.Field{
		logger.F("method", method),
		logger.F("success", success),
		logger.F("duration_ms", durationMs),
	}
	if message != "" {
		fields = append(fields, logger.F("message", message))
	}
	if !success || level == "error" || level == "ERROR" {
		log.Error("前端操作失败", fields...)
		return
	}
	log.Info("前端操作完成", fields...)
}
