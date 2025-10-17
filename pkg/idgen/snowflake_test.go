package idgen

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
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
			name:         "有效参数_中间值",
			datacenterID: 15,
			workerID:     15,
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

// TestNewSnowflakeWithConfig 测试使用配置创建Snowflake实例
func TestNewSnowflakeWithConfig(t *testing.T) {
	t.Run("配置为nil", func(t *testing.T) {
		_, err := NewSnowflakeWithConfig(nil)
		if err == nil {
			t.Error("期望得到错误，但没有返回错误")
		}
	})

	t.Run("自定义时间函数", func(t *testing.T) {
		mockTime := int64(1700000000000)
		config := &SnowflakeConfig{
			DatacenterID: 1,
			WorkerID:     1,
			TimeFunc: func() int64 {
				return mockTime
			},
		}

		sf, err := NewSnowflakeWithConfig(config)
		if err != nil {
			t.Fatalf("创建失败: %v", err)
		}

		// 验证使用了自定义时间函数
		id, err := sf.NextID()
		if err != nil {
			t.Fatalf("生成ID失败: %v", err)
		}
		if id <= 0 {
			t.Errorf("生成的ID应为正数，得到: %d", id)
		}
	})
}

// TestSnowflakeNextID 测试生成ID的基本功能
func TestSnowflakeNextID(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	t.Run("生成单个ID", func(t *testing.T) {
		id, err := sf.NextID()
		if err != nil {
			t.Errorf("生成ID失败: %v", err)
		}
		if id <= 0 {
			t.Errorf("ID应为正数，得到: %d", id)
		}
	})

	t.Run("生成多个ID_顺序递增", func(t *testing.T) {
		var prevID int64
		for i := 0; i < 100; i++ {
			id, err := sf.NextID()
			if err != nil {
				t.Errorf("第%d次生成ID失败: %v", i, err)
			}
			if id <= prevID {
				t.Errorf("ID应该递增，prevID=%d, currentID=%d", prevID, id)
			}
			prevID = id
		}
	})

	t.Run("生成的ID唯一性", func(t *testing.T) {
		ids := make(map[int64]bool)
		count := 10000

		for i := 0; i < count; i++ {
			id, err := sf.NextID()
			if err != nil {
				t.Fatalf("生成ID失败: %v", err)
			}
			if ids[id] {
				t.Errorf("发现重复ID: %d", id)
			}
			ids[id] = true
		}

		if len(ids) != count {
			t.Errorf("生成的唯一ID数量不正确，期望%d, 得到%d", count, len(ids))
		}
	})
}

// TestSnowflakeConcurrency 测试并发安全性
func TestSnowflakeConcurrency(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	goroutines := 100
	idsPerGoroutine := 1000
	totalIDs := goroutines * idsPerGoroutine

	ids := make(chan int64, totalIDs)
	var wg sync.WaitGroup

	// 启动多个goroutine并发生成ID
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
				ids <- id
			}
		}()
	}

	wg.Wait()
	close(ids)

	// 检查唯一性
	idSet := make(map[int64]bool)
	for id := range ids {
		if idSet[id] {
			t.Errorf("并发场景下发现重复ID: %d", id)
		}
		idSet[id] = true
	}

	if len(idSet) != totalIDs {
		t.Errorf("生成的唯一ID数量不正确，期望%d, 得到%d", totalIDs, len(idSet))
	}

	t.Logf("并发测试通过: %d个goroutine共生成%d个唯一ID", goroutines, len(idSet))
}

// TestSnowflakeClockBackward 测试时钟回拨处理
func TestSnowflakeClockBackward(t *testing.T) {
	mockTime := int64(1700000000000)

	config := &SnowflakeConfig{
		DatacenterID: 1,
		WorkerID:     1,
		TimeFunc: func() int64 {
			return mockTime
		},
	}

	sf, err := NewSnowflakeWithConfig(config)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	// 生成第一个ID
	_, err = sf.NextID()
	if err != nil {
		t.Fatalf("生成ID失败: %v", err)
	}

	// 模拟时钟回拨
	mockTime = mockTime - 100 // 回拨100毫秒

	_, err = sf.NextID()
	if err == nil {
		t.Error("时钟回拨应该返回错误")
	}
	if !errors.Is(err, ErrClockMovedBackwards) {
		t.Errorf("期望 ErrClockMovedBackwards 错误, 得到: %v", err)
	}
}

// TestSnowflakeClockBackwardTolerance 测试时钟回拨容忍
func TestSnowflakeClockBackwardTolerance(t *testing.T) {
	mockTime := int64(1700000000000)

	config := &SnowflakeConfig{
		DatacenterID: 1,
		WorkerID:     1,
		TimeFunc: func() int64 {
			result := mockTime
			// 第三次调用开始恢复正常
			if mockTime < 1700000000000 {
				mockTime = 1700000000000
			}
			return result
		},
	}

	sf, err := NewSnowflakeWithConfig(config)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	// 生成第一个ID
	_, err = sf.NextID()
	if err != nil {
		t.Fatalf("生成ID失败: %v", err)
	}

	// 模拟小幅度时钟回拨（在容忍范围内）
	mockTime = mockTime - 3 // 回拨3毫秒

	// 应该能够成功生成ID
	id, err := sf.NextID()
	if err != nil {
		t.Errorf("小幅度时钟回拨应该被容忍, 错误: %v", err)
	}
	if id <= 0 {
		t.Errorf("生成的ID应为正数，得到: %d", id)
	}
}

// TestParseSnowflakeID 测试解析Snowflake ID
func TestParseSnowflakeID(t *testing.T) {
	sf, err := NewSnowflake(5, 10)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("生成ID失败: %v", err)
	}

	// 测试全局函数
	timestamp, datacenterID, workerID, sequence := ParseSnowflakeID(id)

	if datacenterID != 5 {
		t.Errorf("DatacenterID = %d, 期望 5", datacenterID)
	}
	if workerID != 10 {
		t.Errorf("WorkerID = %d, 期望 10", workerID)
	}
	if sequence < 0 || sequence > MaxSequence {
		t.Errorf("Sequence = %d, 应在 [0, %d] 范围内", sequence, MaxSequence)
	}
	if timestamp <= Epoch {
		t.Errorf("Timestamp = %d, 应大于 Epoch %d", timestamp, Epoch)
	}

	// 测试实例方法
	info, err := sf.Parse(id)
	if err != nil {
		t.Fatalf("解析ID失败: %v", err)
	}

	if info.ID != id {
		t.Errorf("Info.ID = %d, 期望 %d", info.ID, id)
	}
	if info.DatacenterID != 5 {
		t.Errorf("Info.DatacenterID = %d, 期望 5", info.DatacenterID)
	}
	if info.WorkerID != 10 {
		t.Errorf("Info.WorkerID = %d, 期望 10", info.WorkerID)
	}
	if info.Time.IsZero() {
		t.Error("Info.Time 不应为零值")
	}
}

// TestParseInvalidSnowflakeID 测试解析无效ID
func TestParseInvalidSnowflakeID(t *testing.T) {
	sf, _ := NewSnowflake(1, 1)

	tests := []struct {
		name string
		id   int64
	}{
		{"零值", 0},
		{"负数", -1},
		{"负数_大值", -1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sf.Parse(tt.id)
			if err == nil {
				t.Error("解析无效ID应该返回错误")
			}
			if !errors.Is(err, ErrInvalidSnowflakeID) {
				t.Errorf("期望 ErrInvalidSnowflakeID, 得到: %v", err)
			}
		})
	}
}

// TestGetTimestamp 测试从ID中提取时间戳
func TestGetTimestamp(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	beforeGen := time.Now()
	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("生成ID失败: %v", err)
	}
	afterGen := time.Now()

	extractedTime := GetTimestamp(id)

	// 提取的时间应该在生成前后的时间范围内
	if extractedTime.Before(beforeGen.Add(-time.Second)) || extractedTime.After(afterGen.Add(time.Second)) {
		t.Errorf("提取的时间 %v 不在预期范围内 [%v, %v]", extractedTime, beforeGen, afterGen)
	}
}

// TestValidateSnowflakeID 测试验证ID有效性
func TestValidateSnowflakeID(t *testing.T) {
	sf, _ := NewSnowflake(1, 1)

	t.Run("有效ID", func(t *testing.T) {
		id, err := sf.NextID()
		if err != nil {
			t.Fatalf("生成ID失败: %v", err)
		}

		err = ValidateSnowflakeID(id)
		if err != nil {
			t.Errorf("有效ID验证失败: %v", err)
		}
	})

	t.Run("无效ID_零值", func(t *testing.T) {
		err := ValidateSnowflakeID(0)
		if err == nil {
			t.Error("零值ID应该验证失败")
		}
	})

	t.Run("无效ID_负数", func(t *testing.T) {
		err := ValidateSnowflakeID(-1)
		if err == nil {
			t.Error("负数ID应该验证失败")
		}
	})
}

// TestSnowflakeIDCount 测试ID计数器
func TestSnowflakeIDCount(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	initialCount := sf.GetIDCount()
	if initialCount != 0 {
		t.Errorf("初始计数应为0, 得到: %d", initialCount)
	}

	count := 100
	for i := 0; i < count; i++ {
		_, err := sf.NextID()
		if err != nil {
			t.Fatalf("生成ID失败: %v", err)
		}
	}

	finalCount := sf.GetIDCount()
	if finalCount != uint64(count) {
		t.Errorf("最终计数应为%d, 得到: %d", count, finalCount)
	}
}

// TestSequenceOverflow 测试序列号溢出处理
func TestSequenceOverflow(t *testing.T) {
	mockTime := int64(1700000000000)
	callCount := 0

	config := &SnowflakeConfig{
		DatacenterID: 1,
		WorkerID:     1,
		TimeFunc: func() int64 {
			// 前4097次调用返回相同时间，第4098次开始返回新时间
			if callCount < 4097 {
				callCount++
				return mockTime
			}
			return mockTime + 1
		},
	}

	sf, err := NewSnowflakeWithConfig(config)
	if err != nil {
		t.Fatalf("创建Snowflake失败: %v", err)
	}

	// 生成4096个ID（耗尽同一毫秒内的序列号）
	for i := 0; i < 4096; i++ {
		_, err := sf.NextID()
		if err != nil {
			t.Fatalf("生成第%d个ID失败: %v", i, err)
		}
	}

	// 第4097个ID应该触发等待下一毫秒
	id, err := sf.NextID()
	if err != nil {
		t.Fatalf("序列号溢出后生成ID失败: %v", err)
	}
	if id <= 0 {
		t.Errorf("生成的ID应为正数，得到: %d", id)
	}
}

// BenchmarkSnowflakeNextID 基准测试：单goroutine生成ID
func BenchmarkSnowflakeNextID(b *testing.B) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		b.Fatalf("创建Snowflake失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sf.NextID()
		if err != nil {
			b.Fatalf("生成ID失败: %v", err)
		}
	}
}

// BenchmarkSnowflakeNextIDParallel 基准测试：并发生成ID
func BenchmarkSnowflakeNextIDParallel(b *testing.B) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		b.Fatalf("创建Snowflake失败: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := sf.NextID()
			if err != nil {
				b.Fatalf("生成ID失败: %v", err)
			}
		}
	})
}

// BenchmarkParseSnowflakeID 基准测试：解析ID
func BenchmarkParseSnowflakeID(b *testing.B) {
	sf, _ := NewSnowflake(1, 1)
	id, _ := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseSnowflakeID(id)
	}
}

// BenchmarkSnowflakeParse 基准测试：实例方法解析ID
func BenchmarkSnowflakeParse(b *testing.B) {
	sf, _ := NewSnowflake(1, 1)
	id, _ := sf.NextID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sf.Parse(id)
		if err != nil {
			b.Fatalf("解析ID失败: %v", err)
		}
	}
}

// ExampleSnowflake_NextID 示例：基本使用
func ExampleSnowflake_NextID() {
	// 创建Snowflake实例
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		fmt.Printf("创建失败: %v\n", err)
		return
	}

	// 生成ID
	id, err := sf.NextID()
	if err != nil {
		fmt.Printf("生成ID失败: %v\n", err)
		return
	}

	fmt.Printf("生成的ID: %d\n", id)
}

// ExampleParseSnowflakeID 示例：解析ID
func ExampleParseSnowflakeID() {
	sf, _ := NewSnowflake(5, 10)
	id, _ := sf.NextID()

	// 解析ID
	timestamp, datacenterID, workerID, sequence := ParseSnowflakeID(id)

	fmt.Printf("时间戳: %d\n", timestamp)
	fmt.Printf("数据中心ID: %d\n", datacenterID)
	fmt.Printf("工作机器ID: %d\n", workerID)
	fmt.Printf("序列号: %d\n", sequence)
}

// ExampleSnowflake_Parse 示例：使用实例方法解析ID
func ExampleSnowflake_Parse() {
	sf, _ := NewSnowflake(5, 10)
	id, _ := sf.NextID()

	// 解析ID获取详细信息
	info, err := sf.Parse(id)
	if err != nil {
		fmt.Printf("解析失败: %v\n", err)
		return
	}

	fmt.Printf("ID: %d\n", info.ID)
	fmt.Printf("时间: %v\n", info.Time)
	fmt.Printf("数据中心ID: %d\n", info.DatacenterID)
	fmt.Printf("工作机器ID: %d\n", info.WorkerID)
	fmt.Printf("序列号: %d\n", info.Sequence)
}
