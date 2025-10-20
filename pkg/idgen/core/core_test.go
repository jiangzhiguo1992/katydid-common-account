package core

import (
	"errors"
	"testing"
)

// TestGeneratorType 测试生成器类型
func TestGeneratorType(t *testing.T) {
	tests := []struct {
		name     string
		genType  GeneratorType
		expected string
		isValid  bool
	}{
		{
			name:     "Snowflake类型",
			genType:  GeneratorTypeSnowflake,
			expected: "snowflake",
			isValid:  true,
		},
		{
			name:     "UUID类型",
			genType:  GeneratorTypeUUID,
			expected: "uuid",
			isValid:  true,
		},
		{
			name:     "Custom类型",
			genType:  GeneratorTypeCustom,
			expected: "custom",
			isValid:  true,
		},
		{
			name:     "无效类型",
			genType:  GeneratorType("invalid"),
			expected: "invalid",
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试String方法
			if got := tt.genType.String(); got != tt.expected {
				t.Errorf("String() = %s, 期望 %s", got, tt.expected)
			}

			// 测试IsValid方法
			if got := tt.genType.IsValid(); got != tt.isValid {
				t.Errorf("IsValid() = %v, 期望 %v", got, tt.isValid)
			}
		})
	}
}

// TestClockBackwardStrategy 测试时钟回拨策略
func TestClockBackwardStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy ClockBackwardStrategy
		expected string
	}{
		{
			name:     "错误策略",
			strategy: StrategyError,
			expected: "Error",
		},
		{
			name:     "等待策略",
			strategy: StrategyWait,
			expected: "Wait",
		},
		{
			name:     "使用上次时间戳策略",
			strategy: StrategyUseLastTimestamp,
			expected: "UseLastTimestamp",
		},
		{
			name:     "未知策略",
			strategy: ClockBackwardStrategy(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.expected {
				t.Errorf("String() = %s, 期望 %s", got, tt.expected)
			}
		})
	}
}

// TestErrors 测试错误定义
func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidWorkerID", ErrInvalidWorkerID},
		{"ErrInvalidDatacenterID", ErrInvalidDatacenterID},
		{"ErrClockMovedBackwards", ErrClockMovedBackwards},
		{"ErrInvalidSnowflakeID", ErrInvalidSnowflakeID},
		{"ErrInvalidBatchSize", ErrInvalidBatchSize},
		{"ErrNilConfig", ErrNilConfig},
		{"ErrGeneratorNotFound", ErrGeneratorNotFound},
		{"ErrGeneratorAlreadyExists", ErrGeneratorAlreadyExists},
		{"ErrInvalidGeneratorType", ErrInvalidGeneratorType},
		{"ErrInvalidKey", ErrInvalidKey},
		{"ErrFactoryNotFound", ErrFactoryNotFound},
		{"ErrMaxGeneratorsReached", ErrMaxGeneratorsReached},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("错误不应为nil")
			}
			if tt.err.Error() == "" {
				t.Error("错误消息不应为空")
			}
		})
	}
}

// TestErrorsIs 测试错误判断
func TestErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "相同错误",
			err:    ErrInvalidWorkerID,
			target: ErrInvalidWorkerID,
			want:   true,
		},
		{
			name:   "不同错误",
			err:    ErrInvalidWorkerID,
			target: ErrInvalidDatacenterID,
			want:   false,
		},
		{
			name:   "包装的错误",
			err:    errors.Join(ErrInvalidWorkerID, errors.New("额外信息")),
			target: ErrInvalidWorkerID,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

// ========== 高并发百万级测试（多维度性能分析） ==========

// TestGeneratorType_Concurrent 测试GeneratorType并发安全性
func TestGeneratorType_Concurrent(t *testing.T) {
	types := []GeneratorType{
		GeneratorTypeSnowflake,
		GeneratorTypeUUID,
		GeneratorTypeCustom,
		GeneratorType("invalid"),
	}

	const goroutines = 1000
	const iterations = 1000

	errors := make(chan error, goroutines*iterations)
	done := make(chan struct{})

	// 并发测试类型验证
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				for _, t := range types {
					_ = t.String()
					_ = t.IsValid()
				}
			}
			done <- struct{}{}
		}()
	}

	// 等待所有协程完成
	for i := 0; i < goroutines; i++ {
		<-done
	}
	close(errors)

	// 检查错误
	for err := range errors {
		if err != nil {
			t.Errorf("并发测试失败: %v", err)
		}
	}
}

// TestClockBackwardStrategy_Concurrent 测试ClockBackwardStrategy并发安全性
func TestClockBackwardStrategy_Concurrent(t *testing.T) {
	strategies := []ClockBackwardStrategy{
		StrategyError,
		StrategyWait,
		StrategyUseLastTimestamp,
	}

	const goroutines = 1000
	const iterations = 1000

	done := make(chan struct{})

	// 并发测试策略
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				for _, s := range strategies {
					_ = s.String()
					_ = s.IsValid()
				}
			}
			done <- struct{}{}
		}()
	}

	// 等待完成
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// TestErrors_Concurrent 测试错误常量并发访问
func TestErrors_Concurrent(t *testing.T) {
	errorList := []error{
		ErrInvalidWorkerID,
		ErrInvalidDatacenterID,
		ErrClockMovedBackwards,
		ErrInvalidSnowflakeID,
		ErrInvalidBatchSize,
		ErrNilConfig,
		ErrGeneratorNotFound,
		ErrGeneratorAlreadyExists,
		ErrInvalidGeneratorType,
		ErrInvalidKey,
		ErrFactoryNotFound,
		ErrMaxGeneratorsReached,
		ErrParserNotFound,
		ErrValidatorNotFound,
		ErrInvalidKeyFormat,
	}

	const goroutines = 1000
	const iterations = 10000

	done := make(chan struct{})

	// 并发读取错误常量
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				for _, err := range errorList {
					_ = err.Error()
					_ = errors.Is(err, err)
				}
			}
			done <- struct{}{}
		}()
	}

	// 等待完成
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// BenchmarkGeneratorType_String 基准测试：GeneratorType.String()
func BenchmarkGeneratorType_String(b *testing.B) {
	gt := GeneratorTypeSnowflake
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gt.String()
	}
}

// BenchmarkGeneratorType_IsValid 基准测试：GeneratorType.IsValid()
func BenchmarkGeneratorType_IsValid(b *testing.B) {
	gt := GeneratorTypeSnowflake
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gt.IsValid()
	}
}

// BenchmarkClockBackwardStrategy_String 基准测试：Strategy.String()
func BenchmarkClockBackwardStrategy_String(b *testing.B) {
	s := StrategyWait
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.String()
	}
}

// BenchmarkClockBackwardStrategy_IsValid 基准测试：Strategy.IsValid()
func BenchmarkClockBackwardStrategy_IsValid(b *testing.B) {
	s := StrategyWait
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.IsValid()
	}
}
