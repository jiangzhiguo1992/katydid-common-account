# Status 状态管理

## 概述

`Status` 类型是一个基于位运算的状态管理系统，支持多个状态的叠加和组合。适用于需要同时跟踪多个状态标志的场景。

## 特性

- ✅ 使用位运算实现高效的状态管理
- ✅ 支持多状态叠加（如：启用 + 可见 + 活跃）
- ✅ 提供丰富的 API 进行状态操作
- ✅ 支持数据库存储（实现了 `driver.Valuer` 和 `sql.Scanner`）
- ✅ 支持 JSON 序列化
- ✅ 内置常用状态组合

## 预定义状态

### 基础状态

| 状态 | 描述 |
|-----|------|
| `StatusNone` | 无状态（0） |
| `StatusEnabled` | 启用状态 |
| `StatusVisible` | 可见状态 |
| `StatusLocked` | 锁定状态 |
| `StatusDeleted` | 删除状态 |
| `StatusActive` | 活跃状态 |
| `StatusVerified` | 已验证状态 |
| `StatusPublished` | 已发布状态 |
| `StatusArchived` | 已归档状态 |
| `StatusFeatured` | 特色/推荐状态 |
| `StatusPinned` | 置顶状态 |
| `StatusHidden` | 隐藏状态 |
| `StatusSuspended` | 暂停/冻结状态 |
| `StatusPending` | 待处理状态 |
| `StatusApproved` | 已批准状态 |
| `StatusRejected` | 已拒绝状态 |
| `StatusDraft` | 草稿状态 |

### 状态组合

| 组合 | 包含的状态 | 描述 |
|-----|-----------|------|
| `StatusNormal` | `StatusEnabled` + `StatusVisible` | 正常状态 |
| `StatusPublicActive` | `StatusEnabled` + `StatusVisible` + `StatusActive` + `StatusPublished` | 公开活跃状态 |
| `StatusPendingReview` | `StatusEnabled` + `StatusPending` | 待审核状态 |
| `StatusSoftDeleted` | `StatusDeleted` + `StatusHidden` | 软删除状态 |

## 基本用法

### 设置状态

```go
import "katydid-common-account/pkg/types"

var status types.Status

// 设置单个状态
status.Set(types.StatusEnabled)
status.Set(types.StatusVisible)

// 批量设置多个状态
status.SetMultiple(types.StatusEnabled, types.StatusVisible, types.StatusActive)

// 使用预定义组合
status = types.StatusNormal // 等同于 StatusEnabled | StatusVisible
```

### 取消状态

```go
// 取消单个状态
status.Unset(types.StatusEnabled)

// 批量取消多个状态
status.UnsetMultiple(types.StatusEnabled, types.StatusVisible)

// 清除所有状态
status.Clear()
```

### 切换状态

```go
// 如果有该状态则取消，没有则设置
status.Toggle(types.StatusEnabled)
```

### 检查状态

```go
// 检查是否包含指定状态
if status.Has(types.StatusEnabled) {
    // 状态包含 Enabled
}

// 检查是否包含任意一个状态
if status.HasAny(types.StatusEnabled, types.StatusVisible) {
    // 至少有一个状态
}

// 检查是否包含所有指定状态
if status.HasAll(types.StatusEnabled, types.StatusVisible) {
    // 同时包含两个状态
}

// 检查是否完全匹配
if status.Is(types.StatusNormal) {
    // 完全等于 StatusNormal
}
```

### 便捷方法

```go
// 内置的便捷检查方法
status.IsEnabled()    // 是否启用
status.IsVisible()    // 是否可见
status.IsLocked()     // 是否锁定
status.IsDeleted()    // 是否删除
status.IsActive()     // 是否活跃
status.IsVerified()   // 是否已验证
status.IsPublished()  // 是否已发布
status.IsNormal()     // 是否正常（启用+可见）
```

## 在模型中使用

### 基础模型

`BaseModel` 已经包含了 `Status` 字段：

```go
type BaseModel struct {
    ID        idgen.ID       `gorm:"primarykey" json:"id"`
    Status    types.Status   `gorm:"column:status;not null;default:0;index" json:"status"`
    CreatedAt time.Time      `gorm:"column:created_at;not null" json:"created_at"`
    UpdatedAt time.Time      `gorm:"column:updated_at;not null" json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}
```

### 使用示例

```go
type User struct {
    models.BaseModel
    Username string `json:"username"`
    Email    string `json:"email"`
}

// 创建用户
user := &User{
    Username: "john",
    Email:    "john@example.com",
}
// BaseModel.BeforeCreate 会自动设置 Status 为 StatusNormal

// 用户完成邮箱验证
user.Status.Set(types.StatusVerified)

// 用户发布内容，成为活跃用户
user.Status.Set(types.StatusActive)

// 检查用户状态
if user.Status.IsNormal() && user.Status.IsVerified() {
    // 正常且已验证的用户
}

// 暂停用户
user.Status.Unset(types.StatusEnabled)
user.Status.Set(types.StatusSuspended)

// 软删除用户
user.Status = types.StatusSoftDeleted
```

## 数据库查询

### 查询特定状态的记录

```go
// 查询所有启用的用户
var users []User
db.Where("status & ? = ?", types.StatusEnabled, types.StatusEnabled).Find(&users)

// 查询正常状态的用户（启用+可见）
db.Where("status & ? = ?", types.StatusNormal, types.StatusNormal).Find(&users)

// 查询未删除的用户
db.Where("status & ? = 0", types.StatusDeleted).Find(&users)
```

### Scope 辅助方法（可选实现）

可以在 repository 中创建辅助方法：

```go
// 查询启用的记录
func WithEnabled(db *gorm.DB) *gorm.DB {
    return db.Where("status & ? = ?", types.StatusEnabled, types.StatusEnabled)
}

// 查询可见的记录
func WithVisible(db *gorm.DB) *gorm.DB {
    return db.Where("status & ? = ?", types.StatusVisible, types.StatusVisible)
}

// 使用
db.Scopes(WithEnabled, WithVisible).Find(&users)
```

## 实际应用场景

### 1. 用户状态管理

```go
// 新用户注册
user.Status = types.StatusNormal

// 邮箱验证后
user.Status.Set(types.StatusVerified)

// 用户活跃
user.Status.Set(types.StatusActive)

// 违规暂停
user.Status.Unset(types.StatusEnabled)
user.Status.Set(types.StatusSuspended)

// 软删除
user.Status.Set(types.StatusDeleted)
user.Status.Set(types.StatusHidden)
```

### 2. 文章/内容管理

```go
// 草稿
article.Status = types.StatusDraft

// 提交审核
article.Status.Set(types.StatusPending)

// 审核通过并发布
article.Status.UnsetMultiple(types.StatusDraft, types.StatusPending)
article.Status.SetMultiple(types.StatusApproved, types.StatusPublished, types.StatusVisible)

// 设为推荐
article.Status.Set(types.StatusFeatured)

// 置顶
article.Status.Set(types.StatusPinned)

// 归档
article.Status.Set(types.StatusArchived)
article.Status.Unset(types.StatusVisible)
```

### 3. 商品状态管理

```go
// 上架商品
product.Status.SetMultiple(types.StatusEnabled, types.StatusVisible, types.StatusPublished)

// 库存不足，暂时下架
product.Status.Unset(types.StatusVisible)

// 锁定商品（禁止修改）
product.Status.Set(types.StatusLocked)

// 特色商品
product.Status.Set(types.StatusFeatured)
```

## JSON 序列化

```go
// 序列化
data, _ := json.Marshal(user)
// {"id":123,"status":3,"username":"john",...}

// 反序列化
json.Unmarshal(data, &user)
```

状态值以整数形式存储，前端可以使用位运算进行判断。

## 性能优势

- **存储效率**：使用单个 uint64 字段存储最多 64 个状态标志
- **查询效率**：使用位运算进行状态过滤，数据库索引友好
- **内存效率**：相比使用多个 bool 字段，大大减少内存占用

## 扩展自定义状态

如果需要添加项目特定的状态：

```go
const (
    StatusCustom1 types.Status = 1 << 20  // 从较大的位开始
    StatusCustom2 types.Status = 1 << 21
    StatusCustom3 types.Status = 1 << 22
)
```

## 注意事项

1. **位运算范围**：使用 `uint64`，最多支持 64 个状态位
2. **数据库类型**：建议使用 `BIGINT` 或 `UNSIGNED BIGINT` 类型
3. **向后兼容**：添加新状态时使用新的位，不要修改已有状态的位值
4. **组合使用**：可以灵活组合多个状态，但要注意业务逻辑的一致性

## 测试

运行测试：

```bash
go test -v katydid-common-account/pkg/types -run TestStatus
```

查看覆盖率：

```bash
go test -v -cover katydid-common-account/pkg/types -run TestStatus
```

