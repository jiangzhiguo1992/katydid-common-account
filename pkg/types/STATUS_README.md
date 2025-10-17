# Status 状态位管理模块

## 概述

`Status` 是一个基于位运算的状态管理类型，支持多状态并存和高效的状态检查。使用 `int64` 作为底层类型，最多支持 63 种状态位的组合。

## 核心特性

### 1. 位运算设计
- 基于 `int64`，固定 8 字节内存占用
- 支持多状态叠加（位或运算）
- O(1) 时间复杂度的状态检查
- 无内存分配的纯位运算

### 2. 分层状态管理
- **System（系统级）**: 最高优先级，系统自动管理
- **Admin（管理员级）**: 中等优先级，管理员手动操作
- **User（用户级）**: 最低优先级，用户自主控制

### 3. 四类预定义状态
- **Deleted（删除）**: 软删除标记，支持回收站
- **Disabled（禁用）**: 暂时不可用，可恢复
- **Hidden（隐藏）**: 不对外展示
- **Unverified（未验证）**: 等待验证或审核

### 4. 业务语义方法
- `CanEnable()`: 检查是否可启用
- `CanVisible()`: 检查是否可见
- `CanVerified()`: 检查是否已验证

## 快速开始

### 基本用法

```go
// 创建状态
var status types.Status

// 设置状态（追加）
status.Set(types.StatusUserDisabled)
status.Set(types.StatusSysHidden)

// 检查状态
if status.Contain(types.StatusUserDisabled) {
    fmt.Println("用户已禁用")
}

// 移除状态
status.Unset(types.StatusUserDisabled)

// 切换状态
status.Toggle(types.StatusUserHidden)
```

### 状态常量

#### 删除状态（位 0-2）
```go
types.StatusSysDeleted   // 系统删除：系统自动标记，通常不可恢复
types.StatusAdmDeleted   // 管理员删除：管理员操作，可能支持恢复
types.StatusUserDeleted  // 用户删除：用户主动删除，通常可恢复
```

#### 禁用状态（位 3-5）
```go
types.StatusSysDisabled  // 系统禁用：系统检测异常后自动禁用
types.StatusAdmDisabled  // 管理员禁用：管理员手动禁用
types.StatusUserDisabled // 用户禁用：用户主动禁用（如账号冻结）
```

#### 隐藏状态（位 6-8）
```go
types.StatusSysHidden    // 系统隐藏：系统根据规则自动隐藏
types.StatusAdmHidden    // 管理员隐藏：管理员手动隐藏内容
types.StatusUserHidden   // 用户隐藏：用户设置为私密/不公开
```

#### 未验证状态（位 9-11）
```go
types.StatusSysUnverified  // 系统未验证：等待系统自动验证
types.StatusAdmUnverified  // 管理员未验证：等待管理员审核
types.StatusUserUnverified // 用户未验证：等待用户完成验证（如邮箱）
```

### 预定义组合常量

```go
// 所有删除状态
types.StatusAllDeleted = StatusSysDeleted | StatusAdmDeleted | StatusUserDeleted

// 所有禁用状态
types.StatusAllDisabled = StatusSysDisabled | StatusAdmDisabled | StatusUserDisabled

// 所有隐藏状态
types.StatusAllHidden = StatusSysHidden | StatusAdmHidden | StatusUserHidden

// 所有未验证状态
types.StatusAllUnverified = StatusSysUnverified | StatusAdmUnverified | StatusUserUnverified

// 所有系统级状态
types.StatusAllSystem = StatusSysDeleted | StatusSysDisabled | StatusSysHidden | StatusSysUnverified

// 所有管理员级状态
types.StatusAllAdmin = StatusAdmDeleted | StatusAdmDisabled | StatusAdmHidden | StatusAdmUnverified

// 所有用户级状态
types.StatusAllUser = StatusUserDeleted | StatusUserDisabled | StatusUserHidden | StatusUserUnverified
```

## 核心操作

### 1. 设置和移除状态

```go
var s types.Status

// Set: 追加状态（保留原有状态）
s.Set(types.StatusUserDisabled)    // s = 32
s.Set(types.StatusSysHidden)        // s = 32 | 64 = 96

// Unset: 移除指定状态
s.Unset(types.StatusUserDisabled)   // s = 64

// Toggle: 切换状态（有则删除，无则添加）
s.Toggle(types.StatusUserHidden)    // 首次：添加
s.Toggle(types.StatusUserHidden)    // 再次：移除

// Clear: 清除所有状态
s.Clear()                            // s = 0
```

### 2. 批量操作

```go
var s types.Status

// SetMultiple: 批量设置
s.SetMultiple(
    types.StatusUserDisabled,
    types.StatusSysHidden,
    types.StatusAdmUnverified,
)

// UnsetMultiple: 批量移除
s.UnsetMultiple(
    types.StatusUserDisabled,
    types.StatusSysHidden,
)
```

### 3. 状态检查

```go
s := types.StatusUserDisabled | types.StatusSysHidden

// Contain: 检查是否包含所有指定状态
if s.Contain(types.StatusUserDisabled) {
    fmt.Println("包含用户禁用状态")
}

// HasAny: 检查是否包含任意一个状态
if s.HasAny(types.StatusUserDisabled, types.StatusAdmDisabled) {
    fmt.Println("包含至少一个禁用状态")
}

// HasAll: 检查是否包含所有状态
if s.HasAll(types.StatusUserDisabled, types.StatusSysHidden) {
    fmt.Println("同时包含两个状态")
}

// Equal: 检查状态是否完全相等
s2 := types.StatusUserDisabled | types.StatusSysHidden
if s.Equal(s2) {
    fmt.Println("状态完全一致")
}
```

### 4. 业务语义检查

```go
var s types.Status

// IsDeleted: 是否被删除（任意级别）
if s.IsDeleted() {
    // 不应该访问或展示
}

// IsDisable: 是否被禁用（任意级别）
if s.IsDisable() {
    // 暂时不可用
}

// IsHidden: 是否被隐藏（任意级别）
if s.IsHidden() {
    // 不对外展示
}

// IsUnverified: 是否未验证（任意级别）
if s.IsUnverified() {
    // 需要验证或审核
}

// CanEnable: 是否可启用（未删除且未禁用）
if s.CanEnable() {
    // 可以启用该功能
}

// CanVisible: 是否可见（可启用且未隐藏）
if s.CanVisible() {
    // 可以对外展示
}

// CanVerified: 是否已验证（可见且已验证）
if s.CanVerified() {
    // 完全可用
}
```

## 数据库使用

### 在模型中使用

```go
type Article struct {
    ID        uint64       `gorm:"primaryKey"`
    Title     string       `gorm:"size:200"`
    Status    types.Status `gorm:"type:bigint;index"` // 使用索引提升查询性能
    CreatedAt time.Time
}

// 创建文章（默认状态）
article := &Article{
    Title:  "示例文章",
    Status: types.StatusNone, // 正常状态
}
db.Create(article)

// 管理员隐藏文章
article.Status.Set(types.StatusAdmHidden)
db.Save(article)

// 查询可见的文章
var articles []Article
db.Where("status & ? = 0", types.StatusAllHidden).Find(&articles)

// 查询已删除的文章
db.Where("status & ? != 0", types.StatusAllDeleted).Find(&articles)

// 查询正常状态的文章（未删除、未禁用、未隐藏）
normalMask := types.StatusAllDeleted | types.StatusAllDisabled | types.StatusAllHidden
db.Where("status & ? = 0", normalMask).Find(&articles)
```

### 状态查询示例

```go
// 1. 查询所有正常可见的文章
db.Model(&Article{}).Where("status = ?", types.StatusNone).Find(&articles)

// 2. 查询被管理员操作过的文章（任意管理员级状态）
db.Model(&Article{}).Where("status & ? != 0", types.StatusAllAdmin).Find(&articles)

// 3. 查询等待审核的文章
db.Model(&Article{}).Where("status & ? != 0", types.StatusAllUnverified).Find(&articles)

// 4. 排除已删除的文章
db.Model(&Article{}).Where("status & ? = 0", types.StatusAllDeleted).Find(&articles)

// 5. 查询用户自己删除的文章（回收站）
db.Model(&Article{}).Where("status & ? = ?", 
    types.StatusAllDeleted, types.StatusUserDeleted).Find(&articles)
```

## 业务场景示例

### 场景1：内容审核流程

```go
// 用户发布文章，默认需要审核
article := &Article{
    Title:  "新文章",
    Status: types.StatusAdmUnverified, // 等待管理员审核
}
db.Create(article)

// 管理员审核通过
article.Status.Unset(types.StatusAdmUnverified)
db.Save(article)

// 管理员审核不通过并隐藏
article.Status.Set(types.StatusAdmHidden)
article.Status.Unset(types.StatusAdmUnverified)
db.Save(article)
```

### 场景2：用户权限管理

```go
// 系统检测到异常行为，自动禁用账号
user.Status.Set(types.StatusSysDisabled)
db.Save(user)

// 用户申诉，管理员解除禁用
user.Status.Unset(types.StatusSysDisabled)
db.Save(user)

// 检查用户是否可以登录
if !user.Status.CanEnable() {
    return errors.New("账号已被禁用或删除")
}
```

### 场景3：软删除和回收站

```go
// 用户删除文章（进入回收站）
article.Status.Set(types.StatusUserDeleted)
db.Save(article)

// 查询回收站中的文章
var deletedArticles []Article
db.Where("status & ? = ?", 
    types.StatusAllDeleted, 
    types.StatusUserDeleted,
).Find(&deletedArticles)

// 从回收站恢复
article.Status.Unset(types.StatusUserDeleted)
db.Save(article)

// 管理员永久删除（无法恢复）
article.Status.Set(types.StatusSysDeleted)
db.Save(article)
// 或直接物理删除
db.Delete(article)
```

### 场景4：内容可见性控制

```go
// 用户设置文章为私密
article.Status.Set(types.StatusUserHidden)

// 检查是否对外可见
if article.Status.CanVisible() {
    // 展示文章
} else {
    // 隐藏或显示"内容不可见"
}

// 管理员强制公开（移除所有隐藏状态）
article.Status.Unset(types.StatusUserHidden)
article.Status.Unset(types.StatusAdmHidden)
article.Status.Unset(types.StatusSysHidden)
```

## 自定义状态扩展

```go
// 从位 12 开始定义自定义状态
const (
    // 业务自定义状态
    StatusCustom1 types.Status = types.StatusExpand51 << 0  // 位 12
    StatusCustom2 types.Status = types.StatusExpand51 << 1  // 位 13
    StatusCustom3 types.Status = types.StatusExpand51 << 2  // 位 14
    // ... 最多可以定义 51 个自定义状态（位 12-62）
)

// 使用自定义状态
var s types.Status
s.Set(StatusCustom1)
if s.Contain(StatusCustom1) {
    // 处理自定义状态
}
```

## 性能优化

### 1. 使用预定义组合常量

```go
// 推荐：使用预定义常量（单次位运算）
if status.HasAny(types.StatusAllDeleted) {
    // 检查任意删除状态
}

// 不推荐：每次调用都要合并（多次位运算）
if status.HasAny(
    types.StatusSysDeleted,
    types.StatusAdmDeleted,
    types.StatusUserDeleted,
) {
    // 性能略低
}
```

### 2. 数据库索引优化

```go
// 在 status 字段上创建索引
type Article struct {
    Status types.Status `gorm:"type:bigint;index"` // 添加索引
}

// 使用位运算查询时，索引仍然有效
db.Where("status & ? = 0", types.StatusAllDeleted).Find(&articles)
```

### 3. 批量操作

```go
// 推荐：批量设置
s.SetMultiple(status1, status2, status3)

// 不推荐：多次单独设置
s.Set(status1)
s.Set(status2)
s.Set(status3)
```

## 最佳实践

### 1. 状态分层使用

```go
// 系统级：自动化操作
if detectSpam(content) {
    status.Set(types.StatusSysHidden)
}

// 管理员级：人工干预
if adminReview.IsRejected() {
    status.Set(types.StatusAdmDeleted)
}

// 用户级：用户自主控制
if user.WantsPrivate() {
    status.Set(types.StatusUserHidden)
}
```

### 2. 业务语义优先

```go
// 推荐：使用业务语义方法
if article.Status.CanVisible() {
    renderArticle(article)
}

// 不推荐：直接位运算判断
if article.Status & types.StatusAllHidden == 0 && 
   article.Status & types.StatusAllDeleted == 0 {
    renderArticle(article)
}
```

### 3. 状态验证

```go
// 从外部输入创建状态时，进行验证
status := types.Status(userInput)
if !status.IsValid() {
    return errors.New("invalid status value")
}
```

### 4. 错误处理

```go
// 数据库读取时检查错误
var article Article
if err := db.First(&article, id).Error; err != nil {
    return err
}

// 验证状态是否合法
if !article.Status.IsValid() {
    log.Warn("检测到非法状态值", article.Status)
}
```

## 常见问题

### Q: 为什么 Set 方法需要指针接收者？
A: 因为需要修改状态本身。值接收者只会修改副本，不会影响原始值。

```go
// 正确用法
var s types.Status
s.Set(types.StatusUserDisabled) // s 被修改

// 错误用法（编译错误）
s := types.Status(0)
s.Set(types.StatusUserDisabled) // 这样写会修改副本，不影响 s
```

### Q: 如何判断状态是否为"正常"？
A: 有两种方式：
```go
// 方式1：检查是否为零值
if status == types.StatusNone {
    // 完全正常
}

// 方式2：检查是否可用
if status.CanVerified() {
    // 业务上可用
}
```

### Q: 多个状态之间是否有优先级？
A: 位运算没有优先级概念，所有状态平等。优先级由业务逻辑决定：

```go
// 业务逻辑示例：删除优先级最高
if status.IsDeleted() {
    return "已删除"
} else if status.IsDisable() {
    return "已禁用"
} else if status.IsHidden() {
    return "已隐藏"
}
```

### Q: 为什么不能使用负数？
A: `int64` 的符号位（第 63 位）为 1 表示负数，会与状态位冲突，导致不可预期的行为。所有状态值应该 >= 0。

### Q: 如何重置所有状态？
A: 使用 `Clear()` 方法或直接赋值为 `StatusNone`：

```go
// 方式1
status.Clear()

// 方式2
status = types.StatusNone
```
