package snowflake

import "sync/atomic"

// Metrics 性能监控指标
type Metrics struct {
	// IDCount 已生成的ID总数
	// 说明：每次成功生成ID时递增
	IDCount atomic.Uint64

	// SequenceOverflow 序列号溢出次数
	// 说明：同一毫秒内序列号达到4095时递增
	// 用途：反映系统负载情况，溢出过多说明负载很高
	SequenceOverflow atomic.Uint64

	// ClockBackward 时钟回拨检测次数
	// 说明：检测到时钟回拨时递增
	// 用途：监控时钟稳定性，频繁回拨需要检查NTP配置
	ClockBackward atomic.Uint64

	// WaitCount 等待下一毫秒的次数
	// 说明：序列号耗尽需要等待时递增
	// 用途：反映系统负载和等待频率
	WaitCount atomic.Uint64

	// TotalWaitTimeNs 总等待时间（纳秒）
	// 说明：累计等待时间，用于计算平均等待时间
	// 用途：评估等待对性能的影响
	TotalWaitTimeNs atomic.Uint64
}

// NewMetrics 创建新的监控指标实例
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Reset 重置所有监控指标
func (m *Metrics) Reset() {
	m.IDCount.Store(0)
	m.SequenceOverflow.Store(0)
	m.ClockBackward.Store(0)
	m.WaitCount.Store(0)
	m.TotalWaitTimeNs.Store(0)
}

// Snapshot 获取当前指标的快照
func (m *Metrics) Snapshot() *Metrics {
	// 创建新的指标对象并复制当前值
	snapshot := NewMetrics()
	snapshot.IDCount.Store(m.IDCount.Load())
	snapshot.SequenceOverflow.Store(m.SequenceOverflow.Load())
	snapshot.ClockBackward.Store(m.ClockBackward.Load())
	snapshot.WaitCount.Store(m.WaitCount.Load())
	snapshot.TotalWaitTimeNs.Store(m.TotalWaitTimeNs.Load())

	return snapshot
}

// ToMap 转换为map格式
func (m *Metrics) ToMap() map[string]uint64 {
	// 读取所有计数器的值
	waitCount := m.WaitCount.Load()
	totalWaitTime := m.TotalWaitTimeNs.Load()

	// 计算平均等待时间（避免除零）
	var avgWaitTime uint64
	if waitCount > 0 {
		avgWaitTime = totalWaitTime / waitCount
	}

	// 构建结果map
	return map[string]uint64{
		"metrics_enabled":   1,                         // 监控已启用
		"id_count":          m.IDCount.Load(),          // ID生成总数
		"sequence_overflow": m.SequenceOverflow.Load(), // 序列号溢出次数
		"clock_backward":    m.ClockBackward.Load(),    // 时钟回拨次数
		"wait_count":        waitCount,                 // 等待次数
		"avg_wait_time_ns":  avgWaitTime,               // 平均等待时间（纳秒）
	}
}
