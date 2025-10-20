package snowflake

import "time"

const (
	// Epoch 起始时间戳 (2024-01-01 00:00:00 UTC)
	Epoch int64 = 1672502400000 // 毫秒时间戳

	// 位数分配
	WorkerIDBits     = 5  // 工作机器ID位数
	DatacenterIDBits = 5  // 数据中心ID位数
	SequenceBits     = 12 // 序列号位数

	// 最大值计算(切记不是个数)
	MaxWorkerID     = -1 ^ (-1 << WorkerIDBits)     // 31 (2^5 - 1) [0, 31]
	MaxDatacenterID = -1 ^ (-1 << DatacenterIDBits) // 31 (2^5 - 1) [0, 31]
	MaxSequence     = -1 ^ (-1 << SequenceBits)     // 4095 (2^12 - 1) [0, 4095]

	// 位移量
	WorkerIDShift     = SequenceBits                                   // 12
	DatacenterIDShift = SequenceBits + WorkerIDBits                    // 17
	TimestampShift    = SequenceBits + WorkerIDBits + DatacenterIDBits // 22

	// 等待下一毫秒时的休眠时间（微秒）
	sleepDuration = 100 * time.Microsecond

	// 时钟回拨最大容忍时间（毫秒）
	maxClockBackwardTolerance = 5

	// 时钟回拨容忍度的绝对上限（毫秒），防止无限等待
	maxClockBackwardToleranceLimit = 1000

	// 批量生成最大数量（支持跨毫秒生成）
	maxBatchSize = 100_000

	// 等待策略最大重试次数
	maxWaitRetries = 10

	// 允许的未来时间容差（毫秒）
	maxFutureTimeTolerance = 60 * 1000 // 1分钟
)
