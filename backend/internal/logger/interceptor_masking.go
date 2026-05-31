package logger

import (
	"fmt"
	"reflect"
	"strings"
)

// maskSensitiveParams 对敏感参数进行脱敏
func (m *MethodInterceptor) maskSensitiveParams(params []interface{}) []interface{} {
	if len(m.sensitiveFields) == 0 {
		return params
	}

	masked := make([]interface{}, len(params))
	for i, param := range params {
		masked[i] = m.maskValue(param)
	}
	return masked
}

// maskValue 对值进行脱敏处理
func (m *MethodInterceptor) maskValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Map:
		return m.maskMap(v)
	case reflect.Struct:
		return m.maskStruct(v)
	case reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return m.maskValue(v.Elem().Interface())
	default:
		return value
	}
}

// maskMap 对 map 进行脱敏
func (m *MethodInterceptor) maskMap(v reflect.Value) interface{} {
	result := make(map[string]interface{})

	iter := v.MapRange()
	for iter.Next() {
		key := fmt.Sprintf("%v", iter.Key().Interface())
		val := iter.Value().Interface()

		if m.isSensitiveField(key) {
			result[key] = "***"
		} else {
			result[key] = m.maskValue(val)
		}
	}

	return result
}

// maskStruct 对结构体进行脱敏
func (m *MethodInterceptor) maskStruct(v reflect.Value) interface{} {
	result := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldName := field.Name
		fieldValue := v.Field(i).Interface()

		if m.isSensitiveField(fieldName) {
			result[fieldName] = "***"
		} else {
			result[fieldName] = m.maskValue(fieldValue)
		}
	}

	return result
}

// maskSensitiveValue 对单个值进行脱敏（用于返回值）
func (m *MethodInterceptor) maskSensitiveValue(fieldName string, value interface{}) interface{} {
	if m.isSensitiveField(fieldName) {
		return "***"
	}
	return m.maskValue(value)
}

// isSensitiveField 检查字段是否为敏感字段
func (m *MethodInterceptor) isSensitiveField(fieldName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sensitiveFields[strings.ToLower(fieldName)]
}

// AddSensitiveField 添加敏感字段
func (m *MethodInterceptor) AddSensitiveField(fieldName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sensitiveFields[strings.ToLower(fieldName)] = true
}

// RemoveSensitiveField 移除敏感字段
func (m *MethodInterceptor) RemoveSensitiveField(fieldName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sensitiveFields, strings.ToLower(fieldName))
}
