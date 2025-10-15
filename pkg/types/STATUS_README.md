# Status 状态管理

## 概述

`Status` 类型是一个基于位运算的状态管理系统，支持多个状态的叠加和组合。适用于需要同时跟踪多个状态标志的场景。

该系统采用**分级状态设计**，每种状态都有三个级别：
- **系统级别（Sys）**：系统控制的状态，优先级最高
- **管理员级别（Adm）**：管理员控制的状态
- **用户级别（User）**：用户自己控制的状态

## 特性

- ✅ 使用 int64 位运算实现高效的状态管理（最多支持63种状态）
- ✅ 支持多状态叠加（如：系统禁用 + 管理员隐藏）
- ✅ 分级状态控制，权限清晰
- ✅ 提供丰富的 API 进行状态操作
- ✅ 支持数据库存储（实现了 `driver.Valuer` 和 `sql.Scanner`）
- ✅ 支持 JSON 序列化

## 预定义状态

### 基础状态（分级设计）

| 状态类型 | 系统级别 | 管理员级别 | 用户级别 |
|---------|---------|-----------|---------|
| **删除** | `StatusSysDeleted` | `StatusAdmDeleted` | `StatusUserDeleted` |
| **禁用** | `StatusSysDisabled` | `StatusAdmDisabled` | `StatusUserDisabled` |
| **隐藏** | `StatusSysHidden` | `StatusAdmHidden` | `StatusUserHidden` |
| **未验证** | `StatusSysUnverified` | `StatusAdmUnverified` | `StatusUserUnverified` |

**位使用情况**：
- 当前使用了 12 个状态位（0-11位）
- 还有 51 个位可用（12-62位）
- 第 63 位是符号位，不使用

## 基本用法

### 设置状态

```go
import "katydid-common-account/pkg/types"

var status types.Status

// 设置单个状态
status.Set(types.StatusUserDisabled)
status.Set(types.StatusSysHidden)

// 批量设置多个状态
status.SetMultiple(types.StatusUserDisabled, types.StatusSysHidden, types.StatusAdmDisabled)

// 常用组合示例
status = types.StatusSysDeleted | types.StatusSysHidden // 软删除状态
```

### 取消状态

```go
// 取消单个状态
status.Unset(types.StatusUserDisabled)

// 批量取消多个状态
status.UnsetMultiple(types.StatusUserDisabled, types.StatusSysHidden)

// 清除所有状态
status.Clear()
```

### 切换状态

```go
// 如果有该状态则取消，没有则设置
status.Toggle(types.StatusUserDisabled)
```

### 检查状态

```go
// 检查是否包含指定状态
if status.Contain(types.StatusUserDisabled) {
    // 状态包含 UserDisabled
}

// 检查是否包含任意一个状态
if status.HasAny(types.StatusUserDisabled, types.StatusAdmDisabled) {
    // 至少有一个禁用状态
}

// 检查是否包含所有指定状态
if status.HasAll(types.StatusSysDeleted, types.StatusSysHidden) {
    // 同时包含两个状态
}

// 检查是否完全匹配
if status.Equal(types.StatusNone) {
    // 完全等于 StatusNone
}
```

### 便捷方法

```go
// 禁用相关
status.IsDisable()        // 是否被任意级别禁用
status.Contain(types.StatusUserDisabled)   // 是否被用户禁用
status.Contain(types.StatusAdmDisabled)    // 是否被管理员禁用
status.Contain(types.StatusSysDisabled)    // 是否被系统禁用

// 隐藏相关
status.IsHidden()         // 是否被任意级别隐藏

// 删除相关
status.IsDeleted()        // 是否被任意级别删除

// 验证相关
status.IsUnverified()     // 是否未验证

// 综合判断
status.IsNormal()         // 是否正常（未禁用、未删除、未隐藏、已验证）
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

// 创建用户 - 默认为正常状态（Status = 0）
user := &User{
    Username: "john",
    Email:    "john@example.com",
}
// Status 默认为 0，即正常状态

// 用户自己禁用账号
user.Status.Set(types.StatusUserDisabled)

// 管理员禁用用户
user.Status.Set(types.StatusAdmDisabled)

// 系统禁用用户
user.Status.Set(types.StatusSysDisabled)

// 检查用户状态
if !user.Status.IsDisable() {
    // 用户可以登录
}

if user.Status.IsDisable() {
    // 用户被某个级别禁用了
}

// 用户隐藏自己的资料
user.Status.Set(types.StatusUserHidden)

// 软删除用户（系统删除+隐藏）
user.Status.SetMultiple(types.StatusSysDeleted, types.StatusSysHidden)
```

## 数据库查询

### 查询特定状态的记录

```go
// 查询所有未被禁用的用户
var users []User
disabledFlags := types.StatusSysDisabled | types.StatusAdmDisabled | types.StatusUserDisabled
db.Where("status & ? = 0", disabledFlags).Find(&users)

// 查询被任意级别禁用的用户
db.Where("status & ? != 0", disabledFlags).Find(&users)

// 查询未删除的用户
deletedFlags := types.StatusSysDeleted | types.StatusAdmDeleted | types.StatusUserDeleted
db.Where("status & ? = 0", deletedFlags).Find(&users)

// 查询正常状态的用户（未禁用、未删除、未隐藏）
allBadFlags := disabledFlags | deletedFlags | 
    (types.StatusSysHidden | types.StatusAdmHidden | types.StatusUserHidden)
db.Where("status & ? = 0", allBadFlags).Find(&users)
```

### Scope 辅助方法（推荐）

在 repository 中创建辅助方法：

```go
// 查询未禁用的记录
func WithNotDisabled(db *gorm.DB) *gorm.DB {
    disabledFlags := types.StatusSysDisabled | types.StatusAdmDisabled | types.StatusUserDisabled
    return db.Where("status & ? = 0", disabledFlags)
}

// 查询未隐藏的记录
func WithNotHidden(db *gorm.DB) *gorm.DB {
    hiddenFlags := types.StatusSysHidden | types.StatusAdmHidden | types.StatusUserHidden
    return db.Where("status & ? = 0", hiddenFlags)
}

// 查询未删除的记录
func WithNotDeleted(db *gorm.DB) *gorm.DB {
    deletedFlags := types.StatusSysDeleted | types.StatusAdmDeleted | types.StatusUserDeleted
    return db.Where("status & ? = 0", deletedFlags)
}

// 使用
db.Scopes(WithNotDisabled, WithNotHidden, WithNotDeleted).Find(&users)
```

## 实际应用场景

### 1. 用户状态管理

```go
// 新用户注册 - 默认正常状态
user.Status = types.StatusNone

// 用户自己禁用账号（注销）
user.Status.Set(types.StatusUserDisabled)

// 用户重新启用账号
user.Status.Unset(types.StatusUserDisabled)

// 管理员禁用违规用户
user.Status.Set(types.StatusAdmDisabled)

// 系统自动禁用长期不活跃用户
user.Status.Set(types.StatusSysDisabled)

// 完全禁用（所有级别）
user.Status.SetMultiple(types.StatusSysDisabled, types.StatusAdmDisabled, types.StatusUserDisabled)

// 用户隐藏自己的资料
user.Status.Set(types.StatusUserHidden)

// 软删除用户
user.Status.SetMultiple(types.StatusSysDeleted, types.StatusSysHidden)
```

### 2. 内容管理

```go
// 文章发布
article.Status = types.StatusNone

// 作者隐藏文章
article.Status.Set(types.StatusUserHidden)

// 管理员隐藏违规文章
article.Status.Set(types.StatusAdmHidden)

// 系统隐藏敏感内容
article.Status.Set(types.StatusSysHidden)

// 软删除文章
article.Status.SetMultiple(types.StatusSysDeleted, types.StatusSysHidden)
```

### 3. 权限控制示例

```go
// 检查是否可以显示给用户
func CanDisplay(status types.Status) bool {
    return !status.IsDisable() && !status.IsHidden() && !status.IsDeleted()
}

// 检查管理员是否可以编辑
func CanAdminEdit(status types.Status) bool {
    // 系统级别的状态管理员也不能改
    return !status.Contain(types.StatusSysDisabled) && !status.Contain(types.StatusSysDeleted)
}

// 检查用户是否可以编辑自己的内容
func CanUserEdit(status types.Status) bool {
    // 被管理员或系统禁用/删除的内容用户不能编辑
    return !status.HasAny(
        types.StatusSysDisabled, types.StatusAdmDisabled,
        types.StatusSysDeleted, types.StatusAdmDeleted,
    )
}
```

## JSON 序列化

```go
// 序列化
data, _ := json.Marshal(user)
// {"id":123,"status":4,"username":"john",...}

// 反序列化
json.Unmarshal(data, &user)
```

状态值以整数形式存储在JSON中。

## 性能优势

- **存储效率**：使用单个 int64 字段存储最多 63 个状态标志
- **查询效率**：使用位运算进行状态过滤，数据库索引友好
- **内存效率**：相比使用多个 bool 字段，大大减少内存占用
- **扩展性强**：目前仅使用 12 位，还有 51 位可供扩展

## 扩展自定义状态

如果需要添加项目特定的状态：

```go
const (
    StatusCustom1 types.Status = 1 << 12  // 从第12位开始
    StatusCustom2 types.Status = 1 << 13
    StatusCustom3 types.Status = 1 << 14
    // ... 最多到第62位
)
```

## 注意事项

1. **位运算范围**：使用 `int64`，最多支持 63 个状态位（0-62位）
2. **数据库类型**：使用 `BIGINT` 类型（所有主流数据库都支持）
3. **向后兼容**：添加新状态时使用新的位，不要修改已有状态的位值
4. **分级设计**：合理使用系统、管理员、用户三个级别，明确权限边界
5. **状态组合**：可以灵活组合多个状态，但要注意业务逻辑的一致性

## 分级状态设计理念

### 为什么要分级？

1. **权限分离**：不同级别的管理员/用户只能操作对应级别的状态
2. **审计追踪**：可以明确知道是谁（系统/管理员/用户）设置的状态
3. **灵活控制**：支持多级别同时存在，如系统禁用+用户隐藏

### 级别优先级

虽然技术上没有优先级（位运算是平等的），但业务上建议：
- **系统级别** > **管理员级别** > **用户级别**
- 系统级别的状态通常不允许管理员修改
- 管理员级别的状态不允许普通用户修改

## 测试

运行测试：

```bash
go test -v katydid-common-account/pkg/types -run TestStatus
```

查看覆盖率：

```bash
go test -v -cover katydid-common-account/pkg/types -run TestStatus
```

## 数据库迁移建议

```sql
-- MySQL
ALTER TABLE your_table 
ADD COLUMN status BIGINT NOT NULL DEFAULT 0,
ADD INDEX idx_status (status);

-- PostgreSQL
ALTER TABLE your_table 
ADD COLUMN status BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_status ON your_table(status);
```

## 常见问题

**Q: 为什么默认值是0（StatusNone）？**  
A: 0 表示没有任何标记，即正常状态。这样设计更符合直觉，且数据库默认值为0。

**Q: 如何判断是否被系统禁用？**  
A: 使用 `status.Contain(types.StatusSysDisabled)` 方法，它只检查系统级别的禁用标志。

**Q: 软删除如何实现？**  
A: 推荐组合使用 `StatusSysDeleted | StatusSysHidden`，既标记删除又隐藏内容。

**Q: 如何检查是否正常状态？**  
A: 使用 `status.IsNormal()` 方法，它检查未被禁用、未删除、未隐藏、已验证。
