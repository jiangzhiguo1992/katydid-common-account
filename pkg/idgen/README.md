# Snowflake ID 生成器

## 📖 简介

Snowflake 是一个高性能的分布式 ID 生成器，基于 Twitter 的 Snowflake 算法实现。

### 特性

- ✅ **高性能**：单实例支持每毫秒生成 4096 个唯一 ID
- ✅ **分布式友好**：支持数据中心 ID 和工作机器 ID，避免冲突
- ✅ **线程安全**：使用互斥锁保证并发安全
- ✅ **批量生成**：支持批量生成 ID，减少锁竞争
- ✅ **时钟回拨处理**：多种策略应对时钟回拨问题
- ✅ **性能监控**：内置监控指标，便于观测
- ✅ **ID 解析**：完整的 ID 解析和验证功能
- ✅ **易于测试**：支持自定义时间函数，便于单元测试

## 🏗️ ID 结构

Snowflake ID 是一个 64 位的正整数，结构如下：

```
+--------------------------------------------------------------------------+
| 1 Bit Unused | 41 Bits Timestamp |  5 Bits DC ID  |  5 Bits Worker ID |  12 Bits Sequence  |
+--------------------------------------------------------------------------+
```

- **时间戳（41位）**：毫秒级时间戳，可使用约 69 年
- **数据中心 ID（5位）**：支持 32 个数据中心（0-31）
- **工作机器 ID（5位）**：每个数据中心支持 32 台机器（0-31）
- **序列号（12位）**：同一毫秒内可生成 4096 个 ID（0-4095）

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "fmt"
    "katydid-common-account/pkg/idgen"
)

func main() {
    // 创建 Snowflake 实例
    // 参数：数据中心ID(0-31), 工作机器ID(0-31)
    sf, err := idgen.NewSnowflake(1, 1)
    if err != nil {
        panic(err)
    }

    // 生成单个 ID
    id, err := sf.NextID()
    if err != nil {
        panic(err)
    }
    fmt.Printf("生成的 ID: %d\n", id)

    // 批量生成 ID（推荐用于批量场景）
    ids, err := sf.NextIDBatch(100)
    if err != nil {
        panic(err)
    }
    fmt.Printf("批量生成了 %d 个 ID\n", len(ids))

    // 解析 ID
    info, err := sf.Parse(id)
    if err != nil {
        panic(err)
    }
    fmt.Printf("ID 信息: %+v\n", info)

    // 获取性能指标
    metrics := sf.GetMetrics()
    fmt.Printf("性能指标: %+v\n", metrics)
}
```

### 高级配置

```go
// 使用配置对象创建实例
sf, err := idgen.NewSnowflakeWithConfig(&idgen.SnowflakeConfig{
    DatacenterID:           1,
    WorkerID:               1,
    ClockBackwardStrategy:  idgen.StrategyWait,  // 时钟回拨策略
    ClockBackwardTolerance: 10,                  // 容忍 10ms 回拨
})
```

## ⚙️ 时钟回拨策略

当检测到系统时钟回拨时，支持三种处理策略：

### 1. StrategyError（默认，最安全）

```go
sf, _ := idgen.NewSnowflakeWithConfig(&idgen.SnowflakeConfig{
    DatacenterID:          1,
    WorkerID:              1,
    ClockBackwardStrategy: idgen.StrategyError,
})
```

- **行为**：直接返回错误
- **优点**：最安全，避免 ID 冲突
- **缺点**：在时钟回拨时服务不可用
- **适用场景**：对数据一致性要求高的场景

### 2. StrategyWait（推荐）

```go
sf, _ := idgen.NewSnowflakeWithConfig(&idgen.SnowflakeConfig{
    DatacenterID:           1,
    WorkerID:               1,
    ClockBackwardStrategy:  idgen.StrategyWait,
    ClockBackwardTolerance: 10, // 容忍 10ms
})
```

- **行为**：等待直到时钟追上
- **优点**：在容忍范围内自动恢复
- **缺点**：可能导致短暂阻塞
- **适用场景**：生产环境推荐使用

### 3. StrategyUseLastTimestamp（不推荐）

```go
sf, _ := idgen.NewSnowflakeWithConfig(&idgen.SnowflakeConfig{
    DatacenterID:          1,
    WorkerID:              1,
    ClockBackwardStrategy: idgen.StrategyUseLastTimestamp,
})
```

- **行为**：使用上次的时间戳
- **优点**：服务始终可用
- **缺点**：可能导致 ID 冲突
- **适用场景**：仅用于特殊场景，不推荐

## 📊 性能监控

### 获取监控指标

```go
metrics := sf.GetMetrics()
fmt.Printf("已生成 ID 总数: %d\n", metrics["id_count"])
fmt.Printf("序列号溢出次数: %d\n", metrics["sequence_overflow"])
fmt.printf("时钟回拨次数: %d\n", metrics["clock_backward"])
fmt.Printf("平均等待时间: %dns\n", metrics["avg_wait_time_ns"])
```

### 可用指标

| 指标 | 说明 |
|------|------|
| `id_count` | 已生成的 ID 总数 |
| `sequence_overflow` | 序列号溢出次数（需要等待下一毫秒） |
| `clock_backward` | 检测到时钟回拨的次数 |
| `wait_count` | 等待下一毫秒的总次数 |
| `avg_wait_time_ns` | 平均等待时间（纳秒） |

## 🔧 API 参考

### 创建实例

```go
// 简单创建
NewSnowflake(datacenterID, workerID int64) (*Snowflake, error)

// 使用配置创建（推荐）
NewSnowflakeWithConfig(config *SnowflakeConfig) (*Snowflake, error)
```

### 生成 ID

```go
// 生成单个 ID
NextID() (int64, error)

// 批量生成 ID（减少锁竞争）
NextIDBatch(n int) ([]int64, error)
```

### ID 解析与验证

```go
// 解析 ID（方法）
Parse(id int64) (*IDInfo, error)

// 解析 ID（全局函数）
ParseSnowflakeID(id int64) (timestamp, datacenterID, workerID, sequence int64)

// 验证 ID 有效性
ValidateSnowflakeID(id int64) error

// 提取时间戳
GetTimestamp(id int64) time.Time
```

### 监控与信息

```go
// 获取性能指标
GetMetrics() map[string]uint64

// 获取已生成的 ID 数量
GetIDCount() uint64

// 获取工作机器 ID
GetWorkerID() int64

// 获取数据中心 ID
GetDatacenterID() int64

// 重置指标（仅用于测试）
ResetMetrics()
```

## 📈 性能基准

运行基准测试：

```bash
cd pkg/idgen
go test -bench=. -benchmem -benchtime=3s
```

### 预期性能指标

| 场景 | 目标性能 |
|------|---------|
| 单线程生成 | >= 100万 ops/s |
| 并发生成（10个goroutine） | >= 80万 ops/s |
| 并发生成（100个goroutine） | >= 50万 ops/s |
| 批量生成（100个/批） | 吞吐量提升 10-20% |
| ID 解析 | >= 1000万 ops/s |

## 🧪 测试

```bash
# 运行所有测试
go test -v

# 运行基准测试
go test -bench=. -benchmem

# 运行并发测试
go test -v -run=TestConcurrency

# 运行批量生成测试
go test -v -run=TestNextIDBatch
```

## 💡 最佳实践

### 1. 合理分配 ID

```go
// 不同数据中心使用不同的 datacenterID
// 北京机房：datacenterID = 1
sf_bj, _ := idgen.NewSnowflake(1, 1)

// 上海机房：datacenterID = 2
sf_sh, _ := idgen.NewSnowflake(2, 1)
```

### 2. 批量生成场景

```go
// 批量初始化数据时使用 NextIDBatch
ids, err := sf.NextIDBatch(1000)
if err != nil {
    return err
}

for i, id := range ids {
    records[i].ID = id
}
```

### 3. 错误处理

```go
id, err := sf.NextID()
if err != nil {
    if errors.Is(err, idgen.ErrClockMovedBackwards) {
        // 处理时钟回拨
        log.Warn("检测到时钟回拨", "error", err)
    }
    return err
}
```

### 4. 生产环境配置

```go
sf, err := idgen.NewSnowflakeWithConfig(&idgen.SnowflakeConfig{
    DatacenterID:           getDatacenterID(),    // 从配置获取
    WorkerID:               getWorkerID(),         // 从配置获取
    ClockBackwardStrategy:  idgen.StrategyWait,   // 容忍短暂回拨
    ClockBackwardTolerance: 10,                    // 容忍 10ms
})
```

## ⚠️ 注意事项

1. **避免 ID 冲突**：确保同一集群中不同实例的 `datacenterID` 和 `workerID` 组合唯一
2. **时钟同步**：使用 NTP 保持服务器时钟同步，避免时钟回拨
3. **实例复用**：创建的 Snowflake 实例应该复用，不要频繁创建
4. **批量生成限制**：单次批量生成最多 4096 个 ID
5. **监控指标**：定期检查 `clock_backward` 指标，及时发现时钟问题

## 🔄 版本历史

### v2.0.0（当前版本）

**新增功能：**
- ✨ 批量生成 ID 接口（`NextIDBatch`）
- ✨ 可配置的时钟回拨策略
- ✨ 增强的性能监控指标
- ✨ 详细的等待时间统计

**改进：**
- 🚀 优化了锁粒度，提升并发性能
- 📊 增加了序列号溢出和时钟回拨的监控
- 📝 完善了文档和示例

### v1.0.0

- 基础的 Snowflake ID 生成功能
- ID 解析和验证
- 基础的时钟回拨处理

## 📚 相关资源

- [Twitter Snowflake 原理](https://github.com/twitter-archive/snowflake/tree/snowflake-2010)
- [分布式 ID 生成方案对比](https://tech.meituan.com/2017/04/21/mt-leaf.html)

## 📄 许可证

本项目遵循项目根目录的许可证。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

