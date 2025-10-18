package idgen

import (
	"errors"
	"sync"
	"testing"
)

// TestNewSnowflake 测试创建Snowflake实例
func TestNewSnowflake(t *testing.T) {
	tests := []struct {
		name         string
		datacenterID int64
		workerID     int64
		wantErr      bool
		expectedErr  error
	}{
		{
			name:         "有效参数_最小值",
			datacenterID: 0,
			workerID:     0,
			wantErr:      false,
		},
		{
			name:         "有效参数_最大值",
			datacenterID: 31,
			workerID:     31,
			wantErr:      false,
		},
		{
			name:         "无效WorkerID_负数",
			datacenterID: 1,
			workerID:     -1,
			wantErr:      true,
			expectedErr:  ErrInvalidWorkerID,
		},
		{
			name:         "无效WorkerID_超出最大值",
			datacenterID: 1,
			workerID:     32,
			wantErr:      true,
			expectedErr:  ErrInvalidWorkerID,
		},
		{
			name:         "无效DatacenterID_负数",
			datacenterID: -1,
			workerID:     1,
			wantErr:      true,
			expectedErr:  ErrInvalidDatacenterID,
		},
		{
			name:         "无效DatacenterID_超出最大值",
			datacenterID: 32,
			workerID:     1,
			wantErr:      true,
			expectedErr:  ErrInvalidDatacenterID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf, err := NewSnowflake(tt.datacenterID, tt.workerID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("期望得到错误，但没有返回错误")
					return
				}
				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("期望错误 %v, 实际得到 %v", tt.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
					return
				}
				if sf == nil {
					t.Error("Snowflake实例不应为nil")
					return
				}
				if sf.GetWorkerID() != tt.workerID {
					t.Errorf("WorkerID = %v, 期望 %v", sf.GetWorkerID(), tt.workerID)
				}
				if sf.GetDatacenterID() != tt.datacenterID {
					t.Errorf("DatacenterID = %v, 期望 %v", sf.GetDatacenterID(), tt.datacenterID)
				}
			}
		})
	}
}

// TestNextID 测试ID生成
func TestNextID(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// 生成多个ID，确保唯一性
	ids := make(map[int64]bool)
	count := 10000

	for i := 0; i < count; i++ {
		id, err := sf.NextID()
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
		t.Errorf("生成了 %d 个ID，期望 %d 个", len(ids), count)
	}
}

// TestNextIDBatch 测试批量生成ID
func TestNextIDBatch(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
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
		{"无效数量_超过最大值", 5000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := sf.NextIDBatch(tt.n)

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

// TestClockBackwardStrategy 测试时钟回拨策略
func TestClockBackwardStrategy(t *testing.T) {
	tests := []struct {
		name     string
		strategy ClockBackwardStrategy
	}{
		{"StrategyError", StrategyError},
		{"StrategyWait", StrategyWait},
		{"StrategyUseLastTimestamp", StrategyUseLastTimestamp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf, err := NewSnowflakeWithConfig(&SnowflakeConfig{
				DatacenterID:          1,
				WorkerID:              1,
				ClockBackwardStrategy: tt.strategy,
			})
			if err != nil {
				t.Fatal(err)
			}

			// 生成一个ID
			_, err = sf.NextID()
			if err != nil {
				t.Fatalf("生成ID失败: %v", err)
			}
		})
	}
}

// TestGetMetrics 测试获取监控指标
func TestGetMetrics(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// 生成一些ID
	count := 1000
	for i := 0; i < count; i++ {
		_, err := sf.NextID()
		if err != nil {
			t.Fatal(err)
		}
	}

	// 获取指标
	metrics := sf.GetMetrics()

	if metrics["id_count"] != uint64(count) {
		t.Errorf("id_count = %d, 期望 %d", metrics["id_count"], count)
	}

	// 检查指标键是否存在
	expectedKeys := []string{"id_count", "sequence_overflow", "clock_backward", "wait_count", "avg_wait_time_ns"}
	for _, key := range expectedKeys {
		if _, ok := metrics[key]; !ok {
			t.Errorf("指标中缺少键: %s", key)
		}
	}
}

// TestConcurrency 测试并发安全性
func TestConcurrency(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
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
				id, err := sf.NextID()
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

// TestParse 测试ID解析
func TestParse(t *testing.T) {
	sf, err := NewSnowflake(10, 20)
	if err != nil {
		t.Fatal(err)
	}

	id, err := sf.NextID()
	if err != nil {
		t.Fatal(err)
	}

	info, err := sf.Parse(id)
	if err != nil {
		t.Fatalf("解析ID失败: %v", err)
	}

	if info.ID != id {
		t.Errorf("ID = %d, 期望 %d", info.ID, id)
	}
	if info.DatacenterID != 10 {
		t.Errorf("DatacenterID = %d, 期望 10", info.DatacenterID)
	}
	if info.WorkerID != 20 {
		t.Errorf("WorkerID = %d, 期望 20", info.WorkerID)
	}
}

// TestValidateSnowflakeID 测试ID验证
func TestValidateSnowflakeID(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	validID, err := sf.NextID()
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
			err := ValidateSnowflakeID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSnowflakeID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// BenchmarkNextID 基准测试：单线程生成ID
func BenchmarkNextID(b *testing.B) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sf.NextID()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNextIDParallel 基准测试：并发生成ID
func BenchmarkNextIDParallel(b *testing.B) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := sf.NextID()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkNextIDBatch 基准测试：批量生成ID
func BenchmarkNextIDBatch(b *testing.B) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sf.NextIDBatch(100)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse 基准测试：解析ID
func BenchmarkParse(b *testing.B) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	id, err := sf.NextID()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sf.Parse(id)
		if err != nil {
			b.Fatal(err)
		}
	}
}
