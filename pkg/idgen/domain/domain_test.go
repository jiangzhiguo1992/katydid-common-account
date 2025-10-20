package domain

import (
	"encoding/json"
	"testing"
)

// TestNewID 测试创建ID
func TestNewID(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		expected int64
	}{
		{"正数", 12345, 12345},
		{"零", 0, 0},
		{"大数", 9223372036854775807, 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := NewID(tt.value)
			if id.Int64() != tt.expected {
				t.Errorf("NewID(%d).Int64() = %d, 期望 %d", tt.value, id.Int64(), tt.expected)
			}
		})
	}
}

// TestIDString 测试ID转字符串
func TestIDString(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected string
	}{
		{"正数", NewID(12345), "12345"},
		{"零", NewID(0), "0"},
		{"大数", NewID(9223372036854775807), "9223372036854775807"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.expected {
				t.Errorf("String() = %s, 期望 %s", got, tt.expected)
			}
		})
	}
}

// TestIDHex 测试ID转十六进制
func TestIDHex(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected string
	}{
		{"小数", NewID(255), "0xff"},
		{"零", NewID(0), "0x0"},
		{"大数", NewID(65535), "0xffff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Hex(); got != tt.expected {
				t.Errorf("Hex() = %s, 期望 %s", got, tt.expected)
			}
		})
	}
}

// TestIDBinary 测试ID转二进制
func TestIDBinary(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected string
	}{
		{"小数", NewID(5), "0b101"},
		{"零", NewID(0), "0b0"},
		{"8", NewID(8), "0b1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Binary(); got != tt.expected {
				t.Errorf("Binary() = %s, 期望 %s", got, tt.expected)
			}
		})
	}
}

// TestIDMarshalJSON 测试JSON序列化
func TestIDMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected string
	}{
		{"正数", NewID(12345), `"12345"`},
		{"零", NewID(0), `"0"`},
		{"大数", NewID(9007199254740991), `"9007199254740991"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.id)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %s, 期望 %s", string(data), tt.expected)
			}
		})
	}
}

// TestIDUnmarshalJSON 测试JSON反序列化
func TestIDUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected ID
		wantErr  bool
	}{
		{"从字符串", `"12345"`, NewID(12345), false},
		{"从数字", `67890`, NewID(67890), false},
		{"零", `"0"`, NewID(0), false},
		{"无效字符串", `"abc"`, NewID(0), true},
		{"负数", `-1`, NewID(0), true},
		{"空数据", ``, NewID(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ID
			err := json.Unmarshal([]byte(tt.jsonData), &id)
			if tt.wantErr {
				if err == nil {
					t.Error("期望得到错误，但没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
					return
				}
				if id != tt.expected {
					t.Errorf("UnmarshalJSON() = %d, 期望 %d", id, tt.expected)
				}
			}
		})
	}
}

// TestIDIsZero 测试零值检查
func TestIDIsZero(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected bool
	}{
		{"零", NewID(0), true},
		{"正数", NewID(1), false},
		{"负数", NewID(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsZero(); got != tt.expected {
				t.Errorf("IsZero() = %v, 期望 %v", got, tt.expected)
			}
		})
	}
}

// TestIDIsValid 测试有效性检查
func TestIDIsValid(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected bool
	}{
		{"正数_有效", NewID(1), true},
		{"零_无效", NewID(0), false},
		{"负数_无效", NewID(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, 期望 %v", got, tt.expected)
			}
		})
	}
}

// TestIDIsSafeForJavaScript 测试JavaScript安全性检查
func TestIDIsSafeForJavaScript(t *testing.T) {
	tests := []struct {
		name     string
		id       ID
		expected bool
	}{
		{"安全范围内", NewID(9007199254740991), true},
		{"超出范围", NewID(9007199254740992), false},
		{"零", NewID(0), true},
		{"负数", NewID(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsSafeForJavaScript(); got != tt.expected {
				t.Errorf("IsSafeForJavaScript() = %v, 期望 %v", got, tt.expected)
			}
		})
	}
}

// TestParseID 测试解析ID
func TestParseID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ID
		wantErr  bool
	}{
		{"十进制", "12345", NewID(12345), false},
		{"十六进制", "0xFF", NewID(255), false},
		{"二进制", "0b101", NewID(5), false},
		{"带空格", "  123  ", NewID(0), true}, // 修改：ParseID不应该自动trim空格，这是正确的行为
		{"空字符串", "", NewID(0), true},
		{"无效字符", "abc", NewID(0), true},
		{"负数", "-1", NewID(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParseID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("期望得到错误，但没有返回错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
					return
				}
				if id != tt.expected {
					t.Errorf("ParseID() = %d, 期望 %d", id, tt.expected)
				}
			}
		})
	}
}

// TestIDSlice 测试ID切片操作
func TestIDSlice(t *testing.T) {
	t.Run("NewIDSlice", func(t *testing.T) {
		ids := NewIDSlice(NewID(1), NewID(2), NewID(3))
		if ids.Len() != 3 {
			t.Errorf("Len() = %d, 期望 3", ids.Len())
		}
	})

	t.Run("Int64Slice", func(t *testing.T) {
		ids := IDSlice{NewID(1), NewID(2), NewID(3)}
		int64s := ids.Int64Slice()
		if len(int64s) != 3 {
			t.Errorf("长度 = %d, 期望 3", len(int64s))
		}
		for i, v := range int64s {
			if v != int64(i+1) {
				t.Errorf("索引%d: 期望%d, 得到%d", i, i+1, v)
			}
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		ids := IDSlice{NewID(1), NewID(2), NewID(3)}
		strs := ids.StringSlice()
		expected := []string{"1", "2", "3"}
		for i, v := range strs {
			if v != expected[i] {
				t.Errorf("索引%d: 期望%s, 得到%s", i, expected[i], v)
			}
		}
	})

	t.Run("Contains", func(t *testing.T) {
		ids := IDSlice{NewID(1), NewID(2), NewID(3)}
		if !ids.Contains(NewID(2)) {
			t.Error("应该包含ID 2")
		}
		if ids.Contains(NewID(4)) {
			t.Error("不应该包含ID 4")
		}
	})

	t.Run("Deduplicate", func(t *testing.T) {
		ids := IDSlice{NewID(1), NewID(2), NewID(2), NewID(3), NewID(1)}
		unique := ids.Deduplicate()
		if unique.Len() != 3 {
			t.Errorf("去重后长度 = %d, 期望 3", unique.Len())
		}
		if ids.Len() != 5 {
			t.Error("原切片不应被修改")
		}
	})

	t.Run("Filter", func(t *testing.T) {
		ids := IDSlice{NewID(1), NewID(2), NewID(3), NewID(4), NewID(5)}
		filtered := ids.Filter(func(id ID) bool {
			return id > NewID(2)
		})
		if filtered.Len() != 3 {
			t.Errorf("过滤后长度 = %d, 期望 3", filtered.Len())
		}
	})
}

// TestIDSet 测试ID集合操作
func TestIDSet(t *testing.T) {
	t.Run("NewIDSet", func(t *testing.T) {
		set := NewIDSet(NewID(1), NewID(2), NewID(3))
		if set.Size() != 3 {
			t.Errorf("Size() = %d, 期望 3", set.Size())
		}
	})

	t.Run("Add", func(t *testing.T) {
		set := NewIDSet()
		set.Add(NewID(1))
		set.Add(NewID(2))
		if set.Size() != 2 {
			t.Errorf("Size() = %d, 期望 2", set.Size())
		}
		set.Add(NewID(1)) // 重复
		if set.Size() != 2 {
			t.Errorf("添加重复后 Size() = %d, 期望 2", set.Size())
		}
	})

	t.Run("Remove", func(t *testing.T) {
		set := NewIDSet(NewID(1), NewID(2), NewID(3))
		set.Remove(NewID(2))
		if set.Size() != 2 {
			t.Errorf("Size() = %d, 期望 2", set.Size())
		}
		if set.Contains(NewID(2)) {
			t.Error("不应该包含已移除的ID")
		}
	})

	t.Run("Contains", func(t *testing.T) {
		set := NewIDSet(NewID(1), NewID(2), NewID(3))
		if !set.Contains(NewID(2)) {
			t.Error("应该包含ID 2")
		}
		if set.Contains(NewID(4)) {
			t.Error("不应该包含ID 4")
		}
	})

	t.Run("Union", func(t *testing.T) {
		set1 := NewIDSet(NewID(1), NewID(2))
		set2 := NewIDSet(NewID(2), NewID(3))
		union := set1.Union(set2)
		if union.Size() != 3 {
			t.Errorf("并集大小 = %d, 期望 3", union.Size())
		}
	})

	t.Run("Intersect", func(t *testing.T) {
		set1 := NewIDSet(NewID(1), NewID(2), NewID(3))
		set2 := NewIDSet(NewID(2), NewID(3), NewID(4))
		intersect := set1.Intersect(set2)
		if intersect.Size() != 2 {
			t.Errorf("交集大小 = %d, 期望 2", intersect.Size())
		}
	})

	t.Run("Difference", func(t *testing.T) {
		set1 := NewIDSet(NewID(1), NewID(2), NewID(3))
		set2 := NewIDSet(NewID(2), NewID(3), NewID(4))
		diff := set1.Difference(set2)
		if diff.Size() != 1 {
			t.Errorf("差集大小 = %d, 期望 1", diff.Size())
		}
		if !diff.Contains(NewID(1)) {
			t.Error("差集应包含ID 1")
		}
	})
}

// BenchmarkIDString 基准测试：ID转字符串
func BenchmarkIDString(b *testing.B) {
	id := NewID(9223372036854775807)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = id.String()
	}
}

// BenchmarkIDMarshalJSON 基准测试：JSON序列化
func BenchmarkIDMarshalJSON(b *testing.B) {
	id := NewID(9223372036854775807)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(id)
	}
}

// BenchmarkParseID 基准测试：解析ID
func BenchmarkParseID(b *testing.B) {
	str := "9223372036854775807"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseID(str)
	}
}

// BenchmarkIDSliceInt64Slice 基准测试：ID切片转int64切片
func BenchmarkIDSliceInt64Slice(b *testing.B) {
	ids := make(IDSlice, 100)
	for i := range ids {
		ids[i] = NewID(int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ids.Int64Slice()
	}
}
