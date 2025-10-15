# ID Generator (idgen)

## 概述

`idgen` 包提供了基于 Snowflake 算法的分布式唯一 ID 生成器，适用于分布式系统中的数据库主键生成。

## 特性

- **分布式友好**: 基于 Twitter Snowflake 算法，支持多数据中心、多机器部署
- **高性能**: 本地生成，无需网络调用，单机每秒可生成 400 万个 ID
- **趋势递增**: 生成的 ID 按时间递增，对数据库索引友好
- **64位整数**: 使用 int64 类型，兼容大多数数据库
- **包含时间信息**: 可以从 ID 中解析出生成时间

## ID 结构

64 位 ID 的组成（从高位到低位）：

```
| 1位符号位(0) | 41位时间戳 | 5位数据中心ID | 5位工作机器ID | 12位序列号 |
```

- **41位时间戳**: 毫秒级时间戳，可使用 69 年
- **5位数据中心ID**: 支持 32 个数据中心
- **5位工作机器ID**: 每个数据中心支持 32 台机器
- **12位序列号**: 每毫秒可生成 4096 个 ID

## 快速开始

### 1. 初始化 ID 生成器

在应用启动时初始化 ID 生成器：

```go
package main

import (
    "katydid-common-account/pkg/idgen"
)

func main() {
    // 初始化: datacenterID=1, workerID=1
    // datacenterID 和 workerID 范围都是 0-31
    err := idgen.Init(1, 1)
    if err != nil {
        panic(err)
    }
}
```

### 2. 在模型中使用

```go
package models

import (
    "katydid-common-account/pkg/idgen"
    "gorm.io/gorm"
)

type User struct {
    ID        idgen.ID `gorm:"primarykey" json:"id"`
    Name      string   `json:"name"`
    Email     string   `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// BeforeCreate GORM 钩子，自动生成 ID
func (u *User) BeforeCreate(tx *gorm.DB) error {
    if u.ID.IsZero() {
        id, err := idgen.NewID()
        if err != nil {
            return err
        }
        u.ID = id
    }
    return nil
}
```

### 3. 使用 BaseModel

项目已提供 `BaseModel`，直接继承即可：

```go
package models

import "katydid-common-account/internal/models"

type User struct {
    models.BaseModel  // 包含 ID, CreatedAt, UpdatedAt, DeletedAt
    Name      string `json:"name"`
    Email     string `json:"email"`
}
```

## API 文档

### 初始化

```go
// 初始化默认生成器
func Init(datacenterID, workerID int64) error

// 创建自定义生成器
func NewSnowflake(datacenterID, workerID int64) (*Snowflake, error)
```

### ID 生成

```go
// 生成新 ID
func NewID() (ID, error)

// 生成新 ID（失败时 panic）
func MustNewID() ID

// 使用默认生成器生成 ID
func NextID() (int64, error)
```

### ID 类型方法

```go
type ID int64

// 转换为 int64
func (id ID) Int64() int64

// 转换为字符串
func (id ID) String() string

// 判断是否为零值
func (id ID) IsZero() bool

// JSON 序列化/反序列化
func (id ID) MarshalJSON() ([]byte, error)
func (id *ID) UnmarshalJSON(data []byte) error

// 数据库读写
func (id *ID) Scan(value interface{}) error
func (id ID) Value() (driver.Value, error)
```

### 工具函数

```go
// 解析 ID，获取各部分信息
func ParseSnowflakeID(id int64) (timestamp, datacenterID, workerID, sequence int64)

// 从 ID 提取时间戳
func GetTimestamp(id int64) time.Time

// 从字符串解析 ID
func ParseIDFromString(s string) (ID, error)
```

## 使用示例

### 生成 ID

```go
// 方式 1: 使用默认生成器
id, err := idgen.NewID()
if err != nil {
    log.Fatal(err)
}

// 方式 2: 直接生成（失败时 panic）
id := idgen.MustNewID()
```

### 解析 ID

```go
id, _ := idgen.NewID()

// 获取 ID 的生成时间
timestamp := idgen.GetTimestamp(id.Int64())
fmt.Println("ID 生成于:", timestamp)

// 解析 ID 的各个组成部分
ts, dc, worker, seq := idgen.ParseSnowflakeID(id.Int64())
fmt.Printf("时间戳: %d, 数据中心: %d, 机器: %d, 序列号: %d\n", ts, dc, worker, seq)
```

### JSON 序列化

```go
user := User{
    Name:  "张三",
    Email: "zhangsan@example.com",
}
db.Create(&user) // ID 自动生成

// JSON 序列化（ID 会转为字符串，避免 JS 精度问题）
data, _ := json.Marshal(user)
// {"id":"123456789","name":"张三","email":"zhangsan@example.com"}
```

## 配置建议

### 开发环境

```go
// 单机开发
idgen.Init(0, 0)
```

### 生产环境

根据部署情况设置不同的 datacenterID 和 workerID：

```go
// 从环境变量或配置文件读取
datacenterID := config.GetInt64("DATACENTER_ID") // 0-31
workerID := config.GetInt64("WORKER_ID")         // 0-31

idgen.Init(datacenterID, workerID)
```

**重要**: 确保在同一个数据中心内，每台机器的 `workerID` 唯一！

## 性能

- 单机 QPS: ~400 万/秒
- 内存占用: < 1KB
- 线程安全: 是

## 注意事项

1. **时钟回拨**: 如果系统时钟回拨，生成 ID 会返回 `ErrClockMovedBackwards` 错误
2. **ID 范围**: datacenterID 和 workerID 必须在 0-31 范围内
3. **唯一性**: 确保每个实例的 (datacenterID, workerID) 组合唯一
4. **数据库字段**: 使用 `BIGINT` 类型存储 ID

## 数据库迁移

创建表时使用 `BIGINT` 类型：

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255),
    email VARCHAR(255),
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
```

对于 GORM，会自动处理类型映射。
