package v2

import (
	"fmt"
	"reflect"
)

// ============================================================================
// 嵌套验证器实现 - 单一职责：专门处理嵌套结构体验证
// 遵循开放封闭原则：对扩展开放，对修改封闭
// ============================================================================

// defaultNestedValidator 默认的嵌套验证器实现
type defaultNestedValidator struct {
	validator Validator
	maxDepth  int
}

// NewNestedValidator 创建嵌套验证器
// 工厂方法模式：封装对象创建逻辑
func NewNestedValidator(validator Validator, maxDepth int) NestedValidator {
	if maxDepth <= 0 {
		maxDepth = 100 // 默认最大深度
	}
	return &defaultNestedValidator{
		validator: validator,
		maxDepth:  maxDepth,
	}
}

// ValidateNested 验证嵌套结构
// 实现 NestedValidator 接口
// 采用深度优先遍历，递归验证所有嵌套字段
func (v *defaultNestedValidator) ValidateNested(data interface{}, scene Scene, maxDepth int) error {
	if data == nil {
		return nil
	}

	if maxDepth <= 0 {
		maxDepth = v.maxDepth
	}

	// 使用内部方法进行递归验证
	collector := GetPooledErrorCollector()
	defer PutPooledErrorCollector(collector)

	v.validateNestedRecursive(data, scene, 0, maxDepth, "", collector)

	if collector.HasErrors() {
		return collector.GetErrors()
	}

	return nil
}

// validateNestedRecursive 递归验证嵌套结构
// 内部方法：封装递归逻辑
func (v *defaultNestedValidator) validateNestedRecursive(
	data interface{},
	scene Scene,
	currentDepth int,
	maxDepth int,
	parentPath string,
	collector ErrorCollector,
) {
	// 防止无限递归
	if currentDepth >= maxDepth {
		collector.AddError(parentPath, fmt.Sprintf("exceeded maximum nesting depth %d", maxDepth))
		return
	}

	if data == nil {
		return
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	// 只处理结构体类型
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()

	// 遍历所有字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)

		// 跳过未导出的字段
		if !structField.IsExported() {
			continue
		}

		fieldPath := v.buildFieldPath(parentPath, structField.Name)

		// 处理不同类型的字段
		v.validateFieldByKind(field, structField, scene, currentDepth, maxDepth, fieldPath, collector)
	}
}

// validateFieldByKind 根据字段类型进行验证
func (v *defaultNestedValidator) validateFieldByKind(
	field reflect.Value,
	structField reflect.StructField,
	scene Scene,
	currentDepth int,
	maxDepth int,
	fieldPath string,
	collector ErrorCollector,
) {
	// 处理指针类型
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return
		}
		field = field.Elem()
	}

	switch field.Kind() {
	case reflect.Struct:
		// 递归验证嵌套结构体
		if v.ShouldValidateNested(field.Interface()) {
			// 先验证当前结构体
			if err := v.validator.Validate(field.Interface(), scene); err != nil {
				v.addValidationError(fieldPath, err, collector)
			}
			// 再递归验证嵌套字段
			v.validateNestedRecursive(field.Interface(), scene, currentDepth+1, maxDepth, fieldPath, collector)
		}

	case reflect.Slice, reflect.Array:
		// 验证切片或数组中的元素
		v.validateSliceElements(field, scene, currentDepth, maxDepth, fieldPath, collector)

	case reflect.Map:
		// 验证 Map 中的值
		v.validateMapValues(field, scene, currentDepth, maxDepth, fieldPath, collector)
	}
}

// validateSliceElements 验证切片元素
func (v *defaultNestedValidator) validateSliceElements(
	slice reflect.Value,
	scene Scene,
	currentDepth int,
	maxDepth int,
	parentPath string,
	collector ErrorCollector,
) {
	for i := 0; i < slice.Len(); i++ {
		elem := slice.Index(i)
		elemPath := fmt.Sprintf("%s[%d]", parentPath, i)

		// 处理指针元素
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}

		// 只验证结构体类型的元素
		if elem.Kind() == reflect.Struct && v.ShouldValidateNested(elem.Interface()) {
			if err := v.validator.Validate(elem.Interface(), scene); err != nil {
				v.addValidationError(elemPath, err, collector)
			}
			v.validateNestedRecursive(elem.Interface(), scene, currentDepth+1, maxDepth, elemPath, collector)
		}
	}
}

// validateMapValues 验证 Map 值
func (v *defaultNestedValidator) validateMapValues(
	mapVal reflect.Value,
	scene Scene,
	currentDepth int,
	maxDepth int,
	parentPath string,
	collector ErrorCollector,
) {
	iter := mapVal.MapRange()
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		keyStr := fmt.Sprintf("%v", key.Interface())
		valuePath := fmt.Sprintf("%s[%s]", parentPath, keyStr)

		// 处理指针值
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				continue
			}
			value = value.Elem()
		}

		// 只验证结构体类型的值
		if value.Kind() == reflect.Struct && v.ShouldValidateNested(value.Interface()) {
			if err := v.validator.Validate(value.Interface(), scene); err != nil {
				v.addValidationError(valuePath, err, collector)
			}
			v.validateNestedRecursive(value.Interface(), scene, currentDepth+1, maxDepth, valuePath, collector)
		}
	}
}

// ShouldValidateNested 判断是否应该验证嵌套字段
// 实现 NestedValidator 接口
// 过滤掉不需要验证的类型（如 time.Time 等）
func (v *defaultNestedValidator) ShouldValidateNested(field interface{}) bool {
	if field == nil {
		return false
	}

	typ := reflect.TypeOf(field)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 排除标准库中的特殊类型
	pkgPath := typ.PkgPath()
	typeName := typ.Name()

	// 排除 time.Time 等标准库类型
	if pkgPath == "time" && typeName == "Time" {
		return false
	}

	// 排除 sql 相关类型
	if pkgPath == "database/sql" {
		return false
	}

	// 排除 json 相关类型
	if pkgPath == "encoding/json" {
		return false
	}

	// 只验证用户自定义的结构体
	return typ.Kind() == reflect.Struct
}

// addValidationError 添加验证错误
func (v *defaultNestedValidator) addValidationError(fieldPath string, err error, collector ErrorCollector) {
	if err == nil {
		return
	}

	// 如果是 ValidationErrors 类型，提取详细错误
	if validationErrs, ok := err.(ValidationErrors); ok {
		for _, verr := range validationErrs {
			// 构建完整的字段路径
			fullPath := fieldPath
			if verr.Field != "" {
				fullPath = fieldPath + "." + verr.Field
			}
			collector.AddFieldError(fullPath, verr.Tag, verr.Param, verr.Message)
		}
	} else {
		collector.AddError(fieldPath, err.Error())
	}
}

// buildFieldPath 构建字段路径
func (v *defaultNestedValidator) buildFieldPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

// ============================================================================
// 嵌套验证辅助函数
// ============================================================================

// IsNestedStruct 检查值是否为嵌套结构体
func IsNestedStruct(value interface{}) bool {
	if value == nil {
		return false
	}

	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ.Kind() == reflect.Struct
}

// GetStructFields 获取结构体的所有字段
// 工具函数：用于反射分析
func GetStructFields(data interface{}) []reflect.StructField {
	if data == nil {
		return nil
	}

	typ := reflect.TypeOf(data)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil
	}

	fields := make([]reflect.StructField, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() {
			fields = append(fields, field)
		}
	}

	return fields
}

// GetNestedStructFields 获取所有嵌套结构体字段
// 工具函数：过滤出需要嵌套验证的字段
func GetNestedStructFields(data interface{}) []reflect.StructField {
	fields := GetStructFields(data)
	nestedFields := make([]reflect.StructField, 0)

	for _, field := range fields {
		typ := field.Type
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		// 检查是否为结构体类型
		if typ.Kind() == reflect.Struct {
			// 排除标准库类型
			pkgPath := typ.PkgPath()
			if pkgPath != "time" && pkgPath != "database/sql" && pkgPath != "encoding/json" {
				nestedFields = append(nestedFields, field)
			}
		}
	}

	return nestedFields
}

// CountNestedDepth 计算结构体的嵌套深度
// 工具函数：用于性能分析和限制设置
func CountNestedDepth(data interface{}) int {
	return countNestedDepthRecursive(data, 0, 100, make(map[reflect.Type]bool))
}

// countNestedDepthRecursive 递归计算嵌套深度
func countNestedDepthRecursive(data interface{}, currentDepth, maxDepth int, visited map[reflect.Type]bool) int {
	if data == nil || currentDepth >= maxDepth {
		return currentDepth
	}

	typ := reflect.TypeOf(data)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 防止循环引用
	if visited[typ] {
		return currentDepth
	}
	visited[typ] = true

	if typ.Kind() != reflect.Struct {
		return currentDepth
	}

	maxChildDepth := currentDepth
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return currentDepth
		}
		val = val.Elem()
	}

	// 遍历所有字段
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)

		if !structField.IsExported() {
			continue
		}

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}

		if field.Kind() == reflect.Struct {
			childDepth := countNestedDepthRecursive(field.Interface(), currentDepth+1, maxDepth, visited)
			if childDepth > maxChildDepth {
				maxChildDepth = childDepth
			}
		}
	}

	return maxChildDepth
}

// HasCircularReference 检查是否存在循环引用
// 安全检查：防止无限递归
func HasCircularReference(data interface{}) bool {
	return hasCircularReferenceRecursive(data, make(map[uintptr]bool))
}

// hasCircularReferenceRecursive 递归检查循环引用
func hasCircularReferenceRecursive(data interface{}, visited map[uintptr]bool) bool {
	if data == nil {
		return false
	}

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return false
	}

	// 获取指针地址
	ptr := val.Pointer()
	if visited[ptr] {
		return true // 发现循环引用
	}
	visited[ptr] = true

	// 递归检查
	if val.IsNil() {
		return false
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return false
	}

	// 检查所有字段
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if hasCircularReferenceRecursive(field.Interface(), visited) {
				return true
			}
		}
	}

	return false
}

// ============================================================================
// 性能优化：字段路径缓存
// ============================================================================

// fieldPathCache 字段路径缓存
// 优化字符串拼接性能
type fieldPathCache struct {
	paths map[string]string
}

// newFieldPathCache 创建字段路径缓存
func newFieldPathCache() *fieldPathCache {
	return &fieldPathCache{
		paths: make(map[string]string),
	}
}

// GetPath 获取或构建字段路径
func (c *fieldPathCache) GetPath(parent, field string) string {
	key := parent + ":" + field
	if path, exists := c.paths[key]; exists {
		return path
	}

	var path string
	if parent == "" {
		path = field
	} else {
		path = parent + "." + field
	}

	c.paths[key] = path
	return path
}
