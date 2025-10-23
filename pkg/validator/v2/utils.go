package v2

import (
	"fmt"
	"strings"
)

// ============================================================================
// 工具函数集 - 单一职责：提供通用的辅助功能
// 设计原则：高内聚低耦合，可复用性强
// ============================================================================

// TruncateString 安全截断字符串
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// BuildFieldPath 构建字段路径
func BuildFieldPath(parent, field string) string {
	if parent == "" {
		return field
	}
	if field == "" {
		return parent
	}
	return parent + "." + field
}

// BuildArrayPath 构建数组路径
func BuildArrayPath(parent string, index int) string {
	if parent == "" {
		return fmt.Sprintf("[%d]", index)
	}
	return fmt.Sprintf("%s[%d]", parent, index)
}

// BuildMapPath 构建Map路径
func BuildMapPath(parent, key string) string {
	if parent == "" {
		return fmt.Sprintf("[%s]", key)
	}
	return fmt.Sprintf("%s[%s]", parent, key)
}

// ExtractJSONTag 提取JSON标签名称
func ExtractJSONTag(tag string) string {
	if tag == "" {
		return ""
	}

	// 分割逗号前的部分
	parts := strings.SplitN(tag, ",", 2)
	if len(parts) == 0 {
		return ""
	}

	name := strings.TrimSpace(parts[0])
	if name == "-" {
		return ""
	}

	return name
}

// ParseValidationTag 解析验证标签
func ParseValidationTag(tag string) map[string]string {
	if tag == "" {
		return nil
	}

	result := make(map[string]string)
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 分割键值对
		kv := strings.SplitN(part, "=", 2)
		key := strings.TrimSpace(kv[0])

		if len(kv) == 2 {
			result[key] = strings.TrimSpace(kv[1])
		} else {
			result[key] = ""
		}
	}

	return result
}

// HasTag 检查标签中是否包含指定的验证规则
func HasTag(tag, target string) bool {
	if tag == "" || target == "" {
		return false
	}

	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// 精确匹配或键值对匹配
		if part == target {
			return true
		}

		// 检查键值对的键部分
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if strings.TrimSpace(kv[0]) == target {
				return true
			}
		}
	}

	return false
}

// IsRequiredField 检查字段是否必填
func IsRequiredField(tag string) bool {
	return HasTag(tag, "required")
}

// IsOmitEmptyField 检查字段是否可以省略空值
func IsOmitEmptyField(tag string) bool {
	return HasTag(tag, "omitempty")
}

// MergeRules 合并多个规则集
func MergeRules(ruleSets ...map[string]string) map[string]string {
	if len(ruleSets) == 0 {
		return nil
	}

	result := make(map[string]string)
	for _, rules := range ruleSets {
		for field, rule := range rules {
			result[field] = rule
		}
	}

	return result
}

// FilterRules 过滤规则（保留指定字段）
func FilterRules(rules map[string]string, fields []string) map[string]string {
	if len(rules) == 0 || len(fields) == 0 {
		return nil
	}

	fieldSet := make(map[string]bool, len(fields))
	for _, field := range fields {
		fieldSet[field] = true
	}

	result := make(map[string]string)
	for field, rule := range rules {
		if fieldSet[field] {
			result[field] = rule
		}
	}

	return result
}

// ExcludeRules 排除规则（移除指定字段）
func ExcludeRules(rules map[string]string, fields []string) map[string]string {
	if len(rules) == 0 {
		return nil
	}

	if len(fields) == 0 {
		// 没有要排除的字段，返回副本
		result := make(map[string]string, len(rules))
		for k, v := range rules {
			result[k] = v
		}
		return result
	}

	fieldSet := make(map[string]bool, len(fields))
	for _, field := range fields {
		fieldSet[field] = true
	}

	result := make(map[string]string)
	for field, rule := range rules {
		if !fieldSet[field] {
			result[field] = rule
		}
	}

	return result
}

// CombineMessages 组合多个错误消息
func CombineMessages(messages ...string) string {
	if len(messages) == 0 {
		return ""
	}

	var parts []string
	for _, msg := range messages {
		msg = strings.TrimSpace(msg)
		if msg != "" {
			parts = append(parts, msg)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "; ")
}

// FormatErrorMessage 格式化错误消息
func FormatErrorMessage(field, tag, param string) string {
	if field == "" {
		return "验证失败"
	}

	if tag == "" {
		return fmt.Sprintf("字段 '%s' 验证失败", field)
	}

	if param == "" {
		return fmt.Sprintf("字段 '%s' 验证失败: %s", field, tag)
	}

	return fmt.Sprintf("字段 '%s' 验证失败: %s(%s)", field, tag, param)
}

// GetDefaultMessage 获取默认错误消息
func GetDefaultMessage(tag, param string) string {
	messages := map[string]string{
		"required":  "此字段为必填项",
		"email":     "请输入有效的邮箱地址",
		"min":       "值不能小于 " + param,
		"max":       "值不能大于 " + param,
		"len":       "长度必须为 " + param,
		"eq":        "值必须等于 " + param,
		"ne":        "值不能等于 " + param,
		"gt":        "值必须大于 " + param,
		"gte":       "值必须大于或等于 " + param,
		"lt":        "值必须小于 " + param,
		"lte":       "值必须小于或等于 " + param,
		"alpha":     "只能包含字母",
		"alphanum":  "只能包含字母和数字",
		"numeric":   "必须是数字",
		"url":       "请输入有效的URL",
		"uri":       "请输入有效的URI",
		"json":      "必须是有效的JSON格式",
		"uuid":      "必须是有效的UUID",
		"ip":        "请输入有效的IP地址",
		"ipv4":      "请输入有效的IPv4地址",
		"ipv6":      "请输入有效的IPv6地址",
		"mac":       "请输入有效的MAC地址",
		"latitude":  "请输入有效的纬度",
		"longitude": "请输入有效的经度",
	}

	if msg, ok := messages[tag]; ok {
		return msg
	}

	return "验证失败"
}

// IsZeroValue 判断是否为零值
func IsZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case int, int8, int16, int32, int64:
		return val == 0
	case uint, uint8, uint16, uint32, uint64:
		return val == 0
	case float32, float64:
		return val == 0
	case bool:
		return !val
	default:
		return false
	}
}

// CloneMap 克隆Map
func CloneMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	clone := make(map[string]string, len(m))
	for k, v := range m {
		clone[k] = v
	}

	return clone
}

// CloneStringSlice 克隆字符串切片
func CloneStringSlice(s []string) []string {
	if s == nil {
		return nil
	}

	clone := make([]string, len(s))
	copy(clone, s)

	return clone
}
