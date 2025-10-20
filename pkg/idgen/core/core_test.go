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

// BenchmarkGeneratorTypeString 基准测试：生成器类型转字符串
func BenchmarkGeneratorTypeString(b *testing.B) {
	genType := GeneratorTypeSnowflake
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = genType.String()
	}
}

// BenchmarkGeneratorTypeIsValid 基准测试：生成器类型验证
func BenchmarkGeneratorTypeIsValid(b *testing.B) {
	genType := GeneratorTypeSnowflake
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = genType.IsValid()
	}
}

// BenchmarkClockBackwardStrategyString 基准测试：策略转字符串
func BenchmarkClockBackwardStrategyString(b *testing.B) {
	strategy := StrategyError
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.String()
	}
}
