package formatter

import (
	"katydid-common-account/pkg/validator/v5/core"
)

// LocalizesErrorFormatter 国际化错误格式化器
// 职责：将错误格式化为国际化配置模板
type LocalizesErrorFormatter struct{}

// NewLocalizesErrorFormatter 创建国际化错误格式化器
func NewLocalizesErrorFormatter() core.IErrorFormatter {
	return &LocalizesErrorFormatter{}
}

// Format 格式化单个错误为国际化模板字符串
func (f *LocalizesErrorFormatter) Format(err core.IFieldError) string {
	if err == nil {
		return ""
	}

	// 生成国际化模板消息
	builder := core.AcquireStringBuilder()
	core.ReleaseStringBuilder(builder)

	builder.Grow(ErrorMessageEstimatedLength / 2)

	if len(err.Namespace()) > 0 && len(err.Tag()) > 0 {
		builder.WriteString(err.Namespace())
		builder.WriteString(".")
		builder.WriteString(err.Tag())

		if len(err.Param()) > 0 {
			builder.WriteString(".")
			builder.WriteString(err.Param())
		}
		return builder.String()
	}

	return err.Message()
}

// FormatAll 格式化所有错误为国际化模板字符串
func (f *LocalizesErrorFormatter) FormatAll(errs []core.IFieldError) string {
	if len(errs) == 0 {
		return ""
	}

	builder := core.AcquireStringBuilder()
	core.ReleaseStringBuilder(builder)

	builder.Grow(len(errs) * (ErrorMessageEstimatedLength / 2))

	for i, err := range errs {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(f.Format(err))
	}

	return builder.String()
}
