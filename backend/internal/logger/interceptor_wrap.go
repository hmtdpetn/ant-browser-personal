package logger

import "time"

// WrapFunc 包装无参数无返回值的函数
func (m *MethodInterceptor) WrapFunc(name string, fn func()) func() {
	if !m.config.Enabled {
		return fn
	}

	return func() {
		ctx := m.beforeCall(name, nil)
		defer m.afterCallRecover(ctx, nil, nil)

		fn()
	}
}

// WrapFuncWithError 包装返回 error 的函数
func (m *MethodInterceptor) WrapFuncWithError(name string, fn func() error) func() error {
	if !m.config.Enabled {
		return fn
	}

	return func() error {
		ctx := m.beforeCall(name, nil)
		var err error

		defer func() {
			m.afterCallRecover(ctx, nil, err)
		}()

		err = fn()
		return err
	}
}

// WrapFuncResult 包装有返回值的函数（使用 interface{}）
func (m *MethodInterceptor) WrapFuncResult(name string, fn func() interface{}) func() interface{} {
	if !m.config.Enabled {
		return fn
	}

	return func() interface{} {
		ctx := m.beforeCall(name, nil)
		var result interface{}

		defer func() {
			m.afterCallRecover(ctx, result, nil)
		}()

		result = fn()
		return result
	}
}

// WrapFuncResultError 包装有返回值和 error 的函数
func (m *MethodInterceptor) WrapFuncResultError(name string, fn func() (interface{}, error)) func() (interface{}, error) {
	if !m.config.Enabled {
		return fn
	}

	return func() (interface{}, error) {
		ctx := m.beforeCall(name, nil)
		var result interface{}
		var err error

		defer func() {
			m.afterCallRecover(ctx, result, err)
		}()

		result, err = fn()
		return result, err
	}
}

// WrapMethod1Arg 包装单参数方法
func (m *MethodInterceptor) WrapMethod1Arg(name string, fn func(interface{}) interface{}) func(interface{}) interface{} {
	if !m.config.Enabled {
		return fn
	}

	return func(p interface{}) interface{} {
		ctx := m.beforeCall(name, []interface{}{p})
		var result interface{}

		defer func() {
			m.afterCallRecover(ctx, result, nil)
		}()

		result = fn(p)
		return result
	}
}

// WrapMethod1ArgError 包装单参数返回 error 的方法
func (m *MethodInterceptor) WrapMethod1ArgError(name string, fn func(interface{}) (interface{}, error)) func(interface{}) (interface{}, error) {
	if !m.config.Enabled {
		return fn
	}

	return func(p interface{}) (interface{}, error) {
		ctx := m.beforeCall(name, []interface{}{p})
		var result interface{}
		var err error

		defer func() {
			m.afterCallRecover(ctx, result, err)
		}()

		result, err = fn(p)
		return result, err
	}
}

// Intercept 通用拦截方法，用于手动记录方法调用
// 返回 CallContext 用于后续调用 Complete 或 Fail
func (m *MethodInterceptor) Intercept(methodName string, params ...interface{}) *CallContext {
	if !m.config.Enabled {
		return &CallContext{
			RequestID:  GenerateRequestID(),
			MethodName: methodName,
			StartTime:  time.Now(),
			Parameters: params,
		}
	}
	return m.beforeCall(methodName, params)
}

// Complete 标记方法调用成功完成
func (m *MethodInterceptor) Complete(ctx *CallContext, result interface{}) {
	if !m.config.Enabled {
		return
	}
	m.afterCall(ctx, result, nil)
}

// Fail 标记方法调用失败
func (m *MethodInterceptor) Fail(ctx *CallContext, err error) {
	if !m.config.Enabled {
		return
	}
	m.afterCall(ctx, nil, err)
}
