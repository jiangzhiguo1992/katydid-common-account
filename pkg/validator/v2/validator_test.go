package v2

import (
	"testing"
)

// ============================================================================
// 测试用例 - User 模型
// ============================================================================

// 预定义的通用场景常量
const (
	SceneCreate Scene = "create" // 创建场景
	SceneUpdate Scene = "update" // 更新场景
	SceneDelete Scene = "delete" // 删除场景
	SceneQuery  Scene = "query"  // 查询场景
)

// User 测试用户模型
type User struct {
	Username        string `json:"username"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Age             int    `json:"age"`
}

// ProvideRules 实现 RuleValidator 接口
func (u *User) ValidateRules() map[Scene]FieldRules {
	return map[Scene]FieldRules{
		SceneCreate: {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Password": "required,min=6",
			"Age":      "omitempty,gte=0,lte=150",
		},
		SceneUpdate: {
			"Username": "omitempty,min=3,max=20",
			"Email":    "omitempty,email",
			"Password": "omitempty,min=6",
		},
	}
}

// ValidateCustom 实现 CustomValidator 接口
func (u *User) ValidateCustom(scene Scene, reporter ErrorReporter) {
	// 跨字段验证：密码和确认密码必须一致
	if u.Password != "" && u.Password != u.ConfirmPassword {
		reporter.ReportMsg(
			"User.ConfirmPassword",
			"password_mismatch",
			"",
			"密码和确认密码不一致",
		)
	}

	// 场景化验证：创建时年龄必须大于等于 18
	if scene == SceneCreate && u.Age > 0 && u.Age < 18 {
		reporter.ReportMsg(
			"User.Age",
			"min_age",
			"18",
			"创建用户时年龄必须大于等于 18 岁",
		)
	}
}

// ============================================================================
// 基础功能测试
// ============================================================================

func TestValidator_Validate_Success(t *testing.T) {
	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		Age:             25,
	}

	validator := NewValidator()
	result := validator.Validate(user, SceneCreate)

	if !result.IsValid() {
		t.Errorf("Expected validation to pass, but got errors: %v", result.Error())
	}
}

func TestValidator_Validate_RequiredFields(t *testing.T) {
	user := &User{
		// 缺少必填字段
		Username: "",
		Email:    "",
		Password: "",
	}

	validator := NewValidator()
	result := validator.Validate(user, SceneCreate)

	if result.IsValid() {
		t.Error("Expected validation to fail, but it passed")
	}

	errors := result.Errors()
	if len(errors) == 0 {
		t.Error("Expected validation errors, but got none")
	}

	// 检查是否包含 required 错误
	hasRequiredError := false
	for _, err := range errors {
		if err.Tag == "required" {
			hasRequiredError = true
			break
		}
	}

	if !hasRequiredError {
		t.Error("Expected required field error")
	}
}

func TestValidator_Validate_MinLength(t *testing.T) {
	user := &User{
		Username:        "ab", // 少于 3 个字符
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
	}

	validator := NewValidator()
	result := validator.Validate(user, SceneCreate)

	if result.IsValid() {
		t.Error("Expected validation to fail, but it passed")
	}

	// 检查所有错误
	allErrors := result.Errors()
	if len(allErrors) == 0 {
		t.Error("Expected validation errors")
		return
	}

	// 查找 username 相关的错误（可能是 min 或其他标签）
	found := false
	for _, err := range allErrors {
		if err.Field == "username" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected username validation error, got errors: %v", allErrors)
	}
}

func TestValidator_Validate_CustomValidation(t *testing.T) {
	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "different", // 确认密码不一致
		Age:             25,
	}

	validator := NewValidator()
	result := validator.Validate(user, SceneCreate)

	if result.IsValid() {
		t.Error("Expected validation to fail due to password mismatch")
	}

	errors := result.ErrorsByTag("password_mismatch")
	if len(errors) == 0 {
		t.Error("Expected password mismatch error")
	}
}

func TestValidator_Validate_SceneValidation(t *testing.T) {
	// 创建场景：年龄小于 18 应该失败
	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		Age:             16,
	}

	validator := NewValidator()
	result := validator.Validate(user, SceneCreate)

	if result.IsValid() {
		t.Error("Expected validation to fail due to age restriction in create scene")
	}

	errors := result.ErrorsByTag("min_age")
	if len(errors) == 0 {
		t.Error("Expected age validation error in create scene")
	}

	// 更新场景：年龄小于 18 应该通过（没有年龄限制）
	result2 := validator.Validate(user, SceneUpdate)

	// 更新场景下，只有密码匹配的检查
	if !result2.IsValid() {
		// 检查是否只是密码不匹配的错误
		errors := result2.Errors()
		for _, err := range errors {
			if err.Tag == "min_age" {
				t.Error("Should not have age validation error in update scene")
			}
		}
	}
}

// ============================================================================
// Result 接口测试
// ============================================================================

func TestValidationResult_IsValid(t *testing.T) {
	result := NewValidationResult()

	if !result.IsValid() {
		t.Error("Expected empty result to be valid")
	}

	result = NewValidationResultWithErrors([]*FieldError{
		NewFieldError("field", "field", "required", ""),
	})

	if result.IsValid() {
		t.Error("Expected result with errors to be invalid")
	}
}

func TestValidationResult_FirstError(t *testing.T) {
	result := NewValidationResult()

	if result.FirstError() != nil {
		t.Error("Expected FirstError to return nil for empty result")
	}

	errors := []*FieldError{
		NewFieldError("field1", "field1", "required", ""),
		NewFieldError("field2", "field2", "min", "3"),
	}

	result = NewValidationResultWithErrors(errors)
	firstError := result.FirstError()

	if firstError == nil {
		t.Error("Expected FirstError to return an error")
		return
	}

	if firstError.Field != "field1" {
		t.Errorf("Expected first error field to be 'field1', got '%s'", firstError.Field)
	}
}

func TestValidationResult_ErrorsByField(t *testing.T) {
	errors := []*FieldError{
		NewFieldError("User.Username", "username", "required", ""),
		NewFieldError("User.Email", "email", "email", ""),
		NewFieldError("User.Username", "username", "min", "3"),
	}

	result := NewValidationResultWithErrors(errors)
	usernameErrors := result.ErrorsByField("username")

	if len(usernameErrors) != 2 {
		t.Errorf("Expected 2 username errors, got %d", len(usernameErrors))
	}
}

func TestValidationResult_ErrorsByTag(t *testing.T) {
	errors := []*FieldError{
		NewFieldError("User.Username", "username", "required", ""),
		NewFieldError("User.Email", "email", "required", ""),
		NewFieldError("User.Password", "password", "min", "6"),
	}

	result := NewValidationResultWithErrors(errors)
	requiredErrors := result.ErrorsByTag("required")

	if len(requiredErrors) != 2 {
		t.Errorf("Expected 2 required errors, got %d", len(requiredErrors))
	}
}

// ============================================================================
// 全局函数测试
// ============================================================================

func TestDefaultValidator(t *testing.T) {
	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		Age:             25,
	}

	// 使用全局默认验证器
	result := Validate(user, SceneCreate)

	if !result.IsValid() {
		t.Errorf("Expected validation to pass, but got errors: %v", result.Error())
	}
}

func TestClearCache(t *testing.T) {
	// 这个测试主要确保 ClearCache 不会 panic
	ClearCache()

	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
		Age:             25,
	}

	result := Validate(user, SceneCreate)

	if !result.IsValid() {
		t.Error("Validation should still work after clearing cache")
	}
}

// ============================================================================
// 边界条件测试
// ============================================================================

func TestValidator_Validate_NilObject(t *testing.T) {
	validator := NewValidator()
	result := validator.Validate(nil, SceneCreate)

	if result.IsValid() {
		t.Error("Expected validation to fail for nil object")
	}

	if result.FirstError() == nil {
		t.Error("Expected error for nil object")
	}
}

func TestValidator_Validate_EmptyScene(t *testing.T) {
	user := &User{
		Username:        "john",
		Email:           "john@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
	}

	validator := NewValidator()
	result := validator.Validate(user, Scene(""))

	// 空场景应该不匹配任何规则，但自定义验证仍会执行
	if !result.IsValid() {
		// 检查是否只有自定义验证的错误
		errors := result.Errors()
		for _, err := range errors {
			// 空场景下不应该有规则验证错误
			if err.Tag == "required" || err.Tag == "min" || err.Tag == "max" {
				t.Errorf("Should not have rule validation errors in empty scene: %v", err)
			}
		}
	}
}

// ============================================================================
// 并发测试
// ============================================================================

func TestValidator_Concurrent(t *testing.T) {
	validator := NewValidator()

	// 启动多个 goroutine 并发验证
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			user := &User{
				Username:        "john",
				Email:           "john@example.com",
				Password:        "password123",
				ConfirmPassword: "password123",
				Age:             25,
			}

			result := validator.Validate(user, SceneCreate)

			if !result.IsValid() {
				t.Errorf("Goroutine %d: Expected validation to pass", id)
			}

			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
