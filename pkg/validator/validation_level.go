package validator

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// StructLevel 结构体级别验证上下文（封装第三方库）
// 用于在结构体级别验证中报告错误和访问当前验证对象
type StructLevel interface {
	// Current 返回当前正在验证的对象
	Current() reflect.Value

	// ReportError 报告字段验证错误
	// value: 字段值
	// fieldName: 字段名（用于显示）
	// jsonName: JSON 字段名
	// tag: 验证标签
	// param: 验证参数
	ReportError(value interface{}, fieldName, jsonName, tag, param string)

	// ReportValidationErrors 报告底层验证错误（高级用法）
	ReportValidationErrors(errs validator.ValidationErrors)
}

// structLevelWrapper 封装第三方库的 StructLevel
type structLevelWrapper struct {
	sl validator.StructLevel
}

// Current 实现 StructLevel 接口
func (w *structLevelWrapper) Current() reflect.Value {
	return w.sl.Current()
}

// ReportError 实现 StructLevel 接口
func (w *structLevelWrapper) ReportError(value interface{}, fieldName, jsonName, tag, param string) {
	w.sl.ReportError(value, fieldName, jsonName, tag, param)
}

// ReportValidationErrors 实现 StructLevel 接口
func (w *structLevelWrapper) ReportValidationErrors(errs validator.ValidationErrors) {
	// 注意：第三方库的 ReportValidationErrors 方法已被弃用
	// 这里我们遍历错误并逐个报告
	for _, err := range errs {
		w.sl.ReportError(err.Value(), err.Field(), err.Field(), err.Tag(), err.Param())
	}
}

// ValidationFunc 自定义验证函数类型（封装第三方库）
// 用于注册自定义验证标签
type ValidationFunc func(fl FieldLevel) bool

// FieldLevel 字段级别验证上下文（封装第三方库）
// 用于在自定义验证函数中访问字段信息
type FieldLevel interface {
	// Field 返回当前字段的反射值
	Field() reflect.Value

	// Param 返回验证标签的参数
	Param() string

	// FieldName 返回字段名
	FieldName() string

	// StructFieldName 返回结构体字段名
	StructFieldName() string

	// Parent 返回父结构体的反射值
	Parent() reflect.Value
}

// fieldLevelWrapper 封装第三方库的 FieldLevel
type fieldLevelWrapper struct {
	fl validator.FieldLevel
}

// Field 实现 FieldLevel 接口
func (w *fieldLevelWrapper) Field() reflect.Value {
	return w.fl.Field()
}

// Param 实现 FieldLevel 接口
func (w *fieldLevelWrapper) Param() string {
	return w.fl.Param()
}

// FieldName 实现 FieldLevel 接口
func (w *fieldLevelWrapper) FieldName() string {
	return w.fl.FieldName()
}

// StructFieldName 实现 FieldLevel 接口
func (w *fieldLevelWrapper) StructFieldName() string {
	return w.fl.StructFieldName()
}

// Parent 实现 FieldLevel 接口
func (w *fieldLevelWrapper) Parent() reflect.Value {
	return w.fl.Parent()
}
