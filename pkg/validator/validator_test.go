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
