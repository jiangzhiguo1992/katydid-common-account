package snowflake

import "sync/atomic"

// Metrics 性能监控指标（单一职责：只负责监控数据）
type Metrics struct {
	IDCount          atomic.Uint64 // 已生成ID总数
	SequenceOverflow atomic.Uint64 // 序列号溢出次数
	ClockBackward    atomic.Uint64 // 时钟回拨次数
	WaitCount        atomic.Uint64 // 等待下一毫秒次数
	TotalWaitTimeNs  atomic.Uint64 // 总等待时间（纳秒）
}

// NewMetrics 创建新的监控指标实例
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Reset 重置所有监控指标
func (m *Metrics) Reset() {
	if m == nil {
		return
	}
	m.IDCount.Store(0)
	m.SequenceOverflow.Store(0)
	m.ClockBackward.Store(0)
	m.WaitCount.Store(0)
	m.TotalWaitTimeNs.Store(0)
}

// Snapshot 获取当前指标的快照（不可变性：返回副本）
func (m *Metrics) Snapshot() *Metrics {
	if m == nil {
		return NewMetrics()
	}

	snapshot := NewMetrics()
	snapshot.IDCount.Store(m.IDCount.Load())
	snapshot.SequenceOverflow.Store(m.SequenceOverflow.Load())
	snapshot.ClockBackward.Store(m.ClockBackward.Load())
	snapshot.WaitCount.Store(m.WaitCount.Load())
	snapshot.TotalWaitTimeNs.Store(m.TotalWaitTimeNs.Load())
	return snapshot
}

// ToMap 转换为map格式（便于序列化和展示）
func (m *Metrics) ToMap() map[string]uint64 {
	if m == nil {
		return map[string]uint64{
			"metrics_enabled": 0,
		}
	}

	waitCount := m.WaitCount.Load()
	var avgWaitTime uint64
	if waitCount > 0 {
		avgWaitTime = m.TotalWaitTimeNs.Load() / waitCount
	}

	return map[string]uint64{
		"metrics_enabled":   1,
		"id_count":          m.IDCount.Load(),
		"sequence_overflow": m.SequenceOverflow.Load(),
		"clock_backward":    m.ClockBackward.Load(),
		"wait_count":        waitCount,
		"avg_wait_time_ns":  avgWaitTime,
	}
}
