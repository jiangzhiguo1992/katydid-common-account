package formatter

import (
	v5 "katydid-common-account/pkg/validator/v5"
	error2 "katydid-common-account/pkg/validator/v5/error"
)

// LocalizesErrorFormatter 国际化错误格式化器
// 职责：将错误格式化为国际化配置模板
type LocalizesErrorFormatter struct{}

// NewLocalizesErrorFormatter 创建国际化错误格式化器
func NewLocalizesErrorFormatter() *LocalizesErrorFormatter {
	return &LocalizesErrorFormatter{}
}

// Format 格式化单个错误为国际化模板字符串
func (f *LocalizesErrorFormatter) Format(err *error2.FieldError) string {
	if err == nil {
		return ""
	}

	// 生成国际化模板消息
	builder := v5.acquireStringBuilder()
	v5.releaseStringBuilder(builder)

	builder.Grow(v5.errorMessageEstimatedLength / 2)

	if len(err.Namespace) > 0 && len(err.Tag) > 0 {
		builder.WriteString(err.Namespace)
		builder.WriteString(".")
		builder.WriteString(err.Tag)

		if len(err.Param) > 0 {
			builder.WriteString(".")
			builder.WriteString(err.Param)
		}
		return builder.String()
	}

	return err.Message
}

// FormatAll 格式化所有错误为国际化模板字符串
func (f *LocalizesErrorFormatter) FormatAll(errs []*error2.FieldError) string {
	if len(errs) == 0 {
		return ""
	}

	builder := v5.acquireStringBuilder()
	v5.releaseStringBuilder(builder)

	builder.Grow(len(errs) * (v5.errorMessageEstimatedLength / 2))

	for i, err := range errs {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(f.Format(err))
	}

	return builder.String()
}
