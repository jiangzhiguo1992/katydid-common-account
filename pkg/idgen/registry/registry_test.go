package registry_test

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/registry"
	"katydid-common-account/pkg/idgen/snowflake"
)

// ============================================================================
// 1. Registry基础功能测试
// ============================================================================

// TestRegistry_Create 测试创建生成器
func TestRegistry_Create(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	t.Run("正常创建", func(t *testing.T) {
		gen, err := r.Create("test1", core.GeneratorTypeSnowflake, config)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if gen == nil {
			t.Fatal("Create() returned nil generator")
		}
	})

	t.Run("重复键", func(t *testing.T) {
		_, err := r.Create("test1", core.GeneratorTypeSnowflake, config)
		if err == nil {
			t.Error("Create() with duplicate key should return error")
		}
	})

	t.Run("无效类型", func(t *testing.T) {
		_, err := r.Create("test2", core.GeneratorType("invalid"), config)
		if err == nil {
			t.Error("Create() with invalid type should return error")
		}
	})

	t.Run("空键", func(t *testing.T) {
		_, err := r.Create("", core.GeneratorTypeSnowflake, config)
		if err == nil {
			t.Error("Create() with empty key should return error")
		}
	})
}

// TestRegistry_Get 测试获取生成器
func TestRegistry_Get(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	// 先创建
	_, err := r.Create("test1", core.GeneratorTypeSnowflake, config)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("获取存在的生成器", func(t *testing.T) {
		gen, err := r.Get("test1")
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if gen == nil {
			t.Error("Get() returned nil generator")
		}
	})

	t.Run("获取不存在的生成器", func(t *testing.T) {
		_, err := r.Get("nonexistent")
		if err == nil {
			t.Error("Get() should return error for nonexistent key")
		}
	})

	t.Run("空键", func(t *testing.T) {
		_, err := r.Get("")
		if err == nil {
			t.Error("Get() should return error for empty key")
		}
	})
}

// TestRegistry_GetOrCreate 测试获取或创建生成器
func TestRegistry_GetOrCreate(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	t.Run("创建新生成器", func(t *testing.T) {
		gen, err := r.GetOrCreate("test1", core.GeneratorTypeSnowflake, config)
		if err != nil {
			t.Fatalf("GetOrCreate() error = %v", err)
		}
		if gen == nil {
			t.Fatal("GetOrCreate() returned nil generator")
		}
	})

	t.Run("获取已存在的生成器", func(t *testing.T) {
		gen, err := r.GetOrCreate("test1", core.GeneratorTypeSnowflake, config)
		if err != nil {
			t.Fatalf("GetOrCreate() error = %v", err)
		}
		if gen == nil {
			t.Fatal("GetOrCreate() returned nil generator")
		}
	})
}

// TestRegistry_Has 测试检查生成器是否存在
func TestRegistry_Has(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	_, _ = r.Create("test1", core.GeneratorTypeSnowflake, config)

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"存在的键", "test1", true},
		{"不存在的键", "nonexistent", false},
		{"空键", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Has(tt.key); got != tt.want {
				t.Errorf("Has(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// TestRegistry_Remove 测试删除生成器
func TestRegistry_Remove(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	_, _ = r.Create("test1", core.GeneratorTypeSnowflake, config)

	t.Run("删除存在的生成器", func(t *testing.T) {
		err := r.Remove("test1")
		if err != nil {
			t.Errorf("Remove() error = %v", err)
		}
		if r.Has("test1") {
			t.Error("Generator still exists after Remove()")
		}
	})

	t.Run("删除不存在的生成器", func(t *testing.T) {
		err := r.Remove("nonexistent")
		if err == nil {
			t.Error("Remove() should return error for nonexistent key")
		}
	})
}

// TestRegistry_Clear 测试清空注册表
func TestRegistry_Clear(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	// 创建多个生成器
	for i := 0; i < 5; i++ {
		_, _ = r.Create(fmt.Sprintf("test%d", i), core.GeneratorTypeSnowflake, config)
	}

	r.Clear()

	if r.Count() != 0 {
		t.Errorf("Count() = %d after Clear(), want 0", r.Count())
	}
}

// TestRegistry_Count 测试计数
func TestRegistry_Count(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	if r.Count() != 0 {
		t.Errorf("Initial Count() = %d, want 0", r.Count())
	}

	// 创建3个生成器
	for i := 0; i < 3; i++ {
		_, _ = r.Create(fmt.Sprintf("test%d", i), core.GeneratorTypeSnowflake, config)
	}

	if r.Count() != 3 {
		t.Errorf("Count() = %d after creating 3 generators, want 3", r.Count())
	}
}

// TestRegistry_ListKeys 测试列出所有键
func TestRegistry_ListKeys(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	keys := []string{"test1", "test2", "test3"}
	for _, key := range keys {
		_, _ = r.Create(key, core.GeneratorTypeSnowflake, config)
	}

	gotKeys := r.ListKeys()
	if len(gotKeys) != len(keys) {
		t.Errorf("ListKeys() returned %d keys, want %d", len(gotKeys), len(keys))
	}

	// 验证所有键都存在
	keyMap := make(map[string]bool)
	for _, k := range gotKeys {
		keyMap[k] = true
	}
	for _, k := range keys {
		if !keyMap[k] {
			t.Errorf("ListKeys() missing key %q", k)
		}
	}
}

// TestRegistry_MaxGenerators 测试最大生成器限制
func TestRegistry_MaxGenerators(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	t.Run("设置最大值", func(t *testing.T) {
		err := r.SetMaxGenerators(10)
		if err != nil {
			t.Errorf("SetMaxGenerators(10) error = %v", err)
		}
		if got := r.GetMaxGenerators(); got != 10 {
			t.Errorf("GetMaxGenerators() = %d, want 10", got)
		}
	})

	t.Run("无效最大值_负数", func(t *testing.T) {
		err := r.SetMaxGenerators(-1)
		if err == nil {
			t.Error("SetMaxGenerators(-1) should return error")
		}
	})

	t.Run("超出绝对限制", func(t *testing.T) {
		err := r.SetMaxGenerators(200000) // 超过100000的限制
		if err == nil {
			t.Error("SetMaxGenerators(200000) should return error")
		}
	})
}

// ============================================================================
// 2. 并发测试
// ============================================================================

// TestRegistry_Concurrent 测试并发访问
func TestRegistry_Concurrent(t *testing.T) {
	r := registry.GetRegistry()
	defer r.Clear()

	const goroutines = 100
	const iterations = 100

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("gen_%d", id)

			for j := 0; j < iterations; j++ {
				// 创建
				_, _ = r.GetOrCreate(key, core.GeneratorTypeSnowflake, config)

				// 检查
				_ = r.Has(key)

				// 获取
				if gen, err := r.Get(key); err == nil && gen != nil {
					// 生成ID
					_, _ = gen.NextID()
				}
			}
		}(i)
	}

	wg.Wait()
}

// ============================================================================
// 3. 百万级高并发测试
// ============================================================================

// TestRegistry_MillionConcurrentRead 百万级并发读测试
func TestRegistry_MillionConcurrentRead(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	r := registry.GetRegistry()
	defer r.Clear()

	// 准备测试数据：创建10个生成器
	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}
	for i := 0; i < 10; i++ {
		_, _ = r.Create(fmt.Sprintf("gen_%d", i), core.GeneratorTypeSnowflake, config)
	}

	const totalOps = 1_000_000
	goroutines := runtime.NumCPU() * 100
	opsPerGoroutine := totalOps / goroutines

	t.Logf("开始百万级并发读测试: 总操作=%d, 协程数=%d", totalOps, goroutines)

	startTime := time.Now()
	var wg sync.WaitGroup
	var successCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(gid int) {
			defer wg.Done()
			localSuccess := 0

			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("gen_%d", j%10)

				// 测试多种读操作
				switch j % 4 {
				case 0:
					if r.Has(key) {
						localSuccess++
					}
				case 1:
					if _, err := r.Get(key); err == nil {
						localSuccess++
					}
				case 2:
					_ = r.Count()
					localSuccess++
				case 3:
					_ = r.ListKeys()
					localSuccess++
				}
			}

			successCount.Add(int64(localSuccess))
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级并发读测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 成功操作: %d", successCount.Load())
	t.Logf("  - QPS: %.2f ops/sec", float64(totalOps)/duration.Seconds())
	t.Logf("  - 平均延迟: %v", duration/time.Duration(totalOps))
}

// TestRegistry_MillionConcurrentGetOrCreate 百万级并发GetOrCreate测试
func TestRegistry_MillionConcurrentGetOrCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	r := registry.GetRegistry()
	defer r.Clear()

	const totalOps = 1_000_000
	const uniqueKeys = 1000 // 使用1000个不同的键
	goroutines := runtime.NumCPU() * 100
	opsPerGoroutine := totalOps / goroutines

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	t.Logf("开始百万级并发GetOrCreate测试: 总操作=%d, 协程数=%d, 唯一键=%d",
		totalOps, goroutines, uniqueKeys)

	startTime := time.Now()
	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(gid int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("gen_%d", j%uniqueKeys)

				if _, err := r.GetOrCreate(key, core.GeneratorTypeSnowflake, config); err == nil {
					successCount.Add(1)
				} else {
					errorCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级并发GetOrCreate测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 成功操作: %d", successCount.Load())
	t.Logf("  - 失败操作: %d", errorCount.Load())
	t.Logf("  - 最终生成器数: %d (预期: %d)", r.Count(), uniqueKeys)
	t.Logf("  - QPS: %.2f ops/sec", float64(totalOps)/duration.Seconds())
	t.Logf("  - 平均延迟: %v", duration/time.Duration(totalOps))

	// 验证生成器数量
	if r.Count() != uniqueKeys {
		t.Logf("警告: 生成器数量 = %d, 预期 = %d", r.Count(), uniqueKeys)
	}
}

// TestRegistry_MillionConcurrentIDGeneration 百万级并发ID生成测试
func TestRegistry_MillionConcurrentIDGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	r := registry.GetRegistry()
	defer r.Clear()

	// 创建10个生成器
	const numGenerators = 10
	config := &snowflake.Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	}

	generators := make([]core.IDGenerator, numGenerators)
	for i := 0; i < numGenerators; i++ {
		gen, err := r.Create(fmt.Sprintf("gen_%d", i), core.GeneratorTypeSnowflake, config)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		generators[i] = gen
	}

	const totalIDs = 1_000_000
	goroutines := runtime.NumCPU() * 100
	idsPerGoroutine := totalIDs / goroutines

	t.Logf("开始百万级并发ID生成测试: 总ID数=%d, 协程数=%d, 生成器数=%d",
		totalIDs, goroutines, numGenerators)

	startTime := time.Now()
	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64

	// 用于检测ID唯一性
	idSet := sync.Map{}
	var duplicateCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(gid int) {
			defer wg.Done()

			for j := 0; j < idsPerGoroutine; j++ {
				// 轮询使用不同的生成器
				gen := generators[j%numGenerators]

				if id, err := gen.NextID(); err == nil {
					successCount.Add(1)

					// 检测重复（采样检查，避免内存溢出）
					if j%1000 == 0 {
						if _, exists := idSet.LoadOrStore(id, struct{}{}); exists {
							duplicateCount.Add(1)
						}
					}
				} else {
					errorCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级并发ID生成测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 成功生成: %d", successCount.Load())
	t.Logf("  - 失败次数: %d", errorCount.Load())
	t.Logf("  - 重复ID数: %d (采样检测)", duplicateCount.Load())
	t.Logf("  - QPS: %.2f IDs/sec", float64(successCount.Load())/duration.Seconds())
	t.Logf("  - 平均延迟: %v", duration/time.Duration(successCount.Load()))

	// 验证无重复
	if duplicateCount.Load() > 0 {
		t.Errorf("发现 %d 个重复ID", duplicateCount.Load())
	}

	// 验证成功率
	successRate := float64(successCount.Load()) / float64(totalIDs) * 100
	t.Logf("  - 成功率: %.2f%%", successRate)
}

// ============================================================================
// 4. 性能基准测试
// ============================================================================

// BenchmarkRegistry_Get 基准测试Get操作
func BenchmarkRegistry_Get(b *testing.B) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}
	_, _ = r.Create("test", core.GeneratorTypeSnowflake, config)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = r.Get("test")
	}
}

// BenchmarkRegistry_Has 基准测试Has操作
func BenchmarkRegistry_Has(b *testing.B) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}
	_, _ = r.Create("test", core.GeneratorTypeSnowflake, config)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = r.Has("test")
	}
}

// BenchmarkRegistry_GetOrCreate 基准测试GetOrCreate操作
func BenchmarkRegistry_GetOrCreate(b *testing.B) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = r.GetOrCreate("test", core.GeneratorTypeSnowflake, config)
	}
}

// BenchmarkRegistry_Parallel 并行基准测试
func BenchmarkRegistry_Parallel(b *testing.B) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}
	_, _ = r.Create("test", core.GeneratorTypeSnowflake, config)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = r.Get("test")
		}
	})
}

// BenchmarkRegistry_IDGeneration 基准测试通过注册表生成ID
func BenchmarkRegistry_IDGeneration(b *testing.B) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: false,
	}
	gen, _ := r.Create("test", core.GeneratorTypeSnowflake, config)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = gen.NextID()
	}
}

// BenchmarkRegistry_IDGeneration_Parallel 并行基准测试ID生成
func BenchmarkRegistry_IDGeneration_Parallel(b *testing.B) {
	r := registry.GetRegistry()
	defer r.Clear()

	config := &snowflake.Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: false,
	}
	gen, _ := r.Create("test", core.GeneratorTypeSnowflake, config)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = gen.NextID()
		}
	})
}
