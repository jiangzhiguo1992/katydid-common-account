package snowflake

import (
	"fmt"
	"time"

	"katydid-common-account/pkg/idgen/core"
)

// Validator Snowflake ID验证器
type Validator struct{}

// ValidateID 全局验证函数
func ValidateID(id int64) error {
	return NewValidator().Validate(id)
}

// NewValidator 创建新的验证器实例
// 说明：验证器是无状态的，可以创建多个实例或共享单个实例
func NewValidator() core.IIDValidator {
	return &Validator{}
}

// Validate 验证Snowflake ID的有效性
// 实现core.IDValidator接口
func (v *Validator) Validate(id int64) error {
	// 验证1：ID必须为正整数
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive, got %d",
			core.ErrInvalidSnowflakeID, id)
	}

	// 提取时间戳部分（通过位运算）
	timestamp := (id >> TimestampShift) + Epoch

	// 验证2：时间戳必须在Epoch之后
	// 说明：如果时间戳早于Epoch，可能是：
	//   - 使用了不同的Epoch生成的ID
	//   - ID格式错误或损坏
	if timestamp < Epoch {
		return fmt.Errorf("%w: timestamp %d is before epoch %d",
			core.ErrInvalidSnowflakeID, timestamp, Epoch)
	}

	// 验证3：时间戳不能太超前
	// 说明：允许一定的时钟误差（maxFutureTimeTolerance = 1分钟）
	// 目的：
	//   - 防止恶意构造未来的ID
	//   - 容忍服务器之间的时钟偏差
	now := time.Now().UnixMilli()
	if timestamp > now+maxFutureTimeTolerance {
		return fmt.Errorf("%w: timestamp %d is too far in the future (current: %d, max tolerance: %d ms)",
			core.ErrInvalidSnowflakeID, timestamp, now, maxFutureTimeTolerance)
	}

	return nil
}

// ValidateBatch 批量验证ID
// 实现core.IDValidator接口
func (v *Validator) ValidateBatch(ids []int64) error {
	if ids == nil {
		return fmt.Errorf("ids slice cannot be nil")
	}

	// 空切片视为有效（边界情况处理）
	if len(ids) == 0 {
		return nil
	}

	// 逐个验证，遇到第一个错误立即返回
	for i, id := range ids {
		if err := v.Validate(id); err != nil {
			return fmt.Errorf("invalid ID at index %d: %w", i, err)
		}
	}

	return nil
}
