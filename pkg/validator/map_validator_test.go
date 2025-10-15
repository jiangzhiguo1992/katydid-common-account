package validator

import (
	"fmt"
	"testing"
)

// TestValidateMap 测试基础的 map 验证功能
func TestValidateMap(t *testing.T) {
	tests := []struct {
		name      string
		extras    map[string]any
		validator *MapValidator
		wantErr   bool
		errMsg    string
	}{
		{
			name: "所有验证都通过",
			extras: map[string]any{
				"required_key": "value",
				"optional_key": "value",
			},
			validator: &MapValidator{
				RequiredKeys: []string{"required_key"},
				AllowedKeys:  []string{"required_key", "optional_key"},
			},
			wantErr: false,
		},
		{
			name: "缺少必填键",
			extras: map[string]any{
				"optional_key": "value",
			},
			validator: &MapValidator{
				RequiredKeys: []string{"required_key"},
			},
			wantErr: true,
		},
		{
			name: "包含不允许的键",
			extras: map[string]any{
				"required_key": "value",
				"invalid_key":  "value",
			},
			validator: &MapValidator{
				RequiredKeys: []string{"required_key"},
				AllowedKeys:  []string{"required_key", "optional_key"},
			},
			wantErr: true,
		},
		{
			name: "自定义键验证失败",
			extras: map[string]any{
				"name": "ab", // 太短
			},
			validator: &MapValidator{
				KeyValidators: map[string]func(value interface{}) error{
					"name": func(value interface{}) error {
						str, ok := value.(string)
						if !ok || len(str) < 3 {
							return fmt.Errorf("name 长度必须至少3个字符")
						}
						return nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "自定义键验证通过",
			extras: map[string]any{
				"name": "alice",
			},
			validator: &MapValidator{
				KeyValidators: map[string]func(value interface{}) error{
					"name": func(value interface{}) error {
						str, ok := value.(string)
						if !ok || len(str) < 3 {
							return fmt.Errorf("name 长度必须至少3个字符")
						}
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "空验证器不报错",
			extras:    map[string]any{"any_key": "any_value"},
			validator: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMap(tt.extras, tt.validator)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMap() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapMustHaveKeys 测试必填键验证
func TestValidateMapMustHaveKeys(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		keys    []string
		wantErr bool
	}{
		{
			name: "包含所有必填键",
			extras: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			keys:    []string{"key1", "key2"},
			wantErr: false,
		},
		{
			name: "缺少一个必填键",
			extras: map[string]any{
				"key1": "value1",
			},
			keys:    []string{"key1", "key2"},
			wantErr: true,
		},
		{
			name:    "缺少所有必填键",
			extras:  map[string]any{},
			keys:    []string{"key1", "key2"},
			wantErr: true,
		},
		{
			name: "空必填键列表",
			extras: map[string]any{
				"key1": "value1",
			},
			keys:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapMustHaveKeys(tt.extras, tt.keys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapMustHaveKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapStringKey 测试字符串类型键验证
func TestValidateMapStringKey(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		key     string
		minLen  int
		maxLen  int
		wantErr bool
	}{
		{
			name:    "字符串长度在范围内",
			extras:  map[string]any{"name": "alice"},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: false,
		},
		{
			name:    "字符串太短",
			extras:  map[string]any{"name": "ab"},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: true,
		},
		{
			name:    "字符串太长",
			extras:  map[string]any{"name": "verylongname"},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: true,
		},
		{
			name:    "不是字符串类型",
			extras:  map[string]any{"name": 123},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: true,
		},
		{
			name:    "键不存在不报错",
			extras:  map[string]any{},
			key:     "name",
			minLen:  3,
			maxLen:  10,
			wantErr: false,
		},
		{
			name:    "只验证最小长度",
			extras:  map[string]any{"name": "alice"},
			key:     "name",
			minLen:  3,
			maxLen:  0,
			wantErr: false,
		},
		{
			name:    "只验证最大长度",
			extras:  map[string]any{"name": "alice"},
			key:     "name",
			minLen:  0,
			maxLen:  10,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapStringKey(tt.extras, tt.key, tt.minLen, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapStringKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapIntKey 测试整数类型键验证
func TestValidateMapIntKey(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		key     string
		min     int
		max     int
		wantErr bool
	}{
		{
			name:    "整数在范围内 - int类型",
			extras:  map[string]any{"age": 25},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: false,
		},
		{
			name:    "整数在范围内 - int64类型",
			extras:  map[string]any{"age": int64(25)},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: false,
		},
		{
			name:    "整数在范围内 - float64类型",
			extras:  map[string]any{"age": float64(25)},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: false,
		},
		{
			name:    "整数小于最小值",
			extras:  map[string]any{"age": -5},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: true,
		},
		{
			name:    "整数大于最大值",
			extras:  map[string]any{"age": 150},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: true,
		},
		{
			name:    "不是数字类型",
			extras:  map[string]any{"age": "twenty"},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: true,
		},
		{
			name:    "键不存在不报错",
			extras:  map[string]any{},
			key:     "age",
			min:     0,
			max:     120,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapIntKey(tt.extras, tt.key, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapIntKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapFloatKey 测试浮点数类型键验证
func TestValidateMapFloatKey(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		key     string
		min     float64
		max     float64
		wantErr bool
	}{
		{
			name:    "浮点数在范围内",
			extras:  map[string]any{"price": 99.99},
			key:     "price",
			min:     0.0,
			max:     999.99,
			wantErr: false,
		},
		{
			name:    "整数可以作为浮点数验证",
			extras:  map[string]any{"price": 100},
			key:     "price",
			min:     0.0,
			max:     999.99,
			wantErr: false,
		},
		{
			name:    "浮点数小于最小值",
			extras:  map[string]any{"price": -10.0},
			key:     "price",
			min:     0.0,
			max:     999.99,
			wantErr: true,
		},
		{
			name:    "浮点数大于最大值",
			extras:  map[string]any{"price": 1000.0},
			key:     "price",
			min:     0.0,
			max:     999.99,
			wantErr: true,
		},
		{
			name:    "不是数字类型",
			extras:  map[string]any{"price": "expensive"},
			key:     "price",
			min:     0.0,
			max:     999.99,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapFloatKey(tt.extras, tt.key, tt.min, tt.max)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapFloatKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapBoolKey 测试布尔类型键验证
func TestValidateMapBoolKey(t *testing.T) {
	tests := []struct {
		name    string
		extras  map[string]any
		key     string
		wantErr bool
	}{
		{
			name:    "布尔值 true",
			extras:  map[string]any{"enabled": true},
			key:     "enabled",
			wantErr: false,
		},
		{
			name:    "布尔值 false",
			extras:  map[string]any{"enabled": false},
			key:     "enabled",
			wantErr: false,
		},
		{
			name:    "不是布尔类型",
			extras:  map[string]any{"enabled": "yes"},
			key:     "enabled",
			wantErr: true,
		},
		{
			name:    "键不存在不报错",
			extras:  map[string]any{},
			key:     "enabled",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapBoolKey(tt.extras, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapBoolKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestValidateMapKey 测试自定义键验证
func TestValidateMapKey(t *testing.T) {
	tests := []struct {
		name          string
		extras        map[string]any
		key           string
		validatorFunc func(value interface{}) error
		wantErr       bool
	}{
		{
			name:   "自定义验证通过",
			extras: map[string]any{"email": "test@example.com"},
			key:    "email",
			validatorFunc: func(value interface{}) error {
				email, ok := value.(string)
				if !ok || len(email) == 0 {
					return fmt.Errorf("email 不能为空")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:   "自定义验证失败",
			extras: map[string]any{"email": ""},
			key:    "email",
			validatorFunc: func(value interface{}) error {
				email, ok := value.(string)
				if !ok || len(email) == 0 {
					return fmt.Errorf("email 不能为空")
				}
				return nil
			},
			wantErr: true,
		},
		{
			name:   "键不存在不报错",
			extras: map[string]any{},
			key:    "email",
			validatorFunc: func(value interface{}) error {
				return fmt.Errorf("不应该执行到这里")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMapKey(tt.extras, tt.key, tt.validatorFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMapKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				t.Logf("错误信息: %v", err)
			}
		})
	}
}

// TestNewMapValidator 测试验证器构造器
func TestNewMapValidator(t *testing.T) {
	t.Run("创建空验证器", func(t *testing.T) {
		v := NewMapValidator()
		if v == nil {
			t.Error("NewMapValidator() should not return nil")
		}
		if v.KeyValidators == nil {
			t.Error("KeyValidators map should be initialized")
		}
	})

	t.Run("链式调用构建验证器", func(t *testing.T) {
		v := NewMapValidator().
			WithRequiredKeys("name", "email").
			WithAllowedKeys("name", "email", "phone").
			WithKeyValidator("email", func(value interface{}) error {
				email, ok := value.(string)
				if !ok || len(email) == 0 {
					return fmt.Errorf("invalid email")
				}
				return nil
			})

		if len(v.RequiredKeys) != 2 {
			t.Errorf("Expected 2 required keys, got %d", len(v.RequiredKeys))
		}
		if len(v.AllowedKeys) != 3 {
			t.Errorf("Expected 3 allowed keys, got %d", len(v.AllowedKeys))
		}
		if len(v.KeyValidators) != 1 {
			t.Errorf("Expected 1 key validator, got %d", len(v.KeyValidators))
		}
	})

	t.Run("AddRequiredKey 方法", func(t *testing.T) {
		v := NewMapValidator().
			AddRequiredKey("key1").
			AddRequiredKey("key2")

		if len(v.RequiredKeys) != 2 {
			t.Errorf("Expected 2 required keys, got %d", len(v.RequiredKeys))
		}
	})

	t.Run("AddAllowedKey 方法", func(t *testing.T) {
		v := NewMapValidator().
			AddAllowedKey("key1").
			AddAllowedKey("key2")

		if len(v.AllowedKeys) != 2 {
			t.Errorf("Expected 2 allowed keys, got %d", len(v.AllowedKeys))
		}
	})
}

// TestMapValidator_ComplexScenario 测试复杂场景
func TestMapValidator_ComplexScenario(t *testing.T) {
	t.Run("电子产品验证场景", func(t *testing.T) {
		extras := map[string]any{
			"brand":    "Apple",
			"model":    "iPhone 15",
			"warranty": 12,
			"price":    999.99,
		}

		validator := NewMapValidator().
			WithRequiredKeys("brand", "warranty").
			WithAllowedKeys("brand", "model", "warranty", "price").
			WithKeyValidator("brand", func(value interface{}) error {
				brand, ok := value.(string)
				if !ok || len(brand) < 2 {
					return fmt.Errorf("品牌名称至少2个字符")
				}
				return nil
			}).
			WithKeyValidator("warranty", func(value interface{}) error {
				warranty, ok := value.(int)
				if !ok || warranty < 1 || warranty > 60 {
					return fmt.Errorf("保修期必须在1-60个月之间")
				}
				return nil
			})

		err := ValidateMap(extras, validator)
		if err != nil {
			t.Errorf("验证失败: %v", err)
		}
	})

	t.Run("用户资料验证场景", func(t *testing.T) {
		extras := map[string]any{
			"twitter": "https://twitter.com/example",
			"github":  "https://github.com/example",
		}

		validator := &MapValidator{
			AllowedKeys: []string{"twitter", "github", "linkedin", "website"},
			KeyValidators: map[string]func(value interface{}) error{
				"twitter": func(value interface{}) error {
					url, ok := value.(string)
					if !ok || len(url) == 0 || len(url) > 200 {
						return fmt.Errorf("twitter URL 长度必须在1-200之间")
					}
					return nil
				},
				"github": func(value interface{}) error {
					url, ok := value.(string)
					if !ok || len(url) == 0 || len(url) > 200 {
						return fmt.Errorf("github URL 长度必须在1-200之间")
					}
					return nil
				},
			},
		}

		err := ValidateMap(extras, validator)
		if err != nil {
			t.Errorf("验证失败: %v", err)
		}
	})

	t.Run("包含无效键应该失败", func(t *testing.T) {
		extras := map[string]any{
			"twitter":  "https://twitter.com/example",
			"facebook": "https://facebook.com/example", // 不允许的键
		}

		validator := &MapValidator{
			AllowedKeys: []string{"twitter", "github", "linkedin"},
		}

		err := ValidateMap(extras, validator)
		if err == nil {
			t.Error("应该返回错误，因为包含不允许的键 facebook")
		}
	})
}
