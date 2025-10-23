package v2

import (
	"testing"
)

// ============================================================================
// 测试用例结构体定义
// ============================================================================

// 常用场景定义
const (
	SceneCreate Scene = 1 << iota // 创建场景 (1)
	SceneUpdate                   // 更新场景 (2)
	SceneDelete                   // 删除场景 (4)
	SceneQuery                    // 查询场景 (8)
)

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Password string `json:"password"`
}

// RuleValidation 实现 RuleProvider 接口
func (u *User) RuleValidation() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3,max=20",
			"Email":    "required,email",
			"Age":      "required,gte=18",
			"Password": "required,min=6",
		},
		SceneUpdate: {
			"Username": "omitempty,min=3,max=20",
			"Email":    "omitempty,email",
			"Age":      "omitempty,gte=18",
		},
	}
}

// CustomValidation 实现 CustomValidator 接口
func (u *User) CustomValidation(scene Scene, report FuncReportError) {
	// 自定义验证：用户名不能是 admin
	if u.Username == "admin" {
		report("User.Username", "forbidden", "admin")
	}

	// 场景化验证：创建时年龄必须小于 100
	if scene == SceneCreate && u.Age > 100 {
		report("User.Age", "max_age", "100")
	}
}

// ============================================================================
// 基础验证测试
// ============================================================================

func TestValidate_Success(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		Password: "password123",
	}

	errs := Validate(user, SceneCreate)
	if errs != nil {
		t.Errorf("Expected no errors, got: %v", errs)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	user := &User{
		Username: "",
		Email:    "",
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Error("Expected validation errors")
	}

	if len(errs) == 0 {
		t.Error("Expected multiple errors")
	}
}

func TestValidate_MinLength(t *testing.T) {
	user := &User{
		Username: "ab", // 少于 3 个字符
		Email:    "test@example.com",
		Age:      25,
		Password: "password123",
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Error("Expected validation error for username length")
	}
}

func TestValidate_EmailFormat(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "invalid-email", // 无效的邮箱格式
		Age:      25,
		Password: "password123",
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Error("Expected validation error for email format")
	}
}

func TestValidate_CustomValidation(t *testing.T) {
	user := &User{
		Username: "admin", // 禁止的用户名
		Email:    "admin@example.com",
		Age:      25,
		Password: "password123",
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Error("Expected custom validation error")
	}

	// 检查是否有 forbidden 错误
	found := false
	for _, err := range errs {
		if err.Tag == "forbidden" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'forbidden' error tag")
	}
}

// ============================================================================
// 场景验证测试
// ============================================================================

func TestValidate_SceneCreate(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		Password: "password123",
	}

	errs := Validate(user, SceneCreate)
	if errs != nil {
		t.Errorf("Expected no errors for create scene, got: %v", errs)
	}
}

func TestValidate_SceneUpdate(t *testing.T) {
	// 更新场景中，字段是可选的
	user := &User{
		Username: "newname",
	}

	errs := Validate(user, SceneUpdate)
	if errs != nil {
		t.Errorf("Expected no errors for update scene, got: %v", errs)
	}
}

func TestValidate_SceneUpdateWithValidation(t *testing.T) {
	user := &User{
		Username: "ab", // 少于 3 个字符
	}

	errs := Validate(user, SceneUpdate)
	if errs == nil {
		t.Error("Expected validation error for username length in update scene")
	}
}

// ============================================================================
// 部分字段验证测试
// ============================================================================

func TestValidateFields_Success(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "invalid", // 邮箱无效，但不验证
		Age:      25,
	}

	// 只验证 Username 字段
	errs := ValidateFields(user, SceneCreate, "Username")
	if errs != nil {
		t.Errorf("Expected no errors, got: %v", errs)
	}
}

func TestValidateFields_Error(t *testing.T) {
	user := &User{
		Username: "ab", // 少于 3 个字符
		Email:    "test@example.com",
	}

	// 只验证 Username 字段
	errs := ValidateFields(user, SceneCreate, "Username")
	if errs == nil {
		t.Error("Expected validation error for username")
	}
}

func TestValidateFields_Multiple(t *testing.T) {
	user := &User{
		Username: "ab",      // 无效
		Email:    "invalid", // 无效
		Age:      25,
	}

	// 验证多个字段
	errs := ValidateFields(user, SceneCreate, "Username", "Email")
	if errs == nil {
		t.Error("Expected validation errors")
	}

	if len(errs) < 2 {
		t.Error("Expected at least 2 errors")
	}
}

// ============================================================================
// 排除字段验证测试
// ============================================================================

func TestValidateExcept_Success(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		// Password 为空，但被排除
	}

	// 排除 Password 字段
	errs := ValidateExcept(user, SceneCreate, "Password")
	if errs != nil {
		t.Errorf("Expected no errors, got: %v", errs)
	}
}

func TestValidateExcept_Error(t *testing.T) {
	user := &User{
		Username: "ab", // 无效
		Email:    "test@example.com",
		Age:      25,
	}

	// 排除 Password，但 Username 仍然无效
	errs := ValidateExcept(user, SceneCreate, "Password")
	if errs == nil {
		t.Error("Expected validation error for username")
	}
}

// ============================================================================
// 类型缓存测试
// ============================================================================

func TestClearTypeCache(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		Password: "password123",
	}

	// 第一次验证
	Validate(user, SceneCreate)

	// 清除缓存
	ClearTypeCache()

	// 再次验证应该仍然正常工作
	errs := Validate(user, SceneCreate)
	if errs != nil {
		t.Errorf("Expected no errors after cache clear, got: %v", errs)
	}
}

// ============================================================================
// 边界情况测试
// ============================================================================

func TestValidate_NilObject(t *testing.T) {
	errs := Validate(nil, SceneCreate)
	if errs == nil {
		t.Error("Expected error for nil object")
	}
}

func TestValidateFields_EmptyFields(t *testing.T) {
	user := &User{
		Username: "testuser",
	}

	// 空字段列表
	errs := ValidateFields(user, SceneCreate)
	if errs != nil {
		t.Errorf("Expected no errors for empty field list, got: %v", errs)
	}
}

func TestValidateExcept_EmptyExclude(t *testing.T) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		Password: "password123",
	}

	// 空排除列表，应该等同于完整验证
	errs := ValidateExcept(user, SceneCreate)
	if errs != nil {
		t.Errorf("Expected no errors, got: %v", errs)
	}
}

// ============================================================================
// 嵌套验证测试
// ============================================================================

type Profile struct {
	Bio     string `json:"bio"`
	Website string `json:"website"`
}

func (p *Profile) RuleValidation() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneCreate: {
			"Bio":     "required,min=10",
			"Website": "omitempty,url",
		},
	}
}

type UserWithProfile struct {
	Username string   `json:"username"`
	Profile  *Profile `json:"profile"`
}

func (u *UserWithProfile) RuleValidation() map[Scene]map[string]string {
	return map[Scene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3",
		},
	}
}

func TestValidate_NestedStruct(t *testing.T) {
	user := &UserWithProfile{
		Username: "testuser",
		Profile: &Profile{
			Bio:     "This is a bio longer than 10 characters",
			Website: "https://example.com",
		},
	}

	errs := Validate(user, SceneCreate)
	if errs != nil {
		t.Errorf("Expected no errors for nested validation, got: %v", errs)
	}
}

func TestValidate_NestedStructError(t *testing.T) {
	user := &UserWithProfile{
		Username: "testuser",
		Profile: &Profile{
			Bio:     "Short", // 少于 10 个字符
			Website: "invalid-url",
		},
	}

	errs := Validate(user, SceneCreate)
	if errs == nil {
		t.Error("Expected validation errors for nested struct")
	}
}

// ============================================================================
// 性能测试
// ============================================================================

func BenchmarkValidate(b *testing.B) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(user, SceneCreate)
	}
}

func BenchmarkValidateFields(b *testing.B) {
	user := &User{
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateFields(user, SceneCreate, "Username", "Email")
	}
}
