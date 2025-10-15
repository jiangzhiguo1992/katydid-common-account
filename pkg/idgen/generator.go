package idgen

import "sync"

var (
	defaultGenerator *Snowflake
	once             sync.Once
)

// Init 初始化默认的ID生成器
// datacenterID: 数据中心ID (0-31)
// workerID: 工作机器ID (0-31)
// TODO:GG 需要初始化调用
func Init(datacenterID, workerID int64) error {
	var err error
	once.Do(func() {
		defaultGenerator, err = NewSnowflake(datacenterID, workerID)
	})
	return err
}

// NextID 使用默认生成器生成ID
func NextID() (int64, error) {
	if defaultGenerator == nil {
		// 如果未初始化，使用默认值 datacenterID=0, workerID=0
		if err := Init(0, 0); err != nil {
			return 0, err
		}
	}
	return defaultGenerator.NextID()
}
