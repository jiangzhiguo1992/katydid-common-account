package validator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationContext 验证上下文，用于传递验证环境信息和收集验证错误
// 设计目标：高内聚低耦合，集中管理验证过程中的所有错误信息
type ValidationContext struct {
	// Scene 验证场景，用于区分不同的业务场景（如：创建、更新、删除等）
	Scene ValidateScene `json:"scene"`
	// Message 总体错误消息（可选），用于描述整体验证失败的原因
	Message string `json:"message,omitempty"`
	// Errors 所有验证错误的集合，每个元素代表一个字段的验证错误
	Errors []*FieldError `json:"errors"`
}

// FieldError 单个字段的验证错误信息
// 设计原则：单一职责 - 只负责描述字段验证错误的详细信息
type FieldError struct {
	// Namespace 字段的完整命名空间路径（如 User.Profile.Email）
	// 用于嵌套结构体的错误定位，支持复杂对象的精确错误追踪
	Namespace string `json:"namespace"`

	// Tag 验证标签，描述验证规则类型（如 required, email, min, max 等）
	Tag string `json:"tag"`

	// Param 验证参数，提供验证规则的具体配置值
	// 例如：min=3 中的 "3"，len=11 中的 "11"
	Param string `json:"param"`

	// Value 字段的实际值（用于 sl.ReportError 的 value 参数）
	// 用于调试和详细错误信息，可能包含敏感信息，谨慎使用
	Value any `json:"value,omitempty"`

	// Message 用户友好的错误消息（可选，用于直接显示给终端用户）
	// 支持国际化，建议使用本地化后的错误消息
	Message string `json:"message,omitempty"`
}

// 预估的错误消息平均长度，用于优化字符串构建时的内存分配
// 通过预分配减少内存重新分配次数，提升性能
const (
	// errorMessageEstimatedLength 单个错误消息的预估长度
	errorMessageEstimatedLength = 80
	// namespaceEstimatedLength 命名空间的预估长度
	namespaceEstimatedLength = 50
	// maxErrorsCapacity 错误列表的最大容量，防止恶意数据导致内存溢出
	maxErrorsCapacity = 1000
	// maxNamespaceLength 最大命名空间长度，防止超长命名空间攻击
	maxNamespaceLength = 512
	// maxMessageLength 最大错误消息长度，防止超长消息攻击
	maxMessageLength = 2048
)

// NewValidationContext 创建验证上下文
// 工厂方法模式，确保对象正确初始化，避免 nil 引用
// 参数：
//
//	scene: 验证场景标识
//
// 返回：
//
//	已初始化的 ValidationContext 实例
func NewValidationContext(scene ValidateScene) *ValidationContext {
	return &ValidationContext{
		Scene:  scene,
		Errors: make([]*FieldError, 0), // 预分配空切片，避免 nil 切片
	}
}

// NewFieldError 创建字段错误
// 工厂方法模式，简化 FieldError 对象的创建过程
// 参数：
//
//	namespace: 字段命名空间（如 User.Profile.Email）
//	tag: 验证标签（required, email, min 等）
//	param: 验证参数（如 min=3 中的 "3"）
//	value: 字段的实际值
//
// 返回：
//
//	已初始化的 FieldError 实例
func NewFieldError(namespace, tag, param string) *FieldError {
	return &FieldError{
		Namespace: namespace,
		Tag:       tag,
		Param:     param,
		Value:     nil, // 不是必需
		Message:   "",  // 不是必需
	}
}

// Error 实现 error 接口，使 ValidationContext 可以作为 error 类型使用
// 提供人类可读的错误信息，用于日志记录和调试
// 返回：格式化的错误信息字符串
func (vc *ValidationContext) Error() string {
	// 无错误情况
	if len(vc.Errors) == 0 {
		if len(vc.Message) == 0 {
			return "validation passed: no errors found"
		}
		return vc.Message
	}

	// 内存优化：预分配足够的容量，减少动态扩容
	var builder strings.Builder
	builder.Grow(len(vc.Errors) * errorMessageEstimatedLength)

	builder.WriteString("validation failed: ")
	for i, err := range vc.Errors {
		if i > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(err.String())
	}

	return builder.String()
}

// String 返回友好的错误信息，用于用户界面显示
// 优先返回自定义的 Message，否则生成默认的错误描述
// 返回：格式化的错误信息字符串
func (fe *FieldError) String() string {
	// 优先使用自定义消息（更友好）
	if fe.Message != "" {
		return fe.Message
	}

	// 生成默认错误消息（用于调试）
	if fe.Namespace != "" && fe.Tag != "" {
		if fe.Param != "" {
			return fmt.Sprintf("field '%s' validation failed on tag '%s' with param '%s'", fe.Namespace, fe.Tag, fe.Param)
		}
		return fmt.Sprintf("field '%s' validation failed on tag '%s'", fe.Namespace, fe.Tag)
	}

	return "field validation failed"
}

// HasErrors 检查是否存在验证错误
// 返回：true 表示存在错误，false 表示验证通过
func (vc *ValidationContext) HasErrors() bool {
	return len(vc.Errors) > 0
}

// AddError 添加单个字段错误
// 参数：
//
//	err: 待添加的字段错误
func (vc *ValidationContext) AddError(err *FieldError) {
	if err == nil {
		return // 防御性编程：忽略 nil 参数
	}

	// 安全检查：防止恶意数据导致内存溢出
	if len(vc.Errors) >= maxErrorsCapacity {
		return // 达到最大容量，拒绝添加更多错误
	}

	// 安全检查：验证字段长度，防止超长数据攻击
	if len(err.Namespace) > maxNamespaceLength {
		err.Namespace = err.Namespace[:maxNamespaceLength] + "..."
	}
	if len(err.Message) > maxMessageLength {
		err.Message = err.Message[:maxMessageLength] + "..."
	}

	vc.Errors = append(vc.Errors, err)
}

// AddErrorByValidator 通过 go-playground/validator 的 FieldError 添加错误
// 适配器模式：将第三方库的错误类型转换为内部错误类型
// 参数：
//
//	verr: validator 库产生的字段错误
func (vc *ValidationContext) AddErrorByValidator(verr validator.FieldError) {
	if verr == nil {
		return
	}

	// 安全检查：防止恶意数据导致内存溢出
	if len(vc.Errors) >= maxErrorsCapacity {
		return
	}

	namespace := verr.Namespace()
	if len(namespace) > maxNamespaceLength {
		namespace = namespace[:maxNamespaceLength] + "..."
	}

	message := verr.Error()
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength] + "..."
	}

	vc.Errors = append(vc.Errors, NewFieldError(
		namespace,
		verr.Tag(),
		verr.Param(),
	).WithValue(verr.Value()).WithMessage(message))
}

// AddErrorByDetail 通过详细信息添加字段错误
// 提供最大的灵活性，允许手动构建错误信息
// 参数：
//
//	namespace: 字段命名空间
//	tag: 验证标签
//	param: 验证参数
//	value: 字段值
//	message: 自定义错误消息
func (vc *ValidationContext) AddErrorByDetail(namespace, tag, param string, value any, message string) {
	// 安全检查：防止恶意数据导致内存溢出
	if len(vc.Errors) >= maxErrorsCapacity {
		return
	}

	// 安全检查：验证字符串长度
	if len(namespace) > maxNamespaceLength {
		namespace = namespace[:maxNamespaceLength] + "..."
	}
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength] + "..."
	}

	vc.Errors = append(vc.Errors, NewFieldError(
		namespace,
		tag,
		param,
	).WithValue(value).WithMessage(message))
}

// AddErrors 批量添加字段错误
// 提高批量操作的效率，减少锁的获取次数
// 参数：
//
//	errors: 待添加的错误列表
func (vc *ValidationContext) AddErrors(errors []*FieldError) {
	if len(errors) == 0 {
		return // 防御性编程：忽略空列表
	}

	// 安全检查：防止恶意数据导致内存溢出
	if len(vc.Errors) >= maxErrorsCapacity {
		return
	}

	// 安全检查：限制批量添加的数量
	remainingCapacity := maxErrorsCapacity - len(vc.Errors)
	if len(errors) > remainingCapacity {
		errors = errors[:remainingCapacity]
	}

	// 内存优化：如果当前容量不足，一次性扩容到所需大小
	requiredCap := len(vc.Errors) + len(errors)
	// 安全检查：限制最大容量
	if requiredCap > maxErrorsCapacity {
		requiredCap = maxErrorsCapacity
	}

	if cap(vc.Errors) < requiredCap {
		newErrors := make([]*FieldError, len(vc.Errors), requiredCap)
		copy(newErrors, vc.Errors)
		vc.Errors = newErrors
	}

	// 对每个错误进行长度验证
	for _, err := range errors {
		if err == nil {
			continue
		}
		// 安全检查：验证字段长度
		if len(err.Namespace) > maxNamespaceLength {
			err.Namespace = err.Namespace[:maxNamespaceLength] + "..."
		}
		if len(err.Message) > maxMessageLength {
			err.Message = err.Message[:maxMessageLength] + "..."
		}
	}

	vc.Errors = append(vc.Errors, errors...)
}

// ToJSON 将验证上下文转换为 JSON 格式
// 用于 API 响应或日志记录
// 返回：
//
//	JSON 字节数组和可能的错误
func (vc *ValidationContext) ToJSON() ([]byte, error) {
	data, err := json.Marshal(vc)
	if err != nil {
		return nil, fmt.Errorf("validation context serialization failed: %w", err)
	}
	return data, nil
}

// GetErrorsByNamespace 按命名空间筛选错误
// 用于获取特定嵌套结构的所有错误
// 参数：
//
//	namespace: 字段命名空间（如 "User.Profile.Email"）
//
// 返回：
//
//	匹配的错误列表
func (vc *ValidationContext) GetErrorsByNamespace(namespace string) []*FieldError {
	if namespace == "" {
		return nil
	}

	// 内存优化：预分配合理的容量
	errors := make([]*FieldError, 0, len(vc.Errors)/4)
	for _, err := range vc.Errors {
		if err.Namespace == namespace {
			errors = append(errors, err)
		}
	}
	return errors
}

// GetErrorsByTag 按验证标签筛选错误
// 用于统计特定类型的验证失败（如所有 required 错误）
// 参数：
//
//	tag: 验证标签（如 "required", "email"）
//
// 返回：
//
//	匹配的错误列表
func (vc *ValidationContext) GetErrorsByTag(tag string) []*FieldError {
	if tag == "" {
		return nil
	}

	// 内存优化：预分配合理的容量
	errors := make([]*FieldError, 0, len(vc.Errors)/4)
	for _, err := range vc.Errors {
		if err.Tag == tag {
			errors = append(errors, err)
		}
	}
	return errors
}

func (fe *FieldError) WithValue(value any) *FieldError {
	fe.Value = value
	return fe
}

// WithMessage 设置自定义错误消息（链式调用）
// 流式接口模式，提升代码可读性和易用性
// 参数：
//
//	message: 自定义错误消息
//
// 返回：
//
//	FieldError 实例，支持链式调用
func (fe *FieldError) WithMessage(message string) *FieldError {
	fe.Message = message
	return fe
}

// ToLocalizes 转换为本地化错误信息模板
// 用于国际化支持，生成可用于翻译的错误键
// 返回：
//
//	key: 本地化键（格式：命名空间.标签）
//	param: 验证参数（用于消息插值）
func (fe *FieldError) ToLocalizes() (key string, param string) {
	// 内存优化：使用 strings.Builder 构建字符串
	var builder strings.Builder
	builder.Grow(namespaceEstimatedLength)

	builder.WriteString(fe.Namespace)
	if fe.Tag != "" {
		builder.WriteString(".")
	}
	builder.WriteString(fe.Tag)

	if fe.Namespace == "" && fe.Tag == "" {
		builder.WriteString(fe.Message)
	} else if fe.Tag == "custom" {
		builder.WriteString(".")
		builder.WriteString(fe.Message)
	}

	return builder.String(), fe.Param
}

// ErrorCount 获取错误总数
// 返回：错误数量
func (vc *ValidationContext) ErrorCount() int {
	return len(vc.Errors)
}

// Clear 清空所有错误
// 用于重用 ValidationContext 实例，减少内存分配
func (vc *ValidationContext) Clear() {
	vc.Errors = vc.Errors[:0] // 保留底层数组，避免重新分配
	vc.Message = ""
}

// Clone 创建验证上下文的深拷贝
// 用于需要复制上下文的场景，避免并发修改问题
// 返回：
//
//	新的 ValidationContext 实例
func (vc *ValidationContext) Clone() *ValidationContext {
	// 深拷贝错误列表
	newErrors := make([]*FieldError, len(vc.Errors))
	for i, err := range vc.Errors {
		if err != nil {
			// 深拷贝每个 FieldError 对象
			newErrors[i] = &FieldError{
				Namespace: err.Namespace,
				Tag:       err.Tag,
				Param:     err.Param,
				Value:     err.Value,
				Message:   err.Message,
			}
		}
	}

	return &ValidationContext{
		Scene:   vc.Scene,
		Message: vc.Message,
		Errors:  newErrors,
	}
}

// SanitizeValues 清除所有错误中的敏感值，防止数据泄露
// 用于记录日志或返回给客户端时，避免暴露敏感信息
// 此方法会修改当前对象，如需保留原始值请先调用 Clone()
func (vc *ValidationContext) SanitizeValues() *ValidationContext {
	for _, err := range vc.Errors {
		if err != nil {
			err.Value = nil
		}
	}
	return vc
}

// GetFirstError 获取第一个错误
// 返回：
//
//	第一个 FieldError，如果没有错误则返回 nil
func (vc *ValidationContext) GetFirstError() *FieldError {
	if len(vc.Errors) > 0 {
		return vc.Errors[0]
	}
	return nil
}
