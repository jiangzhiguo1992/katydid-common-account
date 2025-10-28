package formatter

import (
	"fmt"
	v5 "katydid-common-account/pkg/validator/v5"
	error2 "katydid-common-account/pkg/validator/v5/error"
)

// JSONErrorFormatter JSON错误格式化器
// 职责：将错误格式化为JSON格式
type JSONErrorFormatter struct{}

// NewJSONErrorFormatter 创建JSON错误格式化器
func NewJSONErrorFormatter() *JSONErrorFormatter {
	return &JSONErrorFormatter{}
}

// Format 格式化单个错误为JSON字符串
func (f *JSONErrorFormatter) Format(err *error2.FieldError) string {
	if err == nil {
		return "{}"
	}

	builder := v5.acquireStringBuilder()
	v5.releaseStringBuilder(builder)

	builder.WriteString("{")
	builder.WriteString(fmt.Sprintf(`"namespace":"%s"`, err.Namespace))
	builder.WriteString(fmt.Sprintf(`,"tag":"%s"`, err.Tag))

	if len(err.Param) > 0 {
		builder.WriteString(fmt.Sprintf(`,"param":"%s"`, err.Param))
	}

	if len(err.Message) > 0 {
		builder.WriteString(fmt.Sprintf(`,"message":"%s"`, err.Message))
	}

	builder.WriteString("}")
	return builder.String()
}

// FormatAll 格式化所有错误为JSON数组
func (f *JSONErrorFormatter) FormatAll(errs []*error2.FieldError) string {
	if len(errs) == 0 {
		return "[]"
	}

	builder := v5.acquireStringBuilder()
	v5.releaseStringBuilder(builder)

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
