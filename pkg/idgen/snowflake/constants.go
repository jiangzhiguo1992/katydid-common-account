package snowflake

import "time"

// Epoch Snowflake ID的起始时间戳（Unix毫秒）
// 当前值：2023-01-01 00:00:00 UTC (1672502400000)
// 备选值：2026-01-01 00:00:00 UTC (1767196800000) - 预留未来使用
//
// 说明：
//   - Epoch越早，ID的可用时间越长（41位时间戳可用约69年）
//   - 修改Epoch会使新旧ID不兼容，需谨慎变更
//   - 建议在系统初始化时设置，后续不再修改
const Epoch int64 = 1672502400000

// Snowflake ID结构（64位）：
// +--------------------------------------------------------------------------+
// | 1 Bit Unused | 41 Bits Timestamp | 5 Bits DC ID | 5 Bits Worker ID | 12 Bits Sequence |
// +--------------------------------------------------------------------------+

const (
	// WorkerIDBits 工作机器ID位数
	// 说明：5位可表示32个不同的工作机器（0-31）
	WorkerIDBits = 5

	// DatacenterIDBits 数据中心ID位数
	// 说明：5位可表示32个不同的数据中心（0-31）
	DatacenterIDBits = 5

	// SequenceBits 序列号位数
	// 说明：12位可表示4096个序列号（0-4095）
	// 含义：同一毫秒内最多生成4096个不同的ID
	SequenceBits = 12
)

const (
	// MaxWorkerID 工作机器ID的最大值
	// 计算方式：-1 ^ (-1 << 5) = 31 (二进制: 11111)
	// 有效范围：[0, 31]，共32个值
	MaxWorkerID = -1 ^ (-1 << WorkerIDBits)

	// MaxDatacenterID 数据中心ID的最大值
	// 计算方式：-1 ^ (-1 << 5) = 31 (二进制: 11111)
	// 有效范围：[0, 31]，共32个值
	MaxDatacenterID = -1 ^ (-1 << DatacenterIDBits)

	// MaxSequence 序列号的最大值
	// 计算方式：-1 ^ (-1 << 12) = 4095 (二进制: 111111111111)
	// 有效范围：[0, 4095]，共4096个值
	// 含义：同一毫秒内最多生成4096个唯一ID
	MaxSequence = -1 ^ (-1 << SequenceBits)
)

const (
	// WorkerIDShift 工作机器ID的左移位数
	// 计算：SequenceBits = 12
	// 含义：WorkerID需要左移12位，为序列号留出空间
	WorkerIDShift = SequenceBits

	// DatacenterIDShift 数据中心ID的左移位数
	// 计算：SequenceBits + WorkerIDBits = 12 + 5 = 17
	// 含义：DatacenterID需要左移17位，为序列号和WorkerID留出空间
	DatacenterIDShift = SequenceBits + WorkerIDBits

	// TimestampShift 时间戳的左移位数
	// 计算：SequenceBits + WorkerIDBits + DatacenterIDBits = 12 + 5 + 5 = 22
	// 含义：时间戳需要左移22位，为其他部分留出空间
	TimestampShift = SequenceBits + WorkerIDBits + DatacenterIDBits
)

const (
	// sleepDuration 等待下一毫秒时的休眠时间
	// 说明：当序列号耗尽时，需要等待下一毫秒
	// 值：100微秒（0.1毫秒）
	// 目的：避免过于频繁的循环检查，减少CPU占用
	sleepDuration = 100 * time.Microsecond
)

const (
	// maxClockBackwardTolerance 默认的时钟回拨容忍时间（毫秒）
	// 说明：当回拨时间不超过此值时，生成器会等待时钟追上
	// 值：5毫秒
	// 适用场景：NTP同步导致的微小时钟调整
	maxClockBackwardTolerance = 5

	// maxClockBackwardToleranceLimit 时钟回拨容忍时间的绝对上限（毫秒）
	// 说明：防止配置过大的容忍时间导致长时间阻塞
	// 值：1000毫秒（1秒）
	// 目的：保护系统可用性，避免无限等待
	maxClockBackwardToleranceLimit = 1000

	// maxWaitRetries 等待策略的最大重试次数
	// 说明：等待时钟追上时的最大重试次数
	// 值：10次
	// 目的：防止时钟持续回拨导致无限重试
	maxWaitRetries = 10
)

const (
	// maxBatchSize 批量生成ID的最大数量
	// 说明：一次性可生成的最大ID数量
	// 值：100,000（10万）
	// 目的：
	//   - 防止内存占用过高
	//   - 支持跨毫秒生成（可能需要等待多个毫秒）
	maxBatchSize = 100_000
)

const (
	// maxFutureTimeTolerance 允许的未来时间容差（毫秒）
	// 说明：ID中的时间戳可以比当前时间稍早，但不能太超前
	// 值：60,000毫秒（1分钟）
	// 目的：
	//   - 防止恶意构造的未来ID
	//   - 容忍服务器之间的时钟偏差
	maxFutureTimeTolerance = 60 * 1000
)
