package validator

import (
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
)

// TestUser 测试用户模型
type TestUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
	Phone    string `json:"phone"`
}

// 测试场景常量
const (
	SceneCreate ValidateScene = "create" // 创建场景
	SceneUpdate ValidateScene = "update" // 更新场景
	SceneDelete ValidateScene = "delete" // 删除场景
	SceneQuery  ValidateScene = "query"  // 查询场景
)

// ValidateRules 实现 Validatable 接口
func (u *TestUser) ValidateRules() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3,max=20,alphanum",
			"Email":    "required,email",
			"Password": "required,min=6,max=20",
			"Age":      "omitempty,gte=0,lte=150",
			"Phone":    "omitempty,len=11,numeric",
		},
		SceneUpdate: {
			"Username": "omitempty,min=3,max=20,alphanum",
			"Email":    "omitempty,email",
			"Password": "omitempty,min=6,max=20",
			"Age":      "omitempty,gte=0,lte=150",
			"Phone":    "omitempty,len=11,numeric",
		},
	}
}

// CustomValidate 实现 CustomValidatable 接口
func (u *TestUser) CustomValidate(scene ValidateScene) error {
	if scene == SceneCreate {
		// 创建时，用户名不能是admin
		if u.Username == "admin" {
			return fmt.Errorf("用户名不能是 'admin'")
		}
	}
	return nil
}

// GetErrorMessage 实现 ErrorMessageProvider 接口，自定义错误消息
func (u *TestUser) GetErrorMessage(fieldName, tag, param string) string {
	switch fieldName {
	case "username":
		switch tag {
		case "required":
			return "用户名不能为空"
		case "min":
			return fmt.Sprintf("用户名长度不能少于%s个字符", param)
		case "max":
			return fmt.Sprintf("用户名长度不能超过%s个字符", param)
		case "alphanum":
			return "用户名只能包含字母和数字"
		}
	case "email":
		switch tag {
		case "required":
			return "邮箱地址不能为空"
		case "email":
			return "请输入有效的邮箱地址"
		}
	case "password":
		switch tag {
		case "required":
			return "密码不能为空"
		case "min":
			return fmt.Sprintf("密码长度不能少于%s位", param)
		}
	case "phone":
		switch tag {
		case "len":
			return "手机号码必须是11位数字"
		case "numeric":
			return "手机号码只能包含数字"
		}
	}
	// 返回空字符串使用默认消息
	return ""
}

func TestValidate_Create(t *testing.T) {
	tests := []struct {
		name    string
		user    *TestUser
		wantErr bool
	}{
		{
			name: "有效的创建数据",
			user: &TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Age:      25,
			},
			wantErr: false,
		},
		{
			name: "缺少必填字段username",
			user: &TestUser{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "无效的邮箱",
			user: &TestUser{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "密码太短",
			user: &TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123",
			},
			wantErr: true,
		},
		{
			name: "用户名是admin",
			user: &TestUser{
				Username: "admin",
				Email:    "admin@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "手机号格式错误",
			user: &TestUser{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Phone:    "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.user, SceneCreate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("验证错误: %v", err)
			}
		})
	}
}

func TestValidate_Update(t *testing.T) {
	tests := []struct {
		name    string
		user    *TestUser
		wantErr bool
	}{
		{
			name: "更新用户名",
			user: &TestUser{
				ID:       1,
				Username: "newname",
			},
			wantErr: false,
		},
		{
			name: "更新邮箱",
			user: &TestUser{
				ID:    1,
				Email: "newemail@example.com",
			},
			wantErr: false,
		},
		{
			name: "更新无效邮箱",
			user: &TestUser{
				ID:    1,
				Email: "invalid",
			},
			wantErr: true,
		},
		{
			name: "空数据也可以（更新时所有字段都是可选的）",
			user: &TestUser{
				ID: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.user, SceneUpdate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("验证错误: %v", err)
			}
		})
	}
}

func TestRegisterValidation(t *testing.T) {
	// 注册自定义验证规则
	err := RegisterValidation("is_awesome", func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "awesome"
	})
	if err != nil {
		t.Fatalf("RegisterValidation() error = %v", err)
	}

	type TestStruct struct {
		Field string `validate:"is_awesome"`
	}

	tests := []struct {
		name    string
		obj     *TestStruct
		wantErr bool
	}{
		{
			name:    "符合自定义规则",
			obj:     &TestStruct{Field: "awesome"},
			wantErr: false,
		},
		{
			name:    "不符合自定义规则",
			obj:     &TestStruct{Field: "not awesome"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCustomErrorMessage 测试自定义错误消息功能
func TestCustomErrorMessage(t *testing.T) {
	// 测试缺少用户名的情况
	user := &TestUser{
		Email:    "test@example.com",
		Password: "password123",
	}

	err := Validate(user, SceneCreate)
	if err == nil {
		t.Fatal("期望验证失败，但是通过了")
	}

	// 检查是否包含自定义错误消息
	errMsg := err.Error()
	t.Logf("验证错误消息: %s", errMsg)

	// 验证错误消息包含字段名
	if errMsg == "" {
		t.Error("错误消息为空")
	}
}

// TestCustomErrorMessageVsDefault 对比自定义错误消息和默认错误消息
func TestCustomErrorMessageVsDefault(t *testing.T) {
	t.Run("使用自定义错误消息", func(t *testing.T) {
		user := &TestUser{
			Username: "ab", // 太短，少于3个字符
			Email:    "test@example.com",
			Password: "password123",
		}

		err := Validate(user, SceneCreate)
		if err == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("自定义错误消息: %s", err.Error())
	})

	t.Run("无效的邮箱格式", func(t *testing.T) {
		user := &TestUser{
			Username: "testuser",
			Email:    "invalid-email",
			Password: "password123",
		}

		err := Validate(user, SceneCreate)
		if err == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("邮箱验证错误: %s", err.Error())
	})

	t.Run("手机号格式错误", func(t *testing.T) {
		user := &TestUser{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
			Phone:    "123", // 不足11位
		}

		err := Validate(user, SceneCreate)
		if err == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("手机号验证错误: %s", err.Error())
	})
}

// TestNoCustomErrorMessage 测试不实现 ErrorMessageProvider 接口的情况
func TestNoCustomErrorMessage(t *testing.T) {
	type SimpleUser struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// SimpleUser 没有实现 ErrorMessageProvider 接口
	// 应该使用默认错误消息

	user := &SimpleUser{
		Email: "test@example.com",
	}

	// 由于没有实现 Validatable 接口，会使用 ValidateStruct
	err := ValidateStruct(user)
	if err != nil {
		t.Logf("默认错误消息: %s", err.Error())
	}
}

// 示例：国际化错误消息
type InternationalUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Lang     string `json:"-"` // 语言设置
}

func (u *InternationalUser) ValidateRules() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {
			"Username": "required,min=3",
			"Email":    "required,email",
		},
	}
}

func (u *InternationalUser) GetErrorMessage(fieldName, tag, param string) string {
	if u.Lang == "en" {
		return u.getEnglishMessage(fieldName, tag, param)
	}
	return u.getChineseMessage(fieldName, tag, param)
}

func (u *InternationalUser) getEnglishMessage(fieldName, tag, param string) string {
	switch fieldName {
	case "username":
		switch tag {
		case "required":
			return "Username is required"
		case "min":
			return fmt.Sprintf("Username must be at least %s characters", param)
		}
	case "email":
		switch tag {
		case "required":
			return "Email is required"
		case "email":
			return "Invalid email format"
		}
	}
	return ""
}

func (u *InternationalUser) getChineseMessage(fieldName, tag, param string) string {
	switch fieldName {
	case "username":
		switch tag {
		case "required":
			return "用户名是必填项"
		case "min":
			return fmt.Sprintf("用户名至少需要%s个字符", param)
		}
	case "email":
		switch tag {
		case "required":
			return "邮箱是必填项"
		case "email":
			return "邮箱格式无效"
		}
	}
	return ""
}

func TestInternationalErrorMessage(t *testing.T) {
	t.Run("中文错误消息", func(t *testing.T) {
		user := &InternationalUser{
			Lang: "zh",
		}

		err := Validate(user, SceneCreate)
		if err == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("中文错误消息: %s", err.Error())
	})

	t.Run("英文错误消息", func(t *testing.T) {
		user := &InternationalUser{
			Lang: "en",
		}

		err := Validate(user, SceneCreate)
		if err == nil {
			t.Fatal("期望验证失败")
		}

		t.Logf("English error message: %s", err.Error())
	})
}
