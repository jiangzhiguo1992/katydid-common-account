package idgen

import (
	"fmt"
	"sync"
)

var (
	defaultGenerator *Snowflake
	initOnce         sync.Once
)

// Init 初始化默认的ID生成器
// 参数:
//
//	datacenterID: 数据中心ID，取值范围 [0, 31]
//	workerID: 工作机器ID，取值范围 [0, 31]
//
// 返回:
//
//	error: 初始化失败时返回错误
//
// 注意: 该函数只会执行一次，多次调用将返回第一次初始化的结果
func Init(datacenterID, workerID int64) error {
	var err error
	initOnce.Do(func() {
		defaultGenerator, err = NewSnowflake(datacenterID, workerID)
	})
	return err
}

// NextID 使用默认生成器生成ID
// 如果未初始化，将使用默认值 datacenterID=0, workerID=0 进行初始化
// 返回:
//
//	int64: 生成的唯一ID
//	error: 生成失败时返回错误
func NextID() (int64, error) {
	if defaultGenerator == nil {
		// 如果未初始化，使用默认值进行初始化
		if err := Init(0, 0); err != nil {
			return 0, fmt.Errorf("failed to initialize ID generator: %w", err)
		}
	}
	return defaultGenerator.NextID()
}
