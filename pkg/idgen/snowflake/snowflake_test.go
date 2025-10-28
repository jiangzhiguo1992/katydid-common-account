package snowflake_test

import (
	"katydid-common-account/pkg/idgen/core"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"katydid-common-account/pkg/idgen/snowflake"
)

// ============================================================================
// 1. Snowflake生成器基础功能测试
// ============================================================================

// TestNew 测试创建生成器
func TestNew(t *testing.T) {
	tests := []struct {
		name         string
		datacenterID int64
		workerID     int64
		wantErr      bool
	}{
		{"正常创建", 1, 1, false},
		{"边界值_最小", 0, 0, false},
		{"边界值_最大", 31, 31, false},
		{"无效_datacenterID负数", -1, 1, true},
		{"无效_datacenterID超出", 32, 1, true},
		{"无效_workerID负数", 1, -1, true},
		{"无效_workerID超出", 1, 32, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := snowflake.New(tt.datacenterID, tt.workerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gen == nil {
				t.Error("New() returned nil generator")
			}
		})
	}
}

// TestNewWithConfig 测试使用配置创建生成器
func TestNewWithConfig(t *testing.T) {
	t.Run("nil配置", func(t *testing.T) {
		_, err := snowflake.NewWithConfig(nil)
		if err == nil {
			t.Error("NewWithConfig(nil) should return error")
		}
	})

	t.Run("启用监控", func(t *testing.T) {
		config := &snowflake.Config{
			DatacenterID:  1,
			WorkerID:      1,
			EnableMetrics: true,
		}
		gen, err := snowflake.NewWithConfig(config)
		if err != nil {
			t.Fatalf("NewWithConfig() error = %v", err)
		}
		metrics := gen.GetMetrics()
		if _, ok := metrics["id_count"]; !ok {
			t.Error("Metrics should be enabled")
		}
	})
}

// TestNextID 测试生成单个ID
func TestNextID(t *testing.T) {
	gen, err := snowflake.New(1, 1)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	t.Run("生成单个ID", func(t *testing.T) {
		id, err := gen.NextID()
		if err != nil {
			t.Errorf("NextID() error = %v", err)
		}
		if id <= 0 {
			t.Errorf("NextID() = %d, want positive", id)
		}
	})

	t.Run("生成多个ID_唯一性", func(t *testing.T) {
		ids := make(map[int64]bool)
		const count = 10000

		for i := 0; i < count; i++ {
			id, err := gen.NextID()
			if err != nil {
				t.Fatalf("NextID() error = %v", err)
			}
			if ids[id] {
				t.Fatalf("Duplicate ID detected: %d", id)
			}
			ids[id] = true
		}

		if len(ids) != count {
			t.Errorf("Generated %d unique IDs, want %d", len(ids), count)
		}
	})

	t.Run("ID单调递增", func(t *testing.T) {
		prevID := int64(0)
		for i := 0; i < 1000; i++ {
			id, err := gen.NextID()
			if err != nil {
				t.Fatalf("NextID() error = %v", err)
			}
			if id <= prevID {
				t.Errorf("ID not monotonic: prev=%d, current=%d", prevID, id)
			}
			prevID = id
		}
	})
}

// TestNextIDBatch 测试批量生成ID
func TestNextIDBatch(t *testing.T) {
	gen, err := snowflake.New(1, 1)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		name    string
		n       int
		wantErr bool
	}{
		{"正常批量_小", 10, false},
		{"正常批量_中", 1000, false},
		{"正常批量_大", 10000, false},
		{"无效_零", 0, true},
		{"无效_负数", -1, true},
		{"无效_超出限制", 100001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := gen.NextIDBatch(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextIDBatch(%d) error = %v, wantErr %v", tt.n, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(ids) != tt.n {
					t.Errorf("NextIDBatch(%d) returned %d IDs", tt.n, len(ids))
				}
				// 验证唯一性
				idSet := make(map[int64]bool)
				for _, id := range ids {
					if idSet[id] {
						t.Fatalf("Duplicate ID in batch: %d", id)
					}
					idSet[id] = true
				}
			}
		})
	}
}

// TestGetWorkerID 测试获取WorkerID
func TestGetWorkerID(t *testing.T) {
	tests := []struct {
		name     string
		workerID int64
	}{
		{"WorkerID_0", 0},
		{"WorkerID_15", 15},
		{"WorkerID_31", 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, _ := snowflake.New(1, tt.workerID)
			if got := gen.GetWorkerID(); got != tt.workerID {
				t.Errorf("GetWorkerID() = %d, want %d", got, tt.workerID)
			}
		})
	}
}

// TestGetDatacenterID 测试获取DatacenterID
func TestGetDatacenterID(t *testing.T) {
	tests := []struct {
		name         string
		datacenterID int64
	}{
		{"DatacenterID_0", 0},
		{"DatacenterID_15", 15},
		{"DatacenterID_31", 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, _ := snowflake.New(tt.datacenterID, 1)
			if got := gen.GetDatacenterID(); got != tt.datacenterID {
				t.Errorf("GetDatacenterID() = %d, want %d", got, tt.datacenterID)
			}
		})
	}
}

// TestMetrics 测试性能监控
func TestMetrics(t *testing.T) {
	config := &snowflake.Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	}
	gen, err := snowflake.NewWithConfig(config)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 生成一些ID
	const count = 1000
	for i := 0; i < count; i++ {
		_, _ = gen.NextID()
	}

	t.Run("获取指标", func(t *testing.T) {
		metrics := gen.GetMetrics()
		if metrics == nil {
			t.Fatal("GetMetrics() returned nil")
		}
		idCount, ok := metrics["id_count"]
		if !ok {
			t.Error("Metrics missing 'id_count'")
		}
		if idCount < count {
			t.Errorf("id_count = %d, want >= %d", idCount, count)
		}
	})

	t.Run("重置指标", func(t *testing.T) {
		gen.ResetMetrics()
		idCount := gen.GetIDCount()
		if idCount != 0 {
			t.Errorf("GetIDCount() after reset = %d, want 0", idCount)
		}
	})
}

// TestParseID 测试ID解析
func TestParseID(t *testing.T) {
	gen, _ := snowflake.New(5, 10)

	id, err := gen.NextID()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	info, err := gen.ParseID(id)
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}

	if info.ID != id {
		t.Errorf("ParseID().ID = %d, want %d", info.ID, id)
	}
	if info.DatacenterID != 5 {
		t.Errorf("ParseID().DatacenterID = %d, want 5", info.DatacenterID)
	}
	if info.WorkerID != 10 {
		t.Errorf("ParseID().WorkerID = %d, want 10", info.WorkerID)
	}
	if info.Timestamp <= 0 {
		t.Error("ParseID().Timestamp should be positive")
	}
	if info.Sequence < 0 {
		t.Error("ParseID().Sequence should be non-negative")
	}
}

// TestValidateID 测试ID验证
func TestValidateID(t *testing.T) {
	gen, _ := snowflake.New(1, 1)

	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{"有效ID", 123456789, false},
		{"零值", 0, true},
		{"负数", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.ValidateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID(%d) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// 2. 并发测试
// ============================================================================

// TestConcurrentGeneration 测试并发生成ID
func TestConcurrentGeneration(t *testing.T) {
	gen, err := snowflake.New(1, 1)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	const goroutines = 100
	const idsPerGoroutine = 1000
	const totalIDs = goroutines * idsPerGoroutine

	idChan := make(chan int64, totalIDs)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 并发生成ID
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := gen.NextID()
				if err != nil {
					t.Errorf("NextID() error = %v", err)
					return
				}
				idChan <- id
			}
		}()
	}

	wg.Wait()
	close(idChan)

	// 验证唯一性
	ids := make(map[int64]bool)
	for id := range idChan {
		if ids[id] {
			t.Fatalf("Duplicate ID detected: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != totalIDs {
		t.Errorf("Generated %d unique IDs, want %d", len(ids), totalIDs)
	}
}

// ============================================================================
// 3. 百万级高并发测试
// ============================================================================

// TestMillionConcurrentGeneration 百万级并发ID生成测试
func TestMillionConcurrentGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	config := &snowflake.Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	}
	gen, err := snowflake.NewWithConfig(config)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	const totalIDs = 1_000_000
	goroutines := runtime.NumCPU() * 100
	idsPerGoroutine := totalIDs / goroutines

	t.Logf("开始百万级并发测试:")
	t.Logf("  - 目标ID数: %d", totalIDs)
	t.Logf("  - 协程数: %d", goroutines)
	t.Logf("  - 每协程ID数: %d", idsPerGoroutine)
	t.Logf("  - CPU核心数: %d", runtime.NumCPU())

	startTime := time.Now()
	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64

	// 用于检测ID唯一性（采样）
	idSet := sync.Map{}
	var duplicateCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(gid int) {
			defer wg.Done()

			for j := 0; j < idsPerGoroutine; j++ {
				id, err := gen.NextID()
				if err != nil {
					errorCount.Add(1)
					continue
				}

				successCount.Add(1)

				// 采样检测唯一性（每100个检测一次，避免内存溢出）
				if j%100 == 0 {
					if _, exists := idSet.LoadOrStore(id, struct{}{}); exists {
						duplicateCount.Add(1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 获取最终指标
	metrics := gen.GetMetrics()

	// 输出测试结果
	t.Logf("\n========== 百万级并发测试结果 ==========")
	t.Logf("【耗时统计】")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 平均每秒生成: %.2f IDs/sec", float64(successCount.Load())/duration.Seconds())
	t.Logf("  - 平均延迟: %v/ID", duration/time.Duration(successCount.Load()))

	t.Logf("\n【成功率统计】")
	t.Logf("  - 成功生成: %d", successCount.Load())
	t.Logf("  - 失败次数: %d", errorCount.Load())
	t.Logf("  - 成功率: %.4f%%", float64(successCount.Load())/float64(totalIDs)*100)

	t.Logf("\n【唯一性统计】")
	t.Logf("  - 采样重复数: %d", duplicateCount.Load())
	t.Logf("  - 唯一性检测: %s", func() string {
		if duplicateCount.Load() == 0 {
			return "✓ 通过"
		}
		return "✗ 失败"
	}())

	t.Logf("\n【性能指标】")
	if idCount, ok := metrics["id_count"]; ok {
		t.Logf("  - ID总数: %d", idCount)
	}
	if seqOverflow, ok := metrics["sequence_overflow"]; ok {
		t.Logf("  - 序列号溢出次数: %d", seqOverflow)
	}
	if waitCount, ok := metrics["wait_count"]; ok {
		t.Logf("  - 等待次数: %d", waitCount)
	}
	if totalWaitNs, ok := metrics["total_wait_time_ns"]; ok {
		t.Logf("  - 总等待时间: %v", time.Duration(totalWaitNs))
	}

	// 验证结果
	if duplicateCount.Load() > 0 {
		t.Errorf("检测到 %d 个重复ID", duplicateCount.Load())
	}

	if successCount.Load() < int64(totalIDs*95/100) {
		t.Errorf("成功率过低: %.2f%%, 期望 >= 95%%",
			float64(successCount.Load())/float64(totalIDs)*100)
	}
}

// TestMillionBatchGeneration 百万级批量生成测试
func TestMillionBatchGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级批量测试")
	}

	gen, err := snowflake.New(1, 1)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	const totalIDs = 1_000_000
	const batchSize = 1000
	batches := totalIDs / batchSize

	t.Logf("开始百万级批量生成测试: 总ID=%d, 批次大小=%d, 批次数=%d",
		totalIDs, batchSize, batches)

	startTime := time.Now()
	idSet := make(map[int64]bool, totalIDs)

	for i := 0; i < batches; i++ {
		ids, err := gen.NextIDBatch(batchSize)
		if err != nil {
			t.Fatalf("NextIDBatch() error = %v", err)
		}

		// 验证批次内唯一性
		for _, id := range ids {
			if idSet[id] {
				t.Fatalf("Duplicate ID detected: %d", id)
			}
			idSet[id] = true
		}
	}

	duration := time.Since(startTime)

	t.Logf("百万级批量生成测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 生成ID数: %d", len(idSet))
	t.Logf("  - QPS: %.2f IDs/sec", float64(len(idSet))/duration.Seconds())
	t.Logf("  - 批次处理速度: %.2f batches/sec", float64(batches)/duration.Seconds())

	if len(idSet) != totalIDs {
		t.Errorf("Generated %d unique IDs, want %d", len(idSet), totalIDs)
	}
}

// TestMillionMultiGenerator 百万级多生成器测试
func TestMillionMultiGenerator(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级多生成器测试")
	}

	const numGenerators = 10
	const idsPerGenerator = 100_000
	const totalIDs = numGenerators * idsPerGenerator

	// 创建多个生成器（不同的datacenterID和workerID）
	generators := make([]core.IGenerator, numGenerators)
	for i := 0; i < numGenerators; i++ {
		gen, err := snowflake.New(int64(i/5), int64(i%5))
		if err != nil {
			t.Fatalf("Failed to create generator %d: %v", i, err)
		}
		generators[i] = gen
	}

	t.Logf("开始百万级多生成器测试: 生成器数=%d, 每生成器=%d IDs, 总计=%d",
		numGenerators, idsPerGenerator, totalIDs)

	startTime := time.Now()
	var wg sync.WaitGroup
	var successCount atomic.Int64
	idSet := sync.Map{}
	var duplicateCount atomic.Int64

	wg.Add(numGenerators)
	for i := 0; i < numGenerators; i++ {
		go func(genIdx int) {
			defer wg.Done()
			gen := generators[genIdx]

			for j := 0; j < idsPerGenerator; j++ {
				id, err := gen.NextID()
				if err != nil {
					continue
				}

				successCount.Add(1)

				// 检测跨生成器唯一性
				if _, exists := idSet.LoadOrStore(id, genIdx); exists {
					duplicateCount.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级多生成器测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 成功生成: %d", successCount.Load())
	t.Logf("  - 跨生成器重复: %d", duplicateCount.Load())
	t.Logf("  - QPS: %.2f IDs/sec", float64(successCount.Load())/duration.Seconds())
	t.Logf("  - 每生成器QPS: %.2f IDs/sec",
		float64(successCount.Load())/duration.Seconds()/float64(numGenerators))

	if duplicateCount.Load() > 0 {
		t.Errorf("跨生成器检测到 %d 个重复ID", duplicateCount.Load())
	}
}

// ============================================================================
// 4. 性能基准测试
// ============================================================================

// BenchmarkNextID 基准测试单个ID生成
func BenchmarkNextID(b *testing.B) {
	gen, _ := snowflake.New(1, 1)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = gen.NextID()
	}
}

// BenchmarkNextID_WithMetrics 基准测试带监控的ID生成
func BenchmarkNextID_WithMetrics(b *testing.B) {
	config := &snowflake.Config{
		DatacenterID:  1,
		WorkerID:      1,
		EnableMetrics: true,
	}
	gen, _ := snowflake.NewWithConfig(config)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = gen.NextID()
	}
}

// BenchmarkNextIDBatch 基准测试批量生成
func BenchmarkNextIDBatch(b *testing.B) {
	gen, _ := snowflake.New(1, 1)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = gen.NextIDBatch(100)
	}
}

// BenchmarkNextID_Parallel 并行基准测试ID生成
func BenchmarkNextID_Parallel(b *testing.B) {
	gen, _ := snowflake.New(1, 1)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = gen.NextID()
		}
	})
}

// BenchmarkParseID 基准测试ID解析
func BenchmarkParseID(b *testing.B) {
	gen, _ := snowflake.New(5, 10)
	id, _ := gen.NextID()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = gen.ParseID(id)
	}
}

// BenchmarkValidateID 基准测试ID验证
func BenchmarkValidateID(b *testing.B) {
	gen, _ := snowflake.New(1, 1)
	id, _ := gen.NextID()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = gen.ValidateID(id)
	}
}

// ============================================================================
// 5. 压力测试
// ============================================================================

// TestStressTest 持续压力测试
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过压力测试")
	}

	gen, _ := snowflake.New(1, 1)

	const duration = 10 * time.Second
	const goroutines = 500

	t.Logf("开始压力测试: 持续时间=%v, 协程数=%d", duration, goroutines)

	startTime := time.Now()
	stopTime := startTime.Add(duration)
	var totalOps atomic.Int64
	var errors atomic.Int64

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			localOps := int64(0)

			for time.Now().Before(stopTime) {
				if _, err := gen.NextID(); err != nil {
					errors.Add(1)
				} else {
					localOps++
				}
			}

			totalOps.Add(localOps)
		}()
	}

	wg.Wait()
	actualDuration := time.Since(startTime)

	ops := totalOps.Load()
	t.Logf("压力测试完成:")
	t.Logf("  - 实际耗时: %v", actualDuration)
	t.Logf("  - 总操作数: %d", ops)
	t.Logf("  - 错误次数: %d", errors.Load())
	t.Logf("  - 平均QPS: %.2f ops/sec", float64(ops)/actualDuration.Seconds())
	t.Logf("  - 每协程QPS: %.2f ops/sec",
		float64(ops)/actualDuration.Seconds()/float64(goroutines))
	t.Logf("  - 错误率: %.4f%%", float64(errors.Load())/float64(ops)*100)
}

// ============================================================================
// 6. 内存性能测试
// ============================================================================

// BenchmarkMemoryAllocation 基准测试内存分配
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("创建生成器", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = snowflake.New(1, 1)
		}
	})

	b.Run("生成ID", func(b *testing.B) {
		gen, _ := snowflake.New(1, 1)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = gen.NextID()
		}
	})

	b.Run("批量生成", func(b *testing.B) {
		gen, _ := snowflake.New(1, 1)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = gen.NextIDBatch(100)
		}
	})
}
