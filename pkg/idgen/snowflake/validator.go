package snowflake

import (
	"fmt"
	"time"

	"katydid-common-account/pkg/idgen/core"
)

// Validator Snowflake ID验证器（单一职责：只负责ID验证）
// 实现core.IDValidator接口（里氏替换原则）
type Validator struct{}

// NewValidator 创建新的验证器实例
func NewValidator() *Validator {
	return &Validator{}
}

// Validate 验证Snowflake ID的有效性
// 实现core.IDValidator接口
func (v *Validator) Validate(id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", core.ErrInvalidSnowflakeID)
	}

	// 提取时间戳
	timestamp := (id >> TimestampShift) + Epoch

	// 检查时间戳是否在Epoch之后
	if timestamp < Epoch {
		return fmt.Errorf("%w: timestamp %d is before epoch %d",
			core.ErrInvalidSnowflakeID, timestamp, Epoch)
	}

	// 允许一定的时钟误差，防止恶意构造ID
	now := time.Now().UnixMilli()
	if timestamp > now+maxFutureTimeTolerance {
		return fmt.Errorf("%w: timestamp %d is too far in the future (max tolerance %dms)",
			core.ErrInvalidSnowflakeID, timestamp, maxFutureTimeTolerance)
	}

	return nil
}

// ValidateBatch 批量验证ID
// 实现core.IDValidator接口
func (v *Validator) ValidateBatch(ids []int64) error {
	for i, id := range ids {
		if err := v.Validate(id); err != nil {
			return fmt.Errorf("invalid ID at index %d: %w", i, err)
		}
	}
	return nil
}

// ValidateSnowflakeID 全局验证函数（向后兼容）
func ValidateSnowflakeID(id int64) error {
	return NewValidator().Validate(id)
}
