package types

import (
	"encoding/json"
	"sync"
	"testing"
)

// TestStatus_Set 测试设置状态位
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

// TestStatus_Unset 测试取消状态位
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

// TestStatus_Toggle 测试切换状态位
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

// TestStatus_Has 测试包含检查
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

// TestStatus_HasAny 测试包含任意状态
func TestStatus_HasAny(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	if !s.HasAny(StatusUserDisabled, StatusSysDeleted) {
		t.Error("Expected status to have at least one of the flags")
	}
	if s.HasAny(StatusSysDeleted, StatusAdmDeleted) {
		t.Error("Expected status to not have any of the flags")
	}
}

// TestStatus_HasAll 测试包含所有状态
func TestStatus_HasAll(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	if !s.HasAll(StatusUserDisabled, StatusSysHidden) {
		t.Error("Expected status to have all flags")
	}
	if s.HasAll(StatusUserDisabled, StatusSysHidden, StatusSysDeleted) {
		t.Error("Expected status to not have all flags")
	}
}

// TestStatus_SetMultiple 测试批量设置
func TestStatus_SetMultiple(t *testing.T) {
	var s Status
	s.SetMultiple(StatusUserDisabled, StatusSysHidden, StatusAdmDisabled)
	if !s.HasAll(StatusUserDisabled, StatusSysHidden, StatusAdmDisabled) {
		t.Error("Expected status to have all set flags")
	}
}

// TestStatus_UnsetMultiple 测试批量取消
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

// TestStatus_CanVerified 测试验证状态检查
func TestStatus_CanVerified(t *testing.T) {
	s := StatusNone
	if !s.CanVerified() {
		t.Error("Expected StatusNone to be normal")
	}

	s = StatusUserDisabled
	if s.CanVerified() {
		t.Error("Expected status with UserDisabled to not be normal")
	}

	s = StatusSysHidden
	if s.CanVerified() {
		t.Error("Expected status with SysHidden to not be normal")
	}
}

// TestStatus_JSON 测试 JSON 序列化和反序列化
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

// TestStatus_Value 测试数据库 Value 接口
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

// TestStatus_Scan 测试数据库 Scan 接口
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
		{
			name:    "unsupported type",
			input:   "invalid",
			wantErr: true,
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
			if !tt.wantErr && s != tt.want {
				t.Errorf("Scan() got = %v, want %v", s, tt.want)
			}
		})
	}
}

// TestStatus_HelperMethods 测试辅助方法
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

// TestStatus_CanVisible 测试可见性检查
func TestStatus_CanVisible(t *testing.T) {
	// 测试完全正常状态（应该可见）
	s := StatusNone
	if !s.CanVisible() {
		t.Error("Expected StatusNone to be visible")
	}

	// 测试被禁用（不可见）
	s = StatusUserDisabled
	if s.CanVisible() {
		t.Error("Expected disabled status to not be visible")
	}

	s = StatusAdmDisabled
	if s.CanVisible() {
		t.Error("Expected admin disabled status to not be visible")
	}

	s = StatusSysDisabled
	if s.CanVisible() {
		t.Error("Expected system disabled status to not be visible")
	}

	// 测试被删除（不可见）
	s = StatusUserDeleted
	if s.CanVisible() {
		t.Error("Expected deleted status to not be visible")
	}

	// 测试被隐藏（不可见）
	s = StatusUserHidden
	if s.CanVisible() {
		t.Error("Expected hidden status to not be visible")
	}

	s = StatusAdmHidden
	if s.CanVisible() {
		t.Error("Expected admin hidden status to not be visible")
	}

	s = StatusSysHidden
	if s.CanVisible() {
		t.Error("Expected system hidden status to not be visible")
	}

	// 测试组合状态（禁用+隐藏，不可见）
	s = StatusUserDisabled | StatusUserHidden
	if s.CanVisible() {
		t.Error("Expected disabled and hidden status to not be visible")
	}

	// 测试组合状态（删除+隐藏，不可见）
	s = StatusSysDeleted | StatusSysHidden
	if s.CanVisible() {
		t.Error("Expected deleted and hidden status to not be visible")
	}

	// 测试未验证状态（应该可见，因为 CanVisible 不检查验证状态）
	s = StatusSysUnverified
	if !s.CanVisible() {
		t.Error("Expected unverified status to be visible (CanVisible doesn't check verification)")
	}

	// 测试多级别状态叠加
	s = StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled
	if s.CanVisible() {
		t.Error("Expected all levels disabled to not be visible")
	}

	// 测试边界情况：只要有一个禁用/删除/隐藏就不可见
	s = StatusUserDisabled | StatusSysUnverified // 禁用+未验证
	if s.CanVisible() {
		t.Error("Expected disabled (even with unverified) to not be visible")
	}
}

// TestStatus_CanEnable 测试启用状态检查
func TestStatus_CanEnable(t *testing.T) {
	testCases := []struct {
		name   string
		status Status
		expect bool
	}{
		{"None", StatusNone, true},
		{"Only SysDisabled", StatusSysDisabled, false},
		{"Only AdmDisabled", StatusAdmDisabled, false},
		{"Only UserDisabled", StatusUserDisabled, false},
		{"Only SysDeleted", StatusSysDeleted, false},
		{"Only AdmDeleted", StatusAdmDeleted, false},
		{"Only UserDeleted", StatusUserDeleted, false},
		{"SysDisabled + AdmDeleted", StatusSysDisabled | StatusAdmDeleted, false},
		{"Hidden only", StatusSysHidden, true},
		{"Unverified only", StatusSysUnverified, true},
		{"Hidden + Unverified", StatusSysHidden | StatusSysUnverified, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.status.CanEnable() != tc.expect {
				t.Errorf("Expected %s CanEnable to be %v, got %v", tc.name, tc.expect, tc.status.CanEnable())
			}
		})
	}
}

// TestStatus_PredefinedCombinations 测试预定义组合
func TestStatus_PredefinedCombinations(t *testing.T) {
	// 测试软删除组合
	s := StatusSysDeleted | StatusSysHidden
	if !s.HasAll(StatusSysDeleted, StatusSysHidden) {
		t.Error("Expected to have SysDeleted and SysHidden")
	}

	// 测试全禁用组合
	s = StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled
	if !s.HasAll(StatusSysDisabled, StatusAdmDisabled, StatusUserDisabled) {
		t.Error("Expected to have all disabled flags")
	}
}

// TestStatus_RealWorldScenario 测试真实场景
func TestStatus_RealWorldScenario(t *testing.T) {
	// 模拟用户状态管理
	var userStatus Status

	// 新用户注册，默认正常状态
	if !userStatus.CanVerified() {
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
}

// TestStatus_Merge 测试合并操作
func TestStatus_Merge(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted
	s.Merge(StatusUserDisabled | StatusAdmDeleted)

	if !s.Contain(StatusUserDisabled) {
		t.Error("Expected to contain StatusUserDisabled")
	}
	if !s.Contain(StatusAdmDeleted) {
		t.Error("Expected to contain StatusAdmDeleted")
	}
	if s.Contain(StatusSysHidden) {
		t.Error("Expected to not contain StatusSysHidden after merge")
	}
}

// TestStatus_Clear 测试清空操作
func TestStatus_Clear(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden
	s.Clear()

	if s != StatusNone {
		t.Errorf("Expected StatusNone after clear, got %v", s)
	}
}

// TestStatus_Equal 测试相等性检查
func TestStatus_Equal(t *testing.T) {
	s1 := StatusUserDisabled | StatusSysHidden
	s2 := StatusUserDisabled | StatusSysHidden
	s3 := StatusUserDisabled

	if !s1.Equal(s2) {
		t.Error("Expected s1 to equal s2")
	}
	if s1.Equal(s3) {
		t.Error("Expected s1 to not equal s3")
	}
}

// TestStatus_BitOperations 测试位运算正确性
func TestStatus_BitOperations(t *testing.T) {
	// 验证状态位定义正确
	if StatusSysDeleted != 1 {
		t.Errorf("StatusSysDeleted should be 1, got %d", StatusSysDeleted)
	}
	if StatusAdmDeleted != 2 {
		t.Errorf("StatusAdmDeleted should be 2, got %d", StatusAdmDeleted)
	}
	if StatusUserDeleted != 4 {
		t.Errorf("StatusUserDeleted should be 4, got %d", StatusUserDeleted)
	}

	// 验证位运算不冲突
	s := StatusNone
	s.Set(StatusSysDeleted)
	s.Set(StatusAdmDeleted)

	if !s.HasAll(StatusSysDeleted, StatusAdmDeleted) {
		t.Error("Expected to have both flags")
	}
}

// TestStatus_EdgeCases 测试边缘情况
func TestStatus_EdgeCases(t *testing.T) {
	t.Run("Zero value", func(t *testing.T) {
		var s Status
		if s != StatusNone {
			t.Error("Zero value should be StatusNone")
		}
		if !s.CanVerified() {
			t.Error("Zero value should be verified")
		}
	})

	t.Run("All flags", func(t *testing.T) {
		s := StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted |
			StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled |
			StatusSysHidden | StatusAdmHidden | StatusUserHidden |
			StatusSysUnverified | StatusAdmUnverified | StatusUserUnverified

		if !s.IsDeleted() || !s.IsDisable() || !s.IsHidden() || !s.IsUnverified() {
			t.Error("Expected all checks to be true")
		}
		if s.CanVerified() {
			t.Error("Expected CanVerified to be false with all flags")
		}
	})
}

// TestStatus_JSONBytes 测试 []byte 类型的 Scan
func TestStatus_JSONBytes(t *testing.T) {
	var s Status
	data := []byte("42")

	err := s.Scan(data)
	if err != nil {
		t.Fatalf("Scan([]byte) failed: %v", err)
	}

	if s != Status(42) {
		t.Errorf("Expected Status(42), got %v", s)
	}
}

// TestStatus_ConcurrentRead 测试并发读取（安全）
func TestStatus_ConcurrentRead(t *testing.T) {
	s := StatusUserDisabled | StatusSysHidden | StatusAdmDeleted

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 并发读取是安全的
			_ = s.IsDisable()
			_ = s.IsHidden()
			_ = s.IsDeleted()
			_ = s.CanVisible()
		}()
	}
	wg.Wait()
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

// BenchmarkStatus_HasAny 基准测试：HasAny 检查
func BenchmarkStatus_HasAny(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.HasAny(StatusUserDisabled, StatusAdmDisabled)
	}
}

// BenchmarkStatus_CanVerified 基准测试：CanVerified 检查
func BenchmarkStatus_CanVerified(b *testing.B) {
	s := StatusNone
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.CanVerified()
	}
}

// BenchmarkStatus_JSON 基准测试：JSON 序列化
func BenchmarkStatus_JSON(b *testing.B) {
	s := StatusUserDisabled | StatusSysHidden
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(s)
	}
}
