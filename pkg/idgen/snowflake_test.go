package idgen

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSnowflake_NextID(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	assert.NoError(t, err)

	// 生成ID测试
	id1, err := sf.NextID()
	assert.NoError(t, err)
	assert.Greater(t, id1, int64(0))

	id2, err := sf.NextID()
	assert.NoError(t, err)
	assert.Greater(t, id2, id1)

	// 批量生成测试 - 验证唯一性
	ids := make(map[int64]bool)
	for i := 0; i < 10000; i++ {
		id, err := sf.NextID()
		assert.NoError(t, err)
		assert.False(t, ids[id], "ID should be unique")
		ids[id] = true
		fmt.Println(id)
	}
}

func TestSnowflake_InvalidParameters(t *testing.T) {
	_, err := NewSnowflake(-1, 1)
	assert.ErrorIs(t, err, ErrInvalidDatacenterID)

	_, err = NewSnowflake(32, 1)
	assert.ErrorIs(t, err, ErrInvalidDatacenterID)

	_, err = NewSnowflake(1, -1)
	assert.ErrorIs(t, err, ErrInvalidWorkerID)

	_, err = NewSnowflake(1, 32)
	assert.ErrorIs(t, err, ErrInvalidWorkerID)
}

func TestParseID(t *testing.T) {
	sf, err := NewSnowflake(10, 5)
	assert.NoError(t, err)

	id, err := sf.NextID()
	assert.NoError(t, err)

	_, datacenterID, workerID, sequence := ParseSnowflakeID(id)
	assert.Equal(t, int64(10), datacenterID)
	assert.Equal(t, int64(5), workerID)
	assert.GreaterOrEqual(t, sequence, int64(0))
	assert.Less(t, sequence, int64(4096))

	// 验证时间戳
	idTime := GetTimestamp(id)
	now := time.Now()
	assert.WithinDuration(t, now, idTime, time.Second)
}

func TestID_JSON(t *testing.T) {
	err := Init(1, 1)
	assert.NoError(t, err)

	id, err := NewID()
	assert.NoError(t, err)

	// 测试序列化
	data, err := json.Marshal(id)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// 测试反序列化
	var id2 ID
	err = json.Unmarshal(data, &id2)
	assert.NoError(t, err)
	assert.Equal(t, id, id2)
}

func TestID_String(t *testing.T) {
	id := ID(123456789)
	assert.Equal(t, "123456789", id.String())
	assert.Equal(t, int64(123456789), id.Int64())
	assert.False(t, id.IsZero())

	zeroID := ID(0)
	assert.True(t, zeroID.IsZero())
}

func TestParseIDString(t *testing.T) {
	id, err := ParseIDFromString("123456789")
	assert.NoError(t, err)
	assert.Equal(t, ID(123456789), id)

	id, err = ParseIDFromString("")
	assert.NoError(t, err)
	assert.Equal(t, ID(0), id)

	_, err = ParseIDFromString("invalid")
	assert.Error(t, err)
}

func BenchmarkSnowflake_NextID(b *testing.B) {
	sf, _ := NewSnowflake(1, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sf.NextID()
	}
}

func BenchmarkNextID(b *testing.B) {
	_ = Init(1, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NextID()
	}
}

// 测试基本功能
func TestBasicFunctionality(t *testing.T) {
	// 初始化
	err := Init(1, 1)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 生成ID
	id, err := NewID()
	if err != nil {
		t.Fatalf("生成ID失败: %v", err)
	}

	if id.IsZero() {
		t.Fatal("生成的ID不应该为0")
	}

	t.Logf("成功生成ID: %s", id.String())
}

// 测试并发安全性
func TestConcurrency(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatalf("创建生成器失败: %v", err)
	}

	// 并发生成1000个ID
	const count = 1000
	ids := make(chan int64, count)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < count/10; j++ {
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

	// 验证唯一性
	idMap := make(map[int64]bool)
	for id := range ids {
		if idMap[id] {
			t.Errorf("发现重复ID: %d", id)
		}
		idMap[id] = true
	}

	if len(idMap) != count {
		t.Errorf("期望生成 %d 个ID，实际生成 %d 个", count, len(idMap))
	}

	t.Logf("并发测试通过: 成功生成 %d 个唯一ID", len(idMap))
}

// 测试JSON序列化
func TestJSONSerialization(t *testing.T) {
	id := ID(123456789012345)

	// 序列化
	data, err := id.MarshalJSON()
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	expected := `"123456789012345"`
	if string(data) != expected {
		t.Errorf("期望 %s, 实际 %s", expected, string(data))
	}

	// 反序列化
	var id2 ID
	err = id2.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if id != id2 {
		t.Errorf("期望 %v, 实际 %v", id, id2)
	}

	t.Logf("JSON序列化测试通过")
}
