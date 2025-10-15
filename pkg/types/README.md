# Types Package

提供通用的数据类型定义，可被内部和外部项目使用。

## ExtraFields

`ExtraFields` 是一个灵活的键值对数据结构，用于存储扩展字段到数据库模型中。

### 特性

- ✅ 支持多种数据类型（string, int, float, bool, slice, map）
- ✅ 自动 JSON 序列化/反序列化
- ✅ GORM 数据库集成（实现 `driver.Valuer` 和 `sql.Scanner`）
- ✅ 类型安全的 Getter 方法
- ✅ 丰富的工具方法（Clone, Merge, Clear 等）

### 使用示例

#### 在 Model 中使用

```go
import "katydid-common-account/pkg/types"

type User struct {
    ID       int64         `gorm:"primarykey" json:"id"`
    Username string        `gorm:"column:username" json:"username"`
    Extra    types.ExtraFields `gorm:"column:extra;type:json" json:"extra,omitempty"`
}
```

#### 设置和获取值

```go
user := &User{
    Username: "john",
    Extra:    types.NewExtraFields(),
}

// 设置各种类型的值
user.Extra.Set("phone", "13800138000")
user.Extra.Set("age", 25)
user.Extra.Set("vip", true)
user.Extra.Set("score", 98.5)
user.Extra.Set("tags", []interface{}{"admin", "developer"})
user.Extra.Set("profile", map[string]interface{}{
    "avatar": "https://example.com/avatar.jpg",
    "bio":    "Software Engineer",
})

// 读取值
if phone, ok := user.Extra.GetString("phone"); ok {
    fmt.Println("Phone:", phone)
}

if age, ok := user.Extra.GetInt("age"); ok {
    fmt.Println("Age:", age)
}

if vip, ok := user.Extra.GetBool("vip"); ok {
    fmt.Println("VIP:", vip)
}

if tags, ok := user.Extra.GetSlice("tags"); ok {
    fmt.Println("Tags:", tags)
}
```

#### 数据库操作

```go
// 创建记录（自动序列化为JSON存储）
db.Create(&user)

// 查询记录（自动反序列化）
var u User
db.First(&u, 1)
fmt.Println(u.Extra.GetString("phone"))

// 更新扩展字段
u.Extra.Set("level", 5)
u.Extra.Delete("tags")
db.Save(&u)
```

#### 工具方法

```go
ef := types.NewExtraFields()

// 检查键是否存在
if ef.Has("key") {
    // ...
}

// 获取所有键
keys := ef.Keys()

// 获取键值对数量
count := ef.Len()

// 检查是否为空
isEmpty := ef.IsEmpty()

// 克隆
clone := ef.Clone()

// 合并
ef1.Merge(ef2)

// 清空
ef.Clear()
```

### 数据库字段类型

在不同的数据库中使用以下字段类型：

- **MySQL/MariaDB**: `JSON` 或 `TEXT`
- **PostgreSQL**: `JSONB` 或 `JSON`
- **SQLite**: `TEXT`

GORM 示例：
```go
Extra types.ExtraFields `gorm:"column:extra;type:json"`        // MySQL
Extra types.ExtraFields `gorm:"column:extra;type:jsonb"`       // PostgreSQL
Extra types.ExtraFields `gorm:"column:extra;type:text"`        // SQLite
```

### API 响应

`ExtraFields` 实现了 `json.Marshaler` 和 `json.Unmarshaler` 接口，可以直接用于 API 响应：

```go
// 自动序列化为 JSON
{
  "id": 1,
  "username": "john",
  "extra": {
    "phone": "13800138000",
    "age": 25,
    "vip": true
  }
}
```

