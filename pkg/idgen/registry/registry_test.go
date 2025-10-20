package registry

import (
	"sync"
	"testing"

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

// BenchmarkRegistryCreate 基准测试：创建生成器
func BenchmarkRegistryCreate(b *testing.B) {
	registry := GetRegistry()
	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-create"
		_, _ = registry.Create(key, core.GeneratorTypeSnowflake, config)
		_ = registry.Remove(key)
	}
}

// BenchmarkRegistryGet 基准测试：获取生成器
func BenchmarkRegistryGet(b *testing.B) {
	registry := GetRegistry()
	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	key := "bench-get"
	_, _ = registry.Create(key, core.GeneratorTypeSnowflake, config)
	defer func() { _ = registry.Remove(key) }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.Get(key)
	}
}

// BenchmarkRegistryGetOrCreate 基准测试：获取或创建
func BenchmarkRegistryGetOrCreate(b *testing.B) {
	registry := GetRegistry()
	config := &snowflake.Config{
		DatacenterID: 1,
		WorkerID:     1,
	}

	key := "bench-get-or-create"
	defer func() { _ = registry.Remove(key) }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetOrCreate(key, core.GeneratorTypeSnowflake, config)
	}
}
