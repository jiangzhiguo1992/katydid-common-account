# Status 状态类型

## 概述

`Status` 是一个基于位运算的高性能状态管理类型，支持多状态叠加和高效查询。采用 int64 类型，支持最多 63 种状态位的组合。

## 核心特性

### 1. 高性能设计
- **内存占用**：固定 8 字节
- **时间复杂度**：所有操作 O(1)
- **零内存分配**：位运算无需额外分配
- **线程安全**：值类型天然线程安全

### 2. 状态分层设计

#### 三层管理结构
- **System (系统级)**：最高优先级，系统自动管理
- **Admin (管理员级)**：中等优先级，管理员手动操作
- **User (用户级)**：最低优先级，用户自主控制

#### 四类状态
1. **Deleted (删除)**：软删除标记
2. **Disabled (禁用)**：暂时不可用
3. **Hidden (隐藏)**：不对外展示
4. **Unverified (未验证)**：需要验证

### 3. 预定义状态常量

```go
// 基础状态位 (位 0-11)
StatusSysDeleted      // 系统删除
StatusAdmDeleted      // 管理员删除
StatusUserDeleted     // 用户删除
StatusSysDisabled     // 系统禁用
StatusAdmDisabled     // 管理员禁用
StatusUserDisabled    // 用户禁用
StatusSysHidden       // 系统隐藏
StatusAdmHidden       // 管理员隐藏
StatusUserHidden      // 用户隐藏
StatusSysUnverified   // 系统未验证
StatusAdmUnverified   // 管理员未验证
StatusUserUnverified  // 用户未验证

// 组合常量 (性能优化)
StatusAllDeleted      // 所有删除状态
StatusAllDisabled     // 所有禁用状态
StatusAllHidden       // 所有隐藏状态
StatusAllUnverified   // 所有未验证状态
StatusAllSystem       // 所有系统级状态
StatusAllAdmin        // 所有管理员级状态
StatusAllUser         // 所有用户级状态

// 扩展位
StatusExpand51        // 扩展起始位（位 12 开始）
```

## 使用示例

### 基础操作

```go
package main

import (
    "fmt"
    "github.com/katydid/katydid-common-account/pkg/types"
)

func main() {
    // 创建状态
    var status types.Status
    
    // 设置单个状态
    status.Set(types.StatusUserDisabled)
    fmt.Printf("状态值: %s\n", status) // 输出: Status(32)
    
    // 追加多个状态
    status.Set(types.StatusSysHidden)
    status.Set(types.StatusAdmUnverified)
    
    // 批量设置
    var status2 types.Status
    status2.SetMultiple(
        types.StatusUserDisabled,
        types.StatusSysHidden,
        types.StatusAdmUnverified,
    )
    
    // 检查状态
    if status.Contain(types.StatusUserDisabled) {
        fmt.Println("包含用户禁用状态")
    }
    
    // 移除状态
    status.Unset(types.StatusUserDisabled)
    
    // 批量移除
    status.UnsetMultiple(types.StatusSysHidden, types.StatusAdmUnverified)
    
    // 清除所有状态
    status.Clear()
}
```

### 状态检查

```go
// 检查任意一个状态
if status.HasAny(types.StatusUserDisabled, types.StatusAdmDisabled) {
    fmt.Println("被禁用了（用户或管理员）")
}

// 检查所有状态
if status.HasAll(types.StatusUserDisabled, types.StatusSysHidden) {
    fmt.Println("既被禁用又被隐藏")
}

// 使用预定义组合常量（性能更优）
if status.Contain(types.StatusAllDeleted) {
    fmt.Println("检查所有删除状态")
}
```

### 业务场景方法

```go
// 业务可用性检查
if status.CanEnable() {
    fmt.Println("可以启用")
}

if status.CanVisible() {
    fmt.Println("可以对外展示")
}

if status.CanVerified() {
    fmt.Println("已通过验证，完全可用")
}

// 业务状态检查
if status.IsDeleted() {
    fmt.Println("已被删除")
}

if status.IsDisable() {
    fmt.Println("已被禁用")
}

if status.IsHidden() {
    fmt.Println("已被隐藏")
}

if status.IsUnverified() {
    fmt.Println("未通过验证")
}
```

### 数据库集成

```go
import (
    "database/sql"
    "github.com/katydid/katydid-common-account/pkg/types"
)

type User struct {
    ID     int64
    Name   string
    Status types.Status // 自动支持数据库读写
}

func main() {
    db, _ := sql.Open("mysql", "dsn")
    
    // 写入数据库
    user := User{
        ID:     1,
        Name:   "张三",
        Status: types.StatusUserDisabled | types.StatusSysHidden,
    }
    db.Exec("INSERT INTO users (id, name, status) VALUES (?, ?, ?)",
        user.ID, user.Name, user.Status)
    
    // 从数据库读取
    var loadedUser User
    db.QueryRow("SELECT id, name, status FROM users WHERE id = ?", 1).
        Scan(&loadedUser.ID, &loadedUser.Name, &loadedUser.Status)
    
    fmt.Printf("用户状态: %s\n", loadedUser.Status)
}
```

### JSON 序列化

```go
import (
    "encoding/json"
    "fmt"
    "github.com/katydid/katydid-common-account/pkg/types"
)

type Response struct {
    Code   int          `json:"code"`
    Status types.Status `json:"status"`
}

func main() {
    // 序列化
    resp := Response{
        Code:   200,
        Status: types.StatusUserDisabled | types.StatusSysHidden,
    }
    data, _ := json.Marshal(resp)
    fmt.Println(string(data))
    // 输出: {"code":200,"status":96}
    
    // 反序列化
    var loaded Response
    json.Unmarshal(data, &loaded)
    fmt.Printf("状态: %s\n", loaded.Status)
}
```

### 数据库查询示例

```sql
-- 查询所有已删除的用户（任意级别）
-- StatusAllDeleted = 7 (StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted)
SELECT * FROM users WHERE status & 7 != 0;

-- 查询未被删除且未被禁用的用户
-- StatusAllDeleted = 7, StatusAllDisabled = 56
SELECT * FROM users WHERE status & 63 = 0;

-- 查询用户级禁用的用户
-- StatusUserDisabled = 32
SELECT * FROM users WHERE status & 32 = 32;

-- 查询可见的用户（未删除、未禁用、未隐藏）
-- StatusAllDeleted | StatusAllDisabled | StatusAllHidden = 511
SELECT * FROM users WHERE status & 511 = 0;
```

### 自定义扩展状态

```go
const (
    // 基于 StatusExpand51 扩展自定义状态
    StatusCustomLocked   types.Status = types.StatusExpand51 << 0  // 自定义：锁定
    StatusCustomFrozen   types.Status = types.StatusExpand51 << 1  // 自定义：冻结
    StatusCustomPending  types.Status = types.StatusExpand51 << 2  // 自定义：待处理
)

func main() {
    var status types.Status
    status.Set(StatusCustomLocked)
    
    if status.Contain(StatusCustomLocked) {
        fmt.Println("账户已锁定")
    }
}
```

## 设计原则

### 1. 位运算设计

```
位位置    状态              值
0         StatusSysDeleted   1
1         StatusAdmDeleted   2
2         StatusUserDeleted  4
3         StatusSysDisabled  8
4         StatusAdmDisabled  16
5         StatusUserDisabled 32
6         StatusSysHidden    64
7         StatusAdmHidden    128
8         StatusUserHidden   256
9         StatusSysUnverified 512
10        StatusAdmUnverified 1024
11        StatusUserUnverified 2048
12-62     扩展位（自定义）
63        符号位（不可用）
```

### 2. 业务规则

```
CanEnable()    = !IsDeleted() && !IsDisable()
CanVisible()   = CanEnable() && !IsHidden()
CanVerified()  = CanVisible() && !IsUnverified()
```

### 3. 性能优化建议

1. **使用预定义组合常量**
   ```go
   // 推荐：使用组合常量
   if status.Contain(types.StatusAllDeleted) {
       // ...
   }
   
   // 不推荐：手动组合
   if status.HasAny(types.StatusSysDeleted, types.StatusAdmDeleted, types.StatusUserDeleted) {
       // ...
   }
   ```

2. **批量操作优于单次操作**
   ```go
   // 推荐：批量设置
   status.SetMultiple(types.StatusUserDisabled, types.StatusSysHidden)
   
   // 不推荐：多次单独设置
   status.Set(types.StatusUserDisabled)
   status.Set(types.StatusSysHidden)
   ```

3. **直接位运算最快**
   ```go
   // 最快：直接位运算
   if status & types.StatusAllDeleted != 0 {
       // ...
   }
   
   // 次之：方法调用
   if status.IsDeleted() {
       // ...
   }
   ```

## 错误处理

### 运行时验证

```go
// 验证状态值合法性
status := types.Status(12345)
if !status.IsValid() {
    log.Printf("错误：状态值 %d 不合法", status)
}

// 数据库读取时自动验证
var status types.Status
err := status.Scan(int64(-1))  // 返回错误：负数不允许
if err != nil {
    log.Printf("扫描错误: %v", err)
}

// JSON 反序列化时自动验证
var status types.Status
err := json.Unmarshal([]byte("-1"), &status)  // 返回错误：负数不允许
if err != nil {
    log.Printf("反序列化错误: %v", err)
}
```

### 常见错误信息

- `invalid Status value: negative number %d is not allowed (sign bit conflict)` - 负数值冲突
- `invalid Status value: %d exceeds maximum allowed value %d (overflow)` - 值溢出
- `failed to unmarshal Status from JSON: invalid format, expected integer number` - JSON 格式错误
- `cannot scan type %T into Status: unsupported database type` - 不支持的数据库类型

## 最佳实践

### 1. 数据库设计

```sql
-- 建表时使用 BIGINT 类型
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    name VARCHAR(100),
    status BIGINT NOT NULL DEFAULT 0,
    INDEX idx_status (status)  -- 支持高效索引查询
);
```

### 2. API 设计

```go
// 请求结构
type UpdateUserStatusRequest struct {
    UserID int64        `json:"user_id"`
    Status types.Status `json:"status"`
}

// 响应结构
type UserResponse struct {
    ID     int64        `json:"id"`
    Name   string       `json:"name"`
    Status types.Status `json:"status"`
    
    // 可选：提供更友好的状态描述
    CanEnable   bool `json:"can_enable"`
    CanVisible  bool `json:"can_visible"`
    CanVerified bool `json:"can_verified"`
}

func (r *UserResponse) FromUser(user *User) {
    r.ID = user.ID
    r.Name = user.Name
    r.Status = user.Status
    r.CanEnable = user.Status.CanEnable()
    r.CanVisible = user.Status.CanVisible()
    r.CanVerified = user.Status.CanVerified()
}
```

### 3. 日志记录

```go
import "log"

// 记录状态变化
func UpdateUserStatus(userID int64, oldStatus, newStatus types.Status) {
    log.Printf("[状态变更] 用户ID: %d, 旧状态: %s, 新状态: %s",
        userID, oldStatus, newStatus)
    
    // 可以进一步分析变更内容
    if !oldStatus.IsDeleted() && newStatus.IsDeleted() {
        log.Printf("[警告] 用户 %d 被标记为删除", userID)
    }
}
```

### 4. 测试建议

```go
func TestUserStatus(t *testing.T) {
    user := &User{Status: types.StatusNone}
    
    // 测试状态设置
    user.Status.Set(types.StatusUserDisabled)
    if !user.Status.Contain(types.StatusUserDisabled) {
        t.Error("设置用户禁用状态失败")
    }
    
    // 测试业务逻辑
    if user.Status.CanEnable() {
        t.Error("禁用状态不应该可以启用")
    }
}
```

## 性能基准

```
BenchmarkStatusSet-8              1000000000    0.3 ns/op    0 B/op    0 allocs/op
BenchmarkStatusHasAny-8           1000000000    1.2 ns/op    0 B/op    0 allocs/op
BenchmarkStatusHasAll-8           1000000000    1.2 ns/op    0 B/op    0 allocs/op
BenchmarkStatusIsDeleted-8        1000000000    0.5 ns/op    0 B/op    0 allocs/op
BenchmarkStatusMarshalJSON-8      50000000      30 ns/op     8 B/op    1 allocs/op
BenchmarkStatusUnmarshalJSON-8    30000000      45 ns/op     0 B/op    0 allocs/op
```

## 注意事项

1. **不要使用负数**：负数会导致符号位冲突，运行时会返回错误
2. **扩展位限制**：自定义状态位不要超过位 62（最大有效位）
3. **数据库类型**：确保数据库字段使用 BIGINT 类型
4. **并发安全**：值类型天然并发安全，但指针操作需要加锁
5. **位运算理解**：建议团队成员理解基本的位运算知识

## 迁移指南

### 从字符串状态迁移

```go
// 旧代码：字符串状态
type User struct {
    Status string // "active", "disabled", "deleted"
}

// 新代码：位运算状态
type User struct {
    Status types.Status
}

// 迁移函数
func MigrateStatus(oldStatus string) types.Status {
    switch oldStatus {
    case "active":
        return types.StatusNone
    case "disabled":
        return types.StatusUserDisabled
    case "deleted":
        return types.StatusUserDeleted
    default:
        return types.StatusNone
    }
}
```

## 常见问题

**Q: 为什么使用 int64 而不是 uint64？**
A: int64 更兼容数据库和 JSON，且 63 位已经足够使用。

**Q: 可以同时设置多个级别的相同类型状态吗？**
A: 可以，例如同时设置系统删除和用户删除，这在某些场景下是合理的。

**Q: 如何在前端展示状态？**
A: 建议后端返回状态数值的同时，提供友好的布尔字段（如 `can_enable`）。

**Q: 性能如何？**
A: 所有操作都是 O(1) 时间复杂度，零内存分配，性能极高。

## 版本历史

- **v1.0.0** (2025-10-17): 初始版本，支持基础状态管理和位运算操作

