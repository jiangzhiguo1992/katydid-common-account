package domain_test

import (
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"katydid-common-account/pkg/idgen/domain"
)

// ============================================================================
// 1. ID类型测试
// ============================================================================

// TestNewID 测试ID创建
func TestNewID(t *testing.T) {
	tests := []struct {
		name string
		val  int64
		want int64
	}{
		{"零值", 0, 0},
		{"正数", 123456, 123456},
		{"大数", 9007199254740991, 9007199254740991},
		{"负数", -1, -1}, // ID内部可以是负数，但使用时需验证
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := domain.NewID(tt.val)
			if id.Int64() != tt.want {
				t.Errorf("NewID(%d).Int64() = %d, want %d", tt.val, id.Int64(), tt.want)
			}
		})
	}
}

// TestParseID 测试ID解析
func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"十进制_正常", "123456", 123456, false},
		{"十六进制_小写", "0x1e240", 123456, false},
		{"十六进制_大写", "0X1E240", 123456, false},
		{"二进制", "0b11110001001000000", 123456, false},
		{"零", "0", 0, false},
		{"空字符串", "", 0, true},
		{"无效字符", "abc", 0, true},
		{"负数", "-123", 0, true},
		{"超长字符串", strings.Repeat("1", 101), 0, true},
		{"十六进制_无数字", "0x", 0, true},
		{"二进制_无数字", "0b", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Int64() != tt.want {
				t.Errorf("ParseID(%q) = %d, want %d", tt.input, got.Int64(), tt.want)
			}
		})
	}
}

// TestID_String 测试ID字符串转换
func TestID_String(t *testing.T) {
	tests := []struct {
		name string
		id   domain.ID
		want string
	}{
		{"零", domain.NewID(0), "0"},
		{"正数", domain.NewID(123456), "123456"},
		{"大数", domain.NewID(9007199254740991), "9007199254740991"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestID_Hex 测试ID十六进制转换
func TestID_Hex(t *testing.T) {
	tests := []struct {
		name string
		id   domain.ID
		want string
	}{
		{"零", domain.NewID(0), "0x0"},
		{"正数", domain.NewID(255), "0xff"},
		{"大数", domain.NewID(123456), "0x1e240"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Hex(); got != tt.want {
				t.Errorf("Hex() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// 2. IDSlice测试
// ============================================================================

// TestNewIDSlice 测试IDSlice创建
func TestNewIDSlice(t *testing.T) {
	tests := []struct {
		name string
		ids  []domain.ID
		want int
	}{
		{"空切片", []domain.ID{}, 0},
		{"单个ID", []domain.ID{domain.NewID(1)}, 1},
		{"多个ID", []domain.ID{domain.NewID(1), domain.NewID(2), domain.NewID(3)}, 3},
		{"nil输入", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slice := domain.NewIDSlice(tt.ids...)
			if slice.Len() != tt.want {
				t.Errorf("Len() = %d, want %d", slice.Len(), tt.want)
			}
		})
	}
}

// TestIDSlice_Int64Slice 测试转换为int64切片
func TestIDSlice_Int64Slice(t *testing.T) {
	ids := domain.NewIDSlice(
		domain.NewID(1),
		domain.NewID(2),
		domain.NewID(3),
	)

	got := ids.Int64Slice()
	want := []int64{1, 2, 3}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}

	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Int64Slice()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

// TestIDSlice_StringSlice 测试转换为字符串切片
func TestIDSlice_StringSlice(t *testing.T) {
	ids := domain.NewIDSlice(
		domain.NewID(1),
		domain.NewID(2),
		domain.NewID(3),
	)

	got := ids.StringSlice()
	want := []string{"1", "2", "3"}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}

	for i := range got {
		if got[i] != want[i] {
			t.Errorf("StringSlice()[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

// TestIDSlice_Contains 测试包含检查
func TestIDSlice_Contains(t *testing.T) {
	ids := domain.NewIDSlice(
		domain.NewID(1),
		domain.NewID(2),
		domain.NewID(3),
	)

	tests := []struct {
		name string
		id   domain.ID
		want bool
	}{
		{"存在_第一个", domain.NewID(1), true},
		{"存在_中间", domain.NewID(2), true},
		{"存在_最后", domain.NewID(3), true},
		{"不存在", domain.NewID(4), false},
		{"零值", domain.NewID(0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ids.Contains(tt.id); got != tt.want {
				t.Errorf("Contains(%d) = %v, want %v", tt.id.Int64(), got, tt.want)
			}
		})
	}
}

// TestIDSlice_FirstLast 测试首尾元素访问
func TestIDSlice_FirstLast(t *testing.T) {
	t.Run("非空切片", func(t *testing.T) {
		ids := domain.NewIDSlice(
			domain.NewID(1),
			domain.NewID(2),
			domain.NewID(3),
		)

		first, ok := ids.First()
		if !ok || first.Int64() != 1 {
			t.Errorf("First() = (%d, %v), want (1, true)", first.Int64(), ok)
		}

		last, ok := ids.Last()
		if !ok || last.Int64() != 3 {
			t.Errorf("Last() = (%d, %v), want (3, true)", last.Int64(), ok)
		}
	})

	t.Run("空切片", func(t *testing.T) {
		ids := domain.NewIDSlice()

		_, ok := ids.First()
		if ok {
			t.Error("First() on empty slice should return false")
		}

		_, ok = ids.Last()
		if ok {
			t.Error("Last() on empty slice should return false")
		}
	})
}

// TestIDSlice_Deduplicate 测试去重
func TestIDSlice_Deduplicate(t *testing.T) {
	tests := []struct {
		name string
		ids  domain.IDSlice
		want int
	}{
		{
			"无重复",
			domain.NewIDSlice(domain.NewID(1), domain.NewID(2), domain.NewID(3)),
			3,
		},
		{
			"有重复",
			domain.NewIDSlice(domain.NewID(1), domain.NewID(2), domain.NewID(1), domain.NewID(3), domain.NewID(2)),
			3,
		},
		{
			"全部重复",
			domain.NewIDSlice(domain.NewID(1), domain.NewID(1), domain.NewID(1)),
			1,
		},
		{
			"空切片",
			domain.NewIDSlice(),
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ids.Deduplicate()
			if result.Len() != tt.want {
				t.Errorf("Deduplicate().Len() = %d, want %d", result.Len(), tt.want)
			}
		})
	}
}

// TestIDSlice_Filter 测试过滤
func TestIDSlice_Filter(t *testing.T) {
	ids := domain.NewIDSlice(
		domain.NewID(1),
		domain.NewID(2),
		domain.NewID(3),
		domain.NewID(4),
		domain.NewID(5),
	)

	t.Run("过滤偶数", func(t *testing.T) {
		result := ids.Filter(func(id domain.ID) bool {
			return id.Int64()%2 == 0
		})
		if result.Len() != 2 {
			t.Errorf("Filter() len = %d, want 2", result.Len())
		}
	})

	t.Run("nil谓词", func(t *testing.T) {
		result := ids.Filter(nil)
		if result.Len() != ids.Len() {
			t.Errorf("Filter(nil) len = %d, want %d", result.Len(), ids.Len())
		}
	})
}

// ============================================================================
// 3. IDSet测试
// ============================================================================

// TestNewIDSet 测试IDSet创建
func TestNewIDSet(t *testing.T) {
	tests := []struct {
		name string
		ids  []domain.ID
		want int
	}{
		{"空集合", []domain.ID{}, 0},
		{"单个ID", []domain.ID{domain.NewID(1)}, 1},
		{"多个ID_无重复", []domain.ID{domain.NewID(1), domain.NewID(2), domain.NewID(3)}, 3},
		{"多个ID_有重复", []domain.ID{domain.NewID(1), domain.NewID(2), domain.NewID(1)}, 2},
		{"nil输入", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := domain.NewIDSet(tt.ids...)
			if set.Size() != tt.want {
				t.Errorf("Size() = %d, want %d", set.Size(), tt.want)
			}
		})
	}
}

// TestIDSet_AddRemove 测试添加和删除
func TestIDSet_AddRemove(t *testing.T) {
	set := domain.NewIDSet()

	// 测试添加
	set.Add(domain.NewID(1))
	if !set.Contains(domain.NewID(1)) {
		t.Error("Add() failed, ID not in set")
	}
	if set.Size() != 1 {
		t.Errorf("Size() = %d, want 1", set.Size())
	}

	// 测试重复添加
	set.Add(domain.NewID(1))
	if set.Size() != 1 {
		t.Errorf("Duplicate Add(), Size() = %d, want 1", set.Size())
	}

	// 测试删除
	set.Remove(domain.NewID(1))
	if set.Contains(domain.NewID(1)) {
		t.Error("Remove() failed, ID still in set")
	}
	if set.Size() != 0 {
		t.Errorf("Size() = %d, want 0", set.Size())
	}
}

// TestIDSet_Contains 测试包含检查
func TestIDSet_Contains(t *testing.T) {
	set := domain.NewIDSet(
		domain.NewID(1),
		domain.NewID(2),
		domain.NewID(3),
	)

	tests := []struct {
		name string
		id   domain.ID
		want bool
	}{
		{"存在", domain.NewID(1), true},
		{"不存在", domain.NewID(4), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := set.Contains(tt.id); got != tt.want {
				t.Errorf("Contains(%d) = %v, want %v", tt.id.Int64(), got, tt.want)
			}
		})
	}
}

// TestIDSet_Operations 测试集合操作
func TestIDSet_Operations(t *testing.T) {
	set1 := domain.NewIDSet(domain.NewID(1), domain.NewID(2), domain.NewID(3))
	set2 := domain.NewIDSet(domain.NewID(2), domain.NewID(3), domain.NewID(4))

	t.Run("并集", func(t *testing.T) {
		union := set1.Union(set2)
		if union.Size() != 4 {
			t.Errorf("Union().Size() = %d, want 4", union.Size())
		}
		for _, id := range []int64{1, 2, 3, 4} {
			if !union.Contains(domain.NewID(id)) {
				t.Errorf("Union() missing ID %d", id)
			}
		}
	})

	t.Run("交集", func(t *testing.T) {
		intersect := set1.Intersect(set2)
		if intersect.Size() != 2 {
			t.Errorf("Intersect().Size() = %d, want 2", intersect.Size())
		}
		for _, id := range []int64{2, 3} {
			if !intersect.Contains(domain.NewID(id)) {
				t.Errorf("Intersect() missing ID %d", id)
			}
		}
	})

	t.Run("差集", func(t *testing.T) {
		diff := set1.Difference(set2)
		if diff.Size() != 1 {
			t.Errorf("Difference().Size() = %d, want 1", diff.Size())
		}
		if !diff.Contains(domain.NewID(1)) {
			t.Error("Difference() should contain ID 1")
		}
	})

	t.Run("相等性", func(t *testing.T) {
		set3 := domain.NewIDSet(domain.NewID(1), domain.NewID(2), domain.NewID(3))
		if !set1.Equal(set3) {
			t.Error("Equal() should return true for identical sets")
		}
		if set1.Equal(set2) {
			t.Error("Equal() should return false for different sets")
		}
	})
}

// TestIDSet_Clone 测试克隆
func TestIDSet_Clone(t *testing.T) {
	original := domain.NewIDSet(domain.NewID(1), domain.NewID(2))
	clone := original.Clone()

	// 验证内容相同
	if !original.Equal(clone) {
		t.Error("Clone() should create equal set")
	}

	// 验证独立性
	clone.Add(domain.NewID(3))
	if original.Contains(domain.NewID(3)) {
		t.Error("Modifying clone should not affect original")
	}
}

// ============================================================================
// 4. 百万级高并发测试
// ============================================================================

// TestID_MillionConcurrent 百万级并发测试ID操作
func TestID_MillionConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	const totalOps = 1_000_000
	goroutines := runtime.NumCPU() * 100
	opsPerGoroutine := totalOps / goroutines

	t.Logf("开始百万级并发测试: 总操作=%d, 协程数=%d", totalOps, goroutines)

	startTime := time.Now()
	var wg sync.WaitGroup
	var successCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(gid int) {
			defer wg.Done()
			localSuccess := 0

			for j := 0; j < opsPerGoroutine; j++ {
				id := domain.NewID(int64(gid*opsPerGoroutine + j))

				// 测试多种操作
				_ = id.Int64()
				_ = id.String()
				_ = id.Hex()
				localSuccess++
			}

			successCount.Add(int64(localSuccess))
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级并发测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 成功操作: %d", successCount.Load())
	t.Logf("  - QPS: %.2f ops/sec", float64(totalOps)/duration.Seconds())
}

// TestIDSlice_MillionConcurrent 百万级并发测试IDSlice
func TestIDSlice_MillionConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	const totalOps = 1_000_000
	goroutines := runtime.NumCPU() * 50

	// 创建测试数据
	testSlice := make([]domain.ID, 1000)
	for i := range testSlice {
		testSlice[i] = domain.NewID(int64(i))
	}
	ids := domain.NewIDSlice(testSlice...)

	t.Logf("开始百万级并发测试: 总操作=%d, 协程数=%d", totalOps, goroutines)

	startTime := time.Now()
	var wg sync.WaitGroup
	var opsCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			localOps := 0

			for j := 0; j < totalOps/goroutines; j++ {
				// 测试多种操作
				_ = ids.Len()
				_ = ids.Contains(domain.NewID(int64(j % 1000)))
				_, _ = ids.First()
				_, _ = ids.Last()
				localOps++
			}

			opsCount.Add(int64(localOps))
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级并发测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 总操作数: %d", opsCount.Load())
	t.Logf("  - QPS: %.2f ops/sec", float64(opsCount.Load())/duration.Seconds())
}

// TestIDSet_MillionConcurrent 百万级并发读测试IDSet
func TestIDSet_MillionConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过百万级并发测试")
	}

	const totalOps = 1_000_000
	goroutines := runtime.NumCPU() * 50

	// 创建测试数据
	testIDs := make([]domain.ID, 1000)
	for i := range testIDs {
		testIDs[i] = domain.NewID(int64(i))
	}
	set := domain.NewIDSet(testIDs...)

	t.Logf("开始百万级并发读测试: 总操作=%d, 协程数=%d", totalOps, goroutines)

	startTime := time.Now()
	var wg sync.WaitGroup
	var opsCount atomic.Int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			localOps := 0

			for j := 0; j < totalOps/goroutines; j++ {
				// 只读操作（线程安全）
				_ = set.Size()
				_ = set.Contains(domain.NewID(int64(j % 1000)))
				_ = set.IsEmpty()
				localOps++
			}

			opsCount.Add(int64(localOps))
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("百万级并发读测试完成:")
	t.Logf("  - 总耗时: %v", duration)
	t.Logf("  - 总操作数: %d", opsCount.Load())
	t.Logf("  - QPS: %.2f ops/sec", float64(opsCount.Load())/duration.Seconds())
}

// ============================================================================
// 5. 性能基准测试
// ============================================================================

// BenchmarkNewID 基准测试ID创建
func BenchmarkNewID(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = domain.NewID(int64(i))
	}
}

// BenchmarkParseID 基准测试ID解析
func BenchmarkParseID(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = domain.ParseID("123456789")
	}
}

// BenchmarkID_String 基准测试String转换
func BenchmarkID_String(b *testing.B) {
	id := domain.NewID(123456789)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = id.String()
	}
}

// BenchmarkID_Hex 基准测试Hex转换
func BenchmarkID_Hex(b *testing.B) {
	id := domain.NewID(123456789)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = id.Hex()
	}
}

// BenchmarkIDSlice_Contains 基准测试Contains查找
func BenchmarkIDSlice_Contains(b *testing.B) {
	ids := make([]domain.ID, 1000)
	for i := range ids {
		ids[i] = domain.NewID(int64(i))
	}
	slice := domain.NewIDSlice(ids...)
	searchID := domain.NewID(500)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = slice.Contains(searchID)
	}
}

// BenchmarkIDSlice_Deduplicate 基准测试去重
func BenchmarkIDSlice_Deduplicate(b *testing.B) {
	ids := make([]domain.ID, 1000)
	for i := range ids {
		ids[i] = domain.NewID(int64(i % 100)) // 产生重复
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slice := domain.NewIDSlice(ids...)
		_ = slice.Deduplicate()
	}
}

// BenchmarkIDSet_Add 基准测试集合添加
func BenchmarkIDSet_Add(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		set := domain.NewIDSet()
		for j := 0; j < 100; j++ {
			set.Add(domain.NewID(int64(j)))
		}
	}
}

// BenchmarkIDSet_Contains 基准测试集合查找
func BenchmarkIDSet_Contains(b *testing.B) {
	ids := make([]domain.ID, 1000)
	for i := range ids {
		ids[i] = domain.NewID(int64(i))
	}
	set := domain.NewIDSet(ids...)
	searchID := domain.NewID(500)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = set.Contains(searchID)
	}
}

// BenchmarkIDSet_Union 基准测试并集操作
func BenchmarkIDSet_Union(b *testing.B) {
	set1 := domain.NewIDSet()
	set2 := domain.NewIDSet()
	for i := 0; i < 500; i++ {
		set1.Add(domain.NewID(int64(i)))
		set2.Add(domain.NewID(int64(i + 250)))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = set1.Union(set2)
	}
}

// BenchmarkIDSet_Parallel 并行基准测试集合操作
func BenchmarkIDSet_Parallel(b *testing.B) {
	ids := make([]domain.ID, 1000)
	for i := range ids {
		ids[i] = domain.NewID(int64(i))
	}
	set := domain.NewIDSet(ids...)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = set.Contains(domain.NewID(500))
			_ = set.Size()
		}
	})
}

// ============================================================================
// 6. 内存对比测试
// ============================================================================

// BenchmarkIDSlice_vs_IDSet_Memory 对比切片和集合的内存使用
func BenchmarkIDSlice_vs_IDSet_Memory(b *testing.B) {
	b.Run("IDSlice", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ids := make([]domain.ID, 1000)
			for j := range ids {
				ids[j] = domain.NewID(int64(j))
			}
			_ = domain.NewIDSlice(ids...)
		}
	})

	b.Run("IDSet", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			set := domain.NewIDSet()
			for j := 0; j < 1000; j++ {
				set.Add(domain.NewID(int64(j)))
			}
		}
	})
}
