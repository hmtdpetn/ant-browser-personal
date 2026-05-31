package logger

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InterceptorConfig 拦截器配置
type InterceptorConfig struct {
	Enabled         bool
	LogParameters   bool
	LogResults      bool
	SensitiveFields []string
}

// MethodInterceptor 方法拦截器
// 用于自动记录方法调用的 AOP 组件
type MethodInterceptor struct {
	logger          *Logger
	config          InterceptorConfig
	sensitiveFields map[string]bool
	mu              sync.RWMutex
}

// CallContext 调用上下文
type CallContext struct {
	RequestID  string
	MethodName string
	StartTime  time.Time
	Parameters []interface{}
}

// NewMethodInterceptor 创建新的方法拦截器
func NewMethodInterceptor(logger *Logger, config InterceptorConfig) *MethodInterceptor {
	sensitiveFields := make(map[string]bool)
	for _, field := range config.SensitiveFields {
		sensitiveFields[strings.ToLower(field)] = true
	}

	return &MethodInterceptor{
		logger:          logger,
		config:          config,
		sensitiveFields: sensitiveFields,
	}
}

// GenerateRequestID 生成唯一的请求 ID
func GenerateRequestID() string {
	return uuid.New().String()
}

// SetEnabled 设置拦截器启用状态
func (m *MethodInterceptor) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.Enabled = enabled
}

// IsEnabled 检查拦截器是否启用
func (m *MethodInterceptor) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}

// GetConfig 获取拦截器配置
func (m *MethodInterceptor) GetConfig() InterceptorConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetRequestID 获取调用上下文的请求 ID
func (ctx *CallContext) GetRequestID() string {
	return ctx.RequestID
}

// GetMethodName 获取调用上下文的方法名
func (ctx *CallContext) GetMethodName() string {
	return ctx.MethodName
}

// GetDuration 获取调用耗时（毫秒）
func (ctx *CallContext) GetDuration() int64 {
	return time.Since(ctx.StartTime).Milliseconds()
}
