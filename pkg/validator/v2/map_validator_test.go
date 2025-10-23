package v2

import (
	"fmt"
	"testing"
)

// ============================================================================
// Map 验证器测试
// ============================================================================

func TestMapValidator_RequiredKeys(t *testing.T) {
	data := map[string]any{
		"name": "Product A",
	}

	validator := NewMapValidator().
		WithNamespace("Product.Extras").
		WithRequiredKeys("name", "price")

	errors := validator.Validate(data)

	if len(errors) == 0 {
		t.Error("Expected validation errors for missing required key")
	}

	// 检查是否包含 price 的错误
	hasPrice := false
	for _, err := range errors {
		if err.Field == "price" && err.Tag == "required" {
			hasPrice = true
			break
		}
	}

	if !hasPrice {
		t.Error("Expected error for missing 'price' key")
	}
}

func TestMapValidator_AllowedKeys(t *testing.T) {
	data := map[string]any{
		"name":  "Product A",
		"price": 99.99,
		"extra": "not allowed", // 不在白名单中
	}

	validator := NewMapValidator().
		WithAllowedKeys("name", "price")

	errors := validator.Validate(data)

	if len(errors) == 0 {
		t.Error("Expected validation errors for not allowed key")
	}

	// 检查是否包含 extra 的错误
	hasExtra := false
	for _, err := range errors {
		if err.Field == "extra" && err.Tag == "not_allowed" {
			hasExtra = true
			break
		}
	}

	if !hasExtra {
		t.Error("Expected error for 'extra' key not in allowed list")
	}
}

func TestMapValidator_CustomKeyValidator(t *testing.T) {
	data := map[string]any{
		"quantity": -5, // 无效的数量
	}

	validator := NewMapValidator().
		WithKeyValidator("quantity", func(value any) error {
			qty, ok := value.(int)
			if !ok {
				return nil
			}
			if qty < 0 {
				return fmt.Errorf("quantity must be positive")
			}
			return nil
		})

	errors := validator.Validate(data)

	if len(errors) == 0 {
		t.Error("Expected validation error for negative quantity")
	}
}

func TestMapValidator_NilData(t *testing.T) {
	validator := NewMapValidator().
		WithRequiredKeys("name")

	errors := validator.Validate(nil)

	if len(errors) == 0 {
		t.Error("Expected validation error for nil data when required keys are specified")
	}
}

func TestMapValidator_NilData_NoRequiredKeys(t *testing.T) {
	validator := NewMapValidator()

	errors := validator.Validate(nil)

	if len(errors) != 0 {
		t.Error("Expected no validation errors for nil data when no required keys")
	}
}

// ============================================================================
// 便捷函数测试
// ============================================================================

func TestValidateMapRequired(t *testing.T) {
	data := map[string]any{
		"name": "Product A",
	}

	err := ValidateMapRequired(data, "name", "price")

	if err == nil {
		t.Error("Expected error for missing required key")
	}
}

func TestValidateMapString(t *testing.T) {
	data := map[string]any{
		"name": "AB", // 长度为 2
	}

	// 测试最小长度
	err := ValidateMapString(data, "name", 3, 10)
	if err == nil {
		t.Error("Expected error for string shorter than minimum length")
	}

	// 测试最大长度
	data["name"] = "This is a very long string"
	err = ValidateMapString(data, "name", 1, 10)
	if err == nil {
		t.Error("Expected error for string longer than maximum length")
	}

	// 测试通过
	data["name"] = "Valid"
	err = ValidateMapString(data, "name", 3, 10)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

func TestValidateMapInt(t *testing.T) {
	data := map[string]any{
		"age": 15,
	}

	// 测试最小值
	err := ValidateMapInt(data, "age", 18, 100)
	if err == nil {
		t.Error("Expected error for value less than minimum")
	}

	// 测试最大值
	data["age"] = 120
	err = ValidateMapInt(data, "age", 0, 100)
	if err == nil {
		t.Error("Expected error for value greater than maximum")
	}

	// 测试通过
	data["age"] = 25
	err = ValidateMapInt(data, "age", 18, 100)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// 测试 int64 类型
	data["age"] = int64(30)
	err = ValidateMapInt(data, "age", 18, 100)
	if err != nil {
		t.Errorf("Expected no error for int64, but got: %v", err)
	}

	// 测试 float64 类型（整数）
	data["age"] = float64(35)
	err = ValidateMapInt(data, "age", 18, 100)
	if err != nil {
		t.Errorf("Expected no error for float64 integer, but got: %v", err)
	}

	// 测试非整数的 float64
	data["age"] = 35.5
	err = ValidateMapInt(data, "age", 18, 100)
	if err == nil {
		t.Error("Expected error for non-integer float64")
	}
}

func TestValidateMapBool(t *testing.T) {
	data := map[string]any{
		"active": true,
	}

	err := ValidateMapBool(data, "active")
	if err != nil {
		t.Errorf("Expected no error for bool value, but got: %v", err)
	}

	// 测试非布尔值
	data["active"] = "true"
	err = ValidateMapBool(data, "active")
	if err == nil {
		t.Error("Expected error for non-bool value")
	}
}

// ============================================================================
// 综合示例测试
// ============================================================================

func TestMapValidator_CompleteExample(t *testing.T) {
	// 模拟产品的 Extras 字段
	extras := map[string]any{
		"brand":    "Apple",
		"warranty": 24,
		"color":    "Silver",
	}

	validator := NewMapValidator().
		WithNamespace("Product.Extras").
		WithRequiredKeys("brand", "warranty").
		WithAllowedKeys("brand", "warranty", "color").
		WithKeyValidator("warranty", func(value any) error {
			warranty, ok := value.(int)
			if !ok {
				return fmt.Errorf("warranty must be integer")
			}
			if warranty < 12 || warranty > 60 {
				return fmt.Errorf("warranty must be between 12 and 60 months")
			}
			return nil
		})

	errors := validator.Validate(extras)

	if len(errors) != 0 {
		t.Errorf("Expected no validation errors, but got: %v", errors)
	}

	// 测试无效数据
	invalidExtras := map[string]any{
		"brand":   "Apple",
		"invalid": "field", // 不在允许列表中
	}

	errors = validator.Validate(invalidExtras)

	if len(errors) != 2 { // 缺少 warranty + invalid 字段
		t.Errorf("Expected 2 validation errors, but got %d", len(errors))
	}
}
