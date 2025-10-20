package core_test

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"katydid-common-account/pkg/idgen/core"
)

// ============================================================================
// 1. 基础功能测试
// ============================================================================

// TestGeneratorType 测试生成器类型枚举
func TestGeneratorType(t *testing.T) {
	tests := []struct {
		name      string
		genType   core.GeneratorType
		wantValid bool
		wantStr   string
	}{
		{
			name:      "Snowflake类型_有效",
			genType:   core.GeneratorTypeSnowflake,
			wantValid: true,
			wantStr:   "snowflake",
		},
		{
			name:      "UUID类型_有效",
			genType:   core.GeneratorTypeUUID,
			wantValid: true,
			wantStr:   "uuid",
		},
		{
			name:      "Custom类型_有效",
			genType:   core.GeneratorTypeCustom,
			wantValid: true,
			wantStr:   "custom",
		},
		{
			name:      "无效类型",
			genType:   core.GeneratorType("invalid"),
			wantValid: false,
			wantStr:   "invalid",
		},
		{
			name:      "空类型",
			genType:   core.GeneratorType(""),
			wantValid: false,
			wantStr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试IsValid方法
			if got := tt.genType.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}

			// 测试String方法
			if got := tt.genType.String(); got != tt.wantStr {
				t.Errorf("String() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

// TestClockBackwardStrategy 测试时钟回拨策略
func TestClockBackwardStrategy(t *testing.T) {
	tests := []struct {
		name      string
		strategy  core.ClockBackwardStrategy
		wantValid bool
		wantStr   string
	}{
		{
			name:      "Error策略",
			strategy:  core.StrategyError,
			wantValid: true,
			wantStr:   "Error",
		},
		{
			name:      "Wait策略",
			strategy:  core.StrategyWait,
			wantValid: true,
			wantStr:   "Wait",
		},
		{
			name:      "UseLastTimestamp策略",
			strategy:  core.StrategyUseLastTimestamp,
			wantValid: true,
			wantStr:   "UseLastTimestamp",
		},
		{
			name:      "无效策略_负数",
			strategy:  core.ClockBackwardStrategy(-1),
			wantValid: false,
			wantStr:   "Unknown",
		},
		{
			name:      "无效策略_超出范围",
			strategy:  core.ClockBackwardStrategy(999),
			wantValid: false,
			wantStr:   "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试IsValid方法
			if got := tt.strategy.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}

			// 测试String方法
			if got := tt.strategy.String(); got != tt.wantStr {
				t.Errorf("String() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

// TestErrors 测试错误定义
func TestErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrInvalidWorkerID", core.ErrInvalidWorkerID, "invalid worker id"},
		{"ErrInvalidDatacenterID", core.ErrInvalidDatacenterID, "invalid datacenter id"},
		{"ErrClockMovedBackwards", core.ErrClockMovedBackwards, "clock moved backwards"},
		{"ErrInvalidSnowflakeID", core.ErrInvalidSnowflakeID, "invalid snowflake id"},
		{"ErrInvalidBatchSize", core.ErrInvalidBatchSize, "invalid batch size"},
		{"ErrNilConfig", core.ErrNilConfig, "config cannot be nil"},
		{"ErrGeneratorNotFound", core.ErrGeneratorNotFound, "generator not found"},
		{"ErrGeneratorAlreadyExists", core.ErrGeneratorAlreadyExists, "generator already exists"},
		{"ErrInvalidGeneratorType", core.ErrInvalidGeneratorType, "invalid generator type"},
		{"ErrInvalidKey", core.ErrInvalidKey, "invalid key"},
		{"ErrFactoryNotFound", core.ErrFactoryNotFound, "factory not found"},
		{"ErrMaxGeneratorsReached", core.ErrMaxGeneratorsReached, "maximum number of generators reached"},
		{"ErrParserNotFound", core.ErrParserNotFound, "parser not found"},
		{"ErrValidatorNotFound", core.ErrValidatorNotFound, "validator not found"},
		{"ErrInvalidKeyFormat", core.ErrInvalidKeyFormat, "invalid key format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("错误不应为nil")
			}
			errMsg := tt.err.Error()
			if len(errMsg) == 0 {
				t.Error("错误消息不应为空")
			}
			// 检查错误消息是否包含预期关键字
			if !contains(errMsg, tt.wantMsg) {
				t.Errorf("错误消息 = %q, 应包含 %q", errMsg, tt.wantMsg)
			}
		})
	}
}

// TestErrorsIs 测试errors.Is兼容性
func TestErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "相同错误",
			err:    core.ErrInvalidWorkerID,
			target: core.ErrInvalidWorkerID,
			want:   true,
		},
		{
			name:   "包装的错误",
			err:    fmt.Errorf("wrapped: %w", core.ErrInvalidWorkerID),
			target: core.ErrInvalidWorkerID,
			want:   true,
		},
		{
			name:   "不同错误",
			err:    core.ErrInvalidWorkerID,
			target: core.ErrInvalidDatacenterID,
			want:   false,
		},
		{
			name:   "nil错误",
			err:    nil,
			target: core.ErrInvalidWorkerID,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIDInfo 测试IDInfo结构
func TestIDInfo(t *testing.T) {
	info := &core.IDInfo{
		ID:           123456789,
		Timestamp:    1672502400000,
		DatacenterID: 5,
		WorkerID:     10,
		Sequence:     100,
	}

	// 验证字段赋值
	if info.ID != 123456789 {
		t.Errorf("ID = %v, want %v", info.ID, 123456789)
	}
	if info.Timestamp != 1672502400000 {
		t.Errorf("Timestamp = %v, want %v", info.Timestamp, 1672502400000)
	}
	if info.DatacenterID != 5 {
		t.Errorf("DatacenterID = %v, want %v", info.DatacenterID, 5)
	}
	if info.WorkerID != 10 {
		t.Errorf("WorkerID = %v, want %v", info.WorkerID, 10)
	}
	if info.Sequence != 100 {
		t.Errorf("Sequence = %v, want %v", info.Sequence, 100)
	}
}

// ============================================================================
// 2. 并发测试
// ============================================================================

// TestGeneratorType_Concurrent 测试GeneratorType的并发安全性
func TestGeneratorType_Concurrent(t *testing.T) {
	const goroutines = 1000
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 并发调用IsValid和String
				_ = core.GeneratorTypeSnowflake.IsValid()
				_ = core.GeneratorTypeSnowflake.String()
				_ = core.GeneratorTypeUUID.IsValid()
				_ = core.GeneratorTypeCustom.String()
			}
		}()
	}

	wg.Wait()
}

// TestClockBackwardStrategy_Concurrent 测试ClockBackwardStrategy的并发安全性
func TestClockBackwardStrategy_Concurrent(t *testing.T) {
	const goroutines = 1000
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 并发调用IsValid和String
				_ = core.StrategyError.IsValid()
				_ = core.StrategyError.String()
				_ = core.StrategyWait.IsValid()
				_ = core.StrategyUseLastTimestamp.String()
			}
		}()
	}

	wg.Wait()
}

// ============================================================================
// 3. 百万级高并发测试
// ============================================================================

// TestGeneratorType_MillionConcurrent 百万级并发测试GeneratorType
func TestGeneratorType_MillionConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	const totalOps = 1_000_000
	goroutines := runtime.NumCPU() * 100
	opsPerGoroutine := totalOps / goroutines

	t.Logf("开始百万级并发测试: 总操作=%d, 协程数=%d, 每协程操作=%d",
		totalOps, goroutines, opsPerGoroutine)

	startTime := time.Now()
	var wg sync.WaitGroup
	var successCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			localSuccess := 0

			for j := 0; j < opsPerGoroutine; j++ {
				// 测试不同类型的操作
				switch j % 4 {
				case 0:
					if core.GeneratorTypeSnowflake.IsValid() {
						localSuccess++
					}
				case 1:
					if len(core.GeneratorTypeSnowflake.String()) > 0 {
						localSuccess++
					}
				case 2:
					if core.GeneratorTypeUUID.IsValid() {
						localSuccess++
					}
				case 3:
					if len(core.GeneratorTypeCustom.String()) > 0 {
						localSuccess++
					}
				}
			}

			successCount.Add(int64(localSuccess))
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 统计结果
	t.Logf("百万级并发测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 成功操作: %d", successCount.Load())
	t.Logf("  - QPS: %.2f ops/sec", float64(totalOps)/duration.Seconds())
	t.Logf("  - 平均延迟: %v", duration/time.Duration(totalOps))

	// 验证所有操作都成功
	if successCount.Load() != int64(totalOps) {
		t.Errorf("成功操作数 = %d, 期望 = %d", successCount.Load(), totalOps)
	}
}

// TestClockBackwardStrategy_MillionConcurrent 百万级并发测试ClockBackwardStrategy
func TestClockBackwardStrategy_MillionConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	const totalOps = 1_000_000
	goroutines := runtime.NumCPU() * 100
	opsPerGoroutine := totalOps / goroutines

	t.Logf("开始百万级并发测试: 总操作=%d, 协程数=%d", totalOps, goroutines)

	startTime := time.Now()
	var wg sync.WaitGroup
	var validCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			localValid := 0

			for j := 0; j < opsPerGoroutine; j++ {
				strategy := core.ClockBackwardStrategy(j % 3)
				if strategy.IsValid() {
					localValid++
				}
				_ = strategy.String()
			}

			validCount.Add(int64(localValid))
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级并发测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 有效策略数: %d", validCount.Load())
	t.Logf("  - QPS: %.2f ops/sec", float64(totalOps)/duration.Seconds())
}

// ============================================================================
// 4. 性能基准测试
// ============================================================================

// BenchmarkGeneratorType_IsValid 基准测试IsValid方法
func BenchmarkGeneratorType_IsValid(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = core.GeneratorTypeSnowflake.IsValid()
	}
}

// BenchmarkGeneratorType_String 基准测试String方法
func BenchmarkGeneratorType_String(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = core.GeneratorTypeSnowflake.String()
	}
}

// BenchmarkClockBackwardStrategy_IsValid 基准测试IsValid方法
func BenchmarkClockBackwardStrategy_IsValid(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = core.StrategyError.IsValid()
	}
}

// BenchmarkClockBackwardStrategy_String 基准测试String方法
func BenchmarkClockBackwardStrategy_String(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = core.StrategyError.String()
	}
}

// BenchmarkGeneratorType_Parallel 并行基准测试GeneratorType
func BenchmarkGeneratorType_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = core.GeneratorTypeSnowflake.IsValid()
			_ = core.GeneratorTypeSnowflake.String()
		}
	})
}

// BenchmarkClockBackwardStrategy_Parallel 并行基准测试ClockBackwardStrategy
func BenchmarkClockBackwardStrategy_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = core.StrategyError.IsValid()
			_ = core.StrategyError.String()
		}
	})
}

// BenchmarkErrorsIs 基准测试errors.Is性能
func BenchmarkErrorsIs(b *testing.B) {
	wrappedErr := fmt.Errorf("wrapped: %w", core.ErrInvalidWorkerID)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = errors.Is(wrappedErr, core.ErrInvalidWorkerID)
	}
}

// ============================================================================
// 5. 内存性能测试
// ============================================================================

// BenchmarkIDInfo_Memory 测试IDInfo内存分配
func BenchmarkIDInfo_Memory(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		info := &core.IDInfo{
			ID:           int64(i),
			Timestamp:    1672502400000,
			DatacenterID: 5,
			WorkerID:     10,
			Sequence:     100,
		}
		_ = info
	}
}

// ============================================================================
// 6. 压力测试
// ============================================================================

// TestGeneratorType_StressTest 压力测试GeneratorType
func TestGeneratorType_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	const duration = 5 * time.Second
	const goroutines = 1000

	t.Logf("开始压力测试: 持续时间=%v, 协程数=%d", duration, goroutines)

	startTime := time.Now()
	stopTime := startTime.Add(duration)
	var wg sync.WaitGroup
	var totalOps atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			localOps := 0

			for time.Now().Before(stopTime) {
				_ = core.GeneratorTypeSnowflake.IsValid()
				_ = core.GeneratorTypeSnowflake.String()
				localOps++
			}

			totalOps.Add(int64(localOps))
		}()
	}

	wg.Wait()
	actualDuration := time.Since(startTime)

	// 统计结果
	ops := totalOps.Load()
	t.Logf("压力测试完成:")
	t.Logf("  - 实际耗时: %v", actualDuration)
	t.Logf("  - 总操作数: %d", ops)
	t.Logf("  - QPS: %.2f ops/sec", float64(ops)/actualDuration.Seconds())
	t.Logf("  - 每秒每协程: %.2f ops/sec", float64(ops)/actualDuration.Seconds()/float64(goroutines))
}

// ============================================================================
// 辅助函数
// ============================================================================

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
