package types

import (
	"encoding/json"
	"testing"
)

func TestStatus_Set(t *testing.T) {
	var s Status
	s.Set(StatusUserDisabled)
	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected status to have StatusUserDisabled")
	}

	s.Set(StatusSysHidden)
	if !s.Contain(StatusSysHidden) {
		t.Error("Expected status to have StatusSysHidden")
	}
	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected status to still have StatusUserDisabled")
	}
}

func TestStatus_Unset(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	s.Unset(StatusUserDisabled)
	if s.Contain(StatusUserDisabled) {
		t.Error("Expected status to not have StatusUserDisabled")
	}
	if !s.Contain(StatusSysHidden) {
		t.Error("Expected status to still have StatusSysHidden")
	}
}

func TestStatus_Toggle(t *testing.T) {
	var s Status
	s.Toggle(StatusUserDisabled)
	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected status to have StatusUserDisabled after toggle")
	}

	s.Toggle(StatusUserDisabled)
	if s.Contain(StatusUserDisabled) {
		t.Error("Expected status to not have StatusUserDisabled after second toggle")
	}
}

func TestStatus_Has(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected status to have StatusUserDisabled")
	}
	if !s.Contain(StatusSysHidden) {
		t.Error("Expected status to have StatusSysHidden")
	}
	if s.Contain(StatusSysDeleted) {
		t.Error("Expected status to not have StatusSysDeleted")
	}
}

func TestStatus_HasAny(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	if !s.HasAny(StatusUserDisabled, StatusSysDeleted) {
		t.Error("Expected status to have at least one of the flags")
	}
	if s.HasAny(StatusSysDeleted, StatusAdmDeleted) {
		t.Error("Expected status to not have any of the flags")
	}
}

func TestStatus_HasAll(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	if !s.HasAll(StatusUserDisabled, StatusSysHidden) {
		t.Error("Expected status to have all flags")
	}
	if s.HasAll(StatusUserDisabled, StatusSysHidden, StatusSysDeleted) {
		t.Error("Expected status to not have all flags")
	}
}

func TestStatus_SetMultiple(t *testing.T) {
	var s Status
	s.SetMultiple(StatusUserDisabled, StatusSysHidden, StatusAdmDisabled)
	if !s.HasAll(StatusUserDisabled, StatusSysHidden, StatusAdmDisabled) {
		t.Error("Expected status to have all set flags")
	}
}

func TestStatus_UnsetMultiple(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDisabled
	s.UnsetMultiple(StatusUserDisabled, StatusSysHidden)
	if s.Contain(StatusUserDisabled) || s.Contain(StatusSysHidden) {
		t.Error("Expected unset flags to be removed")
	}
	if !s.Contain(StatusAdmDisabled) {
		t.Error("Expected StatusAdmDisabled to remain")
	}
}

func TestStatus_IsNormal(t *testing.T) {
	s := StatusNone
	if !s.IsNormal() {
		t.Error("Expected StatusNone to be normal")
	}

	s = StatusUserDisabled
	if s.IsNormal() {
		t.Error("Expected status with UserDisabled to not be normal")
	}

	s = StatusSysHidden
	if s.IsNormal() {
		t.Error("Expected status with SysHidden to not be normal")
	}
}

func TestStatus_JSON(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDisabled

	// Marshal
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var s2 Status
	if err := json.Unmarshal(data, &s2); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if s != s2 {
		t.Errorf("Expected %v, got %v", s, s2)
	}
}

func TestStatus_Value(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	val, err := s.Value()
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}
	if val.(int64) != int64(s) {
		t.Errorf("Expected %d, got %d", s, val)
	}
}

func TestStatus_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    Status
		wantErr bool
	}{
		{
			name:  "int64",
			input: int64(3),
			want:  Status(3),
		},
		{
			name:  "int",
			input: 5,
			want:  Status(5),
		},
		{
			name:  "uint64",
			input: uint64(7),
			want:  Status(7),
		},
		{
			name:  "nil",
			input: nil,
			want:  StatusNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Status
			err := s.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if s != tt.want {
				t.Errorf("Scan() got = %v, want %v", s, tt.want)
			}
		})
	}
}

func TestStatus_HelperMethods(t *testing.T) {
	// 测试正常状态
	s := StatusNone
	if s.IsDisable() {
		t.Error("Expected empty status to not be disabled")
	}
	if s.IsHidden() {
		t.Error("Expected empty status to not be hidden")
	}
	if s.IsDeleted() {
		t.Error("Expected empty status to not be deleted")
	}

	// 测试禁用状态
	s = StatusUserDisabled
	if !s.IsDisable() {
		t.Error("Expected UserDisabled to be disabled")
	}
	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected to contain StatusUserDisabled")
	}

	// 测试隐藏状态
	s = StatusSysHidden
	if !s.IsHidden() {
		t.Error("Expected IsHidden to be true")
	}

	// 测试删除状态
	s = StatusSysDeleted
	if !s.IsDeleted() {
		t.Error("Expected IsDeleted to be true")
	}

	// 测试验证状态
	s = StatusNone
	if s.IsUnverified() {
		t.Error("Expected empty status to be verified")
	}

	s = StatusSysUnverified
	if !s.IsUnverified() {
		t.Error("Expected IsUnverified to be true")
	}
}

func TestStatus_PredefinedCombinations(t *testing.T) {
	// Test soft deleted combination
	s := StatusSysDeleted | StatusSysHidden
	if !s.HasAll(StatusSysDeleted, StatusSysHidden) {
		t.Error("Expected to have SysDeleted and SysHidden")
	}

	// Test hard disabled combination
	s = StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled
	if !s.HasAll(StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled) {
		t.Error("Expected to have all disabled flags")
	}
}

func TestStatus_RealWorldScenario(t *testing.T) {
	// 模拟用户状态管理
	var userStatus Status

	// 新用户注册，默认正常状态
	if !userStatus.IsNormal() {
		t.Error("New user should be in normal status")
	}

	// 用户违规，被管理员禁用
	userStatus.Set(StatusAdmDisabled)
	if !userStatus.IsDisable() {
		t.Error("Disabled user should be disabled")
	}

	// 解除禁用
	userStatus.Unset(StatusAdmDisabled)
	if userStatus.IsDisable() {
		t.Error("User should not be disabled after unset")
	}

	// 用户自己隐藏账号
	userStatus.Set(StatusUserHidden)
	if !userStatus.IsHidden() {
		t.Error("User should be hidden")
	}

	// 用户取消隐藏
	userStatus.Unset(StatusUserHidden)
	if userStatus.IsHidden() {
		t.Error("User should not be hidden after unset")
	}

	// 系统软删除
	userStatus = StatusSysDeleted | StatusSysHidden
	if !userStatus.IsDeleted() {
		t.Error("User should be deleted")
	}
	if !userStatus.IsHidden() {
		t.Error("Deleted user should be hidden")
	}
}

func TestStatus_NegativeValueHandling(t *testing.T) {
	// 测试负数值是否会影响位运算
	var s Status = -1 // 所有位都是1（二进制补码表示）

	t.Logf("Negative status value: %d (binary: %b)", s, s)

	// 负数包含所有位，所以应该包含所有状态
	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected -1 to have StatusUserDisabled bit")
	}
	if !s.Contain(StatusSysHidden) {
		t.Error("Expected -1 to have StatusSysHidden bit")
	}

	// 测试从负数取消位
	s.Unset(StatusUserDisabled)
	t.Logf("After unsetting UserDisabled: %d (binary: %b)", s, s)
	if s.Contain(StatusUserDisabled) {
		t.Error("Expected StatusUserDisabled to be unset")
	}

	// 正常使用场景下，状态值应该始终为非负数
	var normalStatus Status
	normalStatus.SetMultiple(StatusUserDisabled, StatusSysHidden, StatusAdmDisabled)
	t.Logf("Normal status value: %d (binary: %b)", normalStatus, normalStatus)

	if normalStatus < 0 {
		t.Error("Normal status should never be negative")
	}
}

func TestStatus_ValueRange(t *testing.T) {
	// 测试所有预定义状态都是正数
	statuses := []Status{
		StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled,
		StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted,
		StatusSysHidden, StatusAdmHidden, StatusUserHidden,
		StatusSysUnverified, StatusAdmUnverified, StatusUserUnverified,
	}

	for _, status := range statuses {
		if status < 0 {
			t.Errorf("Status %d should be positive", status)
		}
		t.Logf("Status value: %d (binary: %b)", status, status)
	}

	// 测试设置多个状态后仍然是正数
	var s Status
	s.SetMultiple(StatusUserDisabled, StatusSysHidden, StatusAdmDisabled,
		StatusSysDeleted, StatusAdmDeleted, StatusUserDeleted)

	if s < 0 {
		t.Error("Status with multiple flags should still be positive")
	}
	t.Logf("Multiple flags status: %d (binary: %b)", s, s)
}

func TestStatus_MaxSafeBits(t *testing.T) {
	// int64 可以安全使用 0-62 位（第63位是符号位）
	// 我们目前使用了 12 个状态位（0-11），非常安全

	// 测试使用高位（但不触及符号位）
	var s Status = 1 << 62 // 使用第62位
	t.Logf("High bit status: %d (binary: %b)", s, s)

	// 确认我们当前的最高状态位是安全的
	highestStatus := StatusUserUnverified
	t.Logf("Highest defined status: %d (binary: %b)", highestStatus, highestStatus)

	// 计算最高状态位使用的是第几位
	var bitPosition int
	for i := 0; i < 64; i++ {
		if highestStatus == (1 << i) {
			bitPosition = i
			break
		}
	}
	t.Logf("Highest status uses bit position: %d (safe range: 0-62)", bitPosition)

	if bitPosition >= 63 {
		t.Error("Status bits should not use bit 63 (sign bit)")
	}
}

func TestStatus_LeveledDisableHideDelete(t *testing.T) {
	// 测试分级禁用、隐藏、删除功能
	var s Status

	// 测试用户级别禁用
	s.Set(StatusUserDisabled)
	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected user disabled")
	}
	if !s.IsDisable() {
		t.Error("Expected to be disabled")
	}

	// 测试管理员级别禁用
	s.Clear()
	s.Set(StatusAdmDisabled)
	if !s.Contain(StatusAdmDisabled) {
		t.Error("Expected admin disabled")
	}
	if !s.IsDisable() {
		t.Error("Expected to be disabled")
	}

	// 测试系统级别禁用
	s.Clear()
	s.Set(StatusSysDisabled)
	if !s.Contain(StatusSysDisabled) {
		t.Error("Expected system disabled")
	}

	// 测试多级别同时禁用
	s = StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled
	if !s.IsDisable() {
		t.Error("Expected to be disabled")
	}
	if s.IsNormal() {
		t.Error("Expected not to be normal when disabled")
	}
}
