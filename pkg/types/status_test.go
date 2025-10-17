package types

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

// TestStatusConstants 测试状态常量定义
func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected int64
	}{
		{"无状态", StatusNone, 0},
		{"系统删除", StatusSysDeleted, 1},
		{"管理员删除", StatusAdmDeleted, 2},
		{"用户删除", StatusUserDeleted, 4},
		{"系统禁用", StatusSysDisabled, 8},
		{"管理员禁用", StatusAdmDisabled, 16},
		{"用户禁用", StatusUserDisabled, 32},
		{"系统隐藏", StatusSysHidden, 64},
		{"管理员隐藏", StatusAdmHidden, 128},
		{"用户隐藏", StatusUserHidden, 256},
		{"系统未验证", StatusSysUnverified, 512},
		{"管理员未验证", StatusAdmUnverified, 1024},
		{"用户未验证", StatusUserUnverified, 2048},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int64(tt.status) != tt.expected {
				t.Errorf("状态常量 %s 值错误: 期望 %d, 实际 %d", tt.name, tt.expected, int64(tt.status))
			}
		})
	}
}

// TestStatusCombinedConstants 测试预定义的组合常量
func TestStatusCombinedConstants(t *testing.T) {
	tests := []struct {
		name     string
		combined Status
		includes []Status
	}{
		{
			"所有删除状态",
			StatusAllDeleted,
			[]Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted},
		},
		{
			"所有禁用状态",
			StatusAllDisabled,
			[]Status{StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled},
		},
		{
			"所有隐藏状态",
			StatusAllHidden,
			[]Status{StatusSysHidden, StatusAdmHidden, StatusUserHidden},
		},
		{
			"所有未验证状态",
			StatusAllUnverified,
			[]Status{StatusSysUnverified, StatusAdmUnverified, StatusUserUnverified},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, include := range tt.includes {
				if !tt.combined.Contain(include) {
					t.Errorf("%s 应该包含 %d", tt.name, include)
				}
			}
		})
	}
}

// TestStatusIsValid 测试状态值验证
func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		valid  bool
	}{
		{"零值有效", StatusNone, true},
		{"正常状态有效", StatusUserDisabled, true},
		{"组合状态有效", StatusUserDisabled | StatusSysHidden, true},
		{"负数无效", Status(-1), false},
		{"最大值有效", MaxStatus, true},
		{"超出最大值无效", MaxStatus + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, 期望 %v", got, tt.valid)
			}
		})
	}
}

// TestStatusSet 测试设置状态位
func TestStatusSet(t *testing.T) {
	var s Status
	s.Set(StatusUserDisabled)
	if !s.Contain(StatusUserDisabled) {
		t.Error("设置状态失败")
	}

	s.Set(StatusSysHidden)
	if !s.Contain(StatusUserDisabled) || !s.Contain(StatusSysHidden) {
		t.Error("追加状态后原有状态丢失")
	}
}

// TestStatusUnset 测试取消状态位
func TestStatusUnset(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
	s.Unset(StatusUserDisabled)

	if s.Contain(StatusUserDisabled) {
		t.Error("取消状态失败")
	}

	if !s.Contain(StatusSysHidden) || !s.Contain(StatusAdmDeleted) {
		t.Error("取消状态时误删了其他状态")
	}
}

// TestStatusToggle 测试切换状态位
func TestStatusToggle(t *testing.T) {
	var s Status

	// 第一次切换：添加
	s.Toggle(StatusUserDisabled)
	if !s.Contain(StatusUserDisabled) {
		t.Error("切换状态失败：未添加")
	}

	// 第二次切换：移除
	s.Toggle(StatusUserDisabled)
	if s.Contain(StatusUserDisabled) {
		t.Error("切换状态失败：未移除")
	}
}

// TestStatusMerge 测试状态合并
func TestStatusMerge(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
	s.Merge(StatusUserDisabled | StatusAdmDeleted)

	if !s.Contain(StatusUserDisabled) || !s.Contain(StatusAdmDeleted) {
		t.Error("合并后应保留的状态丢失")
	}

	if s.Contain(StatusSysHidden) {
		t.Error("合并后不应保留的状态仍存在")
	}
}

// TestStatusContain 测试状态包含检查
func TestStatusContain(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden

	tests := []struct {
		name   string
		check  Status
		expect bool
	}{
		{"包含单个状态", StatusUserDisabled, true},
		{"包含全部状态", StatusUserDisabled | StatusSysHidden, true},
		{"不包含的状态", StatusAdmDeleted, false},
		{"部分包含", StatusUserDisabled | StatusAdmDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Contain(tt.check); got != tt.expect {
				t.Errorf("Contain() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusHasAny 测试任意状态检查
func TestStatusHasAny(t *testing.T) {
	s := StatusUserDisabled

	tests := []struct {
		name   string
		flags  []Status
		expect bool
	}{
		{"包含一个", []Status{StatusUserDisabled, StatusAdmDisabled}, true},
		{"都不包含", []Status{StatusSysDeleted, StatusAdmDeleted}, false},
		{"空切片", []Status{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.HasAny(tt.flags...); got != tt.expect {
				t.Errorf("HasAny() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusHasAll 测试全部状态检查
func TestStatusHasAll(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden

	tests := []struct {
		name   string
		flags  []Status
		expect bool
	}{
		{"都包含", []Status{StatusUserDisabled, StatusSysHidden}, true},
		{"缺少一个", []Status{StatusUserDisabled, StatusAdmDeleted}, false},
		{"空切片", []Status{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.HasAll(tt.flags...); got != tt.expect {
				t.Errorf("HasAll() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusClear 测试清除状态
func TestStatusClear(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	s.Clear()

	if !s.Equal(StatusNone) {
		t.Errorf("清除后状态应为 StatusNone，实际为 %d", s)
	}
}

// TestStatusEqual 测试状态相等
func TestStatusEqual(t *testing.T) {
	s1 := StatusUserDisabled | StatusSysHidden
	s2 := StatusUserDisabled | StatusSysHidden
	s3 := StatusUserDisabled

	if !s1.Equal(s2) {
		t.Error("相同的状态应该相等")
	}

	if s1.Equal(s3) {
		t.Error("不同的状态不应该相等")
	}
}

// TestStatusSetMultiple 测试批量设置
func TestStatusSetMultiple(t *testing.T) {
	var s Status
	s.SetMultiple(StatusUserDisabled, StatusSysHidden, StatusAdmUnverified)

	if !s.HasAll(StatusUserDisabled, StatusSysHidden, StatusAdmUnverified) {
		t.Error("批量设置失败")
	}
}

// TestStatusUnsetMultiple 测试批量取消
func TestStatusUnsetMultiple(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
	s.UnsetMultiple(StatusUserDisabled, StatusSysHidden)

	if s.HasAny(StatusUserDisabled, StatusSysHidden) {
		t.Error("批量取消失败")
	}

	if !s.Contain(StatusAdmDeleted) {
		t.Error("批量取消时误删了其他状态")
	}
}

// TestStatusIsDeleted 测试删除状态检查
func TestStatusIsDeleted(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect bool
	}{
		{"系统删除", StatusSysDeleted, true},
		{"管理员删除", StatusAdmDeleted, true},
		{"用户删除", StatusUserDeleted, true},
		{"未删除", StatusUserDisabled, false},
		{"零值", StatusNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsDeleted(); got != tt.expect {
				t.Errorf("IsDeleted() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusIsDisable 测试禁用状态检查
func TestStatusIsDisable(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect bool
	}{
		{"系统禁用", StatusSysDisabled, true},
		{"管理员禁用", StatusAdmDisabled, true},
		{"用户禁用", StatusUserDisabled, true},
		{"未禁用", StatusSysDeleted, false},
		{"零值", StatusNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsDisable(); got != tt.expect {
				t.Errorf("IsDisable() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusIsHidden 测试隐藏状态检查
func TestStatusIsHidden(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect bool
	}{
		{"系统隐藏", StatusSysHidden, true},
		{"管理员隐藏", StatusAdmHidden, true},
		{"用户隐藏", StatusUserHidden, true},
		{"未隐藏", StatusUserDisabled, false},
		{"零值", StatusNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsHidden(); got != tt.expect {
				t.Errorf("IsHidden() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusIsUnverified 测试未验证状态检查
func TestStatusIsUnverified(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect bool
	}{
		{"系统未验证", StatusSysUnverified, true},
		{"管理员未验证", StatusAdmUnverified, true},
		{"用户未验证", StatusUserUnverified, true},
		{"已验证", StatusUserDisabled, false},
		{"零值", StatusNone, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsUnverified(); got != tt.expect {
				t.Errorf("IsUnverified() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusCanEnable 测试可启用状态检查
func TestStatusCanEnable(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect bool
	}{
		{"正常状态", StatusNone, true},
		{"已删除不可启用", StatusSysDeleted, false},
		{"已禁用不可启用", StatusUserDisabled, false},
		{"仅隐藏可启用", StatusSysHidden, true},
		{"仅未验证可启用", StatusUserUnverified, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.CanEnable(); got != tt.expect {
				t.Errorf("CanEnable() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusCanVisible 测试可见状态检查
func TestStatusCanVisible(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect bool
	}{
		{"正常状态", StatusNone, true},
		{"已删除不可见", StatusSysDeleted, false},
		{"已禁用不可见", StatusUserDisabled, false},
		{"已隐藏不可见", StatusSysHidden, false},
		{"仅未验证可见", StatusUserUnverified, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.CanVisible(); got != tt.expect {
				t.Errorf("CanVisible() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusCanVerified 测试已验证状态检查
func TestStatusCanVerified(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect bool
	}{
		{"正常状态", StatusNone, true},
		{"已删除未验证", StatusSysDeleted, false},
		{"已禁用未验证", StatusUserDisabled, false},
		{"已隐藏未验证", StatusSysHidden, false},
		{"未验证", StatusUserUnverified, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.CanVerified(); got != tt.expect {
				t.Errorf("CanVerified() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusValue 测试数据库 Value 方法
func TestStatusValue(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	val, err := s.Value()

	if err != nil {
		t.Errorf("Value() 返回错误: %v", err)
	}

	if val != int64(s) {
		t.Errorf("Value() = %v, 期望 %d", val, int64(s))
	}
}

// TestStatusScan 测试数据库 Scan 方法
func TestStatusScan(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expect    Status
		expectErr bool
	}{
		{"int64 类型", int64(100), Status(100), false},
		{"int 类型", int(100), Status(100), false},
		{"uint64 类型", uint64(100), Status(100), false},
		{"[]byte 类型", []byte("100"), Status(100), false},
		{"nil 值", nil, StatusNone, false},
		{"负数 int64", int64(-1), StatusNone, true},
		{"负数 int", int(-1), StatusNone, true},
		{"溢出 uint64", uint64(1) << 63, StatusNone, true},
		{"无效类型", "string", StatusNone, true},
		{"无效 JSON", []byte("invalid"), StatusNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Status
			err := s.Scan(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("期望返回错误，但没有")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if s != tt.expect {
					t.Errorf("Scan() = %d, 期望 %d", s, tt.expect)
				}
			}
		})
	}
}

// TestStatusMarshalJSON 测试 JSON 序列化
func TestStatusMarshalJSON(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	data, err := json.Marshal(s)

	if err != nil {
		t.Errorf("MarshalJSON() 错误: %v", err)
	}

	var num int64
	if err := json.Unmarshal(data, &num); err != nil {
		t.Errorf("反序列化失败: %v", err)
	}

	if num != int64(s) {
		t.Errorf("序列化值 = %d, 期望 %d", num, int64(s))
	}
}

// TestStatusUnmarshalJSON 测试 JSON 反序列化
func TestStatusUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expect    Status
		expectErr bool
	}{
		{"正常数字", `100`, Status(100), false},
		{"零值", `0`, StatusNone, false},
		{"负数", `-1`, StatusNone, true},
		{"无效 JSON", `"invalid"`, StatusNone, true},
		{"空字符串", ``, StatusNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Status
			err := json.Unmarshal([]byte(tt.input), &s)

			if tt.expectErr {
				if err == nil {
					t.Error("期望返回错误，但没有")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if s != tt.expect {
					t.Errorf("UnmarshalJSON() = %d, 期望 %d", s, tt.expect)
				}
			}
		})
	}
}

// TestStatusString 测试字符串输出
func TestStatusString(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect string
	}{
		{"零值", StatusNone, "Status(None)"},
		{"正常值", Status(100), "Status(100)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expect {
				t.Errorf("String() = %v, 期望 %v", got, tt.expect)
			}
		})
	}
}

// TestStatusDriverInterfaces 测试数据库接口实现
func TestStatusDriverInterfaces(t *testing.T) {
	var _ driver.Valuer = (*Status)(nil)
	// Note: sql.Scanner 需要指针接收者
}

// TestStatusJSONInterfaces 测试 JSON 接口实现
func TestStatusJSONInterfaces(t *testing.T) {
	var _ json.Marshaler = (*Status)(nil)
	// Note: json.Unmarshaler 需要指针接收者
}

// BenchmarkStatusSet 基准测试：设置状态
func BenchmarkStatusSet(b *testing.B) {
	var s Status
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(StatusUserDisabled)
	}
}

// BenchmarkStatusHasAny 基准测试：检查任意状态
func BenchmarkStatusHasAny(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HasAny(StatusUserDisabled, StatusAdmDisabled)
	}
}

// BenchmarkStatusHasAll 基准测试：检查全部状态
func BenchmarkStatusHasAll(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HasAll(StatusUserDisabled, StatusSysHidden)
	}
}

// BenchmarkStatusIsDeleted 基准测试：删除状态检查
func BenchmarkStatusIsDeleted(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.IsDeleted()
	}
}

// BenchmarkStatusMarshalJSON 基准测试：JSON 序列化
func BenchmarkStatusMarshalJSON(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(s)
	}
}

// BenchmarkStatusUnmarshalJSON 基准测试：JSON 反序列化
func BenchmarkStatusUnmarshalJSON(b *testing.B) {
	data := []byte("100")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s Status
		_ = json.Unmarshal(data, &s)
	}
}
