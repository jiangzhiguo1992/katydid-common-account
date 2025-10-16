package idgen

import (
	"encoding/json"
	"fmt"
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
