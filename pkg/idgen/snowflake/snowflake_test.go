package snowflake

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
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

// ========== 高并发百万级测试（多维度性能分析） ==========

// TestHighConcurrency_Million 测试百万级高并发ID生成
func TestHighConcurrency_Million(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级测试（使用 -short 标志）")
	}

	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name            string
		goroutines      int
		idsPerGoroutine int
		totalIDs        int
	}{
		{"百万_1000协程x1000", 1000, 1000, 1_000_000},
		{"百万_500协程x2000", 500, 2000, 1_000_000},
		{"百万_100协程x10000", 100, 10000, 1_000_000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gen.ResetMetrics()
			startTime := time.Now()

			// 使用map存储ID检查唯一性（内存压力测试）
			idMap := &sync.Map{}
			var duplicateCount int64
			var errorCount int64

			var wg sync.WaitGroup
			for i := 0; i < tc.goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < tc.idsPerGoroutine; j++ {
						id, err := gen.NextID()
						if err != nil {
							t.Errorf("生成ID失败: %v", err)
							atomic.AddInt64(&errorCount, 1)
							return
						}

						// 检查重复
						if _, loaded := idMap.LoadOrStore(id, struct{}{}); loaded {
							t.Errorf("发现重复ID: %d", id)
							atomic.AddInt64(&duplicateCount, 1)
						}
					}
				}()
			}

			wg.Wait()
			elapsed := time.Since(startTime)

			// 统计实际生成的ID数量
			actualCount := int64(0)
			idMap.Range(func(key, value interface{}) bool {
				actualCount++
				return true
			})

			// 性能报告
			idsPerSecond := float64(tc.totalIDs) / elapsed.Seconds()
			avgLatency := elapsed.Nanoseconds() / int64(tc.totalIDs)

			t.Logf("============ 性能报告 ============")
			t.Logf("总ID数: %d", tc.totalIDs)
			t.Logf("协程数: %d", tc.goroutines)
			t.Logf("唯一ID数: %d", actualCount)
			t.Logf("重复ID数: %d", atomic.LoadInt64(&duplicateCount))
			t.Logf("错误数: %d", atomic.LoadInt64(&errorCount))
			t.Logf("总耗时: %v", elapsed)
			t.Logf("吞吐量: %.0f IDs/秒", idsPerSecond)
			t.Logf("平均延迟: %d 纳秒", avgLatency)

			metrics := gen.GetMetrics()
			t.Logf("序列溢出次数: %d", metrics["sequence_overflow"])
			t.Logf("等待次数: %d", metrics["wait_count"])
			t.Logf("总等待时间: %d 纳秒", metrics["total_wait_time_ns"])

			// 断言
			dupCount := atomic.LoadInt64(&duplicateCount)
			errCount := atomic.LoadInt64(&errorCount)
			if dupCount > 0 {
				t.Errorf("发现 %d 个重复ID", dupCount)
			}
			if errCount > 0 {
				t.Errorf("发生 %d 个错误", errCount)
			}
			if actualCount != int64(tc.totalIDs) {
				t.Errorf("唯一ID数 %d 不等于预期 %d", actualCount, tc.totalIDs)
			}
		})
	}
}

// TestMemoryUsage_Million 测试百万级ID生成的内存使用
func TestMemoryUsage_Million(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过内存测试（使用 -short 标志）")
	}

	gen, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	const totalIDs = 1_000_000

	// 记录初始内存
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	runtime.GC()
	runtime.ReadMemStats(&m1)

	startTime := time.Now()
	ids := make([]int64, 0, totalIDs)

	// 生成100万个ID
	for i := 0; i < totalIDs; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("生成ID失败: %v", err)
		}
		ids = append(ids, id)
	}

	elapsed := time.Since(startTime)

	// 记录结束内存
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 计算内存使用
	allocBytes := m2.TotalAlloc - m1.TotalAlloc
	sysBytes := m2.Sys - m1.Sys
	heapAllocBytes := m2.HeapAlloc - m1.HeapAlloc

	t.Logf("============ 内存使用报告 ============")
	t.Logf("生成ID数: %d", totalIDs)
	t.Logf("总耗时: %v", elapsed)
	t.Logf("分配内存: %.2f MB", float64(allocBytes)/(1024*1024))
	t.Logf("系统内存: %.2f MB", float64(sysBytes)/(1024*1024))
	t.Logf("堆内存: %.2f MB", float64(heapAllocBytes)/(1024*1024))
	t.Logf("单ID内存: %.2f 字节", float64(allocBytes)/float64(totalIDs))
	t.Logf("GC次数: %d", m2.NumGC-m1.NumGC)

	// 验证唯一性
	idSet := make(map[int64]bool, totalIDs)
	for _, id := range ids {
		if idSet[id] {
			t.Errorf("发现重复ID: %d", id)
		}
		idSet[id] = true
	}
}

// TestBatchPerformance_Million 测试批量生成性能（百万级）
func TestBatchPerformance_Million(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过批量性能测试（使用 -short 标志）")
	}

	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name      string
		batchSize int
		batches   int
		totalIDs  int
	}{
		{"小批量_1000x1000", 1000, 1000, 1_000_000},
		{"中批量_5000x200", 5000, 200, 1_000_000},
		{"大批量_10000x100", 10000, 100, 1_000_000},
		{"超大批量_50000x20", 50000, 20, 1_000_000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gen.ResetMetrics()
			allIDs := make([]int64, 0, tc.totalIDs)

			startTime := time.Now()
			for i := 0; i < tc.batches; i++ {
				ids, err := gen.NextIDBatch(tc.batchSize)
				if err != nil {
					t.Fatalf("批量生成失败: %v", err)
				}
				allIDs = append(allIDs, ids...)
			}
			elapsed := time.Since(startTime)

			// 检查唯一性
			idMap := make(map[int64]bool, len(allIDs))
			duplicates := 0
			for _, id := range allIDs {
				if idMap[id] {
					duplicates++
				}
				idMap[id] = true
			}

			idsPerSecond := float64(tc.totalIDs) / elapsed.Seconds()

			t.Logf("============ 批量性能报告 ============")
			t.Logf("批次大小: %d", tc.batchSize)
			t.Logf("批次数: %d", tc.batches)
			t.Logf("总ID数: %d", tc.totalIDs)
			t.Logf("唯一ID数: %d", len(idMap))
			t.Logf("重复数: %d", duplicates)
			t.Logf("总耗时: %v", elapsed)
			t.Logf("吞吐量: %.0f IDs/秒", idsPerSecond)
			t.Logf("单批次耗时: %v", elapsed/time.Duration(tc.batches))

			metrics := gen.GetMetrics()
			t.Logf("序列溢出: %d", metrics["sequence_overflow"])
			t.Logf("等待次数: %d", metrics["wait_count"])

			if duplicates > 0 {
				t.Errorf("发现 %d 个重复ID", duplicates)
			}
			if len(allIDs) != tc.totalIDs {
				t.Errorf("生成ID数 %d 不等于预期 %d", len(allIDs), tc.totalIDs)
			}
		})
	}
}

// TestConcurrentBatch_Million 测试并发批量生成（百万级）
func TestConcurrentBatch_Million(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发批量测试（使用 -short 标志）")
	}

	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name       string
		goroutines int
		batchSize  int
		totalIDs   int
	}{
		{"并发批量_100协程x10000批量", 100, 10000, 1_000_000},
		{"并发批量_50协程x20000批量", 50, 20000, 1_000_000},
		{"并发批量_20协程x50000批量", 20, 50000, 1_000_000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gen.ResetMetrics()

			// 计算每个协程需要的批次数
			batchesPerGoroutine := tc.totalIDs / (tc.goroutines * tc.batchSize)

			idMap := &sync.Map{}
			errorCount := int64(0)
			totalGenerated := int64(0)

			startTime := time.Now()
			var wg sync.WaitGroup

			for i := 0; i < tc.goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < batchesPerGoroutine; j++ {
						ids, err := gen.NextIDBatch(tc.batchSize)
						if err != nil {
							t.Errorf("批量生成失败: %v", err)
							errorCount++
							return
						}

						for _, id := range ids {
							if _, loaded := idMap.LoadOrStore(id, struct{}{}); loaded {
								t.Errorf("发现重复ID: %d", id)
							}
							totalGenerated++
						}
					}
				}()
			}

			wg.Wait()
			elapsed := time.Since(startTime)

			// 统计唯一ID数
			uniqueCount := int64(0)
			idMap.Range(func(key, value interface{}) bool {
				uniqueCount++
				return true
			})

			idsPerSecond := float64(totalGenerated) / elapsed.Seconds()

			t.Logf("============ 并发批量性能报告 ============")
			t.Logf("协程数: %d", tc.goroutines)
			t.Logf("批次大小: %d", tc.batchSize)
			t.Logf("生成总数: %d", totalGenerated)
			t.Logf("唯一ID数: %d", uniqueCount)
			t.Logf("错误数: %d", errorCount)
			t.Logf("总耗时: %v", elapsed)
			t.Logf("吞吐量: %.0f IDs/秒", idsPerSecond)

			metrics := gen.GetMetrics()
			t.Logf("序列溢出: %d", metrics["sequence_overflow"])
			t.Logf("等待次数: %d", metrics["wait_count"])
			t.Logf("时钟回拨: %d", metrics["clock_backward"])

			if uniqueCount != totalGenerated {
				t.Errorf("发现重复ID: 唯一数%d != 总数%d", uniqueCount, totalGenerated)
			}
		})
	}
}

// TestStressTest_Sequential 测试顺序压力（最大吞吐量）
func TestStressTest_Sequential(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过顺序压力测试（使用 -short 标志）")
	}

	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	const totalIDs = 5_000_000 // 500万

	startTime := time.Now()
	prevID := int64(0)
	monotonic := true

	for i := 0; i < totalIDs; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("生成ID失败 (第%d个): %v", i+1, err)
		}

		// 检查单调性
		if id <= prevID {
			monotonic = false
			t.Errorf("ID不单调: 前一个=%d, 当前=%d, 位置=%d", prevID, id, i)
		}
		prevID = id
	}

	elapsed := time.Since(startTime)
	idsPerSecond := float64(totalIDs) / elapsed.Seconds()

	metrics := gen.GetMetrics()

	t.Logf("============ 顺序压力测试报告 ============")
	t.Logf("总ID数: %d", totalIDs)
	t.Logf("总耗时: %v", elapsed)
	t.Logf("吞吐量: %.0f IDs/秒", idsPerSecond)
	t.Logf("平均延迟: %d 纳秒", elapsed.Nanoseconds()/int64(totalIDs))
	t.Logf("ID单调性: %v", monotonic)
	t.Logf("序列溢出: %d", metrics["sequence_overflow"])
	t.Logf("等待次数: %d", metrics["wait_count"])
	t.Logf("总等待时间: %.2f 毫秒", float64(metrics["total_wait_time_ns"])/1e6)

	if !monotonic {
		t.Error("ID序列不单调")
	}
}

// TestMultiGenerator_Concurrent 测试多生成器并发（模拟分布式）
func TestMultiGenerator_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过多生成器测试（使用 -short 标志）")
	}

	const (
		numGenerators = 10
		idsPerGen     = 100_000
		totalIDs      = numGenerators * idsPerGen
	)

	generators := make([]*Generator, numGenerators)
	for i := 0; i < numGenerators; i++ {
		gen, err := New(int64(i/5), int64(i%5))
		if err != nil {
			t.Fatalf("创建生成器%d失败: %v", i, err)
		}
		generators[i] = gen
	}

	idMap := &sync.Map{}
	startTime := time.Now()

	var wg sync.WaitGroup
	for i, gen := range generators {
		wg.Add(1)
		go func(genIndex int, g *Generator) {
			defer wg.Done()
			for j := 0; j < idsPerGen; j++ {
				id, err := g.NextID()
				if err != nil {
					t.Errorf("生成器%d生成ID失败: %v", genIndex, err)
					return
				}

				if _, loaded := idMap.LoadOrStore(id, genIndex); loaded {
					t.Errorf("生成器%d产生重复ID: %d", genIndex, id)
				}
			}
		}(i, gen)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	uniqueCount := int64(0)
	idMap.Range(func(key, value interface{}) bool {
		uniqueCount++
		return true
	})

	idsPerSecond := float64(totalIDs) / elapsed.Seconds()

	t.Logf("============ 多生成器并发测试报告 ============")
	t.Logf("生成器数: %d", numGenerators)
	t.Logf("每生成器ID数: %d", idsPerGen)
	t.Logf("总ID数: %d", totalIDs)
	t.Logf("唯一ID数: %d", uniqueCount)
	t.Logf("总耗时: %v", elapsed)
	t.Logf("吞吐量: %.0f IDs/秒", idsPerSecond)

	if uniqueCount != int64(totalIDs) {
		t.Errorf("发现重复ID: 唯一数%d != 总数%d", uniqueCount, totalIDs)
	}
}

// TestLongRunning_Stability 测试长时间运行稳定性
func TestLongRunning_Stability(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过长时间运行测试（使用 -short 标志）")
	}

	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	const (
		duration   = 30 * time.Second
		goroutines = 50
	)

	idMap := &sync.Map{}
	stopChan := make(chan struct{})
	totalGenerated := int64(0)
	errorCount := int64(0)

	startTime := time.Now()

	// 启动生成器协程
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					id, err := gen.NextID()
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						t.Errorf("生成ID失败: %v", err)
						continue
					}

					if _, loaded := idMap.LoadOrStore(id, struct{}{}); loaded {
						t.Errorf("发现重复ID: %d", id)
					}
					atomic.AddInt64(&totalGenerated, 1)
				}
			}
		}()
	}

	// 监控协程
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)
				current := atomic.LoadInt64(&totalGenerated)
				rate := float64(current) / elapsed.Seconds()
				metrics := gen.GetMetrics()

				t.Logf("[%v] 已生成: %d, 速率: %.0f IDs/秒, 溢出: %d, 等待: %d",
					elapsed.Round(time.Second), current, rate,
					metrics["sequence_overflow"], metrics["wait_count"])
			}
		}
	}()

	// 运行指定时间
	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	elapsed := time.Since(startTime)

	uniqueCount := int64(0)
	idMap.Range(func(key, value interface{}) bool {
		uniqueCount++
		return true
	})

	totalGen := atomic.LoadInt64(&totalGenerated)
	errCount := atomic.LoadInt64(&errorCount)
	metrics := gen.GetMetrics()
	idsPerSecond := float64(totalGen) / elapsed.Seconds()

	t.Logf("============ 长时间运行稳定性报告 ============")
	t.Logf("运行时间: %v", elapsed)
	t.Logf("协程数: %d", goroutines)
	t.Logf("生成总数: %d", totalGen)
	t.Logf("唯一ID数: %d", uniqueCount)
	t.Logf("错误数: %d", errCount)
	t.Logf("平均吞吐量: %.0f IDs/秒", idsPerSecond)
	t.Logf("序列溢出: %d", metrics["sequence_overflow"])
	t.Logf("等待次数: %d", metrics["wait_count"])
	t.Logf("时钟回拨: %d", metrics["clock_backward"])

	if uniqueCount != totalGen {
		t.Errorf("发现重复ID: 唯一数%d != 总数%d", uniqueCount, totalGen)
	}
	if errCount > 0 {
		t.Errorf("发生 %d 个错误", errCount)
	}
}

// TestDiagnostic_BasicFunction 诊断测试 - 基础功能
func TestDiagnostic_BasicFunction(t *testing.T) {
	t.Log("=== 开始诊断测试 ===")

	// 测试1: 创建生成器
	t.Log("步骤1: 创建生成器...")
	gen, err := New(1, 1)
	if err != nil {
		t.Fatalf("创建生成器失败: %v", err)
	}
	t.Log("✓ 生成器创建成功")

	// 测试2: 生成单个ID
	t.Log("步骤2: 生成单个ID...")
	id1, err := gen.NextID()
	if err != nil {
		t.Fatalf("生成ID失败: %v", err)
	}
	t.Logf("✓ 生成ID成功: %d", id1)

	// 测试3: 连续生成少量ID
	t.Log("步骤3: 连续生成10个ID...")
	ids := make([]int64, 10)
	for i := 0; i < 10; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("生成第%d个ID失败: %v", i+1, err)
		}
		ids[i] = id
	}
	t.Logf("✓ 成功生成10个ID: %v", ids)

	// 测试4: 验证唯一性
	t.Log("步骤4: 验证唯一性...")
	idMap := make(map[int64]bool)
	for i, id := range ids {
		if idMap[id] {
			t.Fatalf("发现重复ID: %d (位置%d)", id, i)
		}
		idMap[id] = true
	}
	t.Log("✓ 所有ID唯一")

	// 测试5: 批量生成
	t.Log("步骤5: 批量生成100个ID...")
	batchIDs, err := gen.NextIDBatch(100)
	if err != nil {
		t.Fatalf("批量生成失败: %v", err)
	}
	t.Logf("✓ 批量生成成功，数量: %d", len(batchIDs))

	t.Log("=== 诊断测试完成，基础功能正常 ===")
}

// TestDiagnostic_Performance 诊断测试 - 性能测试
func TestDiagnostic_Performance(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatalf("创建生成器失败: %v", err)
	}

	counts := []int{100, 1000, 10000}

	for _, count := range counts {
		t.Run(fmt.Sprintf("生成%d个ID", count), func(t *testing.T) {
			start := time.Now()

			for i := 0; i < count; i++ {
				_, err := gen.NextID()
				if err != nil {
					t.Fatalf("生成失败: %v", err)
				}
			}

			elapsed := time.Since(start)
			rate := float64(count) / elapsed.Seconds()

			t.Logf("数量: %d, 耗时: %v, 速率: %.0f IDs/秒", count, elapsed, rate)
		})
	}
}

// TestDiagnostic_Concurrency 诊断测试 - 并发测试
func TestDiagnostic_Concurrency(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatalf("创建生成器失败: %v", err)
	}

	// 简单并发测试
	t.Run("10协程x100ID", func(t *testing.T) {
		type result struct {
			ids []int64
			err error
		}

		results := make(chan result, 10)

		for i := 0; i < 10; i++ {
			go func() {
				ids := make([]int64, 100)
				var err error
				for j := 0; j < 100; j++ {
					ids[j], err = gen.NextID()
					if err != nil {
						results <- result{nil, err}
						return
					}
				}
				results <- result{ids, nil}
			}()
		}

		allIDs := make(map[int64]bool)
		for i := 0; i < 10; i++ {
			res := <-results
			if res.err != nil {
				t.Fatalf("协程生成失败: %v", res.err)
			}
			for _, id := range res.ids {
				if allIDs[id] {
					t.Fatalf("发现重复ID: %d", id)
				}
				allIDs[id] = true
			}
		}

		t.Logf("✓ 并发测试通过，唯一ID数: %d", len(allIDs))
	})
}

// TestQuickPerformance 快速性能验证测试
func TestQuickPerformance(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// 测试1: 单线程顺序生成性能
	t.Run("单线程_10万", func(t *testing.T) {
		const count = 100_000
		start := time.Now()

		for i := 0; i < count; i++ {
			_, err := gen.NextID()
			if err != nil {
				t.Fatalf("生成失败: %v", err)
			}
		}

		elapsed := time.Since(start)
		rate := float64(count) / elapsed.Seconds()
		t.Logf("单线程: %d IDs, 耗时: %v, 速率: %.0f IDs/秒", count, elapsed, rate)
	})

	// 测试2: 并发测试
	t.Run("并发_10协程x1万", func(t *testing.T) {
		const goroutines = 10
		const perGoroutine = 10_000

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < perGoroutine; j++ {
					_, err := gen.NextID()
					if err != nil {
						t.Errorf("生成失败: %v", err)
					}
				}
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)
		total := goroutines * perGoroutine
		rate := float64(total) / elapsed.Seconds()
		t.Logf("并发: %d IDs, 耗时: %v, 速率: %.0f IDs/秒", total, elapsed, rate)
	})
}

// TestRaceDetection 数据竞争检测测试
func TestRaceDetection(t *testing.T) {
	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 50
	const idsPerGoroutine = 1000

	idMap := &sync.Map{}
	var duplicates int64

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := gen.NextID()
				if err != nil {
					t.Errorf("生成失败: %v", err)
					return
				}

				if _, exists := idMap.LoadOrStore(id, struct{}{}); exists {
					atomic.AddInt64(&duplicates, 1)
					t.Errorf("发现重复ID: %d", id)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	count := int64(0)
	idMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})

	total := int64(goroutines * idsPerGoroutine)
	t.Logf("并发安全测试: 协程=%d, 总数=%d, 唯一=%d, 重复=%d, 耗时=%v",
		goroutines, total, count, atomic.LoadInt64(&duplicates), elapsed)

	if atomic.LoadInt64(&duplicates) > 0 {
		t.Errorf("检测到 %d 个重复ID", atomic.LoadInt64(&duplicates))
	}

	if count != total {
		t.Errorf("ID数量不匹配: 唯一=%d, 期望=%d", count, total)
	}
}

// TestMemoryFootprint 内存占用测试
func TestMemoryFootprint(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	const count = 100_000

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	ids := make([]int64, 0, count)
	start := time.Now()

	for i := 0; i < count; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}

	elapsed := time.Since(start)
	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocDiff := m2.TotalAlloc - m1.TotalAlloc
	heapDiff := m2.HeapAlloc - m1.HeapAlloc

	t.Logf("内存占用测试 (%d IDs):", count)
	t.Logf("  耗时: %v", elapsed)
	t.Logf("  分配内存: %.2f KB", float64(allocDiff)/1024)
	t.Logf("  堆内存: %.2f KB", float64(heapDiff)/1024)
	t.Logf("  单ID内存: %.2f 字节", float64(allocDiff)/float64(count))
	t.Logf("  GC次数: %d", m2.NumGC-m1.NumGC)
}

// TestBatchEfficiency 批量生成效率测试
func TestBatchEfficiency(t *testing.T) {
	gen, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name      string
		batchSize int
		batches   int
	}{
		{"小批量", 100, 1000},
		{"中批量", 1000, 100},
		{"大批量", 10000, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			totalIDs := 0

			for i := 0; i < tc.batches; i++ {
				ids, err := gen.NextIDBatch(tc.batchSize)
				if err != nil {
					t.Fatalf("批量生成失败: %v", err)
				}
				totalIDs += len(ids)
			}

			elapsed := time.Since(start)
			rate := float64(totalIDs) / elapsed.Seconds()

			t.Logf("批量=%d, 批次=%d, 总数=%d, 耗时=%v, 速率=%.0f IDs/秒",
				tc.batchSize, tc.batches, totalIDs, elapsed, rate)
		})
	}
}

// TestSequenceOverflow 序列溢出测试
func TestSequenceOverflow(t *testing.T) {
	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 在一个很短的时间内生成大量ID，触发序列溢出
	const targetCount = 50_000 // 超过单毫秒最大值4096

	start := time.Now()
	ids := make([]int64, 0, targetCount)

	for i := 0; i < targetCount; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("生成失败: %v", err)
		}
		ids = append(ids, id)
	}

	elapsed := time.Since(start)
	metrics := gen.GetMetrics()

	t.Logf("序列溢出测试:")
	t.Logf("  生成数量: %d", targetCount)
	t.Logf("  耗时: %v", elapsed)
	t.Logf("  序列溢出次数: %d", metrics["sequence_overflow"])
	t.Logf("  等待次数: %d", metrics["wait_count"])
	t.Logf("  总等待时间: %.2f ms", float64(metrics["total_wait_time_ns"])/1e6)

	// 验证唯一性
	idSet := make(map[int64]bool)
	for _, id := range ids {
		if idSet[id] {
			t.Errorf("发现重复ID: %d", id)
		}
		idSet[id] = true
	}

	if len(idSet) != targetCount {
		t.Errorf("唯一ID数 %d != 目标数 %d", len(idSet), targetCount)
	}
}

// TestPerformanceComparison 性能对比测试
func TestPerformanceComparison(t *testing.T) {
	scenarios := []struct {
		name       string
		goroutines int
		idsEach    int
	}{
		{"低并发_1x10万", 1, 100_000},
		{"中并发_10x1万", 10, 10_000},
		{"高并发_100x1千", 100, 1_000},
		{"超高并发_1000x100", 1000, 100},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			gen, err := NewWithConfig(&Config{
				DatacenterID:  1,
				WorkerID:      1,
				EnableMetrics: true,
			})
			if err != nil {
				t.Fatal(err)
			}

			idMap := &sync.Map{}
			var errorCount int64

			start := time.Now()
			var wg sync.WaitGroup

			for i := 0; i < scenario.goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < scenario.idsEach; j++ {
						id, err := gen.NextID()
						if err != nil {
							atomic.AddInt64(&errorCount, 1)
							return
						}
						idMap.Store(id, struct{}{})
					}
				}()
			}

			wg.Wait()
			elapsed := time.Since(start)

			count := int64(0)
			idMap.Range(func(k, v interface{}) bool {
				count++
				return true
			})

			total := int64(scenario.goroutines * scenario.idsEach)
			rate := float64(total) / elapsed.Seconds()
			metrics := gen.GetMetrics()

			t.Logf("协程=%d, 每协程=%d, 总数=%d, 唯一=%d, 错误=%d",
				scenario.goroutines, scenario.idsEach, total, count, atomic.LoadInt64(&errorCount))
			t.Logf("  耗时=%v, 速率=%.0f IDs/秒, 溢出=%d, 等待=%d",
				elapsed, rate, metrics["sequence_overflow"], metrics["wait_count"])

			if count != total {
				t.Errorf("ID数量异常: 唯一=%d, 期望=%d", count, total)
			}
		})
	}
}

// BenchmarkPerformanceProfile 性能剖析基准测试
func BenchmarkPerformanceProfile(b *testing.B) {
	gen, err := New(1, 1)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("NextID", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := gen.NextID()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NextID_Parallel", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := gen.NextID()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("NextIDBatch_1000", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := gen.NextIDBatch(1000)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestDetailedMetrics 详细指标测试
func TestDetailedMetrics(t *testing.T) {
	gen, err := NewWithConfig(&Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	const rounds = 5
	const idsPerRound = 20_000

	t.Logf("=== 详细性能指标分析 ===")

	for round := 1; round <= rounds; round++ {
		gen.ResetMetrics()
		start := time.Now()

		for i := 0; i < idsPerRound; i++ {
			_, err := gen.NextID()
			if err != nil {
				t.Fatal(err)
			}
		}

		elapsed := time.Since(start)
		metrics := gen.GetMetrics()
		rate := float64(idsPerRound) / elapsed.Seconds()

		t.Logf("Round %d: %d IDs, 耗时=%v, 速率=%.0f IDs/秒, 溢出=%d, 等待=%d",
			round, idsPerRound, elapsed, rate,
			metrics["sequence_overflow"], metrics["wait_count"])
	}
}
