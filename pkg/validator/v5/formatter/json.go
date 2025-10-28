package formatter

import (
	"fmt"
	"katydid-common-account/pkg/validator/v5/core"
)

// JSONErrorFormatter JSON错误格式化器
// 职责：将错误格式化为JSON格式
type JSONErrorFormatter struct{}

// NewJSONErrorFormatter 创建JSON错误格式化器
func NewJSONErrorFormatter() core.IErrorFormatter {
	return &JSONErrorFormatter{}
}

// Format 格式化单个错误为JSON字符串
func (f *JSONErrorFormatter) Format(err core.IFieldError) string {
	if err == nil {
		return "{}"
	}

	builder := core.AcquireStringBuilder()
	core.ReleaseStringBuilder(builder)

	builder.WriteString("{")
	builder.WriteString(fmt.Sprintf(`"namespace":"%s"`, err.Namespace()))
	builder.WriteString(fmt.Sprintf(`,"tag":"%s"`, err.Tag()))

	if len(err.Param()) > 0 {
		builder.WriteString(fmt.Sprintf(`,"param":"%s"`, err.Param()))
	}

	if len(err.Message()) > 0 {
		builder.WriteString(fmt.Sprintf(`,"message":"%s"`, err.Message()))
	}

	builder.WriteString("}")
	return builder.String()
}

// FormatAll 格式化所有错误为JSON数组
func (f *JSONErrorFormatter) FormatAll(errs []core.IFieldError) string {
	if len(errs) == 0 {
		return "[]"
	}

	builder := core.AcquireStringBuilder()
	core.ReleaseStringBuilder(builder)

	builder.WriteString("[")

	for i, err := range errs {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(f.Format(err))
	}

	builder.WriteString("]")
	return builder.String()
}
