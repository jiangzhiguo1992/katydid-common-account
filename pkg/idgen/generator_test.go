package idgen

import (
	"errors"
	"sync"
	"testing"
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

// TestRegisterFactory 测试注册工厂
func TestRegisterFactory(t *testing.T) {
	registry := GetRegistry()

	tests := []struct {
		name          string
		generatorType GeneratorType
		factory       GeneratorFactory
		wantErr       bool
		expectedErr   error
	}{
		{
			name:          "注册新工厂_成功",
			generatorType: "test-gen",
			factory:       &SnowflakeFactory{},
			wantErr:       false,
		},
		{
			name:          "空类型_失败",
			generatorType: "",
			factory:       &SnowflakeFactory{},
			wantErr:       true,
			expectedErr:   ErrInvalidGeneratorType,
		},
		{
			name:          "工厂为nil_失败",
			generatorType: "test-gen-2",
			factory:       nil,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.RegisterFactory(tt.generatorType, tt.factory)

			if tt.wantErr {
				if err == nil {
					t.Error("期望得到错误，但没有返回错误")
					return
				}
				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("期望错误 %v, 实际得到 %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
			}
		})
	}
}

// TestCreateGenerator 测试创建生成器
func TestCreateGenerator(t *testing.T) {
	registry := GetRegistry()

	t.Run("创建Snowflake生成器_成功", func(t *testing.T) {
		config := &SnowflakeConfig{
			DatacenterID: 1,
			WorkerID:     1,
		}

		gen, err := registry.CreateGenerator("test-sf-1", SnowflakeGeneratorType, config)
		if err != nil {
			t.Fatalf("创建生成器失败: %v", err)
		}

		if gen == nil {
			t.Error("生成器不应为nil")
		}

		// 测试生成ID
		id, err := gen.NextID()
		if err != nil {
			t.Errorf("生成ID失败: %v", err)
		}
		if id <= 0 {
			t.Errorf("生成的ID应为正数，得到: %d", id)
		}
	})

	t.Run("重复创建_返回缓存实例", func(t *testing.T) {
		config := &SnowflakeConfig{
			DatacenterID: 2,
			WorkerID:     2,
		}

		gen1, err := registry.CreateGenerator("test-sf-2", SnowflakeGeneratorType, config)
		if err != nil {
			t.Fatalf("第一次创建失败: %v", err)
		}

		gen2, err := registry.CreateGenerator("test-sf-2", SnowflakeGeneratorType, config)
		if err != nil {
			t.Fatalf("第二次创建失败: %v", err)
		}

		// 应该返回同一个实例
		if gen1 != gen2 {
			t.Error("重复创建应返回缓存的实例")
		}
	})

	t.Run("不存在的生成器类型_失败", func(t *testing.T) {
		// 🔧 修复：提供有效的配置，以便测试能检查到生成器类型不存在的错误
		config := &SnowflakeConfig{
			DatacenterID: 1,
			WorkerID:     1,
		}
		_, err := registry.CreateGenerator("test-unknown", "unknown-type", config)
		if err == nil {
			t.Error("期望得到错误")
		}
		if !errors.Is(err, ErrGeneratorNotFound) {
			t.Errorf("期望 ErrGeneratorNotFound, 得到: %v", err)
		}
	})

	t.Run("无效配置_失败", func(t *testing.T) {
		config := &SnowflakeConfig{
			DatacenterID: 1,
			WorkerID:     100, // 超出范围
		}

		_, err := registry.CreateGenerator("test-invalid", SnowflakeGeneratorType, config)
		if err == nil {
			t.Error("期望得到错误")
		}
	})
}

// TestGetGenerator 测试获取生成器
func TestGetGenerator(t *testing.T) {
	registry := GetRegistry()

	t.Run("获取存在的生成器", func(t *testing.T) {
		config := &SnowflakeConfig{
			DatacenterID: 3,
			WorkerID:     3,
		}

		_, err := registry.CreateGenerator("test-get-1", SnowflakeGeneratorType, config)
		if err != nil {
			t.Fatalf("创建生成器失败: %v", err)
		}

		gen, exists := registry.GetGenerator("test-get-1")
		if !exists {
			t.Error("应该能找到生成器")
		}
		if gen == nil {
			t.Error("生成器不应为nil")
		}
	})

	t.Run("获取不存在的生成器", func(t *testing.T) {
		_, exists := registry.GetGenerator("non-existent")
		if exists {
			t.Error("不应该找到生成器")
		}
	})
}

// TestRemoveGenerator 测试移除生成器
func TestRemoveGenerator(t *testing.T) {
	registry := GetRegistry()

	config := &SnowflakeConfig{
		DatacenterID: 4,
		WorkerID:     4,
	}

	key := "test-remove-1"
	_, err := registry.CreateGenerator(key, SnowflakeGeneratorType, config)
	if err != nil {
		t.Fatalf("创建生成器失败: %v", err)
	}

	// 确认存在
	_, exists := registry.GetGenerator(key)
	if !exists {
		t.Fatal("生成器应该存在")
	}

	// 移除
	registry.RemoveGenerator(key)

	// 确认已移除
	_, exists = registry.GetGenerator(key)
	if exists {
		t.Error("生成器应该已被移除")
	}
}

// TestListGeneratorTypes 测试列出生成器类型
func TestListGeneratorTypes(t *testing.T) {
	registry := GetRegistry()

	types := registry.ListGeneratorTypes()
	if len(types) == 0 {
		t.Error("至少应该有Snowflake类型")
	}

	// 检查是否包含Snowflake类型
	found := false
	for _, typ := range types {
		if typ == SnowflakeGeneratorType {
			found = true
			break
		}
	}
	if !found {
		t.Error("应该包含SnowflakeGeneratorType")
	}
}

// TestSnowflakeFactory 测试Snowflake工厂
func TestSnowflakeFactory(t *testing.T) {
	factory := &SnowflakeFactory{}

	t.Run("创建成功", func(t *testing.T) {
		config := &SnowflakeConfig{
			DatacenterID: 5,
			WorkerID:     5,
		}

		// 安全优化后：工厂接口只接受any类型的config参数
		gen, err := factory.Create(config)
		if err != nil {
			t.Fatalf("创建失败: %v", err)
		}

		if gen == nil {
			t.Error("生成器不应为nil")
		}

		// 验证接口实现
		_, ok := gen.(IDGenerator)
		if !ok {
			t.Error("应该实现IDGenerator接口")
		}
	})

	t.Run("错误的配置类型", func(t *testing.T) {
		// 安全优化后：传入错误的配置类型应该报错
		_, err := factory.Create("invalid-config")
		if err == nil {
			t.Error("期望得到错误")
		}
	})

	t.Run("nil配置", func(t *testing.T) {
		// 安全优化后：nil配置应该报错
		_, err := factory.Create(nil)
		if err == nil {
			t.Error("期望得到错误")
		}
	})
}

// TestNewGenerator 测试全局便捷函数
func TestNewGenerator(t *testing.T) {
	config := &SnowflakeConfig{
		DatacenterID: 7,
		WorkerID:     7,
	}

	gen, err := NewGenerator("global-test-1", SnowflakeGeneratorType, config)
	if err != nil {
		t.Fatalf("创建生成器失败: %v", err)
	}

	if gen == nil {
		t.Error("生成器不应为nil")
	}

	// 测试生成ID
	id, err := gen.NextID()
	if err != nil {
		t.Errorf("生成ID失败: %v", err)
	}
	if id <= 0 {
		t.Errorf("生成的ID应为正数，得到: %d", id)
	}
}

// TestGetGeneratorFromRegistry 测试从注册表获取
func TestGetGeneratorFromRegistry(t *testing.T) {
	config := &SnowflakeConfig{
		DatacenterID: 8,
		WorkerID:     8,
	}

	key := "global-test-2"
	_, err := NewGenerator(key, SnowflakeGeneratorType, config)
	if err != nil {
		t.Fatalf("创建生成器失败: %v", err)
	}

	gen, exists := GetGeneratorFromRegistry(key)
	if !exists {
		t.Error("应该能找到生成器")
	}
	if gen == nil {
		t.Error("生成器不应为nil")
	}
}

// TestGetDefaultGenerator 测试获取默认生成器
func TestGetDefaultGenerator(t *testing.T) {
	gen1, err := GetDefaultGenerator()
	if err != nil {
		t.Fatalf("获取默认生成器失败: %v", err)
	}

	gen2, err := GetDefaultGenerator()
	if err != nil {
		t.Fatalf("第二次获取失败: %v", err)
	}

	// 应该是同一个实例
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

// TestGenerateID 测试全局便捷函数
func TestGenerateID(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("生成ID失败: %v", err)
	}

	if id <= 0 {
		t.Errorf("生成的ID应为正数，得到: %d", id)
	}

	// 生成多个，验证唯一性
	ids := make(map[int64]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		if err != nil {
			t.Fatalf("第%d次生成失败: %v", i, err)
		}
		if ids[id] {
			t.Errorf("发现重复ID: %d", id)
		}
		ids[id] = true
	}
}

// TestRegistryConcurrency 测试注册表的并发安全性
func TestRegistryConcurrency(t *testing.T) {
	registry := GetRegistry()

	goroutines := 50
	var wg sync.WaitGroup

	// 并发创建生成器
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			config := &SnowflakeConfig{
				DatacenterID: int64(idx % 32),
				WorkerID:     int64(idx % 32),
			}

			key := "concurrent-test"
			gen, err := registry.CreateGenerator(key, SnowflakeGeneratorType, config)
			if err != nil {
				t.Errorf("创建生成器失败: %v", err)
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

// BenchmarkCreateGenerator 基准测试：创建生成器
func BenchmarkCreateGenerator(b *testing.B) {
	registry := GetRegistry()
	config := &SnowflakeConfig{
		DatacenterID: 1,
		WorkerID:     1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-gen"
		_, err := registry.CreateGenerator(key, SnowflakeGeneratorType, config)
		if err != nil {
			b.Fatalf("创建失败: %v", err)
		}
		// 清理以便下次测试
		registry.RemoveGenerator(key)
	}
}

// BenchmarkGetGenerator 基准测试：获取生成器
func BenchmarkGetGenerator(b *testing.B) {
	registry := GetRegistry()
	config := &SnowflakeConfig{
		DatacenterID: 1,
		WorkerID:     1,
	}

	key := "bench-get"
	_, _ = registry.CreateGenerator(key, SnowflakeGeneratorType, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetGenerator(key)
	}
}

// BenchmarkGenerateID 基准测试：使用全局便捷函数生成ID
func BenchmarkGenerateID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateID()
		if err != nil {
			b.Fatalf("生成ID失败: %v", err)
		}
	}
}
