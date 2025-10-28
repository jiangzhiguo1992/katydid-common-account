package formatter

import (
	"fmt"
	"strings"

	"katydid-common-account/pkg/validator/v6/core"
)

// DefaultFormatter 默认错误格式化器
// 职责：格式化错误信息
// 设计原则：单一职责
type DefaultFormatter struct{}

// NewDefaultFormatter 创建默认格式化器
func NewDefaultFormatter() core.ErrorFormatter {
	return &DefaultFormatter{}
}

// Format 格式化单个错误
func (f *DefaultFormatter) Format(err *core.FieldError) string {
	if err == nil {
		return ""
	}

	// 优先使用自定义消息
	if err.Message != "" {
		return err.Message
	}

	// 生成默认消息
	var builder strings.Builder
	builder.Grow(80)

	if err.Namespace != "" {
		builder.WriteString("字段 '")
		builder.WriteString(err.Namespace)
		builder.WriteString("' ")
	}

	builder.WriteString("验证失败")

	if err.Tag != "" {
		builder.WriteString("，规则: ")
		builder.WriteString(err.Tag)
	}

	if err.Param != "" {
		builder.WriteString("，参数: ")
		builder.WriteString(err.Param)
	}

	return builder.String()
}

// FormatAll 格式化所有错误
func (f *DefaultFormatter) FormatAll(errs []*core.FieldError) string {
	if len(errs) == 0 {
		return "验证通过"
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	var builder strings.Builder
	builder.Grow(len(errs) * 80)

	builder.WriteString(fmt.Sprintf("验证失败，共 %d 个错误:\n", len(errs)))

	for i, err := range errs {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, f.Format(err)))
	}

	return builder.String()
}

// I18nFormatter 国际化错误格式化器
// 职责：支持多语言错误消息
type I18nFormatter struct {
	locale   string
	messages map[string]map[string]string // map[locale]map[tag]message
}

// NewI18nFormatter 创建国际化格式化器
func NewI18nFormatter(locale string) core.ErrorFormatter {
	f := &I18nFormatter{
		locale:   locale,
		messages: make(map[string]map[string]string),
	}

	// 加载默认消息
	f.loadDefaultMessages()

	return f
}

// loadDefaultMessages 加载默认消息
func (f *I18nFormatter) loadDefaultMessages() {
	// 中文消息
	f.messages["zh"] = map[string]string{
		"required": "不能为空",
		"min":      "长度不能小于 %s",
		"max":      "长度不能大于 %s",
		"email":    "格式不正确",
		"len":      "长度必须为 %s",
	}

	// 英文消息
	f.messages["en"] = map[string]string{
		"required": "is required",
		"min":      "must be at least %s",
		"max":      "must be at most %s",
		"email":    "must be a valid email",
		"len":      "must be %s characters",
	}
}

// Format 格式化单个错误
func (f *I18nFormatter) Format(err *core.FieldError) string {
	if err == nil {
		return ""
	}

	// 优先使用自定义消息
	if err.Message != "" {
		return err.Message
	}

	// 获取本地化消息
	template := f.getMessageTemplate(err.Tag)
	if template == "" {
		// 回退到默认格式
		return fmt.Sprintf("%s validation failed on tag '%s'", err.Namespace, err.Tag)
	}

	// 格式化消息
	var msg string
	if err.Param != "" {
		msg = fmt.Sprintf(template, err.Param)
	} else {
		msg = template
	}

	// 添加字段名
	if err.Namespace != "" {
		return fmt.Sprintf("%s %s", err.Namespace, msg)
	}

	return msg
}

// FormatAll 格式化所有错误
func (f *I18nFormatter) FormatAll(errs []*core.FieldError) string {
	if len(errs) == 0 {
		return f.getValidationPassedMessage()
	}

	if len(errs) == 1 {
		return f.Format(errs[0])
	}

	var builder strings.Builder
	builder.Grow(len(errs) * 80)

	builder.WriteString(f.getValidationFailedMessage(len(errs)))
	builder.WriteString("\n")

	for i, err := range errs {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, f.Format(err)))
	}

	return builder.String()
}

// getMessageTemplate 获取消息模板
func (f *I18nFormatter) getMessageTemplate(tag string) string {
	if localeMessages, ok := f.messages[f.locale]; ok {
		if template, ok := localeMessages[tag]; ok {
			return template
		}
	}

	// 回退到英文
	if localeMessages, ok := f.messages["en"]; ok {
		if template, ok := localeMessages[tag]; ok {
			return template
		}
	}

	return ""
}

// getValidationPassedMessage 获取验证通过消息
func (f *I18nFormatter) getValidationPassedMessage() string {
	if f.locale == "zh" {
		return "验证通过"
	}
	return "validation passed"
}

// getValidationFailedMessage 获取验证失败消息
func (f *I18nFormatter) getValidationFailedMessage(count int) string {
	if f.locale == "zh" {
		return fmt.Sprintf("验证失败，共 %d 个错误:", count)
	}
	return fmt.Sprintf("validation failed with %d errors:", count)
}
