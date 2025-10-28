package formatter

import (
	"fmt"
	"katydid-common-account/pkg/validator/v5/core"
)

// ErrorMessageEstimatedLength 预估的错误消息平均长度，用于优化字符串构建时的内存分配
const ErrorMessageEstimatedLength = 80

// NormalErrorFormatter 普通错误格式化器
type NormalErrorFormatter struct{}

// NewNormalErrorFormatter 创建普通错误格式化器
func NewNormalErrorFormatter() core.IErrorFormatter {
	return &NormalErrorFormatter{}
}

// Format 格式化单个错误
func (f *NormalErrorFormatter) Format(err core.IFieldError) string {
	if err == nil {
		return ""
	}

	// 优先使用自定义消息
	if len(err.Message()) > 0 {
		return err.Message()
	}

	// 生成普通消息
	builder := core.AcquireStringBuilder()
	core.ReleaseStringBuilder(builder)

	builder.Grow(ErrorMessageEstimatedLength)

	if len(err.Namespace()) > 0 {
		builder.WriteString("字段 '")
		builder.WriteString(err.Namespace())
		builder.WriteString("' ")
	}

	builder.WriteString("验证失败")

	if len(err.Tag()) > 0 {
		builder.WriteString("，规则: ")
		builder.WriteString(err.Tag())
	}

	if len(err.Param()) > 0 {
		builder.WriteString("，参数: ")
		builder.WriteString(err.Param())
	}

	if err.Value != nil {
		builder.WriteString("，值: ")
		builder.WriteString(fmt.Sprintf("%v", err.Value()))
	}

	return builder.String()
}

// FormatAll 格式化所有错误
func (f *NormalErrorFormatter) FormatAll(errs []core.IFieldError) string {
	if len(errs) == 0 {
		return "验证通过"
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	builder := core.AcquireStringBuilder()
	core.ReleaseStringBuilder(builder)

	builder.Grow(len(errs) * ErrorMessageEstimatedLength)

	builder.WriteString(fmt.Sprintf("验证失败，共 %d 个错误:\n", len(errs)))

	for i, err := range errs {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, f.Format(err)))
	}

	return builder.String()
}
