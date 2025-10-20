package snowflake

import (
	"sync"
	"testing"
	"time"
)

// TestNew 测试创建Snowflake生成器
func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		datacenterID int64
		workerID     int64
		wantErr      bool
	}{
		{"有效参数_最小值", 0, 0, false},
		{"有效参数_最大值", 31, 31, false},
		{"有效参数_中间值", 15, 15, false},
		{"无效WorkerID_负数", 1, -1, true},
		{"无效WorkerID_超出", 1, 32, true},
		{"无效DatacenterID_负数", -1, 1, true},
		{"无效DatacenterID_超出", 32, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := New(tt.datacenterID, tt.workerID)
			if tt.wantErr {
				if err == nil {
					t.Error("期望得到错误，但没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
					return
				}
				if gen == nil {
					t.Error("生成器不应为nil")
				}
			}
		})
	}
}

// TestNewWithConfig 测试使用配置创建
func TestNewWithConfig(t *testing.T) {
	t.Run("有效配置", func(t *testing.T) {
		config := &Config{
			DatacenterID:  1,
			WorkerID:      1,
			EnableMetrics: true,
		}

		gen, err := NewWithConfig(config)
		if err != nil {
			t.Fatalf("创建失败: %v", err)
		}
		if gen == nil {
			t.Error("生成器不应为nil")
		}
		if gen.GetDatacenterID() != 1 {
			t.Errorf("DatacenterID = %d, 期望 1", gen.GetDatacenterID())
		}
		if gen.GetWorkerID() != 1 {
			t.Errorf("WorkerID = %d, 期望 1", gen.GetWorkerID())
		}
	})

	t.Run("nil配置", func(t *testing.T) {
		_, err := NewWithConfig(nil)
		if err == nil {
			t.Error("期望得到错误")
		}
	})
}

// TestNextID 测试ID生成
func TestNextID(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// 生成多个ID，验证唯一性
	ids := make(map[int64]bool)
	count := 10000

	for i := 0; i < count; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("生成ID失败: %v", err)
		}
		if id <= 0 {
			t.Errorf("ID应为正数，得到: %d", id)
		}
		if ids[id] {
			t.Errorf("发现重复ID: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("生成了 %d 个唯一ID，期望 %d 个", len(ids), count)
	}
}

// TestNextIDBatch 测试批量生成ID
func TestNextIDBatch(t *testing.T) {
	gen, err := New(2, 2)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"批量生成10个", 10, false},
		{"批量生成100个", 100, false},
		{"批量生成1000个", 1000, false},
		{"无效数量_负数", -1, true},
		{"无效数量_零", 0, true},
		{"无效数量_超过最大值", 150000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := gen.NextIDBatch(tt.n)
			if tt.wantErr {
				if err == nil {
					t.Error("期望得到错误，但没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
					return
				}
				if len(ids) != tt.n {
					t.Errorf("生成了 %d 个ID，期望 %d 个", len(ids), tt.n)
				}

				// 检查唯一性
				idMap := make(map[int64]bool)
				for _, id := range ids {
					if idMap[id] {
						t.Errorf("发现重复ID: %d", id)
					}
					idMap[id] = true
				}
			}
		})
	}
}

// TestGetWorkerID 测试获取WorkerID
func TestGetWorkerID(t *testing.T) {
	gen, err := New(3, 5)
	if err != nil {
		t.Fatal(err)
	}

	if gen.GetWorkerID() != 5 {
		t.Errorf("GetWorkerID() = %d, 期望 5", gen.GetWorkerID())
	}
}

// TestGetDatacenterID 测试获取DatacenterID
func TestGetDatacenterID(t *testing.T) {
	gen, err := New(7, 3)
	if err != nil {
		t.Fatal(err)
	}

	if gen.GetDatacenterID() != 7 {
		t.Errorf("GetDatacenterID() = %d, 期望 7", gen.GetDatacenterID())
	}
}

// TestGetMetrics 测试获取监控指标
func TestGetMetrics(t *testing.T) {
	config := &Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	}

	gen, err := NewWithConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	// 生成一些ID
	count := 100
	for i := 0; i < count; i++ {
		_, err := gen.NextID()
		if err != nil {
			t.Fatal(err)
		}
	}

	metrics := gen.GetMetrics()
	if metrics["id_count"] != uint64(count) {
		t.Errorf("id_count = %d, 期望 %d", metrics["id_count"], count)
	}
}

// TestResetMetrics 测试重置监控指标
func TestResetMetrics(t *testing.T) {
	config := &Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	}

	gen, err := NewWithConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	// 生成一些ID
	for i := 0; i < 50; i++ {
		_, _ = gen.NextID()
	}

	gen.ResetMetrics()

	if gen.GetIDCount() != 0 {
		t.Errorf("重置后 IDCount = %d, 期望 0", gen.GetIDCount())
	}
}

// TestParseID 测试解析ID
func TestParseID(t *testing.T) {
	gen, err := New(5, 10)
	if err != nil {
		t.Fatal(err)
	}

	id, err := gen.NextID()
	if err != nil {
		t.Fatal(err)
	}

	info, err := gen.ParseID(id)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if info.DatacenterID != 5 {
		t.Errorf("DatacenterID = %d, 期望 5", info.DatacenterID)
	}
	if info.WorkerID != 10 {
		t.Errorf("WorkerID = %d, 期望 10", info.WorkerID)
	}
}

// TestValidateID 测试验证ID
func TestValidateID(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	validID, err := gen.NextID()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{"有效ID", validID, false},
		{"无效ID_负数", -1, true},
		{"无效ID_零", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.ValidateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConcurrency 测试并发安全性
func TestConcurrency(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	goroutines := 100
	idsPerGoroutine := 100
	results := make(chan int64, goroutines*idsPerGoroutine)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := gen.NextID()
				if err != nil {
					t.Errorf("生成ID失败: %v", err)
					return
				}
				results <- id
			}
		}()
	}

	wg.Wait()
	close(results)

	// 检查唯一性
	ids := make(map[int64]bool)
	for id := range results {
		if ids[id] {
			t.Errorf("发现重复ID: %d", id)
		}
		ids[id] = true
	}

	expectedCount := goroutines * idsPerGoroutine
	if len(ids) != expectedCount {
		t.Errorf("生成了 %d 个唯一ID，期望 %d 个", len(ids), expectedCount)
	}
}

// TestConfig 测试配置
func TestConfig(t *testing.T) {
	t.Run("Validate_有效配置", func(t *testing.T) {
		config := &Config{
			DatacenterID: 1,
			WorkerID:     1,
		}
		if err := config.Validate(); err != nil {
			t.Errorf("验证失败: %v", err)
		}
	})

	t.Run("Validate_无效配置", func(t *testing.T) {
		config := &Config{
			DatacenterID: 100,
			WorkerID:     1,
		}
		if err := config.Validate(); err == nil {
			t.Error("期望得到错误")
		}
	})

	t.Run("Clone", func(t *testing.T) {
		config := &Config{
			DatacenterID:  1,
			WorkerID:      2,
			EnableMetrics: true,
		}
		cloned := config.Clone()
		if cloned.DatacenterID != config.DatacenterID {
			t.Error("克隆的配置不匹配")
		}
		// 修改克隆不应影响原配置
		cloned.DatacenterID = 10
		if config.DatacenterID == 10 {
			t.Error("修改克隆影响了原配置")
		}
	})
}

// TestParser 测试解析器
func TestParser(t *testing.T) {
	parser := NewParser()
	gen, _ := New(5, 10)
	id, _ := gen.NextID()

	t.Run("Parse", func(t *testing.T) {
		info, err := parser.Parse(id)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if info.DatacenterID != 5 {
			t.Errorf("DatacenterID = %d, 期望 5", info.DatacenterID)
		}
		if info.WorkerID != 10 {
			t.Errorf("WorkerID = %d, 期望 10", info.WorkerID)
		}
	})

	t.Run("ExtractTimestamp", func(t *testing.T) {
		timestamp := parser.ExtractTimestamp(id)
		if timestamp <= 0 {
			t.Error("时间戳应为正数")
		}
	})

	t.Run("ExtractTimestampAsTime", func(t *testing.T) {
		tm := parser.ExtractTimestampAsTime(id)
		if tm.IsZero() {
			t.Error("时间不应为零值")
		}
		if tm.After(time.Now()) {
			t.Error("时间不应在未来")
		}
	})

	t.Run("ExtractDatacenterID", func(t *testing.T) {
		dcID := parser.ExtractDatacenterID(id)
		if dcID != 5 {
			t.Errorf("DatacenterID = %d, 期望 5", dcID)
		}
	})

	t.Run("ExtractWorkerID", func(t *testing.T) {
		wID := parser.ExtractWorkerID(id)
		if wID != 10 {
			t.Errorf("WorkerID = %d, 期望 10", wID)
		}
	})

	t.Run("ExtractSequence", func(t *testing.T) {
		seq := parser.ExtractSequence(id)
		if seq < 0 {
			t.Error("序列号不应为负数")
		}
	})
}

// TestValidator 测试验证器
func TestValidator(t *testing.T) {
	validator := NewValidator()
	gen, _ := New(1, 1)
	validID, _ := gen.NextID()

	t.Run("Validate_有效ID", func(t *testing.T) {
		err := validator.Validate(validID)
		if err != nil {
			t.Errorf("验证失败: %v", err)
		}
	})

	t.Run("Validate_无效ID", func(t *testing.T) {
		tests := []struct {
			name string
			id   int64
		}{
			{"负数", -1},
			{"零", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validator.Validate(tt.id)
				if err == nil {
					t.Error("期望得到错误")
				}
			})
		}
	})

	t.Run("ValidateBatch", func(t *testing.T) {
		ids := []int64{validID, validID + 1, validID + 2}
		err := validator.ValidateBatch(ids)
		if err != nil {
			t.Errorf("批量验证失败: %v", err)
		}

		invalidIDs := []int64{validID, -1, validID + 2}
		err = validator.ValidateBatch(invalidIDs)
		if err == nil {
			t.Error("期望得到错误")
		}
	})
}

// TestValidateSnowflakeID 测试全局验证函数
func TestValidateSnowflakeID(t *testing.T) {
	gen, _ := New(1, 1)
	validID, _ := gen.NextID()

	if err := ValidateSnowflakeID(validID); err != nil {
		t.Errorf("验证失败: %v", err)
	}

	if err := ValidateSnowflakeID(-1); err == nil {
		t.Error("期望得到错误")
	}
}

// TestParseSnowflakeID 测试全局解析函数
func TestParseSnowflakeID(t *testing.T) {
	gen, _ := New(7, 15)
	id, _ := gen.NextID()

	timestamp, datacenterID, workerID, sequence := ParseSnowflakeID(id)

	if datacenterID != 7 {
		t.Errorf("DatacenterID = %d, 期望 7", datacenterID)
	}
	if workerID != 15 {
		t.Errorf("WorkerID = %d, 期望 15", workerID)
	}
	if timestamp <= 0 {
		t.Error("时间戳应为正数")
	}
	if sequence < 0 {
		t.Error("序列号不应为负数")
	}
}

// TestGetTimestamp 测试全局时间戳提取函数
func TestGetTimestamp(t *testing.T) {
	gen, _ := New(1, 1)
	id, _ := gen.NextID()

	tm := GetTimestamp(id)
	if tm.IsZero() {
		t.Error("时间不应为零值")
	}
	if tm.After(time.Now()) {
		t.Error("时间不应在未来")
	}
}

// BenchmarkNextID 基准测试：生成ID
func BenchmarkNextID(b *testing.B) {
	gen, err := New(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.NextID()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNextIDParallel 基准测试：并发生成ID
func BenchmarkNextIDParallel(b *testing.B) {
	gen, err := New(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gen.NextID()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkNextIDBatch 基准测试：批量生成ID
func BenchmarkNextIDBatch(b *testing.B) {
	gen, err := New(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.NextIDBatch(100)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseID 基准测试：解析ID
func BenchmarkParseID(b *testing.B) {
	gen, _ := New(1, 1)
	id, _ := gen.NextID()
	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(id)
	}
}

// BenchmarkValidateID 基准测试：验证ID
func BenchmarkValidateID(b *testing.B) {
	gen, _ := New(1, 1)
	id, _ := gen.NextID()
	validator := NewValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(id)
	}
}
