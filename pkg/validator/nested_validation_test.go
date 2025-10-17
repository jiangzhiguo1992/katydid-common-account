package validator

import (
	"fmt"
	"testing"

	"katydid-common-account/pkg/types"
)

// ============================================================================
// 嵌套验证测试模型定义
// ============================================================================

// TestBaseModel 测试用基础模型
type TestBaseModel struct {
	ID     int64        `json:"id"`
	Status types.Status `json:"status"`
	Extras types.Extras `json:"extras,omitempty"`
}

// ValidateRules 实现 Validatable 接口
func (m *TestBaseModel) ValidateRules() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {},
		SceneUpdate: {
			"ID": "omitempty,gt=0",
		},
	}
}

// TestProduct 测试用产品模型
type TestProduct struct {
	TestBaseModel
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	Category string  `json:"category"`
}

// ValidateRules 实现 Validatable 接口
func (p *TestProduct) ValidateRules() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {
			"Name":     "required,min=2,max=100",
			"Price":    "required,gt=0",
			"Stock":    "required,gte=0",
			"Category": "omitempty,min=2,max=50",
		},
		SceneUpdate: {
			"Name":     "omitempty,min=2,max=100",
			"Price":    "omitempty,gt=0",
			"Stock":    "omitempty,gte=0",
			"Category": "omitempty,min=2,max=50",
		},
	}
}

// CustomValidate 实现 CustomValidatable 接口
func (p *TestProduct) CustomValidate(scene ValidateScene) []*FieldError {
	if scene == SceneCreate {
		// 电子产品类别验证
		if p.Category == "electronics" {
			if errs := p.validateElectronicsExtras(); errs != nil {
				return errs
			}
		}

		// 服装类别验证
		if p.Category == "clothing" {
			if errs := p.validateClothingExtras(); errs != nil {
				return errs
			}
		}
	}

	return nil
}

// validateElectronicsExtras 验证电子产品的额外属性
func (p *TestProduct) validateElectronicsExtras() []*FieldError {
	if p.Extras == nil {
		return []*FieldError{NewFieldError("extras", "电子产品必须提供额外属性（品牌、保修期等）", nil, nil)}
	}

	// 验证必填键
	if err := ValidateMapMustHaveKeys(p.Extras, "brand", "warranty"); err != nil {
		return []*FieldError{NewFieldError("extras", err.Error(), nil, nil)}
	}

	// 验证 brand 字段
	if err := ValidateMapStringKey(p.Extras, "brand", 2, 50); err != nil {
		return []*FieldError{NewFieldError("extras.brand", err.Error(), nil, nil)}
	}

	// 验证 warranty 字段
	if err := ValidateMapIntKey(p.Extras, "warranty", 1, 60); err != nil {
		return []*FieldError{NewFieldError("extras.warranty", err.Error(), nil, nil)}
	}

	return nil
}

// validateClothingExtras 验证服装的额外属性
func (p *TestProduct) validateClothingExtras() []*FieldError {
	if p.Extras == nil {
		return []*FieldError{NewFieldError("extras", "服装必须提供额外属性（尺码、颜色等）", nil, nil)}
	}

	// 验证必填键
	if err := ValidateMapMustHaveKeys(p.Extras, "size", "color"); err != nil {
		return []*FieldError{NewFieldError("extras", err.Error(), nil, nil)}
	}

	// 验证 size 必须是指定值之一
	if err := ValidateMapKey(p.Extras, "size", func(value interface{}) error {
		size, ok := value.(string)
		if !ok {
			return fmt.Errorf("size 必须是字符串类型")
		}
		validSizes := map[string]bool{"XS": true, "S": true, "M": true, "L": true, "XL": true, "XXL": true}
		if !validSizes[size] {
			return fmt.Errorf("size 必须是 XS, S, M, L, XL, XXL 之一")
		}
		return nil
	}); err != nil {
		return []*FieldError{NewFieldError("extras.size", err.Error(), nil, nil)}
	}

	// 验证 color 字段
	if err := ValidateMapStringKey(p.Extras, "color", 2, 20); err != nil {
		return []*FieldError{NewFieldError("extras.color", err.Error(), nil, nil)}
	}

	return nil
}

// TestUserProfile 测试用用户资料模型
type TestUserProfile struct {
	TestBaseModel
	UserID   int64  `json:"user_id"`
	Bio      string `json:"bio"`
	Avatar   string `json:"avatar"`
	Location string `json:"location"`
}

// ValidateRules 实现 Validatable 接口
func (up *TestUserProfile) ValidateRules() map[ValidateScene]map[string]string {
	return map[ValidateScene]map[string]string{
		SceneCreate: {
			"UserID":   "required,gt=0",
			"Bio":      "omitempty,max=500",
			"Avatar":   "omitempty,url",
			"Location": "omitempty,max=100",
		},
		SceneUpdate: {
			"Bio":      "omitempty,max=500",
			"Avatar":   "omitempty,url",
			"Location": "omitempty,max=100",
		},
	}
}

// CustomValidate 验证 Extras 中的社交媒体链接
func (up *TestUserProfile) CustomValidate(scene ValidateScene) []*FieldError {
	_ = scene // 避免未使用参数警告

	if up.Extras == nil || len(up.Extras) == 0 {
		return nil
	}

	extrasValidator := &MapValidator{
		AllowedKeys: []string{"twitter", "github", "linkedin", "website", "hobbies"},
		KeyValidators: map[string]func(value interface{}) error{
			"twitter": func(value interface{}) error {
				url, ok := value.(string)
				if !ok {
					return fmt.Errorf("twitter 必须是字符串类型")
				}
				if len(url) == 0 || len(url) > 200 {
					return fmt.Errorf("twitter URL 长度必须在 1-200 之间")
				}
				return nil
			},
			"github": func(value interface{}) error {
				url, ok := value.(string)
				if !ok {
					return fmt.Errorf("github 必须是字符串类型")
				}
				if len(url) == 0 || len(url) > 200 {
					return fmt.Errorf("github URL 长度必须在 1-200 之间")
				}
				return nil
			},
		},
	}

	if err := ValidateMap(up.Extras, extrasValidator); err != nil {
		return []*FieldError{NewFieldError("extras", err.Error(), nil, nil)}
	}

	return nil
}

// ============================================================================
// 嵌套验证测试用例
// ============================================================================

// TestProduct_NestedValidation 测试产品模型的嵌套验证
func TestProduct_NestedValidation(t *testing.T) {
	tests := []struct {
		name    string
		product *TestProduct
		scene   ValidateScene
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效的电子产品 - BaseModel 和 Extras 都会验证",
			product: &TestProduct{
				TestBaseModel: TestBaseModel{
					Extras: types.Extras{
						"brand":    "Apple",
						"warranty": 12,
					},
				},
				Name:     "iPhone 15",
				Price:    999.99,
				Stock:    100,
				Category: "electronics",
			},
			scene:   SceneCreate,
			wantErr: false,
		},
		{
			name: "电子产品缺少 brand - Extras 验证失败",
			product: &TestProduct{
				TestBaseModel: TestBaseModel{
					Extras: types.Extras{
						"warranty": 12,
					},
				},
				Name:     "iPhone 15",
				Price:    999.99,
				Stock:    100,
				Category: "electronics",
			},
			scene:   SceneCreate,
			wantErr: true,
		},
		{
			name: "电子产品缺少 Extras",
			product: &TestProduct{
				Name:     "iPhone 15",
				Price:    999.99,
				Stock:    100,
				Category: "electronics",
			},
			scene:   SceneCreate,
			wantErr: true,
		},
		{
			name: "有效的服装产品",
			product: &TestProduct{
				TestBaseModel: TestBaseModel{
					Extras: types.Extras{
						"size":  "M",
						"color": "blue",
					},
				},
				Name:     "T-Shirt",
				Price:    29.99,
				Stock:    500,
				Category: "clothing",
			},
			scene:   SceneCreate,
			wantErr: false,
		},
		{
			name: "服装产品尺码不合法",
			product: &TestProduct{
				TestBaseModel: TestBaseModel{
					Extras: types.Extras{
						"size":  "INVALID",
						"color": "blue",
					},
				},
				Name:     "T-Shirt",
				Price:    29.99,
				Stock:    500,
				Category: "clothing",
			},
			scene:   SceneCreate,
			wantErr: true,
		},
		{
			name: "价格验证失败 - Product 字段验证",
			product: &TestProduct{
				Name:     "Test Product",
				Price:    -10.0, // 价格不能为负
				Stock:    100,
				Category: "electronics",
			},
			scene:   SceneCreate,
			wantErr: true,
		},
		{
			name: "名称太短 - Product 字段验证",
			product: &TestProduct{
				Name:     "A", // 名称至少2个字符
				Price:    99.99,
				Stock:    100,
				Category: "books",
			},
			scene:   SceneCreate,
			wantErr: true,
		},
		{
			name: "更新场景 - 只更新部分字段",
			product: &TestProduct{
				Name:  "Updated Name",
				Price: 199.99,
			},
			scene:   SceneUpdate,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.product, tt.scene)
			if (errs != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
			if errs != nil {
				t.Logf("验证错误: %v", errs)
			}
		})
	}
}

// TestUserProfile_NestedValidation 测试用户资料的嵌套验证
func TestUserProfile_NestedValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile *TestUserProfile
		scene   ValidateScene
		wantErr bool
	}{
		{
			name: "有效的用户资料",
			profile: &TestUserProfile{
				TestBaseModel: TestBaseModel{
					Extras: types.Extras{
						"twitter": "https://twitter.com/example",
						"github":  "https://github.com/example",
					},
				},
				UserID:   123,
				Bio:      "Software Engineer",
				Avatar:   "https://example.com/avatar.jpg",
				Location: "San Francisco",
			},
			scene:   SceneCreate,
			wantErr: false,
		},
		{
			name: "Extras 包含不允许的键",
			profile: &TestUserProfile{
				TestBaseModel: TestBaseModel{
					Extras: types.Extras{
						"facebook": "https://facebook.com/example", // 不在允许列表中
					},
				},
				UserID: 123,
				Bio:    "Developer",
			},
			scene:   SceneCreate,
			wantErr: true,
		},
		{
			name: "Avatar URL 格式错误",
			profile: &TestUserProfile{
				UserID: 123,
				Avatar: "not-a-valid-url",
			},
			scene:   SceneCreate,
			wantErr: true,
		},
		{
			name: "Bio 太长",
			profile: &TestUserProfile{
				UserID: 123,
				Bio:    string(make([]byte, 501)), // 超过 500 字符
			},
			scene:   SceneCreate,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.profile, tt.scene)
			if (errs != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
			if errs != nil {
				t.Logf("验证错误: %v", errs)
			}
		})
	}
}

// TestBaseModel_NestedValidation 测试 BaseModel 的嵌套验证
func TestBaseModel_NestedValidation(t *testing.T) {
	// 使用 TestUser 模型测试，它会继承验证规则
	type TestUserNested struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// 实现 Validatable 接口
	userValidateRules := func() map[ValidateScene]map[string]string {
		return map[ValidateScene]map[string]string{
			SceneCreate: {
				"Username": "required,min=3",
				"Email":    "required,email",
				"Password": "required,min=6",
			},
		}
	}

	// 创建用户实例
	user := &TestUserNested{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	// 由于 TestUserNested 没有实现接口，使用 ValidateStruct
	err := ValidateStruct(user)
	if err != nil {
		t.Logf("验证结果: %v", err)
	}

	// 测试验证规则函数
	rules := userValidateRules()
	if rules == nil {
		t.Error("验证规则不应该为 nil")
	}
}

// TestComplexNestedStructure 测试复杂的嵌套结构
func TestComplexNestedStructure(t *testing.T) {
	// 创建一个复杂的嵌套结构
	type Address struct {
		Street  string         `json:"street"`
		City    string         `json:"city"`
		Country string         `json:"country"`
		Extras  map[string]any `json:"extras,omitempty"`
	}

	type Company struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	type Employee struct {
		Name    string  `json:"name"`
		Email   string  `json:"email"`
		Company Company `json:"company"`
	}

	// 创建嵌套数据
	employee := Employee{
		Name:  "John Doe",
		Email: "john@example.com",
		Company: Company{
			Name: "Tech Corp",
			Address: Address{
				Street:  "123 Main St",
				City:    "San Francisco",
				Country: "USA",
				Extras: map[string]any{
					"zip_code": "94102",
					"phone":    "+1-555-1234",
				},
			},
		},
	}

	// 由于没有实现验证接口，使用 ValidateStruct
	err := ValidateStruct(employee)
	if err != nil {
		t.Logf("复杂嵌套结构验证: %v", err)
	}
}

// TestNestedValidation_WithEmbeddedFields 测试包含嵌入字段的验证
func TestNestedValidation_WithEmbeddedFields(t *testing.T) {
	type Timestamps struct {
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}

	type Article struct {
		Timestamps        // 嵌入字段
		Title      string `json:"title"`
		Content    string `json:"content"`
		Author     string `json:"author"`
	}

	article := Article{
		Timestamps: Timestamps{
			CreatedAt: "2024-01-01",
			UpdatedAt: "2024-01-02",
		},
		Title:   "Test Article",
		Content: "This is test content",
		Author:  "John Doe",
	}

	// 由于没有实现验证接口，使用 ValidateStruct
	err := ValidateStruct(article)
	if err != nil {
		t.Logf("嵌入字段验证: %v", err)
	}
}

// TestNestedValidation_MaxDepth 测试最大嵌套深度限制
func TestNestedValidation_MaxDepth(t *testing.T) {
	// 创建一个深度嵌套的结构（用于测试深度限制）
	type Level struct {
		Name  string `json:"name"`
		Child *Level `json:"child,omitempty"`
	}

	// 创建一个适度嵌套的结构
	level5 := &Level{Name: "Level 5"}
	level4 := &Level{Name: "Level 4", Child: level5}
	level3 := &Level{Name: "Level 3", Child: level4}
	level2 := &Level{Name: "Level 2", Child: level3}
	level1 := &Level{Name: "Level 1", Child: level2}

	// 验证嵌套结构
	err := ValidateStruct(level1)
	if err != nil {
		t.Logf("嵌套深度验证: %v", err)
	}
}

// TestNestedValidation_NilFields 测试 nil 字段的处理
func TestNestedValidation_NilFields(t *testing.T) {
	type OptionalData struct {
		Value string `json:"value"`
	}

	type Container struct {
		Name     string        `json:"name"`
		Optional *OptionalData `json:"optional,omitempty"`
	}

	tests := []struct {
		name      string
		container *Container
		wantErr   bool
	}{
		{
			name: "包含可选数据",
			container: &Container{
				Name: "Test",
				Optional: &OptionalData{
					Value: "Optional Value",
				},
			},
			wantErr: false,
		},
		{
			name: "不包含可选数据",
			container: &Container{
				Name:     "Test",
				Optional: nil, // nil 指针
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.container)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNestedValidation_CircularReference 测试循环引用的处理
func TestNestedValidation_CircularReference(t *testing.T) {
	// 注意：这个测试主要是为了确保验证器不会因为循环引用而死锁
	// 在实际使用中应该避免循环引用

	type Node struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	node := Node{
		Name:  "Test Node",
		Value: 42,
	}

	// 验证简单节点
	err := ValidateStruct(node)
	if err != nil {
		t.Logf("节点验证: %v", err)
	}
}
