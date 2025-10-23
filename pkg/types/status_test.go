// filepath: /Users/jiang/Workspace/Resource/9_katydid/Code/katydid-common-account/pkg/types/status_test.go
package types

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
)

// ============================================================================
// 基础功能测试
// ============================================================================

// TestStatus_Set 测试状态设置
func TestStatus_Set(t *testing.T) {
	tests := []struct {
		name     string
		initial  Status
		setValue Status
		want     Status
	}{
		{"零值设置", StatusNone, StatusSysDeleted, StatusSysDeleted},
		{"覆盖单个状态", StatusSysDeleted, StatusAdmDeleted, StatusAdmDeleted},
		{"覆盖多个状态", StatusSysDeleted | StatusAdmDeleted, StatusUserDeleted, StatusUserDeleted},
		{"设置组合状态", StatusNone, StatusAllDeleted, StatusAllDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.Set(tt.setValue)
			if s != tt.want {
				t.Errorf("Set() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_Clear 测试清除所有状态
func TestStatus_Clear(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
	}{
		{"清除零值", StatusNone},
		{"清除单个状态", StatusSysDeleted},
		{"清除多个状态", StatusAllDeleted},
		{"清除复杂状态", StatusAllDeleted | StatusAllDisabled | StatusAllHidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.Clear()
			if s != StatusNone {
				t.Errorf("Clear() = %v, want StatusNone", s)
			}
		})
	}
}

// TestStatus_Add 测试添加状态位
func TestStatus_Add(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		add     Status
		want    Status
	}{
		{"添加到零值", StatusNone, StatusSysDeleted, StatusSysDeleted},
		{"添加相同状态", StatusSysDeleted, StatusSysDeleted, StatusSysDeleted},
		{"添加不同状态", StatusSysDeleted, StatusAdmDeleted, StatusSysDeleted | StatusAdmDeleted},
		{"添加组合状态", StatusSysDeleted, StatusAllDisabled, StatusSysDeleted | StatusAllDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.Add(tt.add)
			if s != tt.want {
				t.Errorf("Add() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_Del 测试删除状态位
func TestStatus_Del(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		del     Status
		want    Status
	}{
		{"从零值删除", StatusNone, StatusSysDeleted, StatusNone},
		{"删除存在的状态", StatusSysDeleted | StatusAdmDeleted, StatusSysDeleted, StatusAdmDeleted},
		{"删除不存在的状态", StatusSysDeleted, StatusAdmDeleted, StatusSysDeleted},
		{"删除组合状态", StatusAllDeleted | StatusAllDisabled, StatusAllDeleted, StatusAllDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.Del(tt.del)
			if s != tt.want {
				t.Errorf("Del() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_AddMultiple 测试批量添加
func TestStatus_AddMultiple(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		flags   []Status
		want    Status
	}{
		{"添加空列表", StatusNone, []Status{}, StatusNone},
		{"添加单个", StatusNone, []Status{StatusSysDeleted}, StatusSysDeleted},
		{"添加多个", StatusNone, []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted}, StatusAllDeleted},
		{"添加重复", StatusSysDeleted, []Status{StatusSysDeleted, StatusAdmDeleted}, StatusSysDeleted | StatusAdmDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.AddMultiple(tt.flags...)
			if s != tt.want {
				t.Errorf("AddMultiple() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_DelMultiple 测试批量删除
func TestStatus_DelMultiple(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		flags   []Status
		want    Status
	}{
		{"删除空列表", StatusAllDeleted, []Status{}, StatusAllDeleted},
		{"删除单个", StatusAllDeleted, []Status{StatusSysDeleted}, StatusAdmDeleted | StatusUserDeleted},
		{"删除多个", StatusAllDeleted, []Status{StatusSysDeleted, StatusAdmDeleted}, StatusUserDeleted},
		{"删除全部", StatusAllDeleted, []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted}, StatusNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.DelMultiple(tt.flags...)
			if s != tt.want {
				t.Errorf("DelMultiple() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_Toggle 测试切换状态位
func TestStatus_Toggle(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		toggle  Status
		want    Status
	}{
		{"切换零值", StatusNone, StatusSysDeleted, StatusSysDeleted},
		{"切换已存在", StatusSysDeleted, StatusSysDeleted, StatusNone},
		{"切换不存在", StatusSysDeleted, StatusAdmDeleted, StatusSysDeleted | StatusAdmDeleted},
		{"切换组合", StatusSysDeleted | StatusAdmDeleted, StatusAllDeleted, StatusUserDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.Toggle(tt.toggle)
			if s != tt.want {
				t.Errorf("Toggle() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_ToggleMultiple 测试批量切换
func TestStatus_ToggleMultiple(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		flags   []Status
		want    Status
	}{
		{"切换空列表", StatusNone, []Status{}, StatusNone},
		{"切换单个不存在", StatusNone, []Status{StatusSysDeleted}, StatusSysDeleted},
		{"切换单个已存在", StatusSysDeleted, []Status{StatusSysDeleted}, StatusNone},
		{"切换多个混合", StatusSysDeleted | StatusAdmDeleted, []Status{StatusAdmDeleted, StatusUserDeleted}, StatusSysDeleted | StatusUserDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.ToggleMultiple(tt.flags...)
			if s != tt.want {
				t.Errorf("ToggleMultiple() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_And 测试状态位与运算
func TestStatus_And(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		and     Status
		want    Status
	}{
		{"零值与运算", StatusNone, StatusSysDeleted, StatusNone},
		{"相同状态", StatusSysDeleted, StatusSysDeleted, StatusSysDeleted},
		{"保留交集", StatusAllDeleted, StatusSysDeleted | StatusAdmDeleted, StatusSysDeleted | StatusAdmDeleted},
		{"无交集", StatusSysDeleted, StatusAdmDeleted, StatusNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.And(tt.and)
			if s != tt.want {
				t.Errorf("And() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_AndMultiple 测试批量与运算
func TestStatus_AndMultiple(t *testing.T) {
	tests := []struct {
		name    string
		initial Status
		flags   []Status
		want    Status
	}{
		{"空列表", StatusAllDeleted, []Status{}, StatusAllDeleted}, // 空列表时保持不变
		{"单个", StatusAllDeleted, []Status{StatusSysDeleted}, StatusSysDeleted},
		{"多个", StatusAllDeleted | StatusAllDisabled, []Status{StatusAllDeleted, StatusSysDisabled}, StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted | StatusSysDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.initial
			s.AndMultiple(tt.flags...)
			if s != tt.want {
				t.Errorf("AndMultiple() = %v, want %v", s, tt.want)
			}
		})
	}
}

// ============================================================================
// 状态查询测试
// ============================================================================

// TestStatus_Has 测试单个状态检查
func TestStatus_Has(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		check  Status
		want   bool
	}{
		{"零值检查零", StatusNone, StatusNone, false}, // 零值特殊处理
		{"零值检查非零", StatusNone, StatusSysDeleted, false},
		{"单个匹配", StatusSysDeleted, StatusSysDeleted, true},
		{"单个不匹配", StatusSysDeleted, StatusAdmDeleted, false},
		{"多个全匹配", StatusAllDeleted, StatusAllDeleted, true},
		{"多个部分匹配", StatusAllDeleted, StatusSysDeleted, true},
		{"检查组合-全匹配", StatusSysDeleted | StatusAdmDeleted, StatusSysDeleted | StatusAdmDeleted, true},
		{"检查组合-部分匹配", StatusSysDeleted | StatusAdmDeleted, StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.Has(tt.check); got != tt.want {
				t.Errorf("Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_HasAny 测试任意状态检查
func TestStatus_HasAny(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		flags  []Status
		want   bool
	}{
		{"空列表", StatusSysDeleted, []Status{}, false},
		{"单个匹配", StatusSysDeleted, []Status{StatusSysDeleted}, true},
		{"单个不匹配", StatusSysDeleted, []Status{StatusAdmDeleted}, false},
		{"多个有一个匹配", StatusSysDeleted, []Status{StatusSysDeleted, StatusAdmDeleted}, true},
		{"多个都不匹配", StatusSysDeleted, []Status{StatusAdmDeleted, StatusUserDeleted}, false},
		{"零值检查", StatusNone, []Status{StatusSysDeleted}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.HasAny(tt.flags...); got != tt.want {
				t.Errorf("HasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_HasAll 测试全部状态检查
func TestStatus_HasAll(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		flags  []Status
		want   bool
	}{
		{"空列表", StatusSysDeleted, []Status{}, true},
		{"单个匹配", StatusSysDeleted, []Status{StatusSysDeleted}, true},
		{"单个不匹配", StatusSysDeleted, []Status{StatusAdmDeleted}, false},
		{"多个全匹配", StatusAllDeleted, []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted}, true},
		{"多个部分匹配", StatusSysDeleted | StatusAdmDeleted, []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted}, false},
		{"零值检查", StatusNone, []Status{StatusSysDeleted}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.HasAll(tt.flags...); got != tt.want {
				t.Errorf("HasAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_ActiveFlags 测试获取活动状态位
func TestStatus_ActiveFlags(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   []Status
	}{
		{"零值", StatusNone, nil},
		{"单个状态", StatusSysDeleted, []Status{StatusSysDeleted}},
		{"三个状态", StatusAllDeleted, []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted}},
		{"复杂组合", StatusSysDeleted | StatusAdmDisabled | StatusUserHidden,
			[]Status{StatusSysDeleted, StatusAdmDisabled, StatusUserHidden}},
		{"高位状态", StatusExpand51, []Status{StatusExpand51}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.ActiveFlags()
			if len(got) != len(tt.want) {
				t.Errorf("ActiveFlags() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i, flag := range got {
				if flag != tt.want[i] {
					t.Errorf("ActiveFlags()[%d] = %v, want %v", i, flag, tt.want[i])
				}
			}
		})
	}
}

// TestStatus_Diff 测试状态差异比较
func TestStatus_Diff(t *testing.T) {
	tests := []struct {
		name        string
		s1          Status
		s2          Status
		wantAdded   Status
		wantRemoved Status
	}{
		{"相同状态", StatusSysDeleted, StatusSysDeleted, StatusNone, StatusNone},
		{"完全不同", StatusSysDeleted, StatusAdmDeleted, StatusSysDeleted, StatusAdmDeleted},
		{"添加状态", StatusSysDeleted | StatusAdmDeleted, StatusSysDeleted, StatusAdmDeleted, StatusNone},
		{"移除状态", StatusSysDeleted, StatusSysDeleted | StatusAdmDeleted, StatusNone, StatusAdmDeleted},
		{"混合变化", StatusSysDeleted | StatusAdmDeleted, StatusAdmDeleted | StatusUserDeleted,
			StatusSysDeleted, StatusUserDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAdded, gotRemoved := tt.s1.Diff(tt.s2)
			if gotAdded != tt.wantAdded {
				t.Errorf("Diff() added = %v, want %v", gotAdded, tt.wantAdded)
			}
			if gotRemoved != tt.wantRemoved {
				t.Errorf("Diff() removed = %v, want %v", gotRemoved, tt.wantRemoved)
			}
		})
	}
}

// ============================================================================
// 业务状态检查测试
// ============================================================================

// TestStatus_IsDeleted 测试删除状态检查
func TestStatus_IsDeleted(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"零值", StatusNone, false},
		{"系统删除", StatusSysDeleted, true},
		{"管理员删除", StatusAdmDeleted, true},
		{"用户删除", StatusUserDeleted, true},
		{"全部删除", StatusAllDeleted, true},
		{"其他状态", StatusSysDisabled, false},
		{"混合状态", StatusSysDeleted | StatusAdmDisabled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsDeleted(); got != tt.want {
				t.Errorf("IsDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_IsDisable 测试禁用状态检查
func TestStatus_IsDisable(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"零值", StatusNone, false},
		{"系统禁用", StatusSysDisabled, true},
		{"管理员禁用", StatusAdmDisabled, true},
		{"用户禁用", StatusUserDisabled, true},
		{"全部禁用", StatusAllDisabled, true},
		{"其他状态", StatusSysDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsDisable(); got != tt.want {
				t.Errorf("IsDisable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_IsHidden 测试隐藏状态检查
func TestStatus_IsHidden(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"零值", StatusNone, false},
		{"系统隐藏", StatusSysHidden, true},
		{"管理员隐藏", StatusAdmHidden, true},
		{"用户隐藏", StatusUserHidden, true},
		{"全部隐藏", StatusAllHidden, true},
		{"其他状态", StatusSysDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsHidden(); got != tt.want {
				t.Errorf("IsHidden() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_IsReview 测试审核状态检查
func TestStatus_IsReview(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"零值", StatusNone, false},
		{"系统审核", StatusSysReview, true},
		{"管理员审核", StatusAdmReview, true},
		{"用户审核", StatusUserReview, true},
		{"全部审核", StatusAllReview, true},
		{"其他状态", StatusSysDeleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsReview(); got != tt.want {
				t.Errorf("IsReview() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_CanEnable 测试可启用状态检查
func TestStatus_CanEnable(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"零值", StatusNone, true},
		{"隐藏状态", StatusSysHidden, true},
		{"审核状态", StatusSysReview, true},
		{"删除状态", StatusSysDeleted, false},
		{"禁用状态", StatusSysDisabled, false},
		{"删除+禁用", StatusSysDeleted | StatusSysDisabled, false},
		{"隐藏+审核", StatusSysHidden | StatusSysReview, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.CanEnable(); got != tt.want {
				t.Errorf("CanEnable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_CanVisible 测试可见状态检查
func TestStatus_CanVisible(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"零值", StatusNone, true},
		{"审核状态", StatusSysReview, true},
		{"删除状态", StatusSysDeleted, false},
		{"禁用状态", StatusSysDisabled, false},
		{"隐藏状态", StatusSysHidden, false},
		{"删除+禁用+隐藏", StatusSysDeleted | StatusSysDisabled | StatusSysHidden, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.CanVisible(); got != tt.want {
				t.Errorf("CanVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_CanActive 测试完全激活状态检查
func TestStatus_CanActive(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"零值", StatusNone, true},
		{"删除状态", StatusSysDeleted, false},
		{"禁用状态", StatusSysDisabled, false},
		{"隐藏状态", StatusSysHidden, false},
		{"审核状态", StatusSysReview, false},
		{"任意组合", StatusSysDeleted | StatusSysDisabled | StatusSysHidden | StatusSysReview, false},
		{"扩展状态", StatusExpand51, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.CanActive(); got != tt.want {
				t.Errorf("CanActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// 格式化测试
// ============================================================================

// TestStatus_String 测试字符串格式化
func TestStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{"零值", StatusNone, "Status(0)[0 bits]"},
		{"单个状态", StatusSysDeleted, "Status(1)[1 bits]"},
		{"三个状态", StatusAllDeleted, "Status(7)[3 bits]"},
		{"复杂状态", StatusAllDeleted | StatusAllDisabled, "Status(63)[6 bits]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_BitCount 测试位计数
func TestStatus_BitCount(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   int
	}{
		{"零值", StatusNone, 0},
		{"单位", StatusSysDeleted, 1},
		{"三位", StatusAllDeleted, 3},
		{"六位", StatusAllDeleted | StatusAllDisabled, 6},
		{"十二位", StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllReview, 12},
		{"高位", StatusExpand51, 1},
		{"连续位", Status(0xFF), 8},
		{"稀疏位", Status(0x101), 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.BitCount(); got != tt.want {
				t.Errorf("BitCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================================
// 数据库接口测试
// ============================================================================

// TestStatus_Value 测试 driver.Valuer 接口
func TestStatus_Value(t *testing.T) {
	tests := []struct {
		name    string
		status  Status
		want    driver.Value
		wantErr bool
	}{
		{"零值", StatusNone, int64(0), false},
		{"正常值", StatusSysDeleted, int64(1), false},
		{"组合值", StatusAllDeleted, int64(7), false},
		{"负数", Status(-1), nil, true},
		{"最大合法值", MaxStatus, int64(MaxStatus), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.status.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatus_Scan 测试 sql.Scanner 接口
func TestStatus_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		want    Status
		wantErr bool
	}{
		{"nil值", nil, StatusNone, false},
		{"int64零值", int64(0), StatusNone, false},
		{"int64正常值", int64(7), StatusAllDeleted, false},
		{"int类型", int(7), StatusAllDeleted, false},
		{"uint64类型", uint64(7), StatusAllDeleted, false},
		{"JSON字节", []byte("7"), StatusAllDeleted, false},
		{"负数", int64(-1), StatusNone, true},
		{"不支持的类型", "invalid", StatusNone, true},
		{"无效JSON", []byte("invalid"), StatusNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Status
			err := s.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && s != tt.want {
				t.Errorf("Scan() = %v, want %v", s, tt.want)
			}
		})
	}
}

// ============================================================================
// JSON 序列化测试
// ============================================================================

// TestStatus_MarshalJSON 测试 JSON 序列化
func TestStatus_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		status  Status
		want    string
		wantErr bool
	}{
		{"零值", StatusNone, "0", false},
		{"单个状态", StatusSysDeleted, "1", false},
		{"组合状态", StatusAllDeleted, "7", false},
		{"复杂状态", StatusAllDeleted | StatusAllDisabled, "63", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.status.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("MarshalJSON() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

// TestStatus_UnmarshalJSON 测试 JSON 反序列化
func TestStatus_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    Status
		wantErr bool
	}{
		{"零值", "0", StatusNone, false},
		{"单个状态", "1", StatusSysDeleted, false},
		{"组合状态", "7", StatusAllDeleted, false},
		{"null值", "null", StatusNone, false},
		{"空数据", "", StatusNone, true},
		{"负数", "-1", StatusNone, true},
		{"非数字", `"invalid"`, StatusNone, true},
		{"无效JSON", "invalid", StatusNone, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Status
			err := s.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && s != tt.want {
				t.Errorf("UnmarshalJSON() = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_JSON_RoundTrip 测试 JSON 往返转换
func TestStatus_JSON_RoundTrip(t *testing.T) {
	tests := []Status{
		StatusNone,
		StatusSysDeleted,
		StatusAllDeleted,
		StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllReview,
		StatusExpand51,
	}

	for _, original := range tests {
		t.Run(original.String(), func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Unmarshal
			var decoded Status
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Verify
			if decoded != original {
				t.Errorf("Round trip failed: got %v, want %v", decoded, original)
			}
		})
	}
}

// ============================================================================
// 边界条件测试
// ============================================================================

// TestStatus_EdgeCases 测试边界条件
func TestStatus_EdgeCases(t *testing.T) {
	t.Run("零值特殊处理", func(t *testing.T) {
		s := StatusNone
		if s.Has(StatusNone) {
			t.Error("StatusNone.Has(StatusNone) should return false")
		}
	})

	t.Run("最大状态值", func(t *testing.T) {
		s := MaxStatus
		bitCount := s.BitCount()
		if bitCount != 63 {
			t.Errorf("MaxStatus bit count = %d, want 63", bitCount)
		}
	})

	t.Run("高位状态", func(t *testing.T) {
		s := Status(1 << 62)
		if !s.Has(Status(1 << 62)) {
			t.Error("High bit status check failed")
		}
	})

	t.Run("所有预定义状态", func(t *testing.T) {
		allPredefined := StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllReview
		if allPredefined.BitCount() != 12 {
			t.Errorf("All predefined status bit count = %d, want 12", allPredefined.BitCount())
		}
	})
}

// TestStatus_Constants 测试常量定义
func TestStatus_Constants(t *testing.T) {
	t.Run("删除状态组", func(t *testing.T) {
		if StatusAllDeleted != (StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted) {
			t.Error("StatusAllDeleted constant mismatch")
		}
	})

	t.Run("禁用状态组", func(t *testing.T) {
		if StatusAllDisabled != (StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled) {
			t.Error("StatusAllDisabled constant mismatch")
		}
	})

	t.Run("隐藏状态组", func(t *testing.T) {
		if StatusAllHidden != (StatusSysHidden | StatusAdmHidden | StatusUserHidden) {
			t.Error("StatusAllHidden constant mismatch")
		}
	})

	t.Run("审核状态组", func(t *testing.T) {
		if StatusAllReview != (StatusSysReview | StatusAdmReview | StatusUserReview) {
			t.Error("StatusAllReview constant mismatch")
		}
	})

	t.Run("扩展起始位", func(t *testing.T) {
		if StatusExpand51 != Status(1<<12) {
			t.Error("StatusExpand51 constant mismatch")
		}
	})
}

// ============================================================================
// 性能基准测试 - 百万级单线程测试
// ============================================================================

// BenchmarkStatus_Set 基准测试：设置状态
func BenchmarkStatus_Set(b *testing.B) {
	s := StatusNone
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(StatusSysDeleted)
	}
}

// BenchmarkStatus_Add 基准测试：添加状态
func BenchmarkStatus_Add(b *testing.B) {
	s := StatusNone
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Add(StatusSysDeleted)
	}
}

// BenchmarkStatus_Del 基准测试：删除状态
func BenchmarkStatus_Del(b *testing.B) {
	s := StatusAllDeleted
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Del(StatusSysDeleted)
		s.Add(StatusSysDeleted) // 恢复状态
	}
}

// BenchmarkStatus_Has 基准测试：检查单个状态
func BenchmarkStatus_Has(b *testing.B) {
	s := StatusAllDeleted
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Has(StatusSysDeleted)
	}
}

// BenchmarkStatus_HasAny 基准测试：检查任意状态
func BenchmarkStatus_HasAny(b *testing.B) {
	s := StatusAllDeleted
	flags := []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.HasAny(flags...)
	}
}

// BenchmarkStatus_HasAll 基准测试：检查所有状态
func BenchmarkStatus_HasAll(b *testing.B) {
	s := StatusAllDeleted
	flags := []Status{StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.HasAll(flags...)
	}
}

// BenchmarkStatus_Toggle 基准测试：切换状态
func BenchmarkStatus_Toggle(b *testing.B) {
	s := StatusNone
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Toggle(StatusSysDeleted)
	}
}

// BenchmarkStatus_BitCount 基准测试：位计数
func BenchmarkStatus_BitCount(b *testing.B) {
	s := StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllReview
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.BitCount()
	}
}

// BenchmarkStatus_ActiveFlags 基准测试：获取活动状态
func BenchmarkStatus_ActiveFlags(b *testing.B) {
	s := StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllReview
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.ActiveFlags()
	}
}

// BenchmarkStatus_String 基准测试：字符串格式化
func BenchmarkStatus_String(b *testing.B) {
	s := StatusAllDeleted | StatusAllDisabled
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.String()
	}
}

// BenchmarkStatus_IsDeleted 基准测试：删除状态检查
func BenchmarkStatus_IsDeleted(b *testing.B) {
	s := StatusSysDeleted | StatusAdmDisabled
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.IsDeleted()
	}
}

// BenchmarkStatus_CanActive 基准测试：激活状态检查
func BenchmarkStatus_CanActive(b *testing.B) {
	s := StatusExpand51
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.CanActive()
	}
}

// BenchmarkStatus_MarshalJSON 基准测试：JSON 序列化
func BenchmarkStatus_MarshalJSON(b *testing.B) {
	s := StatusAllDeleted | StatusAllDisabled
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.MarshalJSON()
	}
}

// BenchmarkStatus_UnmarshalJSON 基准测试：JSON 反序列化
func BenchmarkStatus_UnmarshalJSON(b *testing.B) {
	data := []byte("63")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s Status
		_ = s.UnmarshalJSON(data)
	}
}

// BenchmarkStatus_Value 基准测试：数据库 Value
func BenchmarkStatus_Value(b *testing.B) {
	s := StatusAllDeleted
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Value()
	}
}

// BenchmarkStatus_Scan 基准测试：数据库 Scan
func BenchmarkStatus_Scan(b *testing.B) {
	value := int64(7)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s Status
		_ = s.Scan(value)
	}
}

// ============================================================================
// 百万级复杂场景性能测试
// ============================================================================

// BenchmarkStatus_MillionOperations_Sequential 百万次顺序操作
func BenchmarkStatus_MillionOperations_Sequential(b *testing.B) {
	b.Run("1M-Set", func(b *testing.B) {
		s := StatusNone
		b.ResetTimer()
		for i := 0; i < 1000000; i++ {
			s.Set(StatusSysDeleted)
		}
	})

	b.Run("1M-Add-Del", func(b *testing.B) {
		s := StatusNone
		b.ResetTimer()
		for i := 0; i < 1000000; i++ {
			s.Add(StatusSysDeleted)
			s.Del(StatusSysDeleted)
		}
	})

	b.Run("1M-Toggle", func(b *testing.B) {
		s := StatusNone
		b.ResetTimer()
		for i := 0; i < 1000000; i++ {
			s.Toggle(StatusSysDeleted)
		}
	})

	b.Run("1M-Has", func(b *testing.B) {
		s := StatusAllDeleted
		b.ResetTimer()
		for i := 0; i < 1000000; i++ {
			_ = s.Has(StatusSysDeleted)
		}
	})

	b.Run("1M-BitCount", func(b *testing.B) {
		s := StatusAllDeleted | StatusAllDisabled
		b.ResetTimer()
		for i := 0; i < 1000000; i++ {
			_ = s.BitCount()
		}
	})
}

// BenchmarkStatus_MillionOperations_Mixed 百万次混合操作
func BenchmarkStatus_MillionOperations_Mixed(b *testing.B) {
	s := StatusNone
	b.ResetTimer()

	for i := 0; i < 1000000; i++ {
		// 添加状态
		s.Add(StatusSysDeleted)
		s.Add(StatusAdmDisabled)

		// 检查状态
		_ = s.Has(StatusSysDeleted)
		_ = s.IsDeleted()
		_ = s.CanEnable()

		// 修改状态
		s.Toggle(StatusUserHidden)
		s.Del(StatusAdmDisabled)

		// 查询位数
		_ = s.BitCount()

		// 清理
		s.Clear()
	}
}

// BenchmarkStatus_MemoryAllocation 内存分配测试
func BenchmarkStatus_MemoryAllocation(b *testing.B) {
	b.Run("Add-NoAlloc", func(b *testing.B) {
		s := StatusNone
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.Add(StatusSysDeleted)
		}
	})

	b.Run("Has-NoAlloc", func(b *testing.B) {
		s := StatusAllDeleted
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = s.Has(StatusSysDeleted)
		}
	})

	b.Run("ActiveFlags-WithAlloc", func(b *testing.B) {
		s := StatusAllDeleted | StatusAllDisabled
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = s.ActiveFlags()
		}
	})

	b.Run("String-WithAlloc", func(b *testing.B) {
		s := StatusAllDeleted
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = s.String()
		}
	})
}

// BenchmarkStatus_CompareOperations 操作对比测试
func BenchmarkStatus_CompareOperations(b *testing.B) {
	b.Run("SingleAdd-vs-AddMultiple", func(b *testing.B) {
		b.Run("SingleAdd", func(b *testing.B) {
			s := StatusNone
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Add(StatusSysDeleted)
				s.Add(StatusAdmDeleted)
				s.Add(StatusUserDeleted)
			}
		})

		b.Run("AddMultiple", func(b *testing.B) {
			s := StatusNone
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.AddMultiple(StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted)
			}
		})
	})

	b.Run("Has-vs-HasAny-vs-HasAll", func(b *testing.B) {
		s := StatusAllDeleted

		b.Run("Has", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.Has(StatusSysDeleted)
			}
		})

		b.Run("HasAny", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.HasAny(StatusSysDeleted, StatusAdmDeleted)
			}
		})

		b.Run("HasAll", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.HasAll(StatusSysDeleted, StatusAdmDeleted)
			}
		})
	})
}

// BenchmarkStatus_DifferentBitCounts 不同位数性能对比
func BenchmarkStatus_DifferentBitCounts(b *testing.B) {
	scenarios := []struct {
		name   string
		status Status
	}{
		{"1-bit", StatusSysDeleted},
		{"3-bits", StatusAllDeleted},
		{"6-bits", StatusAllDeleted | StatusAllDisabled},
		{"12-bits", StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllReview},
		{"Sparse-bits", Status(0x100000001)}, // 第0位和第32位
	}

	for _, sc := range scenarios {
		b.Run(sc.name+"-BitCount", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = sc.status.BitCount()
			}
		})

		b.Run(sc.name+"-ActiveFlags", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = sc.status.ActiveFlags()
			}
		})

		b.Run(sc.name+"-String", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = sc.status.String()
			}
		})
	}
}

// BenchmarkStatus_JSONRoundTrip 百万次 JSON 往返测试
func BenchmarkStatus_JSONRoundTrip(b *testing.B) {
	statuses := []Status{
		StatusNone,
		StatusSysDeleted,
		StatusAllDeleted,
		StatusAllDeleted | StatusAllDisabled | StatusAllHidden | StatusAllReview,
	}

	for _, original := range statuses {
		b.Run(original.String(), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				data, _ := json.Marshal(original)
				var decoded Status
				_ = json.Unmarshal(data, &decoded)
			}
		})
	}
}

// BenchmarkStatus_DatabaseRoundTrip 百万次数据库往返测试
func BenchmarkStatus_DatabaseRoundTrip(b *testing.B) {
	s := StatusAllDeleted
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Value
		val, _ := s.Value()

		// Scan
		var decoded Status
		_ = decoded.Scan(val)
	}
}
