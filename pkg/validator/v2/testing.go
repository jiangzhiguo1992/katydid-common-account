package v2

import (
	"testing"
)

// ============================================================================
// 测试辅助函数 - 单一职责：提供测试相关的工具
// 设计原则：提高测试的可读性和可维护性
// ============================================================================

// TestValidator 测试验证器（简化测试代码）
type TestValidator struct {
	validator Validator
	t         *testing.T
}

// NewTestValidator 创建测试验证器
func NewTestValidator(t *testing.T) *TestValidator {
	v, err := NewSimpleValidator()
	if err != nil {
		t.Fatalf("创建验证器失败: %v", err)
	}

	return &TestValidator{
		validator: v,
		t:         t,
	}
}

// MustPass 验证必须通过
func (tv *TestValidator) MustPass(data interface{}, scene Scene) {
	err := tv.validator.Validate(data, scene)
	if err != nil {
		tv.t.Errorf("验证应该通过但失败了: %v", err)
	}
}

// MustFail 验证必须失败
func (tv *TestValidator) MustFail(data interface{}, scene Scene) {
	err := tv.validator.Validate(data, scene)
	if err == nil {
		tv.t.Error("验证应该失败但通过了")
	}
}

// MustFailWithField 验证必须失败且包含指定字段错误
func (tv *TestValidator) MustFailWithField(data interface{}, scene Scene, field string) {
	err := tv.validator.Validate(data, scene)
	if err == nil {
		tv.t.Errorf("验证应该失败但通过了")
		return
	}

	if validationErrors, ok := err.(ValidationErrors); ok {
		for _, e := range validationErrors {
			if e.Field == field {
				return // 找到了指定字段的错误
			}
		}
		tv.t.Errorf("验证失败但未包含字段 '%s' 的错误", field)
	}
}

// MustFailWithTag 验证必须失败且包含指定标签错误
func (tv *TestValidator) MustFailWithTag(data interface{}, scene Scene, tag string) {
	err := tv.validator.Validate(data, scene)
	if err == nil {
		tv.t.Errorf("验证应该失败但通过了")
		return
	}

	if validationErrors, ok := err.(ValidationErrors); ok {
		for _, e := range validationErrors {
			if e.Tag == tag {
				return // 找到了指定标签的错误
			}
		}
		tv.t.Errorf("验证失败但未包含标签 '%s' 的错误", tag)
	}
}

// AssertErrorCount 断言错误数量
func (tv *TestValidator) AssertErrorCount(data interface{}, scene Scene, expectedCount int) {
	err := tv.validator.Validate(data, scene)

	if expectedCount == 0 {
		if err != nil {
			tv.t.Errorf("期望无错误，但得到: %v", err)
		}
		return
	}

	if err == nil {
		tv.t.Errorf("期望 %d 个错误，但验证通过了", expectedCount)
		return
	}

	if validationErrors, ok := err.(ValidationErrors); ok {
		actualCount := len(validationErrors)
		if actualCount != expectedCount {
			tv.t.Errorf("期望 %d 个错误，实际得到 %d 个", expectedCount, actualCount)
		}
	}
}

// ============================================================================
// Mock 对象 - 用于测试
// ============================================================================

// MockRuleProvider Mock规则提供者
type MockRuleProvider struct {
	Rules SceneRules
}

// GetRules 实现 RuleProvider 接口
func (m *MockRuleProvider) GetRules(scene Scene) map[string]string {
	return m.Rules.Get(scene)
}

// MockCustomValidator Mock自定义验证器
type MockCustomValidator struct {
	ValidateFunc func(scene Scene, collector ErrorCollector)
}

// CustomValidate 实现 CustomValidator 接口
func (m *MockCustomValidator) CustomValidate(scene Scene, collector ErrorCollector) {
	if m.ValidateFunc != nil {
		m.ValidateFunc(scene, collector)
	}
}

// MockErrorMessageProvider Mock错误消息提供者
type MockErrorMessageProvider struct {
	Messages map[string]string
}

// GetErrorMessage 实现 ErrorMessageProvider 接口
func (m *MockErrorMessageProvider) GetErrorMessage(field, tag, param string) string {
	key := field + "." + tag
	if msg, ok := m.Messages[key]; ok {
		return msg
	}
	return ""
}

// ============================================================================
// 基准测试辅助
// ============================================================================

// BenchmarkValidator 基准测试验证器
type BenchmarkValidator struct {
	validator Validator
}

// NewBenchmarkValidator 创建基准测试验证器
func NewBenchmarkValidator() (*BenchmarkValidator, error) {
	v, err := NewDefaultValidator()
	if err != nil {
		return nil, err
	}

	return &BenchmarkValidator{
		validator: v,
	}, nil
}

// Validate 执行验证（用于基准测试）
func (bv *BenchmarkValidator) Validate(data interface{}, scene Scene) error {
	return bv.validator.Validate(data, scene)
}
