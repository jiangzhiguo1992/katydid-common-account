package types

import (
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

// TestStatusSetAndUnset 测试设置和取消状态位
func TestStatusSetAndUnset(t *testing.T) {
	var s Status

	// 测试 Set 方法（应该使用 |= 追加状态）
	s.Set(StatusUserDisabled)
	if !s.Contain(StatusUserDisabled) {
		t.Error("Set 后应该包含 StatusUserDisabled")
	}

	// 追加另一个状态，原有状态应该保留
	s.Set(StatusSysHidden)
	if !s.Contain(StatusUserDisabled) {
		t.Error("追加状态后，原有的 StatusUserDisabled 应该保留")
	}
	if !s.Contain(StatusSysHidden) {
		t.Error("追加状态后，应该包含 StatusSysHidden")
	}

	// 测试 Unset 方法
	s.Unset(StatusUserDisabled)
	if s.Contain(StatusUserDisabled) {
		t.Error("Unset 后不应该包含 StatusUserDisabled")
	}
	if !s.Contain(StatusSysHidden) {
		t.Error("Unset 后，其他状态应该保留")
	}
}

// TestStatusToggle 测试状态切换
func TestStatusToggle(t *testing.T) {
	var s Status

	// 首次切换：添加状态
	s.Toggle(StatusUserDisabled)
	if !s.Contain(StatusUserDisabled) {
		t.Error("首次 Toggle 应该添加状态")
	}

	// 再次切换：移除状态
	s.Toggle(StatusUserDisabled)
	if s.Contain(StatusUserDisabled) {
		t.Error("再次 Toggle 应该移除状态")
	}
}

// TestStatusMerge 测试状态过滤
func TestStatusMerge(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
	s.Merge(StatusUserDisabled | StatusAdmDeleted)

	if !s.Contain(StatusUserDisabled) {
		t.Error("Merge 后应该保留 StatusUserDisabled")
	}
	if !s.Contain(StatusAdmDeleted) {
		t.Error("Merge 后应该保留 StatusAdmDeleted")
	}
	if s.Contain(StatusSysHidden) {
		t.Error("Merge 后不应该包含 StatusSysHidden")
	}
}

// TestStatusContain 测试状态包含检查
func TestStatusContain(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden

	tests := []struct {
		name   string
		flag   Status
		expect bool
	}{
		{"包含单个状态", StatusUserDisabled, true},
		{"包含所有状态", StatusUserDisabled | StatusSysHidden, true},
		{"不包含的状态", StatusAdmDeleted, false},
		{"部分包含", StatusUserDisabled | StatusAdmDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Contain(tt.flag); got != tt.expect {
				t.Errorf("Contain(%d) = %v, 期望 %v", tt.flag, got, tt.expect)
			}
		})
	}
}

// TestStatusHasAnyAndHasAll 测试批量状态检查
func TestStatusHasAnyAndHasAll(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden

	// 测试 HasAny
	if !s.HasAny(StatusUserDisabled, StatusAdmDeleted) {
		t.Error("HasAny: 应该返回 true（包含第一个）")
	}
	if s.HasAny(StatusAdmDeleted, StatusSysDeleted) {
		t.Error("HasAny: 应该返回 false（都不包含）")
	}
	if s.HasAny() {
		t.Error("HasAny: 空参数应该返回 false")
	}

	// 测试 HasAll
	if !s.HasAll(StatusUserDisabled, StatusSysHidden) {
		t.Error("HasAll: 应该返回 true（都包含）")
	}
	if s.HasAll(StatusUserDisabled, StatusAdmDeleted) {
		t.Error("HasAll: 应该返回 false（缺少第二个）")
	}
	if !s.HasAll() {
		t.Error("HasAll: 空参数应该返回 true")
	}
}

// TestStatusBatchOperations 测试批量操作
func TestStatusBatchOperations(t *testing.T) {
	// 测试 SetMultiple
	var s Status
	s.SetMultiple(StatusUserDisabled, StatusSysHidden, StatusAdmUnverified)
	if !s.HasAll(StatusUserDisabled, StatusSysHidden, StatusAdmUnverified) {
		t.Error("SetMultiple 后应该包含所有指定的状态")
	}

	// 测试 UnsetMultiple
	s.UnsetMultiple(StatusUserDisabled, StatusSysHidden)
	if s.Contain(StatusUserDisabled) || s.Contain(StatusSysHidden) {
		t.Error("UnsetMultiple 后不应该包含被移除的状态")
	}
	if !s.Contain(StatusAdmUnverified) {
		t.Error("UnsetMultiple 后应该保留其他状态")
	}
}

// TestStatusBusinessLogic 测试业务逻辑方法
func TestStatusBusinessLogic(t *testing.T) {
	tests := []struct {
		name       string
		status     Status
		isDeleted  bool
		isDisable  bool
		isHidden   bool
		canEnable  bool
		canVisible bool
	}{
		{
			"正常状态",
			StatusNone,
			false, false, false, true, true,
		},
		{
			"用户删除",
			StatusUserDeleted,
			true, false, false, false, false,
		},
		{
			"系统禁用",
			StatusSysDisabled,
			false, true, false, false, false,
		},
		{
			"管理员隐藏",
			StatusAdmHidden,
			false, false, true, true, false,
		},
		{
			"删除且禁用",
			StatusUserDeleted | StatusSysDisabled,
			true, true, false, false, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsDeleted(); got != tt.isDeleted {
				t.Errorf("IsDeleted() = %v, 期望 %v", got, tt.isDeleted)
			}
			if got := tt.status.IsDisable(); got != tt.isDisable {
				t.Errorf("IsDisable() = %v, 期望 %v", got, tt.isDisable)
			}
			if got := tt.status.IsHidden(); got != tt.isHidden {
				t.Errorf("IsHidden() = %v, 期望 %v", got, tt.isHidden)
			}
			if got := tt.status.CanEnable(); got != tt.canEnable {
				t.Errorf("CanEnable() = %v, 期望 %v", got, tt.canEnable)
			}
			if got := tt.status.CanVisible(); got != tt.canVisible {
				t.Errorf("CanVisible() = %v, 期望 %v", got, tt.canVisible)
			}
		})
	}
}

// TestStatusDatabaseScan 测试数据库扫描
func TestStatusDatabaseScan(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expected  Status
		shouldErr bool
	}{
		{"int64 类型", int64(5), Status(5), false},
		{"int 类型", int(10), Status(10), false},
		{"uint64 类型", uint64(15), Status(15), false},
		{"[]byte 类型", []byte("20"), Status(20), false},
		{"nil 值", nil, StatusNone, false},
		{"负数错误", int64(-1), StatusNone, true},
		{"uint64 溢出", uint64(MaxStatus) + 1, StatusNone, true},
		{"不支持的类型", "invalid", StatusNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Status
			err := s.Scan(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Error("期望返回错误，但没有错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误，但得到: %v", err)
				}
				if s != tt.expected {
					t.Errorf("Scan() = %d, 期望 %d", s, tt.expected)
				}
			}
		})
	}
}

// TestStatusJSONSerialization 测试 JSON 序列化
func TestStatusJSONSerialization(t *testing.T) {
	original := StatusUserDisabled | StatusSysHidden

	// 序列化
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 反序列化
	var decoded Status
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	// 验证
	if decoded != original {
		t.Errorf("反序列化后的值 %d 不等于原始值 %d", decoded, original)
	}

	// 测试负数错误
	negativeJSON := []byte("-1")
	var s Status
	err = s.UnmarshalJSON(negativeJSON)
	if err == nil {
		t.Error("反序列化负数应该返回错误")
	}
}

// TestStatusValue 测试数据库 Value 方法
func TestStatusValue(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	val, err := s.Value()
	if err != nil {
		t.Fatalf("Value() 返回错误: %v", err)
	}

	int64Val, ok := val.(int64)
	if !ok {
		t.Fatalf("Value() 应该返回 int64 类型，实际返回 %T", val)
	}

	if int64Val != int64(s) {
		t.Errorf("Value() = %d, 期望 %d", int64Val, int64(s))
	}
}

// TestStatusString 测试字符串表示
func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		contains string
	}{
		{StatusNone, "None"},
		{StatusUserDisabled, "32"},
		{StatusSysHidden, "64"},
	}

	for _, tt := range tests {
		str := tt.status.String()
		if str == "" {
			t.Errorf("String() 不应该返回空字符串")
		}
	}
}

// TestStatusClear 测试清除所有状态
func TestStatusClear(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
	s.Clear()

	if s != StatusNone {
		t.Errorf("Clear() 后状态应该为 StatusNone，实际为 %d", s)
	}
}

// TestStatusEqual 测试状态相等性
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

// BenchmarkStatus_Set 基准测试：Set 操作
func BenchmarkStatus_Set(b *testing.B) {
	var s Status
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(StatusUserDisabled)
	}
}

// BenchmarkStatus_Contain 基准测试：Contain 检查
func BenchmarkStatus_Contain(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Contain(StatusUserDisabled)
	}
}

// BenchmarkStatus_HasAll 基准测试：HasAll 检查
func BenchmarkStatus_HasAll(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.HasAll(StatusUserDisabled, StatusSysHidden)
	}
}

// BenchmarkStatus_JSONMarshal 基准测试：JSON 序列化
func BenchmarkStatus_JSONMarshal(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(s)
	}
}
