package registry

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"katydid-common-account/pkg/idgen/core"
	"katydid-common-account/pkg/idgen/snowflake"
)

// TestGetRegistry 测试获取注册表单例
func TestGetRegistry(t *testing.T) {
	r1 := GetRegistry()
	r2 := GetRegistry()

	if r1 != r2 {
		t.Error("GetRegistry应该返回相同的单例实例")
	}

	if r1 == nil {
		t.Error("注册表实例不应为nil")
	}
}

// TestRegistryCreate 测试创建生成器
func TestRegistryCreate(t *testing.T) {
	registry := GetRegistry()
	defer registry.Clear() // 清理

	t.Run("创建成功", func(t *testing.T) {
		config := &snowflake.Config{
			DatacenterID: 1,
			WorkerID:     1,
		}

		gen, err := registry.Create("test-gen-1", core.GeneratorTypeSnowflake, config)
		if err != nil {
			t.Fatalf("创建失败: %v", err)
		}
		if gen == nil {
			t.Error("生成器不应为nil")
		}
	})

	t.Run("无效键_空字符串", func(t *testing.T) {
		config := &snowflake.Config{
			DatacenterID: 1,
			WorkerID:     1,
		}
		_, err := registry.Create("", core.GeneratorTypeSnowflake, config)
		if err == nil {
			t.Error("期望得到错误")
		}
	})

	t.Run("无效生成器类型", func(t *testing.T) {
		config := &snowflake.Config{
			DatacenterID: 1,
			WorkerID:     1,
		}
		_, err := registry.Create("test-invalid", core.GeneratorType("invalid"), config)
		if err == nil {
			t.Error("期望得到错误")
		}
	})

	t.Run("重复创建", func(t *testing.T) {
		config := &snowflake.Config{
			DatacenterID: 2,
			WorkerID:     2,
		}

		key := "test-duplicate"
		_, err := registry.Create(key, core.GeneratorTypeSnowflake, config)
		if err != nil {
			t.Fatalf("第一次创建失败: %v", err)
		}

		_, err = registry.Create(key, core.GeneratorTypeSnowflake, config)
		if err == nil {
			t.Error("期望得到重复错误")
		}
	})
}

// TestRegistryGet 测试获取生成器
func TestRegistryGet(t *testing.T) {
	registry := GetRegistry()
	defer registry.Clear()

	config := &snowflake.Config{
		DatacenterID: 3,
		WorkerID:     3,
	}

	key := "test-get-1"
	_, err := registry.Create(key, core.GeneratorTypeSnowflake, config)
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}

	t.Run("获取存在的生成器", func(t *testing.T) {
		gen, err := registry.Get(key)
		if err != nil {
			t.Errorf("获取失败: %v", err)
		}
		if gen == nil {
			t.Error("生成器不应为nil")
		}
	})

	t.Run("获取不存在的生成器", func(t *testing.T) {
		_, err := registry.Get("non-existent")
		if err == nil {
			t.Error("期望得到错误")
		}
	})
}

// TestRegistryGetOrCreate 测试获取或创建
func TestRegistryGetOrCreate(t *testing.T) {
	registry := GetRegistry()
	defer registry.Clear()

	config := &snowflake.Config{
		DatacenterID: 4,
		WorkerID:     4,
	}

	key := "test-get-or-create"

	gen1, err := registry.GetOrCreate(key, core.GeneratorTypeSnowflake, config)
	if err != nil {
		t.Fatalf("第一次GetOrCreate失败: %v", err)
	}

	gen2, err := registry.GetOrCreate(key, core.GeneratorTypeSnowflake, config)
	if err != nil {
		t.Fatalf("第二次GetOrCreate失败: %v", err)
	}

	if gen1 != gen2 {
		t.Error("应该返回相同的实例")
	}
}

// TestRegistryHas 测试检查存在性
func TestRegistryHas(t *testing.T) {
	registry := GetRegistry()
	defer registry.Clear()

	config := &snowflake.Config{
		DatacenterID: 5,
		WorkerID:     5,
	}

	key := "test-has"
	_, _ = registry.Create(key, core.GeneratorTypeSnowflake, config)

	if !registry.Has(key) {
		t.Error("应该存在")
	}

	if registry.Has("non-existent") {
		t.Error("不应该存在")
	}
}

// TestRegistryRemove 测试移除生成器
func TestRegistryRemove(t *testing.T) {
	registry := GetRegistry()
	defer registry.Clear()

	config := &snowflake.Config{
		DatacenterID: 6,
		WorkerID:     6,
	}

	key := "test-remove"
	_, _ = registry.Create(key, core.GeneratorTypeSnowflake, config)

	if !registry.Has(key) {
		t.Fatal("生成器应该存在")
	}

	err := registry.Remove(key)
	if err != nil {
		t.Errorf("移除失败: %v", err)
	}

	if registry.Has(key) {
		t.Error("生成器应该已被移除")
	}
}

// TestRegistryClear 测试清空注册表
func TestRegistryClear(t *testing.T) {
	registry := GetRegistry()

	config := &snowflake.Config{
		DatacenterID: 7,
		WorkerID:     7,
	}

	// 创建几个生成器
	for i := 0; i < 3; i++ {
		_, _ = registry.Create("test-clear-"+string(rune(i)), core.GeneratorTypeSnowflake, config)
	}

	registry.Clear()

	// 验证已清空
	if registry.Has("test-clear-0") {
		t.Error("注册表应该已被清空")
	}
}

// TestFactoryRegistry 测试工厂注册表
func TestFactoryRegistry(t *testing.T) {
	factoryRegistry := GetFactoryRegistry()

	t.Run("获取Snowflake工厂", func(t *testing.T) {
		factory, err := factoryRegistry.Get(core.GeneratorTypeSnowflake)
		if err != nil {
			t.Errorf("获取工厂失败: %v", err)
		}
		if factory == nil {
			t.Error("工厂不应为nil")
		}
	})

	t.Run("获取不存在的工厂", func(t *testing.T) {
		_, err := factoryRegistry.Get(core.GeneratorType("unknown"))
		if err == nil {
			t.Error("期望得到错误")
		}
	})

	t.Run("Has方法", func(t *testing.T) {
		if !factoryRegistry.Has(core.GeneratorTypeSnowflake) {
			t.Error("应该有Snowflake工厂")
		}
		if factoryRegistry.Has(core.GeneratorType("unknown")) {
			t.Error("不应该有unknown工厂")
		}
	})

	t.Run("List方法", func(t *testing.T) {
		types := factoryRegistry.List()
		if len(types) == 0 {
			t.Error("至少应该有一个工厂类型")
		}
		found := false
		for _, t := range types {
			if t == core.GeneratorTypeSnowflake {
				found = true
				break
			}
		}
		if !found {
			t.Error("应该包含Snowflake类型")
		}
	})
}

// TestSnowflakeFactory 测试Snowflake工厂
func TestSnowflakeFactory(t *testing.T) {
	factory := NewSnowflakeFactory()

	t.Run("创建成功", func(t *testing.T) {
		config := &snowflake.Config{
			DatacenterID: 8,
			WorkerID:     8,
		}

		gen, err := factory.Create(config)
		if err != nil {
			t.Fatalf("创建失败: %v", err)
		}
		if gen == nil {
			t.Error("生成器不应为nil")
		}

		// 验证接口实现
		_, ok := gen.(core.IDGenerator)
		if !ok {
			t.Error("应该实现IDGenerator接口")
		}
	})

	t.Run("无效配置类型", func(t *testing.T) {
		_, err := factory.Create("invalid")
		if err == nil {
			t.Error("期望得到错误")
		}
	})

	t.Run("nil配置", func(t *testing.T) {
		_, err := factory.Create(nil)
		if err == nil {
			t.Error("期望得到错误")
		}
	})
}

// TestParserRegistry 测试解析器注册表
func TestParserRegistry(t *testing.T) {
	parserRegistry := GetParserRegistry()

	t.Run("获取Snowflake解析器", func(t *testing.T) {
		parser, err := parserRegistry.Get(core.GeneratorTypeSnowflake)
		if err != nil {
			t.Errorf("获取解析器失败: %v", err)
		}
		if parser == nil {
			t.Error("解析器不应为nil")
		}
	})

	t.Run("Has方法", func(t *testing.T) {
		if !parserRegistry.Has(core.GeneratorTypeSnowflake) {
			t.Error("应该有Snowflake解析器")
		}
	})
}

// TestValidatorRegistry 测试验证器注册表
func TestValidatorRegistry(t *testing.T) {
	validatorRegistry := GetValidatorRegistry()

	t.Run("获取Snowflake验证器", func(t *testing.T) {
		validator, err := validatorRegistry.Get(core.GeneratorTypeSnowflake)
		if err != nil {
			t.Errorf("获取验证器失败: %v", err)
		}
		if validator == nil {
			t.Error("验证器不应为nil")
		}
	})

	t.Run("Has方法", func(t *testing.T) {
		if !validatorRegistry.Has(core.GeneratorTypeSnowflake) {
			t.Error("应该有Snowflake验证器")
		}
	})
}

// TestGetDefaultGenerator 测试获取默认生成器
func TestGetDefaultGenerator(t *testing.T) {
	gen1, err := GetDefaultGenerator()
	if err != nil {
		t.Fatalf("获取默认生成器失败: %v", err)
	}
	if gen1 == nil {
		t.Error("默认生成器不应为nil")
	}

	gen2, err := GetDefaultGenerator()
	if err != nil {
		t.Fatalf("第二次获取失败: %v", err)
	}

	if gen1 != gen2 {
		t.Error("应该返回相同的默认生成器实例")
	}

	// 测试生成ID
	id, err := gen1.NextID()
	if err != nil {
		t.Errorf("生成ID失败: %v", err)
	}
	if id <= 0 {
		t.Errorf("生成的ID应为正数，得到: %d", id)
	}
}

// TestGetOrCreateDefaultGenerator 测试获取或创建默认生成器
func TestGetOrCreateDefaultGenerator(t *testing.T) {
	gen, err := GetOrCreateDefaultGenerator()
	if err != nil {
		t.Fatalf("获取失败: %v", err)
	}
	if gen == nil {
		t.Error("生成器不应为nil")
	}
}

// TestRegistryConcurrency 测试注册表并发安全性
func TestRegistryConcurrency(t *testing.T) {
	registry := GetRegistry()
	defer registry.Clear()

	goroutines := 50
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			config := &snowflake.Config{
				DatacenterID: int64(idx % 32),
				WorkerID:     int64(idx % 32),
			}

			key := "concurrent-test"
			gen, err := registry.GetOrCreate(key, core.GeneratorTypeSnowflake, config)
			if err != nil {
				t.Errorf("GetOrCreate失败: %v", err)
				return
			}

			// 生成一些ID
			for j := 0; j < 10; j++ {
				_, err := gen.NextID()
				if err != nil {
					t.Errorf("生成ID失败: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}

// ========== 高并发百万级测试（多维度性能分析） ==========

// TestRegistry_ConcurrentCreate 测试并发创建生成器
func TestRegistry_ConcurrentCreate(t *testing.T) {
	registry := &Registry{
		generators:    make(map[string]core.IDGenerator),
		maxGenerators: 1000,
	}

	const goroutines = 100
	const generatorsPerGoroutine = 10

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < generatorsPerGoroutine; j++ {
				key := fmt.Sprintf("gen_%d_%d", idx, j)
				config := &snowflake.Config{
					DatacenterID: int64(idx % 32),
					WorkerID:     int64(j % 32),
				}
				_, err := registry.Create(key, core.GeneratorTypeSnowflake, config)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("成功创建: %d, 失败: %d", successCount, errorCount)
	t.Logf("注册表大小: %d", len(registry.generators))

	expectedSuccess := int64(goroutines * generatorsPerGoroutine)
	if successCount != expectedSuccess {
		t.Errorf("成功数 %d 不等于期望 %d", successCount, expectedSuccess)
	}
}

// TestRegistry_ConcurrentGet 测试并发获取生成器
func TestRegistry_ConcurrentGet(t *testing.T) {
	registry := GetRegistry()
	registry.Clear()

	// 预先创建一些生成器
	const numGenerators = 100
	for i := 0; i < numGenerators; i++ {
		key := fmt.Sprintf("test_gen_%d", i)
		config := &snowflake.Config{
			DatacenterID: 1,
			WorkerID:     int64(i % 32),
		}
		_, err := registry.Create(key, core.GeneratorTypeSnowflake, config)
		if err != nil {
			t.Fatalf("创建生成器失败: %v", err)
		}
	}

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("test_gen_%d", j%numGenerators)
				_, err := registry.Get(key)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	t.Logf("成功获取: %d, 失败: %d", successCount, errorCount)

	if errorCount > 0 {
		t.Errorf("不应有获取失败: %d", errorCount)
	}
}

// TestRegistry_ConcurrentGetOrCreate 测试并发GetOrCreate
func TestRegistry_ConcurrentGetOrCreate(t *testing.T) {
	registry := &Registry{
		generators:    make(map[string]core.IDGenerator),
		maxGenerators: 1000,
	}

	const goroutines = 100
	const keys = 10 // 多个协程竞争同样的key

	var wg sync.WaitGroup
	var successCount int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("shared_gen_%d", idx%keys)
			config := &snowflake.Config{
				DatacenterID: 1,
				WorkerID:     1,
			}
			_, err := registry.GetOrCreate(key, core.GeneratorTypeSnowflake, config)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("成功: %d, 注册表大小: %d", successCount, len(registry.generators))

	if len(registry.generators) != keys {
		t.Errorf("注册表大小 %d 不等于期望 %d", len(registry.generators), keys)
	}
}

// TestRegistry_ConcurrentReadWrite 测试并发读写操作
func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	registry := &Registry{
		generators:    make(map[string]core.IDGenerator),
		maxGenerators: 1000,
	}

	// 预创建一些生成器
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("initial_gen_%d", i)
		config := &snowflake.Config{
			DatacenterID: 1,
			WorkerID:     int64(i % 32),
		}
		registry.Create(key, core.GeneratorTypeSnowflake, config)
	}

	const goroutines = 50
	const operations = 100

	var wg sync.WaitGroup
	var reads int64
	var writes int64
	var deletes int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				switch j % 3 {
				case 0: // 读操作
					key := fmt.Sprintf("initial_gen_%d", j%50)
					_, _ = registry.Get(key)
					atomic.AddInt64(&reads, 1)

				case 1: // 写操作
					key := fmt.Sprintf("new_gen_%d_%d", idx, j)
					config := &snowflake.Config{
						DatacenterID: int64(idx % 32),
						WorkerID:     int64(j % 32),
					}
					_, _ = registry.Create(key, core.GeneratorTypeSnowflake, config)
					atomic.AddInt64(&writes, 1)

				case 2: // 删除操作
					key := fmt.Sprintf("new_gen_%d_%d", idx, j-1)
					_ = registry.Remove(key)
					atomic.AddInt64(&deletes, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("读操作: %d, 写操作: %d, 删除操作: %d", reads, writes, deletes)
	t.Logf("最终注册表大小: %d", len(registry.generators))
}

// TestRegistry_StressTest 测试注册表压力测试
func TestRegistry_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	registry := &Registry{
		generators:    make(map[string]core.IDGenerator),
		maxGenerators: 10000,
	}

	const goroutines = 100
	const operationsPerGoroutine = 1000

	var wg sync.WaitGroup
	var createCount int64
	var getCount int64
	var removeCount int64
	var errorCount int64

	startTime := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("stress_gen_%d_%d", idx, j)

				// 创建
				config := &snowflake.Config{
					DatacenterID: int64(idx % 32),
					WorkerID:     int64(j % 32),
				}
				_, err := registry.Create(key, core.GeneratorTypeSnowflake, config)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				atomic.AddInt64(&createCount, 1)

				// 获取
				_, err = registry.Get(key)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&getCount, 1)
				}

				// 删除（部分）
				if j%2 == 0 {
					err = registry.Remove(key)
					if err == nil {
						atomic.AddInt64(&removeCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	t.Logf("============ 压力测试报告 ============")
	t.Logf("总耗时: %v", elapsed)
	t.Logf("创建数: %d", createCount)
	t.Logf("获取数: %d", getCount)
	t.Logf("删除数: %d", removeCount)
	t.Logf("错误数: %d", errorCount)
	t.Logf("最终大小: %d", len(registry.generators))
}

// TestFactoryRegistry_Concurrent 测试工厂注册表并发安全性
func TestFactoryRegistry_Concurrent(t *testing.T) {
	fr := GetFactoryRegistry()

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 并发获取工厂
				_, _ = fr.Get(core.GeneratorTypeSnowflake)

				// 并发检查
				_ = fr.Has(core.GeneratorTypeSnowflake)

				// 并发列出
				_ = fr.List()
			}
		}()
	}

	wg.Wait()
}

// TestParserRegistry_Concurrent 测试解析器注册表并发安全性
func TestParserRegistry_Concurrent(t *testing.T) {
	pr := GetParserRegistry()

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// 并发获取解析器
				_, _ = pr.Get(core.GeneratorTypeSnowflake)

				// 并发检查
				_ = pr.Has(core.GeneratorTypeSnowflake)
			}
		}()
	}

	wg.Wait()
}

// TestRegistry_MaxGeneratorsLimit 测试最大生成器数量限制
func TestRegistry_MaxGeneratorsLimit(t *testing.T) {
	registry := &Registry{
		generators:    make(map[string]core.IDGenerator),
		maxGenerators: 100, // 设置较小的限制
	}

	const goroutines = 20
	const attemptsPerGoroutine = 10

	var wg sync.WaitGroup
	var successCount int64
	var reachedLimitCount int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < attemptsPerGoroutine; j++ {
				key := fmt.Sprintf("limit_gen_%d_%d", idx, j)
				config := &snowflake.Config{
					DatacenterID: int64(idx % 32),
					WorkerID:     int64(j % 32),
				}
				_, err := registry.Create(key, core.GeneratorTypeSnowflake, config)
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				} else if err.Error() == core.ErrMaxGeneratorsReached.Error() ||
					strings.Contains(err.Error(), "maximum number of generators reached") {
					atomic.AddInt64(&reachedLimitCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("成功创建: %d", successCount)
	t.Logf("达到限制: %d", reachedLimitCount)
	t.Logf("最终大小: %d (限制: %d)", len(registry.generators), registry.maxGenerators)

	if len(registry.generators) > registry.maxGenerators {
		t.Errorf("注册表大小 %d 超过限制 %d", len(registry.generators), registry.maxGenerators)
	}
}

// BenchmarkRegistry_Create 基准测试：创建生成器
func BenchmarkRegistry_Create(b *testing.B) {
	registry := &Registry{
		generators:    make(map[string]core.IDGenerator),
		maxGenerators: 100000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_gen_%d", i)
		config := &snowflake.Config{
			DatacenterID: 1,
			WorkerID:     1,
		}
		_, _ = registry.Create(key, core.GeneratorTypeSnowflake, config)
	}
}

// BenchmarkRegistry_Get 基准测试：获取生成器
func BenchmarkRegistry_Get(b *testing.B) {
	registry := GetRegistry()
	registry.Clear()

	// 预创建生成器
	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}
	registry.Create("test_gen", core.GeneratorTypeSnowflake, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.Get("test_gen")
	}
}

// BenchmarkRegistry_GetOrCreate 基准测试：GetOrCreate
func BenchmarkRegistry_GetOrCreate(b *testing.B) {
	registry := &Registry{
		generators:    make(map[string]core.IDGenerator),
		maxGenerators: 100000,
	}

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_gen_%d", i%100) // 重复使用100个key
		_, _ = registry.GetOrCreate(key, core.GeneratorTypeSnowflake, config)
	}
}

// BenchmarkRegistry_ConcurrentGet 基准测试：并发获取
func BenchmarkRegistry_ConcurrentGet(b *testing.B) {
	registry := GetRegistry()
	registry.Clear()

	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}
	registry.Create("test_gen", core.GeneratorTypeSnowflake, config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = registry.Get("test_gen")
		}
	})
}
