# IDGen - 分布式ID生成器

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

`idgen` 是一个高性能、线程安全的分布式ID生成器包，实现了 Snowflake 算法。该包设计遵循 SOLID 原则，提供了灵活的配置选项和丰富的功能。

## 目录

- [特性](#特性)
- [架构设计](#架构设计)
- [安装](#安装)
- [快速开始](#快速开始)
- [详细使用](#详细使用)
- [性能](#性能)
- [设计原则](#设计原则)
- [API文档](#api文档)
- [常见问题](#常见问题)
- [最佳实践](#最佳实践)

## 特性

### 核心功能
- ✅ **Snowflake算法**: 实现Twitter的Snowflake分布式ID生成算法
- ✅ **高性能**: 单实例每毫秒可生成4096个唯一ID
- ✅ **线程安全**: 使用互斥锁保证并发安全，无数据竞争
- ✅ **时钟回拨处理**: 智能检测和处理时钟回拨问题
- ✅ **零依赖**: 仅使用Go标准库，无第三方依赖
- ✅ **易于使用**: 提供多种使用方式，从简单到高级

### 高级特性
- 🎯 **接口隔离**: IDGenerator和IDParser接口分离
- 🎯 **工厂模式**: 支持多种ID生成器类型扩展
- 🎯 **注册表模式**: 统一管理多个生成器实例
- 🎯 **批量生成**: 支持批量生成ID，提高效率
- 🎯 **ID封装**: 提供ID类型，支持JSON序列化和多种格式转换
- 🎯 **集合操作**: IDSet和IDSlice提供丰富的集合操作

## 架构设计

### Snowflake ID结构

```
+------------------+------------------+------------------+------------------+
| 41位时间戳        | 5位数据中心ID     | 5位工作机器ID     | 12位序列号        |
+------------------+------------------+------------------+------------------+
```

- **时间戳**: 41位，精确到毫秒，可使用约69年
- **数据中心ID**: 5位，支持32个数据中心
- **工作机器ID**: 5位，每个数据中心支持32台机器
- **序列号**: 12位，每毫秒每台机器可生成4096个ID

### 组件架构

```
┌─────────────────────────────────────────────┐
│         IDGenerator Interface               │
│  (接口隔离原则 - 只定义生成功能)              │
└─────────────────────────────────────────────┘
                    ▲
                    │ implements
                    │
┌─────────────────────────────────────────────┐
│         Snowflake Struct                    │
│  - 线程安全的互斥锁                          │
│  - 时钟回拨检测                              │
│  - 序列号管理                                │
│  - 性能计数器                                │
└─────────────────────────────────────────────┘
                    │
                    │ managed by
                    ▼
┌─────────────────────────────────────────────┐
│      GeneratorRegistry (单例)               │
│  - 工厂注册管理                              │
│  - 生成器实例缓存                            │
│  - 线程安全的读写锁                          │
└─────────────────────────────────────────────┘
```

## 安装

```bash
go get github.com/jiangzhiguo1992/katydid-common-account/pkg/idgen
```

## 快速开始

### 1. 最简单的使用方式

```go
package main

import (
    "fmt"
    "github.com/yourusername/katydid-common-account/pkg/idgen"
)

func main() {
    // 使用默认生成器（datacenterID=0, workerID=0）
    id, err := idgen.GenerateID()
    if err != nil {
        panic(err)
    }
    fmt.Printf("生成的ID: %d\n", id)
}
```

### 2. 创建自定义生成器

```go
package main

import (
    "fmt"
    "github.com/yourusername/katydid-common-account/pkg/idgen"
)

func main() {
    // 创建Snowflake生成器
    // datacenterID: 数据中心ID (0-31)
    // workerID: 工作机器ID (0-31)
    sf, err := idgen.NewSnowflake(1, 1)
    if err != nil {
        panic(err)
    }

    // 生成ID
    id, err := sf.NextID()
    if err != nil {
        panic(err)
    }
    fmt.Printf("生成的ID: %d\n", id)
}
```

### 3. 使用配置创建生成器

```go
package main

import (
    "fmt"
    "github.com/yourusername/katydid-common-account/pkg/idgen"
)

func main() {
    // 使用配置创建（推荐方式，便于扩展）
    config := &idgen.SnowflakeConfig{
        DatacenterID: 1,
        WorkerID:     1,
    }
    
    sf, err := idgen.NewSnowflakeWithConfig(config)
    if err != nil {
        panic(err)
    }

    // 生成多个ID
    for i := 0; i < 10; i++ {
        id, err := sf.NextID()
        if err != nil {
            panic(err)
        }
        fmt.Printf("ID %d: %d\n", i+1, id)
    }
}
```

## 详细使用

### 解析ID

```go
// 方式1: 使用全局函数
timestamp, datacenterID, workerID, sequence := idgen.ParseSnowflakeID(id)
fmt.Printf("时间戳: %d, 数据中心: %d, 工作机器: %d, 序列号: %d\n", 
    timestamp, datacenterID, workerID, sequence)

// 方式2: 使用实例方法（获取更详细信息）
info, err := sf.Parse(id)
if err != nil {
    panic(err)
}
fmt.Printf("ID信息: %+v\n", info)
fmt.Printf("生成时间: %v\n", info.Time)

// 方式3: 只提取时间戳
ts := idgen.GetTimestamp(id)
fmt.Printf("ID生成时间: %v\n", ts)
```

### 使用ID类型

```go
// 创建ID实例
id := idgen.NewID(123456789)

// 类型转换
fmt.Printf("Int64: %d\n", id.Int64())
fmt.Printf("String: %s\n", id.String())
fmt.Printf("Hex: %s\n", id.Hex())
fmt.Printf("Binary: %s\n", id.Binary())

// 检查有效性
if id.IsValid() {
    fmt.Println("ID有效")
}

// JSON序列化（自动转为字符串，避免JavaScript精度问题）
data, err := json.Marshal(id)
fmt.Printf("JSON: %s\n", data)

// 解析ID信息
info, err := id.Parse()
if err == nil {
    fmt.Printf("数据中心: %d, 工作机器: %d\n", 
        info.DatacenterID, info.WorkerID)
}
```

### 批量生成ID

```go
// 使用默认生成器批量生成
ids, err := idgen.GenerateIDs(100)
if err != nil {
    panic(err)
}
fmt.Printf("批量生成了 %d 个ID\n", len(ids))

// 使用自定义生成器批量生成
sf, _ := idgen.NewSnowflake(1, 1)
batch := idgen.NewBatchIDGenerator(sf)
ids, err = batch.Generate(1000)
if err != nil {
    panic(err)
}
```

### ID集合操作

```go
// 创建ID集合
set := idgen.NewIDSet(
    idgen.NewID(1),
    idgen.NewID(2),
    idgen.NewID(3),
)

// 添加和检查
set.Add(idgen.NewID(4))
if set.Contains(idgen.NewID(1)) {
    fmt.Println("集合包含ID 1")
}

// 集合操作
set2 := idgen.NewIDSet(
    idgen.NewID(3),
    idgen.NewID(4),
    idgen.NewID(5),
)

union := set.Union(set2)        // 并集
intersect := set.Intersect(set2) // 交集
diff := set.Difference(set2)     // 差集

fmt.Printf("并集大小: %d\n", union.Size())
```

### ID切片操作

```go
ids := idgen.IDSlice{
    idgen.NewID(1),
    idgen.NewID(2),
    idgen.NewID(3),
    idgen.NewID(2), // 重复
}

// 去重
unique := ids.Deduplicate()
fmt.Printf("去重后: %d 个ID\n", len(unique))

// 过滤
filtered := ids.Filter(func(id idgen.ID) bool {
    return id > idgen.NewID(1)
})

// 转换
int64Slice := ids.Int64Slice()
stringSlice := ids.StringSlice()
```

### 使用注册表管理多个生成器

```go
// 注册和创建生成器
gen1, err := idgen.NewGenerator("server-1", idgen.SnowflakeGeneratorType, 
    &idgen.SnowflakeConfig{
        DatacenterID: 1,
        WorkerID:     1,
    })

gen2, err := idgen.NewGenerator("server-2", idgen.SnowflakeGeneratorType,
    &idgen.SnowflakeConfig{
        DatacenterID: 1,
        WorkerID:     2,
    })

// 后续从注册表获取
gen, exists := idgen.GetGeneratorFromRegistry("server-1")
if exists {
    id, _ := gen.NextID()
    fmt.Printf("生成的ID: %d\n", id)
}

// 列出所有生成器类型
types := idgen.GetRegistry().ListGeneratorTypes()
fmt.Printf("支持的生成器类型: %v\n", types)
```

### 验证ID

```go
// 验证ID的有效性
err := idgen.ValidateSnowflakeID(id)
if err != nil {
    fmt.Printf("ID无效: %v\n", err)
} else {
    fmt.Println("ID有效")
}
```

### 监控统计

```go
sf, _ := idgen.NewSnowflake(1, 1)

// 生成一些ID
for i := 0; i < 1000; i++ {
    sf.NextID()
}

// 获取统计信息
count := sf.GetIDCount()
fmt.Printf("已生成 %d 个ID\n", count)

// 获取配置信息
fmt.Printf("数据中心ID: %d\n", sf.GetDatacenterID())
fmt.Printf("工作机器ID: %d\n", sf.GetWorkerID())
```

## 性能

### 基准测试结果

```
BenchmarkSnowflakeNextID-8              5000000    250 ns/op    0 B/op    0 allocs/op
BenchmarkSnowflakeNextIDParallel-8     10000000    150 ns/op    0 B/op    0 allocs/op
BenchmarkParseSnowflakeID-8            50000000     30 ns/op    0 B/op    0 allocs/op
```

### 性能特点

- **单goroutine**: 每秒可生成约 400万 个ID
- **并发场景**: 多goroutine并发时性能更优
- **零内存分配**: 生成和解析ID过程无额外内存分配
- **低CPU占用**: 使用休眠代替忙等待

### 性能优化要点

1. **避免频繁创建实例**: 复用Snowflake实例
2. **合理配置datacenterID和workerID**: 避免ID冲突
3. **使用注册表缓存**: 避免重复创建生成器
4. **批量生成**: 大量ID需求时使用批量生成接口

## 设计原则

本包严格遵循SOLID设计原则：

### 1. 单一职责原则 (SRP)

- `Snowflake`: 只负责ID生成
- `IDParser`: 只负责ID解析
- `GeneratorRegistry`: 只负责生成器管理
- `ID`: 只负责ID的表示和转换

### 2. 开放封闭原则 (OCP)

- 通过`SnowflakeConfig`扩展配置，无需修改核心代码
- 支持自定义时间函数，便于测试
- 可注册新的生成器工厂，支持扩展

### 3. 里氏替换原则 (LSP)

- `Snowflake`实现了`IDGenerator`接口，可替换使用
- 所有实现相同接口的生成器可互换

### 4. 接口隔离原则 (ISP)

- `IDGenerator`和`IDParser`接口分离
- 客户端只依赖需要的接口

### 5. 依赖倒置原则 (DIP)

- 依赖抽象接口`IDGenerator`而非具体实现
- 时间函数可注入，便于测试和扩展

## API文档

### 核心类型

#### Snowflake

```go
type Snowflake struct {
    // 私有字段...
}

// 创建方法
func NewSnowflake(datacenterID, workerID int64) (*Snowflake, error)
func NewSnowflakeWithConfig(config *SnowflakeConfig) (*Snowflake, error)

// 核心方法
func (s *Snowflake) NextID() (int64, error)
func (s *Snowflake) Parse(id int64) (*IDInfo, error)

// 辅助方法
func (s *Snowflake) GetIDCount() uint64
func (s *Snowflake) GetWorkerID() int64
func (s *Snowflake) GetDatacenterID() int64
```

#### ID

```go
type ID int64

// 创建和转换
func NewID(value int64) ID
func (id ID) Int64() int64
func (id ID) String() string
func (id ID) Hex() string
func (id ID) Binary() string

// 检查方法
func (id ID) IsZero() bool
func (id ID) IsValid() bool

// 解析
func (id ID) Parse() (*IDInfo, error)

// JSON序列化
func (id ID) MarshalJSON() ([]byte, error)
func (id *ID) UnmarshalJSON(data []byte) error
```

### 全局函数

```go
// 便捷生成
func GenerateID() (int64, error)
func GenerateIDs(count int) ([]int64, error)

// 解析和验证
func ParseSnowflakeID(id int64) (timestamp, datacenterID, workerID, sequence int64)
func GetTimestamp(id int64) time.Time
func ValidateSnowflakeID(id int64) error

// 生成器管理
func NewGenerator(key string, generatorType GeneratorType, config interface{}) (IDGenerator, error)
func GetGeneratorFromRegistry(key string) (IDGenerator, bool)
func GetDefaultGenerator() (IDGenerator, error)
```

### 错误类型

```go
var (
    ErrInvalidWorkerID        error  // 无效的工作机器ID
    ErrInvalidDatacenterID    error  // 无效的数据中心ID
    ErrClockMovedBackwards    error  // 时钟回拨
    ErrInvalidSnowflakeID     error  // 无效的Snowflake ID
    ErrTimestampOverflow      error  // 时间戳溢出
    ErrGeneratorNotFound      error  // 生成器未找到
    ErrGeneratorAlreadyExists error  // 生成器已存在
)
```

## 常见问题

### Q1: 如何选择datacenterID和workerID？

**A:** 在分布式环境中：
- `datacenterID`: 表示数据中心或区域（0-31）
- `workerID`: 表示该数据中心内的机器编号（0-31）

确保每个服务实例使用唯一的组合，避免ID冲突。

### Q2: 时钟回拨如何处理？

**A:** 本包提供两层保护：
1. **容忍范围**: 5毫秒内的回拨会自动等待恢复
2. **超出范围**: 返回`ErrClockMovedBackwards`错误

建议使用NTP同步时钟，避免大幅度时钟回拨。

### Q3: 为什么JSON序列化时ID是字符串？

**A:** JavaScript的Number类型只能安全表示53位整数，Snowflake ID是63位，会导致精度丢失。使用字符串可以完整保留ID值。

### Q4: 可以在同一毫秒内生成多少个ID？

**A:** 单个Snowflake实例每毫秒最多生成4096个ID（2^12）。如果需要更高吞吐量，可以：
- 使用多个实例（不同workerID）
- 使用多个数据中心

### Q5: 线程安全吗？

**A:** 是的，完全线程安全。使用`sync.Mutex`保护并发访问，可以安全地在多个goroutine中使用同一个实例。

### Q6: 有内存泄漏风险吗？

**A:** 没有。本包：
- 不使用无界的缓存
- 注册表使用固定大小的map
- 生成ID过程零内存分配
- 所有集合操作返回新实例，不持有旧引用

## 最佳实践

### 1. 生产环境配置

```go
// 从环境变量或配置文件读取
datacenterID := getDatacenterIDFromConfig()
workerID := getWorkerIDFromInstance()

// 创建生成器
sf, err := idgen.NewSnowflake(datacenterID, workerID)
if err != nil {
    log.Fatalf("初始化ID生成器失败: %v", err)
}

// 注册到全局注册表
idgen.GetRegistry().CreateGenerator(
    "main-generator",
    idgen.SnowflakeGeneratorType,
    &idgen.SnowflakeConfig{
        DatacenterID: datacenterID,
        WorkerID:     workerID,
    },
)
```

### 2. 使用单例模式

```go
var (
    generator idgen.IDGenerator
    once      sync.Once
)

func GetIDGenerator() idgen.IDGenerator {
    once.Do(func() {
        var err error
        generator, err = idgen.NewSnowflake(1, 1)
        if err != nil {
            panic(fmt.Sprintf("初始化ID生成器失败: %v", err))
        }
    })
    return generator
}
```

### 3. 错误处理

```go
id, err := sf.NextID()
if err != nil {
    if errors.Is(err, idgen.ErrClockMovedBackwards) {
        // 时钟回拨，记录日志并重试或返回错误
        log.Errorf("检测到时钟回拨: %v", err)
        // 可以选择等待一段时间后重试
    } else if errors.Is(err, idgen.ErrTimestampOverflow) {
        // 时间戳溢出，这是严重问题
        log.Fatalf("时间戳溢出: %v", err)
    } else {
        log.Errorf("生成ID失败: %v", err)
    }
    return err
}
```

### 4. 监控和告警

```go
// 定期监控ID生成统计
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    var lastCount uint64
    for range ticker.C {
        currentCount := sf.GetIDCount()
        rate := currentCount - lastCount
        log.Infof("ID生成速率: %d/分钟, 总计: %d", rate, currentCount)
        lastCount = currentCount
        
        // 如果速率异常，触发告警
        if rate > 240000 { // 超过每秒4000个
            log.Warnf("ID生成速率异常高: %d/分钟", rate)
        }
    }
}()
```

### 5. 测试建议

```go
func TestYourService(t *testing.T) {
    // 使用自定义时间函数进行测试
    mockTime := int64(1700000000000)
    config := &idgen.SnowflakeConfig{
        DatacenterID: 1,
        WorkerID:     1,
        TimeFunc: func() int64 {
            return mockTime
        },
    }
    
    sf, err := idgen.NewSnowflakeWithConfig(config)
    require.NoError(t, err)
    
    // 测试逻辑...
    id, err := sf.NextID()
    require.NoError(t, err)
    assert.Greater(t, id, int64(0))
    
    // 模拟时间推进
    mockTime += 1000
    id2, err := sf.NextID()
    require.NoError(t, err)
    assert.Greater(t, id2, id)
}
```

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

[MIT License](LICENSE)

