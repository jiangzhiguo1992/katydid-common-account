package v2

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ============================================================================
// 安全验证功能 - 单一职责：提供安全检查和防护
// 设计原则：防御性编程，保护系统免受恶意输入
// ============================================================================

const (
	// 安全限制常量
	maxFieldNameLength  = 256         // 最大字段名长度
	maxRuleLength       = 1024        // 最大规则长度
	maxMessageLength    = 2048        // 最大错误消息长度
	maxMapSize          = 10000       // 最大Map大小
	maxSliceSize        = 10000       // 最大切片大小
	maxStringLength     = 1024 * 1024 // 最大字符串长度（1MB）
	maxNestedDepth      = 100         // 最大嵌套深度
	maxValidationErrors = 1000        // 最大错误数量
)

// SecurityValidator 安全验证器
// 提供额外的安全检查和防护措施
type SecurityValidator struct {
	validator Validator
	config    SecurityConfig
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	// EnableLengthCheck 启用长度检查
	EnableLengthCheck bool

	// EnableDepthCheck 启用深度检查
	EnableDepthCheck bool

	// EnableSizeCheck 启用大小检查
	EnableSizeCheck bool

	// EnableDangerousPatternCheck 启用危险模式检查
	EnableDangerousPatternCheck bool

	// MaxDepth 最大深度
	MaxDepth int

	// MaxErrors 最大错误数
	MaxErrors int
}

// DefaultSecurityConfig 默认安全配置
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		EnableLengthCheck:           true,
		EnableDepthCheck:            true,
		EnableSizeCheck:             true,
		EnableDangerousPatternCheck: true,
		MaxDepth:                    maxNestedDepth,
		MaxErrors:                   maxValidationErrors,
	}
}

// NewSecurityValidator 创建安全验证器
func NewSecurityValidator(v Validator, config SecurityConfig) *SecurityValidator {
	if v == nil {
		v = defaultGlobalValidator
	}
	return &SecurityValidator{
		validator: v,
		config:    config,
	}
}

// Validate 安全验证
func (sv *SecurityValidator) Validate(data interface{}, scene Scene) error {
	// 执行预验证安全检查
	if err := sv.preValidateSecurityCheck(data); err != nil {
		return err
	}

	// 执行正常验证
	return sv.validator.Validate(data, scene)
}

// preValidateSecurityCheck 预验证安全检查
func (sv *SecurityValidator) preValidateSecurityCheck(data interface{}) error {
	if data == nil {
		return nil
	}

	// 检查数据大小
	if sv.config.EnableSizeCheck {
		if err := sv.checkDataSize(data); err != nil {
			return err
		}
	}

	return nil
}

// checkDataSize 检查数据大小
func (sv *SecurityValidator) checkDataSize(data interface{}) error {
	// 实现数据大小检查逻辑
	// 这里可以根据需要添加更详细的检查
	return nil
}

// ============================================================================
// 字段名安全检查
// ============================================================================

// ValidateFieldName 验证字段名安全性
func ValidateFieldName(name string) error {
	if name == "" {
		return fmt.Errorf("字段名不能为空")
	}

	if len(name) > maxFieldNameLength {
		return fmt.Errorf("字段名长度超过限制 %d", maxFieldNameLength)
	}

	// 检查是否包含危险字符
	if strings.ContainsAny(name, "\x00\n\r\t") {
		return fmt.Errorf("字段名包含非法字符")
	}

	// 检查UTF-8有效性
	if !utf8.ValidString(name) {
		return fmt.Errorf("字段名不是有效的UTF-8字符串")
	}

	return nil
}

// ValidateKeyName 验证键名安全性（用于Map）
func ValidateKeyName(key string) error {
	if key == "" {
		return fmt.Errorf("键名不能为空")
	}

	if len(key) > maxFieldNameLength {
		return fmt.Errorf("键名长度超过限制 %d", maxFieldNameLength)
	}

	// 检查危险字符
	if strings.ContainsAny(key, "\x00\n\r\t") {
		return fmt.Errorf("键名包含非法字符")
	}

	if !utf8.ValidString(key) {
		return fmt.Errorf("键名不是有效的UTF-8字符串")
	}

	return nil
}

// ============================================================================
// 规则安全检查
// ============================================================================

// ValidateRule 验证规则安全性
func ValidateRule(rule string) error {
	if rule == "" {
		return nil // 空规则是允许的
	}

	if len(rule) > maxRuleLength {
		return fmt.Errorf("规则长度超过限制 %d", maxRuleLength)
	}

	// 检查UTF-8有效性
	if !utf8.ValidString(rule) {
		return fmt.Errorf("规则不是有效的UTF-8字符串")
	}

	// 检查是否包含危险模式
	if strings.Contains(rule, "../") || strings.Contains(rule, "..\\") {
		return fmt.Errorf("规则包含危险路径遍历模式")
	}

	return nil
}

// ============================================================================
// 消息安全检查
// ============================================================================

// SanitizeMessage 清理错误消息
func SanitizeMessage(message string) string {
	if message == "" {
		return ""
	}

	// 限制长度
	if len(message) > maxMessageLength {
		message = message[:maxMessageLength-3] + "..."
	}

	// 移除控制字符
	message = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1 // 删除字符
		}
		return r
	}, message)

	return message
}

// ============================================================================
// 数据大小检查
// ============================================================================

// CheckMapSize 检查Map大小
func CheckMapSize(m map[string]interface{}) error {
	if len(m) > maxMapSize {
		return fmt.Errorf("Map大小 %d 超过限制 %d", len(m), maxMapSize)
	}
	return nil
}

// CheckSliceSize 检查切片大小
func CheckSliceSize(length int) error {
	if length > maxSliceSize {
		return fmt.Errorf("切片大小 %d 超过限制 %d", length, maxSliceSize)
	}
	return nil
}

// CheckStringLength 检查字符串长度
func CheckStringLength(s string) error {
	if len(s) > maxStringLength {
		return fmt.Errorf("字符串长度 %d 超过限制 %d", len(s), maxStringLength)
	}
	return nil
}

// ============================================================================
// 深度检查
// ============================================================================

// CheckDepth 检查嵌套深度
func CheckDepth(depth, maxDepth int) error {
	if depth > maxDepth {
		return fmt.Errorf("嵌套深度 %d 超过限制 %d", depth, maxDepth)
	}
	return nil
}

// ============================================================================
// 危险模式检查
// ============================================================================

// dangerousPatterns 危险模式列表
var dangerousPatterns = []string{
	"<script",
	"javascript:",
	"onerror=",
	"onclick=",
	"eval(",
	"setTimeout(",
	"setInterval(",
	"../",
	"..\\",
}

// ContainsDangerousPattern 检查是否包含危险模式
func ContainsDangerousPattern(s string) bool {
	lower := strings.ToLower(s)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// ValidateSafeString 验证安全字符串
func ValidateSafeString(s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("不是有效的UTF-8字符串")
	}

	if ContainsDangerousPattern(s) {
		return fmt.Errorf("包含危险模式")
	}

	return nil
}
