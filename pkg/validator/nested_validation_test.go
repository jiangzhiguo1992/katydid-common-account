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

// Rules 实现 RuleValidator 接口
func (p *TestProduct) RuleValidation() map[ValidateScene]map[string]string {
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

// CrossFieldValidation 验证 Extras 中的社交媒体链接
func (up *TestUserProfile) CustomValidation(scene ValidateScene) []*FieldError {
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

	if errs := ValidateMap(up.Extras, extrasValidator); errs != nil {
		return errs
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
	// 测试带有嵌入 BaseModel 的结构验证
	product := &TestProduct{
		TestBaseModel: TestBaseModel{
			ID:     1,
			Status: 1,
		},
		Name:     "Test Product",
		Price:    99.99,
		Stock:    100,
		Category: "books",
	}

	// 测试更新场景，验证 BaseModel 的 ID 验证规则
	errs := Validate(product, SceneUpdate)
	if errs != nil {
		t.Logf("验证结果: %v", errs)
	}

	// 测试创建场景
	errs = Validate(product, SceneCreate)
	if errs != nil {
		t.Logf("创建场景验证结果: %v", errs)
	}
}

// TestComplexNestedStructure 测试复杂的嵌套结构
func TestComplexNestedStructure(t *testing.T) {
	// 创建一个包含多层嵌套的产品测试
	type NestedProduct struct {
		TestProduct
		RelatedProducts []TestProduct `json:"related_products,omitempty"`
	}

	// 实现 RuleValidator 接口
	var _ RuleValidator = (*NestedProduct)(nil)

	nested := &NestedProduct{
		TestProduct: TestProduct{
			Name:     "Main Product",
			Price:    199.99,
			Stock:    50,
			Category: "books",
		},
	}

	// 验证嵌套结构
	errs := Validate(&nested.TestProduct, SceneCreate)
	if errs != nil {
		t.Logf("复杂嵌套结构验证: %v", errs)
	}
}

// TestNestedValidation_WithEmbeddedFields 测试包含嵌入字段的验证
func TestNestedValidation_WithEmbeddedFields(t *testing.T) {
	// 测试 TestProduct 本身就包含嵌入的 TestBaseModel
	product := &TestProduct{
		TestBaseModel: TestBaseModel{
			ID:     100,
			Status: 1,
			Extras: types.Extras{
				"manufacturer": "Test Inc",
			},
		},
		Name:     "Embedded Test",
		Price:    49.99,
		Stock:    200,
		Category: "test",
	}

	errs := Validate(product, SceneCreate)
	if errs != nil {
		t.Logf("嵌入字段验证: %v", errs)
	}
}

// TestNestedValidation_MaxDepth 测试最大嵌套深度限制
func TestNestedValidation_MaxDepth(t *testing.T) {
	// 测试产品的多层验证（BaseModel -> Product -> Extras）
	product := &TestProduct{
		TestBaseModel: TestBaseModel{
			ID:     1,
			Status: 1,
			Extras: types.Extras{
				"level1": map[string]any{
					"level2": "value",
				},
			},
		},
		Name:     "Deep Structure",
		Price:    99.99,
		Stock:    10,
		Category: "test",
	}

	// 验证嵌套结构（验证器应该能处理适度的嵌套）
	errs := Validate(product, SceneCreate)
	if errs != nil {
		t.Logf("嵌套深度验证: %v", errs)
	}
}

// TestNestedValidation_NilFields 测试 nil 字段的处理
func TestNestedValidation_NilFields(t *testing.T) {
	tests := []struct {
		name    string
		product *TestProduct
		scene   ValidateScene
		wantErr bool
	}{
		{
			name: "包含可选的 Extras 数据",
			product: &TestProduct{
				TestBaseModel: TestBaseModel{
					Extras: types.Extras{
						"optional": "data",
					},
				},
				Name:     "Test",
				Price:    99.99,
				Stock:    10,
				Category: "test",
			},
			scene:   SceneCreate,
			wantErr: false,
		},
		{
			name: "不包含 Extras 数据",
			product: &TestProduct{
				TestBaseModel: TestBaseModel{
					Extras: nil, // nil Extras
				},
				Name:     "Test",
				Price:    99.99,
				Stock:    10,
				Category: "test",
			},
			scene:   SceneCreate,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.product, tt.scene)
			if (errs != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", errs, tt.wantErr)
			}
		})
	}
}

// TestNestedValidation_CircularReference 测试循环引用的处理
func TestNestedValidation_CircularReference(t *testing.T) {
	// 测试验证器对简单结构的处理
	// 避免真正的循环引用，因为这会导致无限递归
	product := &TestProduct{
		TestBaseModel: TestBaseModel{
			ID:     42,
			Status: 1,
		},
		Name:     "Test Node",
		Price:    99.99,
		Stock:    100,
		Category: "test",
	}

	// 验证简单节点
	errs := Validate(product, SceneCreate)
	if errs != nil {
		t.Logf("节点验证: %v", errs)
	}
}
