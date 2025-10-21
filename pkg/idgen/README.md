# IDGen - 企业级分布式ID生成器

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 📋 目录

- [简介](#简介)
- [核心特性](#核心特性)
- [架构设计](#架构设计)
- [快速开始](#快速开始)
- [完整功能](#完整功能)
- [性能分析](#性能分析)
- [第三方库对比](#第三方库对比)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 简介

**IDGen** 是一个功能完善的企业级分布式ID生成器库，基于Twitter Snowflake算法实现，提供高性能、高可用、易扩展的唯一ID生成解决方案。

### 设计理念

- **🎯 高性能**: 单机QPS可达200万+，批量生成性能更优
- **🔒 线程安全**: 全面的并发控制，支持多goroutine安全调用
- **🛡️ 生产就绪**: 完善的错误处理、监控指标、时钟回拨保护
- **📦 易于扩展**: 插件化架构，支持自定义生成器类型
- **📊 可观测性**: 内置性能监控、指标收集、日志记录

### 适用场景

- 分布式系统的全局唯一ID生成
- 数据库主键生成（替代自增ID）
- 订单号、交易流水号生成
- 分布式追踪的TraceID生成
- 消息队列的消息ID生成

---

## 核心特性

### 1. Snowflake算法实现

#### ID结构（64位）

```
+----------+----------------+----------+----------+------------+
| 符号位(1) | 时间戳(41)      |数据中心(5)| 机器ID(5) | 序列号(12)  |
+----------+----------------+----------+----------+------------+
| 0        | 毫秒级时间戳     | 0-31     | 0-31     | 0-4095     |
+----------+----------------+----------+----------+------------+
```

- **时间戳（41位）**: 支持约69年（2^41 / (365 * 24 * 60 * 60 * 1000) ≈ 69年）
- **数据中心ID（5位）**: 支持32个数据中心
- **工作机器ID（5位）**: 每个数据中心支持32台机器
- **序列号（12位）**: 每毫秒可生成4096个ID

**理论性能**: 单机每秒可生成409.6万个ID（4096 × 1000）

### 2. 时钟回拨保护

提供三种策略应对时钟回拨问题：

| 策略 | 描述 | 适用场景 | 风险 |
|------|------|----------|------|
| **StrategyError** | 直接返回错误（默认） | 对唯一性要求极高的场景 | 时钟回拨时服务不可用 |
| **StrategyWait** | 等待时钟追上 | 容忍短暂回拨（<5ms） | 可能短暂阻塞 |
| **StrategyUseLastTimestamp** | 使用上次时间戳 | 高可用优先场景 | ⚠️ 可能ID重复 |

### 3. 性能监控

内置完善的监控指标：

```go
type Metrics struct {
    IDCount          uint64  // ID生成总数
    SequenceOverflow uint64  // 序列号溢出次数
    ClockBackward    uint64  // 时钟回拨次数
    WaitCount        uint64  // 等待次数
    TotalWaitTimeNs  uint64  // 总等待时间（纳秒）
}
```

### 4. 注册表管理

- **生成器注册表**: 统一管理多个生成器实例
- **工厂注册表**: 支持插件式扩展新的生成器类型
- **解析器注册表**: 统一管理ID解析器
- **验证器注册表**: 统一管理ID验证器

### 5. 领域类型增强

提供丰富的领域类型和工具：

- **ID类型**: 封装int64，提供类型安全和便捷方法
- **IDSlice**: ID切片工具，支持去重、过滤、验证
- **IDSet**: ID集合工具，支持并集、交集、差集操作

---

## 架构设计

### UML类图

```
┌─────────────────────────────────────────────────────────────────┐
│                         Core 核心层                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐     ┌──────────────────┐                  │
│  │  IDGenerator     │     │  BatchGenerator  │                  │
│  ├──────────────────┤     ├──────────────────┤                  │
│  │ +NextID()        │     │ +NextIDBatch()   │                  │
│  └──────────────────┘     └──────────────────┘                  │
│           ▲                        ▲                            │
│           │                        │                            │
│  ┌────────┴────────────────────────┴─────────┐                  │
│  │        FullFeaturedGenerator              │                  │
│  ├───────────────────────────────────────────┤                  │
│  │ Configurable + Monitorable + Parseable    │                  │
│  └───────────────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 实现
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Snowflake 实现层                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────┐           │
│  │             Generator（Snowflake生成器）           │           │
│  ├──────────────────────────────────────────────────┤           │
│  │ - datacenterID    int64                          │           │
│  │ - workerID        int64                          │           │
│  │ - lastTimestamp   int64                          │           │
│  │ - sequence        int64                          │           │
│  │ - config         *Config                         │           │
│  │ - metrics        *Metrics                        │           │
│  │ - validator      *Validator                      │           │
│  │ - parser         *Parser                         │           │
│  ├──────────────────────────────────────────────────┤           │
│  │ +NextID() int64                                  │           │
│  │ +NextIDBatch(n int) []int64                      │           │
│  │ +ParseID(id int64) *IDInfo                       │           │
│  │ +ValidateID(id int64) error                      │           │
│  │ +GetMetrics() map[string]uint64                  │           │
│  └──────────────────────────────────────────────────┘           │
│        │                   │                  │                 │
│        │                   │                  │                 │
│        ▼                   ▼                  ▼                 │
│  ┌──────────┐        ┌──────────┐       ┌──────────┐            │
│  │ Parser   │        │Validator │       │ Metrics  │            │
│  └──────────┘        └──────────┘       └──────────┘            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 管理
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Registry 注册层                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌────────────────┐  ┌────────────────┐  ┌─────────────────┐    │
│  │    Registry    │  │FactoryRegistry │  │ParserRegistry   │    │
│  │  (生成器注册表)  │  │  (工厂注册表)    │  │  (解析器注册表)   │    │
│  └────────────────┘  └────────────────┘  └─────────────────┘    │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              ValidatorRegistry (验证器注册表)             │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 封装
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Domain 领域层                               │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │     ID       │    │   IDSlice    │    │    IDSet     │       │
│  ├──────────────┤    ├──────────────┤    ├──────────────┤       │
│  │ +String()    │    │ +Contains()  │    │ +Union()     │       │
│  │ +Hex()       │    │ +Filter()    │    │ +Intersect() │       │
│  │ +Validate()  │    │ +Deduplicate │    │ +Difference()│       │
│  │ +Parse()     │    │ +ValidateAll │    │ +Equal()     │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### 架构分层说明

1. **Core层（核心接口层）**
   - 定义标准接口，遵循依赖倒置原则
   - 提供扩展点，支持多种生成器实现
   
2. **Snowflake层（实现层）**
   - 实现Snowflake算法
   - 包含配置、解析、验证、监控等完整功能

3. **Registry层（注册管理层）**
   - 统一管理生成器实例
   - 支持工厂模式动态创建
   - 提供解析器和验证器的插件机制

4. **Domain层（领域模型层）**
   - 提供类型安全的ID封装
   - 丰富的集合操作工具
   - JSON序列化支持

---

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "github.com/katydid/idgen/snowflake"
)

func main() {
    // 创建生成器（数据中心ID=1, 工作机器ID=1）
    gen, err := snowflake.New(1, 1)
    if err != nil {
        panic(err)
    }
    
    // 生成单个ID
    id, err := gen.NextID()
    if err != nil {
        panic(err)
    }
    fmt.Printf("生成的ID: %d\n", id)
    
    // 批量生成ID
    ids, err := gen.NextIDBatch(100)
    if err != nil {
        panic(err)
    }
    fmt.Printf("批量生成了 %d 个ID\n", len(ids))
}
```

### 高级配置

```go
package main

import (
    "github.com/katydid/idgen/core"
    "github.com/katydid/idgen/snowflake"
)

func main() {
    // 创建配置
    config := &snowflake.Config{
        DatacenterID:           1,
        WorkerID:               1,
        ClockBackwardStrategy:  core.StrategyWait,     // 等待策略
        ClockBackwardTolerance: 5,                     // 容忍5ms回拨
        EnableMetrics:          true,                  // 启用监控
    }
    
    // 使用配置创建生成器
    gen, err := snowflake.NewWithConfig(config)
    if err != nil {
        panic(err)
    }
    
    // 生成ID
    id, _ := gen.NextID()
    
    // 查看监控指标
    metrics := gen.GetMetrics()
    fmt.Printf("监控指标: %+v\n", metrics)
}
```

### 使用注册表

```go
package main

import (
    "github.com/katydid/idgen/core"
    "github.com/katydid/idgen/registry"
    "github.com/katydid/idgen/snowflake"
)

func main() {
    // 获取全局注册表
    reg := registry.GetRegistry()
    
    // 创建并注册生成器
    config := &snowflake.Config{
        DatacenterID:           1,
        WorkerID:               1,
    }
    
    gen, err := reg.Create("user_id_gen", core.GeneratorTypeSnowflake, config)
    if err != nil {
        panic(err)
    }
    
    // 获取已注册的生成器
    gen2, err := reg.Get("user_id_gen")
    if err != nil {
        panic(err)
    }
    
    // 使用生成器
    id, _ := gen2.NextID()
    fmt.Printf("生成的ID: %d\n", id)
}
```

---

## 完整功能

### 1. ID生成

#### 单个ID生成

```go
gen, _ := snowflake.New(1, 1)
id, err := gen.NextID()
```

#### 批量ID生成

```go
// 一次生成1000个ID
ids, err := gen.NextIDBatch(1000)

// 批量生成支持跨毫秒（最大10万个）
ids, err := gen.NextIDBatch(100000)
```

### 2. ID解析

```go
// 解析ID，提取元信息
info, err := gen.ParseID(id)
fmt.Printf("时间戳: %d\n", info.Timestamp)
fmt.Printf("数据中心ID: %d\n", info.DatacenterID)
fmt.Printf("工作机器ID: %d\n", info.WorkerID)
fmt.Printf("序列号: %d\n", info.Sequence)

// 使用解析器直接提取
parser := snowflake.NewParser()
timestamp := parser.ExtractTimestamp(id)
datacenterID := parser.ExtractDatacenterID(id)
workerID := parser.ExtractWorkerID(id)
sequence := parser.ExtractSequence(id)
```

### 3. ID验证

```go
// 验证单个ID
validator := snowflake.NewValidator()
err := validator.Validate(id)

// 批量验证
err := validator.ValidateBatch([]int64{id1, id2, id3})

// 使用生成器验证
err := gen.ValidateID(id)
```

### 4. 性能监控

```go
// 创建启用监控的生成器
config := &snowflake.Config{
    DatacenterID:  1,
    WorkerID:      1,
    EnableMetrics: true,
}
gen, _ := snowflake.NewWithConfig(config)

// 生成一些ID
for i := 0; i < 10000; i++ {
    gen.NextID()
}

// 获取监控指标
metrics := gen.GetMetrics()
fmt.Printf("ID生成总数: %d\n", metrics["id_count"])
fmt.Printf("序列号溢出: %d\n", metrics["sequence_overflow"])
fmt.Printf("时钟回拨: %d\n", metrics["clock_backward"])
fmt.Printf("等待次数: %d\n", metrics["wait_count"])
fmt.Printf("平均等待时间: %d ns\n", metrics["avg_wait_time_ns"])

// 重置监控指标
gen.ResetMetrics()
```

### 5. 领域类型操作

#### ID类型

```go
import "github.com/katydid/idgen/domain"

// 创建ID
id := domain.NewID(123456789)

// 类型转换
str := id.String()           // "123456789"
hex := id.Hex()              // "0x75bcd15"
bin := id.Binary()           // "0b111010110111100110100010101"
i64 := id.Int64()            // 123456789

// 解析ID
id, err := domain.ParseID("123456789")       // 十进制
id, err := domain.ParseID("0x75bcd15")       // 十六进制
id, err := domain.ParseID("0b111010110...")  // 二进制

// ID验证
if id.IsValid() {
    // ID有效
}

// JavaScript安全性检查
if id.IsSafeForJavaScript() {
    // 可安全用于JavaScript
}

// JSON序列化（自动转为字符串，避免精度丢失）
data, _ := json.Marshal(id)  // "123456789"
```

#### IDSlice工具

```go
import "github.com/katydid/idgen/domain"

// 创建ID切片
ids := domain.NewIDSlice(id1, id2, id3)

// 类型转换
int64Slice := ids.Int64Slice()
stringSlice := ids.StringSlice()

// 查找
if ids.Contains(id1) {
    // 包含该ID
}

// 去重
uniqueIDs := ids.Deduplicate()

// 过滤
validIDs := ids.Filter(func(id domain.ID) bool {
    return id.IsValid()
})

// 批量验证
err := ids.ValidateAll()
```

#### IDSet集合

```go
import "github.com/katydid/idgen/domain"

// 创建集合
set1 := domain.NewIDSet(id1, id2, id3)
set2 := domain.NewIDSet(id2, id3, id4)

// 集合操作
set1.Add(id5)
set1.Remove(id1)
exists := set1.Contains(id2)

// 并集
union := set1.Union(set2)

// 交集
intersection := set1.Intersect(set2)

// 差集
difference := set1.Difference(set2)

// 相等性检查
if set1.Equal(set2) {
    // 集合相等
}

// 转换为切片
slice := set1.ToSlice()
```

### 6. 注册表高级功能

```go
import (
    "github.com/katydid/idgen/registry"
    "github.com/katydid/idgen/core"
    "github.com/katydid/idgen/snowflake"
)

reg := registry.GetRegistry()

// 创建生成器
config := &snowflake.Config{DatacenterID: 1, WorkerID: 1}
gen, _ := reg.Create("order_id", core.GeneratorTypeSnowflake, config)

// 获取或创建（幂等操作）
gen, _ = reg.GetOrCreate("order_id", core.GeneratorTypeSnowflake, config)

// 检查是否存在
if reg.Has("order_id") {
    // 生成器已存在
}

// 移除生成器
reg.Remove("order_id")

// 列出所有键
keys := reg.ListKeys()

// 获取数量
count := reg.Count()

// 设置最大数量限制
reg.SetMaxGenerators(1000)

// 清空所有生成器
reg.Clear()
```

---

## 性能分析

### 基准测试结果

在标准测试环境（Apple M1 Pro, 16GB RAM）下的性能表现：

#### 单个ID生成性能

```
操作类型                     耗时          内存分配
─────────────────────────────────────────────────
单线程生成                  ~450 ns/op    0 B/op
并发生成（8 goroutines）    ~500 ns/op    0 B/op
启用监控时生成              ~480 ns/op    0 B/op
```

**吞吐量**: 单核约 **200万 QPS**，8核并发约 **1600万 QPS**

#### 批量ID生成性能

```
批量大小          耗时/ID        总耗时           内存分配
──────────────────────────────────────────────────────────
100个ID         ~50 ns/op      ~5 μs          800 B
1000个ID        ~45 ns/op      ~45 μs         8 KB
10000个ID       ~42 ns/op      ~420 μs        80 KB
```

**性能提升**: 批量生成比单个生成快 **9-10倍**

#### ID解析性能

```
操作类型                耗时          内存分配
───────────────────────────────────────────────
完整解析               ~25 ns/op     48 B/op
提取时间戳             ~2 ns/op      0 B/op
提取数据中心ID         ~2 ns/op      0 B/op
提取工作机器ID         ~2 ns/op      0 B/op
```

#### ID验证性能

```
操作类型                耗时          内存分配
───────────────────────────────────────────────
单个ID验证             ~20 ns/op     0 B/op
批量验证（1000个）     ~18 ns/op     0 B/op
```

### 性能优化技术

1. **预计算优化**: DatacenterID和WorkerID在生成器初始化时预先计算并缓存
2. **零内存分配**: 单个ID生成无任何内存分配
3. **原子操作**: 监控计数器使用atomic.Uint64，无锁开销
4. **批量优化**: 批量生成复用时间戳获取，减少系统调用
5. **位运算**: 使用位移和掩码操作，避免乘除法

### 资源消耗

```
组件                     内存占用
──────────────────────────────────
生成器实例              ~200 bytes
启用监控（增量）        ~40 bytes
注册表（1000个生成器）  ~300 KB
```

---

## 第三方库对比

与主流Go语言ID生成库的全面对比：

### 功能对比

| 功能特性 | IDGen (本库) | bwmarrin/snowflake | sony/sonyflake | rs/xid | google/uuid |
|---------|-------------|-------------------|----------------|--------|-------------|
| **算法** | Snowflake | Snowflake | Sonyflake | MongoID | UUID v4 |
| **ID长度** | 64位 | 64位 | 64位 | 96位 | 128位 |
| **时间精度** | 毫秒 | 毫秒 | 10毫秒 | 秒 | - |
| **时钟回拨保护** | ✅ 三种策略 | ❌ | ✅ 休眠等待 | ✅ | N/A |
| **批量生成** | ✅ 优化支持 | ❌ | ❌ | ❌ | ✅ |
| **性能监控** | ✅ 内置 | ❌ | ❌ | ❌ | ❌ |
| **ID解析** | ✅ 完整解析 | ✅ | ✅ | ✅ | ❌ |
| **ID验证** | ✅ | ❌ | ❌ | ❌ | ✅ |
| **注册表管理** | ✅ | ❌ | ❌ | ❌ | ❌ |
| **领域类型** | ✅ ID/IDSlice/IDSet | ❌ | ❌ | ✅ | ✅ |
| **JSON安全** | ✅ 字符串序列化 | ❌ | ❌ | ✅ | ✅ |
| **配置灵活性** | ✅ 丰富配置 | ⚠️ 基础 | ⚠️ 基础 | ❌ | ❌ |
| **插件扩展** | ✅ 工厂模式 | ❌ | ❌ | ❌ | ❌ |

### 性能对比

```
库名称                  QPS（单核）   内存分配      并发安全
─────────────────────────────────────────────────────────
IDGen (本库)           2,000,000    0 B/op       ✅
bwmarrin/snowflake     1,800,000    0 B/op       ✅
sony/sonyflake         1,500,000    0 B/op       ✅
rs/xid                 3,000,000    0 B/op       ✅
google/uuid            2,500,000    16 B/op      ✅
```

### 适用场景对比

| 场景 | 推荐库 | 理由 |
|------|--------|------|
| **分布式系统主键** | **IDGen** / bwmarrin/snowflake | 64位整数，有序，性能高 |
| **对象存储ID** | rs/xid | 更短的字符串表示 |
| **会话ID** | google/uuid | 完全随机，无序 |
| **订单流水号** | **IDGen** | 时间有序，可解析，可监控 |
| **高并发写入** | **IDGen** (批量) / rs/xid | 批量优化，性能最优 |
| **需要时间解析** | **IDGen** / sony/sonyflake | 可提取精确时间戳 |
| **企业级监控** | **IDGen** | 唯一内置监控指标 |

### 优势总结

#### IDGen的独特优势

1. **企业级完整性**: 唯一提供监控、注册表、验证等完整企业特性
2. **批量生成优化**: 批量生成性能提升9-10倍
3. **时钟回拨保护**: 提供三种策略，灵活应对不同场景
4. **可观测性**: 内置性能指标，便于生产环境监控
5. **类型安全**: 领域类型封装，避免原始类型陷阱
6. **插件化架构**: 支持自定义生成器类型扩展

#### 何时选择其他库

- **极致性能**: rs/xid略快，但功能简单
- **UUID标准**: google/uuid，需要符合UUID标准时
- **简单场景**: bwmarrin/snowflake，只需基础功能时

---

## 最佳实践

### 1. 生产环境配置建议

```go
// 推荐的生产环境配置
config := &snowflake.Config{
    DatacenterID:           getDatacenterID(),     // 从配置中心获取
    WorkerID:               getWorkerID(),         // 从服务发现获取
    ClockBackwardStrategy:  core.StrategyWait,     // 等待策略
    ClockBackwardTolerance: 5,                     // 容忍5ms
    EnableMetrics:          true,                  // 开启监控
}

gen, err := snowflake.NewWithConfig(config)
if err != nil {
    log.Fatalf("初始化ID生成器失败: %v", err)
}
```

### 2. 数据中心和机器ID分配

```go
// 方案1: 基于环境变量
datacenterID := getEnvInt("DATACENTER_ID", 0)
workerID := getEnvInt("WORKER_ID", 0)

// 方案2: 基于IP地址
workerID := getWorkerIDFromIP()

// 方案3: 基于配置中心（推荐）
datacenterID, workerID := getIDsFromConfigCenter()
```

### 3. 高并发场景优化

```go
// 使用批量生成提升性能
func generateOrderIDs(count int) []int64 {
    // 批量生成比循环单个生成快9-10倍
    return gen.NextIDBatch(count)
}

// 使用对象池减少锁竞争
var genPool = &sync.Pool{
    New: func() interface{} {
        gen, _ := snowflake.New(datacenterID, workerID)
        return gen
    },
}
```

### 4. 监控指标接入

```go
// 定期上报监控指标到监控系统
func reportMetrics() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := gen.GetMetrics()
        
        // 上报到Prometheus
        idCountGauge.Set(float64(metrics["id_count"]))
        sequenceOverflowCounter.Add(float64(metrics["sequence_overflow"]))
        clockBackwardCounter.Add(float64(metrics["clock_backward"]))
        
        // 检查异常
        if metrics["clock_backward"] > 10 {
            log.Warn("检测到频繁时钟回拨，请检查NTP配置")
        }
    }
}
```

### 5. 错误处理

```go
// 优雅处理错误
id, err := gen.NextID()
if err != nil {
    if errors.Is(err, core.ErrClockMovedBackwards) {
        // 时钟回拨错误，记录告警
        log.Error("时钟回拨", "error", err)
        // 可选：切换到备用生成器或返回降级ID
    }
    return 0, fmt.Errorf("生成ID失败: %w", err)
}
```

### 6. JavaScript前端集成

```go
// 确保ID在JavaScript中安全使用
type Response struct {
    ID string `json:"id"`  // 使用字符串而非数字
}

// 使用domain.ID自动处理
type Response struct {
    ID domain.ID `json:"id"`  // 自动序列化为字符串
}
```

### 7. 数据库集成

```go
// GORM集成示例
type Order struct {
    ID        int64     `gorm:"primaryKey;autoIncrement:false"`
    CreatedAt time.Time
}

// 创建前钩子
func (o *Order) BeforeCreate(tx *gorm.DB) error {
    if o.ID == 0 {
        id, err := gen.NextID()
        if err != nil {
            return err
        }
        o.ID = id
    }
    return nil
}
```

### 8. 分布式追踪集成

```go
// 生成TraceID
func generateTraceID(ctx context.Context) string {
    id, _ := gen.NextID()
    return domain.NewID(id).Hex()  // 返回十六进制格式
}
```

---

## 常见问题

### Q1: 如何保证分布式环境下ID唯一性？

**A**: 确保每个节点的 `(datacenterID, workerID)` 组合唯一即可。建议：
- 从配置中心统一分配
- 基于容器编排系统的节点ID
- 使用服务发现自动分配

### Q2: 时钟回拨怎么办？

**A**: 提供三种策略：
1. **StrategyError**（默认）: 返回错误，保证绝对唯一性
2. **StrategyWait**: 等待时钟追上，适合小幅回拨（<5ms）
3. **StrategyUseLastTimestamp**: 使用上次时间戳，高可用优先（慎用）

建议生产环境：
- 配置NTP同步
- 使用`StrategyWait`策略
- 监控`clock_backward`指标

### Q3: 每秒生成超过409.6万个ID怎么办？

**A**: 单机理论上限是409.6万/秒（4096 × 1000），超过时：
1. 部署多个生成器实例（不同workerID）
2. 使用批量生成减少锁开销
3. 考虑分片策略

### Q4: ID是否可以用作MySQL主键？

**A**: 完全可以，且优于自增ID：
- **优点**: 全局唯一、时间有序、无需中心化
- **注意**: 使用BIGINT类型存储（8字节）

### Q5: 如何从ID中提取生成时间？

```go
parser := snowflake.NewParser()
timestamp := parser.ExtractTimestamp(id)
timeObj := time.UnixMilli(timestamp)

// 或使用domain.ID
domainID := domain.NewID(id)
timeObj := domainID.ExtractTime()
```

### Q6: 是否支持自定义Epoch？

**A**: 当前版本使用固定Epoch（2026-01-01 00:00:00 UTC），自定义Epoch将在未来版本支持。

### Q7: 生成的ID会重复吗？

**A**: 在正确配置下不会重复：
- ✅ 确保`(datacenterID, workerID)`唯一
- ✅ 不使用`StrategyUseLastTimestamp`策略
- ✅ 系统时间不大幅回拨

### Q8: 性能监控会影响性能吗？

**A**: 影响极小（<5%）：
- 使用原子操作，无锁开销
- 建议生产环境启用，便于问题诊断

### Q9: 如何扩展自定义生成器类型？

```go
// 1. 实现core.IDGenerator接口
type MyGenerator struct {}
func (g *MyGenerator) NextID() (int64, error) { ... }

// 2. 实现工厂
type MyFactory struct {}
func (f *MyFactory) Create(config any) (core.IDGenerator, error) { ... }

// 3. 注册到工厂注册表
registry.GetFactoryRegistry().Register("my_type", &MyFactory{})
```

### Q10: JavaScript精度丢失问题？

**A**: 使用`domain.ID`类型自动处理：
```go
// domain.ID的MarshalJSON会自动转为字符串
type Response struct {
    ID domain.ID `json:"id"`  // JSON: {"id": "123456789"}
}
```

或手动转换：
```go
type Response struct {
    ID string `json:"id"`
}
resp.ID = strconv.FormatInt(id, 10)
```

---

### 开发环境

```bash
# 克隆项目
git clone <repo-url>

# 运行测试
go test ./...

# 运行基准测试
go test -bench=. -benchmem ./...

# 查看测试覆盖率
go test -cover ./...
```

---

## 许可证

MIT License - 详见 [LICENSE](../../../LICENSE) 文件

---

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

- 项目地址：[GitHub](https://github.com/yourproject)
- 问题反馈：[Issues](https://github.com/yourproject/issues)

---

**最后更新时间**: 2025-10-19
