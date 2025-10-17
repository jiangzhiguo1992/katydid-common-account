# Extras 扩展字段模块

## 概述

`Extras` 是一个灵活的键值对存储类型，用于在不修改数据库表结构的情况下动态扩展字段。基于 `map[string]any` 实现，支持存储任意类型的数据，并提供类型安全的访问方法。

## 核心特性

### 1. 类型安全
- 提供类型化的 Get 方法（如 `GetString`、`GetInt`、`GetFloat64` 等）
- 自动类型转换，支持数值类型之间的智能转换
- 完整的边界检查，防止溢出和精度丢失

### 2. 数据库集成
- 实现 `driver.Valuer` 和 `sql.Scanner` 接口
- 自动 JSON 序列化/反序列化
- 支持 NULL 值处理

### 3. 性能优化
- 预分配容量支持（`NewExtrasWithCapacity`）
- 空值优化，避免不必要的内存分配
- O(1) 时间复杂度的查询操作

### 4. 防御性编程
- 空键名自动忽略
- nil 值安全处理
- 类型转换失败时返回零值和 false

## 快速开始

### 基本用法

```go
// 创建实例
extras := types.NewExtras()

// 设置值
extras.Set("username", "alice")
extras.Set("age", 25)
extras.Set("active", true)
extras.Set("score", 98.5)

// 获取值（类型安全）
if name, ok := extras.GetString("username"); ok {
    fmt.Println("用户名:", name)
}

if age, ok := extras.GetInt("age"); ok {
    fmt.Println("年龄:", age)
}

// 检查键是否存在
if extras.Has("username") {
    fmt.Println("用户名字段存在")
}

// 删除键
extras.Delete("temp_field")
```

### 预分配容量

```go
// 如果预知字段数量，可以预分配容量以提升性能
extras := types.NewExtrasWithCapacity(10)
```

### 条件设置

```go
// 值为 nil 时自动删除键
extras.SetOrDel("optional", getValue()) // 如果 getValue() 返回 nil，键会被删除
```

## 类型转换

`Extras` 提供了强大的自动类型转换功能，特别是对数值类型之间的转换。

### 基本类型转换

```go
extras := types.NewExtras()

// 设置 int8 类型
extras.Set("small_num", int8(100))

// 自动转换为 int
if num, ok := extras.GetInt("small_num"); ok {
    fmt.Println("转换成功:", num) // 输出: 100
}
```

### JSON 数值处理

```go
// JSON 反序列化后的数字是 float64，也能正确转换
data := []byte(`{"count": 42}`)
json.Unmarshal(data, &extras)
if count, ok := extras.GetInt("count"); ok {
    fmt.Println("计数:", count) // 输出: 42
}
```

### 浮点数转整数

```go
// 仅支持整数值的浮点数
extras.Set("float_int", 42.0)
if num, ok := extras.GetInt("float_int"); ok {
    fmt.Println("成功:", num) // 输出: 42
}

// 带小数部分的浮点数转换失败
extras.Set("float_frac", 42.5)
if _, ok := extras.GetInt("float_frac"); !ok {
    fmt.Println("失败: 42.5 不是整数值")
}
```

## 支持的类型

### 基础类型
- `string`: 字符串
- `bool`: 布尔值
- `[]byte`: 字节切片

### 整数类型
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`

### 浮点类型
- `float32`, `float64`

### 复杂类型
- `[]any`: 任意类型切片
- `map[string]any`: 嵌套对象
- `Extras`: 嵌套扩展字段
- 各类型的切片（如 `[]string`, `[]int` 等）

## 数据库使用

### 在模型中使用

```go
type User struct {
    ID        uint64         `gorm:"primaryKey"`
    Name      string         `gorm:"size:100"`
    Extras    types.Extras   `gorm:"type:json"`
    CreatedAt time.Time
}

// 设置扩展字段
user := &User{
    Name: "Alice",
    Extras: types.NewExtras(),
}
user.Extras.Set("nickname", "小艾")
user.Extras.Set("vip_level", 5)
user.Extras.Set("tags", []string{"active", "premium"})

// 保存到数据库（自动序列化为 JSON）
db.Create(user)

// 从数据库读取（自动反序列化）
var loadedUser User
db.First(&loadedUser, user.ID)

// 访问扩展字段
if nickname, ok := loadedUser.Extras.GetString("nickname"); ok {
    fmt.Println("昵称:", nickname)
}
```

## 高级操作

### 克隆和合并

```go
// 克隆（浅拷贝）
original := types.NewExtras()
original.Set("key", "value")
clone := original.Clone()

// 合并（覆盖相同键）
extras1 := types.NewExtras()
extras1.Set("a", 1)
extras1.Set("b", 2)

extras2 := types.NewExtras()
extras2.Set("b", 3)
extras2.Set("c", 4)

extras1.Merge(extras2) // extras1 现在包含: a=1, b=3, c=4
```

### 遍历所有键

```go
extras := types.NewExtras()
extras.Set("key1", "value1")
extras.Set("key2", "value2")

// 获取所有键
keys := extras.Keys()
for _, key := range keys {
    if value, ok := extras.Get(key); ok {
        fmt.Printf("%s: %v\n", key, value)
    }
}

// 检查是否为空
if extras.IsEmpty() {
    fmt.Println("没有扩展字段")
}
```

### 清空所有字段

```go
extras := types.NewExtras()
extras.Set("key1", "value1")
extras.Set("key2", "value2")

// 清空所有字段（保留内存）
extras.Clear()

// 或者重新赋值为 nil（释放内存）
extras = nil
```

## 性能考虑

### 内存分配
- 基础空 map：~48 字节
- 使用 `NewExtrasWithCapacity(n)` 预分配容量可避免多次扩容
- 建议单条记录不超过 64KB（数据库性能考虑）

### 并发安全
- `map` 类型非线程安全
- 并发读取是安全的
- 并发写入需要使用 `sync.RWMutex` 保护

```go
type SafeExtras struct {
    mu     sync.RWMutex
    extras types.Extras
}

func (s *SafeExtras) Set(key string, value any) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.extras.Set(key, value)
}

func (s *SafeExtras) GetString(key string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.extras.GetString(key)
}
```

### 操作复杂度
- Get 操作：O(1) 哈希查找
- Set 操作：O(1) 平均情况
- Clone 操作：O(n)，n 为键值对数量
- Merge 操作：O(m)，m 为被合并 map 的大小

## 最佳实践

### 1. 键名规范

```go
// ✅ 推荐：使用清晰的键名
extras.Set("user_level", 5)
extras.Set("last_login_ip", "192.168.1.1")

// ❌ 不推荐：使用空字符串或模糊的键名
extras.Set("", "value")  // 会被自动忽略
extras.Set("x", "value") // 语义不清
```

### 2. 类型选择

```go
// ✅ 推荐：使用合适的类型
extras.Set("count", 42)         // int
extras.Set("price", 99.99)      // float64
extras.Set("active", true)      // bool

// ❌ 不推荐：滥用字符串
extras.Set("count", "42")       // 需要手动转换
extras.Set("active", "true")    // 布尔值应该用 bool
```

### 3. 错误处理

```go
// ✅ 推荐：始终检查返回值
if value, ok := extras.GetInt("age"); ok {
    // 使用 value
} else {
    // 处理键不存在或类型不匹配的情况
}

// ❌ 不推荐：忽略错误
value, _ := extras.GetInt("age")
// value 可能是零值
```

### 4. 空值处理

```go
// ✅ 推荐：使用 SetOrDel 自动清理 nil 值
extras.SetOrDel("optional", getValue())

// 如果需要存储 nil，使用 Set
extras.Set("nullable", nil)
```

### 5. 性能优化

```go
// ✅ 推荐：预知字段数量时预分配容量
extras := types.NewExtrasWithCapacity(10) // 预期10个字段

// ✅ 推荐：批量操作时复用变量
for _, item := range items {
    extras.Set(item.Key, item.Value)
}

// ❌ 不推荐：频繁创建新实例
for _, item := range items {
    extras := types.NewExtras() // 每次循环都创建
    extras.Set(item.Key, item.Value)
}
```

## 常见问题

### Q: 为什么 GetInt 返回 false？

可能的原因：
1. 键不存在
2. 类型不匹配（如存储的是字符串 "42"）
3. 数值溢出（如 uint64 最大值转 int）
4. 浮点数不是整数值（如 42.5）

### Q: JSON 反序列化后为什么类型不对？

JSON 反序列化会将数字统一转换为 `float64`。`Extras` 的 Get 方法已经处理了这种情况，会自动转换。

```go
data := []byte(`{"age": 25}`)
var extras types.Extras
json.Unmarshal(data, &extras)

// 虽然 JSON 中是整数，反序列化后是 float64
// 但 GetInt 会自动转换
age, ok := extras.GetInt("age") // ok = true, age = 25
```

### Q: 如何存储自定义结构体？

可以将结构体序列化为 `map[string]any` 或使用 JSON 序列化：

```go
type CustomData struct {
    Field1 string
    Field2 int
}

// 方式1：转换为 map
data := map[string]any{
    "Field1": "value",
    "Field2": 42,
}
extras.Set("custom", data)

// 方式2：JSON 序列化为字符串
jsonData, _ := json.Marshal(customData)
extras.Set("custom", string(jsonData))
```

### Q: 为什么建议不超过 64KB？

数据库性能考虑：
- JSON 字段过大会影响查询性能
- 索引效率降低
- 网络传输开销增加
- 建议将大量数据拆分到独立表

### Q: 空键名会被如何处理？

空字符串键名会被自动忽略，这是防御性编程的一部分：

```go
extras.Set("", "value")  // 不会产生任何效果
extras.SetOrDel("", nil) // 不会产生任何效果
```

### Q: 并发场景如何使用？

`Extras` 基于 `map`，非线程安全。并发场景需要自行加锁：

```go
type SafeExtras struct {
    mu     sync.RWMutex
    extras types.Extras
}

// 写操作使用写锁
func (s *SafeExtras) Set(key string, value any) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.extras.Set(key, value)
}

// 读操作使用读锁
func (s *SafeExtras) GetString(key string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.extras.GetString(key)
}
```

## 注意事项

### 边界检查

模块提供完整的边界检查机制：

- **数值溢出检查**：uint64 → int 转换会检查是否溢出
- **浮点数整数值检查**：42.0 可以转为 int，42.5 不行
- **NULL 值安全处理**：数据库 NULL 值会被正确处理

### 错误信息

所有错误信息遵循统一规范：
- 使用英文错误前缀（`failed to`, `invalid`, `cannot`）
- 包含具体的错误上下文（类型、值、原因）
- 支持错误链（使用 `%w`）

示例：
```go
return fmt.Errorf("failed to scan Extras: unsupported database type %T, expected []byte or string", value)
return fmt.Errorf("failed to marshal Extras to JSON: %w", err)
```

### 测试覆盖

模块具有高测试覆盖率（~95%），包括：
- 所有数值类型转换测试
- 完整的边界情况测试（空值、nil、溢出）
- 并发读取安全性测试
- 空键名防御测试
- 空切片优化测试

## API 参考

### 创建实例

- `NewExtras() Extras` - 创建新实例
- `NewExtrasWithCapacity(capacity int) Extras` - 创建预分配容量的实例

### 基本操作

- `Set(key string, value any)` - 设置键值对
- `SetOrDel(key string, value any)` - 设置或删除（value 为 nil 时删除）
- `Get(key string) (any, bool)` - 获取原始值
- `Delete(key string)` - 删除键
- `Has(key string) bool` - 检查键是否存在
- `Clear()` - 清空所有键值对
- `IsEmpty() bool` - 检查是否为空
- `Keys() []string` - 获取所有键

### 类型化获取

- `GetString(key string) (string, bool)` - 获取字符串
- `GetInt(key string) (int, bool)` - 获取整数
- `GetInt64(key string) (int64, bool)` - 获取 int64
- `GetUint64(key string) (uint64, bool)` - 获取 uint64
- `GetFloat64(key string) (float64, bool)` - 获取 float64
- `GetBool(key string) (bool, bool)` - 获取布尔值
- `GetBytes(key string) ([]byte, bool)` - 获取字节切片
- `GetStringSlice(key string) ([]string, bool)` - 获取字符串切片
- `GetIntSlice(key string) ([]int, bool)` - 获取整数切片
- `GetMap(key string) (map[string]any, bool)` - 获取映射
- `GetExtras(key string) (Extras, bool)` - 获取嵌套 Extras

### 高级操作

- `Clone() Extras` - 克隆（浅拷贝）
- `Merge(other Extras)` - 合并另一个 Extras（覆盖相同键）

### 数据库接口

- `Value() (driver.Value, error)` - 实现 driver.Valuer 接口
- `Scan(value any) error` - 实现 sql.Scanner 接口
