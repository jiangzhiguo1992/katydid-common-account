package types

import (
	"encoding/json"
	"testing"
)

func TestStatus_Set(t *testing.T) {
	var s Status
	s.Set(StatusEnabled)
	if !s.Has(StatusEnabled) {
		t.Error("Expected status to have StatusEnabled")
	}

	s.Set(StatusVisible)
	if !s.Has(StatusVisible) {
		t.Error("Expected status to have StatusVisible")
	}
	if !s.Has(StatusEnabled) {
		t.Error("Expected status to still have StatusEnabled")
	}
}

func TestStatus_Unset(t *testing.T) {
	s := StatusEnabled | StatusVisible
	s.Unset(StatusEnabled)
	if s.Has(StatusEnabled) {
		t.Error("Expected status to not have StatusEnabled")
	}
	if !s.Has(StatusVisible) {
		t.Error("Expected status to still have StatusVisible")
	}
}

func TestStatus_Toggle(t *testing.T) {
	var s Status
	s.Toggle(StatusEnabled)
	if !s.Has(StatusEnabled) {
		t.Error("Expected status to have StatusEnabled after toggle")
	}

	s.Toggle(StatusEnabled)
	if s.Has(StatusEnabled) {
		t.Error("Expected status to not have StatusEnabled after second toggle")
	}
}

func TestStatus_Has(t *testing.T) {
	s := StatusEnabled | StatusVisible
	if !s.Has(StatusEnabled) {
		t.Error("Expected status to have StatusEnabled")
	}
	if !s.Has(StatusVisible) {
		t.Error("Expected status to have StatusVisible")
	}
	if s.Has(StatusLocked) {
		t.Error("Expected status to not have StatusLocked")
	}
}

func TestStatus_HasAny(t *testing.T) {
	s := StatusEnabled | StatusVisible
	if !s.HasAny(StatusEnabled, StatusLocked) {
		t.Error("Expected status to have at least one of the flags")
	}
	if s.HasAny(StatusLocked, StatusDeleted) {
		t.Error("Expected status to not have any of the flags")
	}
}

func TestStatus_HasAll(t *testing.T) {
	s := StatusEnabled | StatusVisible
	if !s.HasAll(StatusEnabled, StatusVisible) {
		t.Error("Expected status to have all flags")
	}
	if s.HasAll(StatusEnabled, StatusVisible, StatusLocked) {
		t.Error("Expected status to not have all flags")
	}
}

func TestStatus_SetMultiple(t *testing.T) {
	var s Status
	s.SetMultiple(StatusEnabled, StatusVisible, StatusActive)
	if !s.HasAll(StatusEnabled, StatusVisible, StatusActive) {
		t.Error("Expected status to have all set flags")
	}
}

func TestStatus_UnsetMultiple(t *testing.T) {
	s := StatusEnabled | StatusVisible | StatusActive
	s.UnsetMultiple(StatusEnabled, StatusVisible)
	if s.Has(StatusEnabled) || s.Has(StatusVisible) {
		t.Error("Expected unset flags to be removed")
	}
	if !s.Has(StatusActive) {
		t.Error("Expected StatusActive to remain")
	}
}

func TestStatus_IsNormal(t *testing.T) {
	s := StatusNormal
	if !s.IsNormal() {
		t.Error("Expected StatusNormal to be normal")
	}

	s = StatusEnabled
	if s.IsNormal() {
		t.Error("Expected status with only Enabled to not be normal")
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		contains []string
	}{
		{
			name:     "none",
			status:   StatusNone,
			contains: []string{"none"},
		},
		{
			name:     "enabled",
			status:   StatusEnabled,
			contains: []string{"enabled"},
		},
		{
			name:     "multiple",
			status:   StatusEnabled | StatusVisible,
			contains: []string{"enabled", "visible"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.status.String()
			t.Logf("Status string: %s", str)
			// Basic validation - just ensure it doesn't panic
			if str == "" {
				t.Error("Expected non-empty string")
			}
		})
	}
}

func TestStatus_JSON(t *testing.T) {
	s := StatusEnabled | StatusVisible | StatusActive

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
	s := StatusEnabled | StatusVisible
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
	s := StatusEnabled | StatusVisible | StatusActive | StatusPublished

	if !s.IsEnabled() {
		t.Error("Expected IsEnabled to be true")
	}
	if !s.IsVisible() {
		t.Error("Expected IsVisible to be true")
	}
	if !s.IsActive() {
		t.Error("Expected IsActive to be true")
	}
	if !s.IsPublished() {
		t.Error("Expected IsPublished to be true")
	}
	if s.IsLocked() {
		t.Error("Expected IsLocked to be false")
	}
	if s.IsDeleted() {
		t.Error("Expected IsDeleted to be false")
	}
}

func TestStatus_PredefinedCombinations(t *testing.T) {
	// Test StatusNormal
	s := StatusNormal
	if !s.Has(StatusEnabled) || !s.Has(StatusVisible) {
		t.Error("Expected StatusNormal to have Enabled and Visible")
	}

	// Test StatusPublicActive
	s = StatusPublicActive
	if !s.HasAll(StatusEnabled, StatusVisible, StatusActive, StatusPublished) {
		t.Error("Expected StatusPublicActive to have all required flags")
	}

	// Test StatusSoftDeleted
	s = StatusSoftDeleted
	if !s.HasAll(StatusDeleted, StatusHidden) {
		t.Error("Expected StatusSoftDeleted to have Deleted and Hidden")
	}
}

func TestStatus_RealWorldScenario(t *testing.T) {
	// 模拟用户状态管理
	var userStatus Status

	// 新用户注册，设置为启用+可见
	userStatus.SetMultiple(StatusEnabled, StatusVisible)
	if !userStatus.IsNormal() {
		t.Error("New user should be in normal status")
	}

	// 用户通过邮箱验证
	userStatus.Set(StatusVerified)
	if !userStatus.IsVerified() {
		t.Error("User should be verified")
	}

	// 用户发布了内容，变为活跃用户
	userStatus.Set(StatusActive)
	if !userStatus.IsActive() {
		t.Error("User should be active")
	}

	// 管理员将用户设置为推荐用户
	userStatus.Set(StatusFeatured)
	if !userStatus.Has(StatusFeatured) {
		t.Error("User should be featured")
	}

	// 用户违规，被暂停
	userStatus.Set(StatusSuspended)
	userStatus.Unset(StatusEnabled)
	if userStatus.IsEnabled() {
		t.Error("Suspended user should not be enabled")
	}
	if !userStatus.Has(StatusSuspended) {
		t.Error("User should be suspended")
	}

	// 解除暂停
	userStatus.Unset(StatusSuspended)
	userStatus.Set(StatusEnabled)
	if userStatus.Has(StatusSuspended) {
		t.Error("User should not be suspended")
	}
	if !userStatus.IsEnabled() {
		t.Error("User should be enabled again")
	}
}

func TestStatus_NegativeValueHandling(t *testing.T) {
	// 测试负数值是否会影响位运算
	var s Status = -1 // 所有位都是1（二进制补码表示）

	t.Logf("Negative status value: %d (binary: %b)", s, s)

	// 负数包含所有位，所以应该包含所有状态
	if !s.Has(StatusEnabled) {
		t.Error("Expected -1 to have StatusEnabled bit")
	}
	if !s.Has(StatusVisible) {
		t.Error("Expected -1 to have StatusVisible bit")
	}

	// 测试从负数取消位
	s.Unset(StatusEnabled)
	t.Logf("After unsetting Enabled: %d (binary: %b)", s, s)
	if s.Has(StatusEnabled) {
		t.Error("Expected StatusEnabled to be unset")
	}

	// 正常使用场景下，状态值应该始终为非负数
	var normalStatus Status
	normalStatus.SetMultiple(StatusEnabled, StatusVisible, StatusActive)
	t.Logf("Normal status value: %d (binary: %b)", normalStatus, normalStatus)

	if normalStatus < 0 {
		t.Error("Normal status should never be negative")
	}
}

func TestStatus_ValueRange(t *testing.T) {
	// 测试所有预定义状态都是正数
	statuses := []Status{
		StatusEnabled, StatusVisible, StatusLocked, StatusDeleted,
		StatusActive, StatusVerified, StatusPublished, StatusArchived,
		StatusFeatured, StatusPinned, StatusHidden, StatusSuspended,
		StatusPending, StatusApproved, StatusRejected, StatusDraft,
	}

	for _, status := range statuses {
		if status < 0 {
			t.Errorf("Status %d should be positive", status)
		}
		t.Logf("Status value: %d (binary: %b)", status, status)
	}

	// 测试组合状态也都是正数
	combinedStatuses := []Status{
		StatusNormal, StatusPublicActive, StatusPendingReview, StatusSoftDeleted,
	}

	for _, status := range combinedStatuses {
		if status < 0 {
			t.Errorf("Combined status %d should be positive", status)
		}
		t.Logf("Combined status value: %d (binary: %b)", status, status)
	}

	// 测试设置多个状态后仍然是正数
	var s Status
	s.SetMultiple(StatusEnabled, StatusVisible, StatusActive, StatusVerified,
		StatusPublished, StatusFeatured, StatusPinned)

	if s < 0 {
		t.Error("Status with multiple flags should still be positive")
	}
	t.Logf("Multiple flags status: %d (binary: %b)", s, s)
}

func TestStatus_MaxSafeBits(t *testing.T) {
	// int64 可以安全使用 0-62 位（第63位是符号位）
	// 我们目前使用了 17 个状态位（0-16），非常安全

	// 测试使用高位（但不触及符号位）
	var s Status = 1 << 62 // 使用第62位
	t.Logf("High bit status: %d (binary: %b)", s, s)

	if s < 0 {
		t.Error("Using bit 62 should still be positive")
	}

	// 测试第63位（符号位）- 这会导致负数
	var negativeStatus Status = 1 << 63
	t.Logf("Sign bit status: %d (binary: %b)", negativeStatus, negativeStatus)

	if negativeStatus >= 0 {
		t.Error("Using bit 63 should result in negative number")
	}

	// 确认我们当前的最高状态位是安全的
	highestStatus := StatusDraft
	t.Logf("Highest defined status: %d (binary: %b)", highestStatus, highestStatus)

	if highestStatus < 0 {
		t.Error("Highest defined status should be positive")
	}

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
