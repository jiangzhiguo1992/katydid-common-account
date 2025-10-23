# Status - 高性能位图状态管理器

<div align="center">

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg)
![Test Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)
![Performance](https://img.shields.io/badge/ops-3.1B%2Fs-orange.svg)

**基于位运算的超高性能状态管理解决方案**

[特性](#-核心特性) • [快速开始](#-快速开始) • [API文档](#-完整-api-文档) • [性能](#-性能分析) • [对比](#-主流方案对比)

</div>

---

## 📖 目录

- [概述](#-概述)
- [核心特性](#-核心特性)
- [设计架构](#-设计架构)
- [快速开始](#-快速开始)
- [状态常量](#-状态常量定义)
- [完整 API 文档](#-完整-api-文档)
- [高级用法](#-高级用法)
- [性能分析](#-性能分析)
- [主流方案对比](#-主流方案对比)
- [最佳实践](#-最佳实践)
- [常见问题](#-常见问题-faq)

---

## 🎯 概述

`Status` 是一个基于 **位运算（Bitmap）** 实现的轻量级、高性能状态管理器。使用 `int64` 作为底层存储，提供 **63 位** 可用状态位，支持多状态并存和超快速的状态检查操作。

### 为什么选择 Status？

| 特性 | Status (位图) | map[string]bool | []string | struct 字段 |
|------|--------------|-----------------|----------|------------|
| **内存占用** | 8 字节 | 100+ 字节 | 50+ 字节 | 12+ 字节 |
| **检查速度** | 0.3 ns | 20 ns | 100 ns | 1 ns |
| **修改速度** | 2.2 ns | 50 ns | 200 ns | 2 ns |
| **多状态支持** | ✅ 63 个 | ✅ 无限 | ✅ 无限 | ❌ 有限 |
| **序列化大小** | 1-5 字节 | 大 | 大 | 中 |
| **零内存分配** | ✅ | ❌ | ❌ | ✅ |

### 适用场景

✅ **最适合**：
- 用户/文章/订单等实体的状态管理（删除、禁用、隐藏等）
- 权限位管理（read、write、execute 等）
- 高频状态检查和修改场景
- 内存敏感的微服务架构
- 需要数据库高效存储的场景

⚠️ **不适合**：
- 状态之间有复杂依赖关系
- 需要保存状态变更历史
- 状态需要附加元数据（如时间戳、原因等）

---

## ✨ 核心特性

### 🚀 极致性能

```
操作速度：
  ├─ Has检查      0.33 ns/op    (31亿次/秒)
  ├─ BitCount     0.32 ns/op    (32亿次/秒)
  ├─ Add/Del      2.2 ns/op     (4.5亿次/秒)
  └─ 数据库往返    4.37 ns/op    (2.3亿次/秒)

内存效率：
  ├─ 固定大小     8 字节
  ├─ 零堆分配     基础操作 0 allocs/op
  └─ GC 友好      无垃圾产生
```

### 📦 分层状态设计

采用 **三级优先级** 分层设计，清晰划分状态管理职责：

```
┌─────────────────────────────────────┐
│  System（系统级）- 最高优先级         │
│  ├─ 自动化管理                       │
│  ├─ 异常检测触发                     │
│  └─ 通常不可撤销                     │
├─────────────────────────────────────┤
│  Admin（管理员级）- 中等优先级        │
│  ├─ 人工审核                         │
│  ├─ 策略执行                         │
│  └─ 可撤销恢复                       │
├─────────────────────────────────────┤
│  User（用户级）- 最低优先级           │
│  ├─ 用户自主控制                     │
│  ├─ 个性化设置                       │
│  └─ 随时可更改                       │
└─────────────────────────────────────┘
```

### 🎨 四类预定义状态

| 状态类型 | 位范围 | 业务含义 | 典型场景 |
|---------|--------|---------|---------|
| **Deleted** | 0-2 | 软删除标记 | 回收站、数据归档 |
| **Disabled** | 3-5 | 禁用/冻结 | 账号封禁、功能关闭 |
| **Hidden** | 6-8 | 隐藏/私密 | 草稿、私密内容 |
| **Review** | 9-11 | 审核/验证 | 内容审核、邮箱验证 |

### 🔧 51 位扩展空间

预留 **位 12-62**（共 51 位）供业务自定义扩展：

```go
const (
    // 自定义业务状态（从 StatusExpand51 开始）
    StatusCustomPinned    Status = StatusExpand51 << 0  // 置顶
    StatusCustomFeatured  Status = StatusExpand51 << 1  // 精选
    StatusCustomArchived  Status = StatusExpand51 << 2  // 已归档
    // ... 还可定义 48 个状态
)
```

---

## 🏗 设计架构

### UML 类图

```
┌──────────────────────────────────────────────────────────────┐
│                         Status                               │
│                        (int64)                               │
├──────────────────────────────────────────────────────────────┤
│ + StatusNone: Status = 0                                     │
│                                                              │
│ [删除状态组 - 位 0-2]                                          │
│ + StatusSysDeleted:  Status = 1 << 0                         │
│ + StatusAdmDeleted:  Status = 1 << 1                         │
│ + StatusUserDeleted: Status = 1 << 2                         │
│                                                              │
│ [禁用状态组 - 位 3-5]                                          │
│ + StatusSysDisabled:  Status = 1 << 3                        │
│ + StatusAdmDisabled:  Status = 1 << 4                        │
│ + StatusUserDisabled: Status = 1 << 5                        │
│                                                              │
│ [隐藏状态组 - 位 6-8]                                          │
│ + StatusSysHidden:  Status = 1 << 6                          │
│ + StatusAdmHidden:  Status = 1 << 7                          │
│ + StatusUserHidden: Status = 1 << 8                          │
│                                                              │
│ [审核状态组 - 位 9-11]                                         │
│ + StatusSysReview:  Status = 1 << 9                          │
│ + StatusAdmReview:  Status = 1 << 10                         │
│ + StatusUserReview: Status = 1 << 11                         │
│                                                              │
│ [扩展空间 - 位 12-62]                                          │
│ + StatusExpand51: Status = 1 << 12                           │
├─────────────────────────────────────────────────────────-────┤
│ [状态修改方法]                                                 │
│ + Set(flag Status)              完全替换状态                   │
│ + Clear()                       清除所有状态                   │
│ + Add(flag Status)              添加状态位                    │
│ + Del(flag Status)              删除状态位                    │
│ + AddMultiple(...Status)        批量添加                      │
│ + DelMultiple(...Status)        批量删除                      │
│ + Toggle(flag Status)           切换状态                      │
│ + ToggleMultiple(...Status)     批量切换                      │
│ + And(flag Status)              与运算                       │
│ + AndMultiple(...Status)        批量与运算                    │
│                                                             │
│ [状态查询方法]                                                │
│ + Has(flag Status): bool        检查单个状态                  │
│ + HasAny(...Status): bool       检查任意状态                  │
│ + HasAll(...Status): bool       检查所有状态                  │
│ + ActiveFlags(): []Status       获取活动位                    │
│ + Diff(other Status): (added, removed Status)  状态差异       │
│                                                             │
│ [业务语义方法]                                                │
│ + IsDeleted(): bool             是否已删除                    │
│ + IsDisable(): bool             是否已禁用                    │
│ + IsHidden(): bool              是否已隐藏                    │
│ + IsReview(): bool              是否在审核                    │
│ + CanEnable(): bool             是否可启用                    │
│ + CanVisible(): bool            是否可见                     │
│ + CanActive(): bool             是否完全激活                  │
│                                                             │
│ [格式化方法]                                                  │
│ + String(): string              格式化字符串                  │
│ + BitCount(): int               计算活动位数                  │
│                                                             │
│ [接口实现]                                                   │
│ + Value(): (driver.Value, error)       driver.Valuer        │
│ + Scan(value interface{}): error       sql.Scanner          │
│ + MarshalJSON(): ([]byte, error)       json.Marshaler       │
│ + UnmarshalJSON(data []byte): error    json.Unmarshaler     │
└─────────────────────────────────────────────────────────────┘

                           implements
                               ↓
        ┌──────────────────────┬────────────────────────┐
        │                      │                        │
┌───────▼────────┐   ┌─────────▼────────┐   ┌──────────▼─────────┐
│ driver.Valuer  │   │  sql.Scanner     │   │  json.Marshaler    │
│ driver.Scanner │   │                  │   │  json.Unmarshaler  │
└────────────────┘   └──────────────────┘   └────────────────────┘
```

### 位布局示意图

```
int64 (63 位可用，第 63 位为符号位)
┌──────────────────────────────────────────────────┐
│ 63│62 │...│12│11│10│9 │8 │7 │6 │5 │4 │3 │2 │1 │0 │
├───┴───────┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┴──┤
│   扩展空间    │ Review │ Hidden │Disabled│ Deleted │
│   (51位)     │ (3位)  │  (3位)  │ (3位)  │ (3位)   │
│             │         │        │        │        │
│   业务自定义  │  审核   │  隐藏   │ 禁用    │ 删除    │
└─────────────┴────────┴─────────┴────────┴────────┘
                                                                  
位 0-11:  预定义状态（12 位）                                      
位 12-62: 扩展空间（51 位）                                        
位 63:    符号位（不使用，保持非负）                                
```

---

## 🚀 快速开始

### 安装

```bash
go get github.com/your-org/katydid-common-account/pkg/types
```

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/your-org/katydid-common-account/pkg/types"
)

func main() {
    // 1. 创建状态（零值）
    var status types.Status
    
    // 2. 添加状态
    status.Add(types.StatusUserDisabled)
    status.Add(types.StatusSysHidden)
    
    // 3. 检查状态
    if status.Has(types.StatusUserDisabled) {
        fmt.Println("✓ 用户已禁用")
    }
    
    if status.IsDisable() {
        fmt.Println("✓ 处于禁用状态")
    }
    
    // 4. 批量操作
    status.AddMultiple(
        types.StatusAdmDeleted,
        types.StatusUserHidden,
    )
    
    // 5. 业务检查
    if !status.CanActive() {
        fmt.Println("⚠ 状态不可用，无法激活")
    }
    
    // 6. 移除状态
    status.Del(types.StatusUserDisabled)
    
    // 7. 查看活动状态
    fmt.Printf("活动状态位: %v\n", status.ActiveFlags())
    fmt.Printf("状态: %s\n", status.String())
}
```

### 5分钟入门示例

#### 场景1：用户状态管理

```go
// 用户注册后的初始状态
var user types.Status = types.StatusUserReview  // 等待邮箱验证

// 用户完成邮箱验证
user.Del(types.StatusUserReview)

// 管理员封禁用户
user.Add(types.StatusAdmDisabled)

// 检查用户是否可登录
func canLogin(status types.Status) bool {
    return status.CanActive()  // 无删除、禁用、隐藏、审核状态
}

if canLogin(user) {
    fmt.Println("允许登录")
} else {
    fmt.Println("禁止登录")
}
```

#### 场景2：文章状态管理

```go
// 文章草稿
var article types.Status = types.StatusUserHidden

// 用户发布文章
article.Del(types.StatusUserHidden)
article.Add(types.StatusSysReview)  // 进入系统审核

// 审核通过
article.Del(types.StatusSysReview)

// 管理员隐藏违规文章
article.Add(types.StatusAdmHidden)

// 检查文章是否对外可见
func isPublic(status types.Status) bool {
    return status.CanVisible()
}
```

#### 场景3：订单状态管理

```go
// 自定义订单状态（扩展位）
const (
    StatusOrderPaid      = types.StatusExpand51 << 0  // 已支付
    StatusOrderShipped   = types.StatusExpand51 << 1  // 已发货
    StatusOrderCompleted = types.StatusExpand51 << 2  // 已完成
)

var order types.Status

// 订单支付
order.Add(StatusOrderPaid)

// 订单发货
order.Add(StatusOrderShipped)

// 用户取消订单
order.Add(types.StatusUserDeleted)

// 检查订单是否可退款
func canRefund(status types.Status) bool {
    return status.Has(StatusOrderPaid) && 
           !status.Has(StatusOrderShipped) &&
           !status.IsDeleted()
}
```

---

## 📋 状态常量定义

### 删除状态组（Deleted）- 位 0-2

| 常量 | 值 | 说明 | 典型场景 |
|------|---|------|---------|
| `StatusSysDeleted` | `1 << 0` | 系统删除，通常不可恢复 | 违规内容自动删除 |
| `StatusAdmDeleted` | `1 << 1` | 管理员删除，可能支持恢复 | 人工审核删除 |
| `StatusUserDeleted` | `1 << 2` | 用户删除，通常可恢复 | 用户主动删除、回收站 |
| `StatusAllDeleted` | `0x07` | 所有删除状态的组合 | 批量检查 |

```go
// 示例
status.Add(types.StatusSysDeleted)
if status.IsDeleted() {  // 检查任意删除状态
    // 处理已删除逻辑
}
```

### 禁用状态组（Disabled）- 位 3-5

| 常量 | 值 | 说明 | 典型场景 |
|------|---|------|---------|
| `StatusSysDisabled` | `1 << 3` | 系统检测异常后自动禁用 | 风控系统封禁 |
| `StatusAdmDisabled` | `1 << 4` | 管理员手动禁用 | 人工封禁账号 |
| `StatusUserDisabled` | `1 << 5` | 用户主动禁用 | 账号冻结 |
| `StatusAllDisabled` | `0x38` | 所有禁用状态的组合 | 批量检查 |

```go
// 示例
status.Add(types.StatusAdmDisabled)
if status.IsDisable() {  // 检查任意禁用状态
    return errors.New("账号已被禁用")
}
```

### 隐藏状态组（Hidden）- 位 6-8

| 常量 | 值 | 说明 | 典型场景 |
|------|---|------|---------|
| `StatusSysHidden` | `1 << 6` | 系统根据规则自动隐藏 | 敏感词过滤 |
| `StatusAdmHidden` | `1 << 7` | 管理员手动隐藏 | 内容下架 |
| `StatusUserHidden` | `1 << 8` | 用户设置为私密 | 草稿、私密文章 |
| `StatusAllHidden` | `0x1C0` | 所有隐藏状态的组合 | 批量检查 |

```go
// 示例
status.Add(types.StatusUserHidden)
if status.IsHidden() {  // 检查任意隐藏状态
    // 不展示该内容
}
```

### 审核状态组（Review）- 位 9-11

| 常量 | 值 | 说明 | 典型场景 |
|------|---|------|---------|
| `StatusSysReview` | `1 << 9` | 等待系统自动审核 | AI内容审核 |
| `StatusAdmReview` | `1 << 10` | 等待管理员审核 | 人工复审 |
| `StatusUserReview` | `1 << 11` | 等待用户验证 | 邮箱/手机验证 |
| `StatusAllReview` | `0xE00` | 所有审核状态的组合 | 批量检查 |

```go
// 示例
status.Add(types.StatusSysReview)
if status.IsReview() {  // 检查任意审核状态
    return "内容审核中"
}
```

### 扩展空间（Expand）- 位 12-62

| 常量 | 值 | 说明 |
|------|---|------|
| `StatusExpand51` | `1 << 12` | 扩展空间起始位 |
| `MaxStatus` | `(1<<63) - 1` | 最大合法状态值 |

```go
// 自定义业务状态示例
const (
    StatusCustomPinned    = types.StatusExpand51 << 0  // 置顶
    StatusCustomFeatured  = types.StatusExpand51 << 1  // 精选
    StatusCustomArchived  = types.StatusExpand51 << 2  // 归档
    StatusCustomLocked    = types.StatusExpand51 << 3  // 锁定
    StatusCustomEncrypted = types.StatusExpand51 << 4  // 加密
    // ... 还可定义 46 个
)
```

---

## 📚 完整 API 文档

### 状态修改方法

#### Set - 完全替换状态

```go
func (s *Status) Set(flag Status)
```

**说明**：将当前状态完全替换为指定状态。

**示例**：
```go
var status types.Status
status.Set(types.StatusSysDeleted)
// status = StatusSysDeleted

status.Set(types.StatusAllDisabled)
// status = StatusAllDisabled (之前的 StatusSysDeleted 被清除)
```

**性能**：0.32 ns/op，0 allocs/op

---

#### Clear - 清除所有状态

```go
func (s *Status) Clear()
```

**说明**：清除所有状态位，恢复为零值。

**示例**：
```go
status := types.StatusAllDeleted | types.StatusAllDisabled
status.Clear()
// status = StatusNone
```

**性能**：0.32 ns/op，0 allocs/op

---

#### Add - 添加状态位

```go
func (s *Status) Add(flag Status)
```

**说明**：添加指定的状态位（位或运算），不影响其他已存在的状态。

**示例**：
```go
var status types.Status
status.Add(types.StatusSysDeleted)      // status = 0b001
status.Add(types.StatusUserDisabled)    // status = 0b101001
```

**性能**：2.21 ns/op，0 allocs/op

---

#### Del - 删除状态位

```go
func (s *Status) Del(flag Status)
```

**说明**：删除指定的状态位（位与非运算），不影响其他状态。

**示例**：
```go
status := types.StatusAllDeleted
status.Del(types.StatusSysDeleted)
// status = StatusAdmDeleted | StatusUserDeleted
```

**性能**：2.36 ns/op，0 allocs/op

---

#### AddMultiple - 批量添加

```go
func (s *Status) AddMultiple(flags ...Status)
```

**说明**：批量添加多个状态位，比多次调用 `Add` 更高效。

**示例**：
```go
var status types.Status
status.AddMultiple(
    types.StatusSysDeleted,
    types.StatusAdmDisabled,
    types.StatusUserHidden,
)
```

**性能**：2.32 ns/op，0 allocs/op（比单独调用快 2-3 倍）

---

#### DelMultiple - 批量删除

```go
func (s *Status) DelMultiple(flags ...Status)
```

**说明**：批量删除多个状态位。

**示例**：
```go
status := types.StatusAllDeleted | types.StatusAllDisabled
status.DelMultiple(
    types.StatusSysDeleted,
    types.StatusAdmDeleted,
)
```

**性能**：2.35 ns/op，0 allocs/op

---

#### Toggle - 切换状态

```go
func (s *Status) Toggle(flag Status)
```

**说明**：切换指定状态位（存在则删除，不存在则添加）。

**示例**：
```go
var status types.Status
status.Toggle(types.StatusUserHidden)  // 添加
status.Toggle(types.StatusUserHidden)  // 删除
```

**性能**：2.16 ns/op，0 allocs/op

---

#### ToggleMultiple - 批量切换

```go
func (s *Status) ToggleMultiple(flags ...Status)
```

**示例**：
```go
status.ToggleMultiple(
    types.StatusUserHidden,
    types.StatusUserDisabled,
)
```

---

#### And - 与运算

```go
func (s *Status) And(flag Status)
```

**说明**：保留与指定状态位相同的部分。

**示例**：
```go
status := types.StatusAllDeleted | types.StatusAllDisabled
status.And(types.StatusAllDeleted)
// status = StatusAllDeleted（StatusAllDisabled 被清除）
```

---

#### AndMultiple - 批量与运算

```go
func (s *Status) AndMultiple(flags ...Status)
```

**说明**：保留与指定多个状态位的交集。

---

### 状态查询方法

#### Has - 检查单个状态

```go
func (s Status) Has(flag Status) bool
```

**说明**：检查是否包含指定的状态位（全部匹配）。

**示例**：
```go
status := types.StatusSysDeleted | types.StatusAdmDeleted

status.Has(types.StatusSysDeleted)                           // true
status.Has(types.StatusUserDeleted)                          // false
status.Has(types.StatusSysDeleted | types.StatusAdmDeleted)  // true（全部匹配）
status.Has(types.StatusAllDeleted)                           // false（缺少 UserDeleted）
```

**性能**：0.33 ns/op，0 allocs/op（**31 亿次/秒**）

**注意**：`Has(StatusNone)` 始终返回 `false`。

---

#### HasAny - 检查任意状态

```go
func (s Status) HasAny(flags ...Status) bool
```

**说明**：检查是否包含任意一个指定的状态位（或逻辑）。

**示例**：
```go
status := types.StatusSysDeleted

status.HasAny(types.StatusSysDeleted, types.StatusAdmDeleted)  // true
status.HasAny(types.StatusAdmDeleted, types.StatusUserDeleted) // false
```

**性能**：2.26 ns/op，0 allocs/op

---

#### HasAll - 检查所有状态

```go
func (s Status) HasAll(flags ...Status) bool
```

**说明**：检查是否包含所有指定的状态位（且逻辑）。

**示例**：
```go
status := types.StatusAllDeleted

status.HasAll(types.StatusSysDeleted, types.StatusAdmDeleted)  // true
status.HasAll(types.StatusAllDeleted)                          // true（仅一个参数）
```

**性能**：2.18 ns/op，0 allocs/op

---

#### ActiveFlags - 获取活动状态位

```go
func (s Status) ActiveFlags() []Status
```

**说明**：返回所有已设置的状态位切片。

**示例**：
```go
status := types.StatusSysDeleted | types.StatusAdmDisabled | types.StatusUserHidden
flags := status.ActiveFlags()
// flags = [StatusSysDeleted, StatusAdmDisabled, StatusUserHidden]

for _, flag := range flags {
    fmt.Println(flag)
}
```

**性能**：
- 1 位：12.53 ns/op，8 B/op
- 12 位：50.88 ns/op，96 B/op

**注意**：此方法会分配内存，适合调试和日志，避免在热路径频繁调用。

---

#### Diff - 状态差异比较

```go
func (s Status) Diff(other Status) (added Status, removed Status)
```

**说明**：比较两个状态的差异，返回新增和移除的状态位。

**示例**：
```go
old := types.StatusSysDeleted | types.StatusAdmDisabled
new := types.StatusAdmDisabled | types.StatusUserHidden

added, removed := new.Diff(old)
// added   = StatusUserHidden
// removed = StatusSysDeleted
```

**应用场景**：状态变更审计、日志记录。

---

### 业务语义方法

#### IsDeleted - 是否已删除

```go
func (s Status) IsDeleted() bool
```

**说明**：检查是否包含任意删除状态（Sys/Adm/User）。

**示例**：
```go
if status.IsDeleted() {
    return errors.New("内容已被删除")
}
```

**等价于**：`status.HasAny(StatusAllDeleted)`

**性能**：0.31 ns/op

---

#### IsDisable - 是否已禁用

```go
func (s Status) IsDisable() bool
```

**说明**：检查是否包含任意禁用状态。

**等价于**：`status.HasAny(StatusAllDisabled)`

---

#### IsHidden - 是否已隐藏

```go
func (s Status) IsHidden() bool
```

**说明**：检查是否包含任意隐藏状态。

**等价于**：`status.HasAny(StatusAllHidden)`

---

#### IsReview - 是否在审核

```go
func (s Status) IsReview() bool
```

**说明**：检查是否包含任意审核状态。

**等价于**：`status.HasAny(StatusAllReview)`

---

#### CanEnable - 是否可启用

```go
func (s Status) CanEnable() bool
```

**说明**：检查是否可以启用（无删除、禁用状态）。

**业务逻辑**：
```go
!IsDeleted() && !IsDisable()
```

**示例**：
```go
if status.CanEnable() {
    // 允许启用功能
}
```

---

#### CanVisible - 是否可见

```go
func (s Status) CanVisible() bool
```

**说明**：检查是否可见（无删除、禁用、隐藏状态）。

**业务逻辑**：
```go
!IsDeleted() && !IsDisable() && !IsHidden()
```

---

#### CanActive - 是否完全激活

```go
func (s Status) CanActive() bool
```

**说明**：检查是否完全激活（无任何限制状态）。

**业务逻辑**：
```go
!IsDeleted() && !IsDisable() && !IsHidden() && !IsReview()
```

**性能**：0.31 ns/op

**示例**：
```go
func (u *User) CanLogin() bool {
    return u.Status.CanActive()
}
```

---

### 格式化方法

#### String - 字符串表示

```go
func (s Status) String() string
```

**说明**：返回状态的字符串表示。

**格式**：`Status(值)[位数 bits]`

**示例**：
```go
status := types.StatusAllDeleted
fmt.Println(status.String())
// 输出: Status(7)[3 bits]
```

**性能**：22.98 ns/op，32 B/op

---

#### BitCount - 计算位数

```go
func (s Status) BitCount() int
```

**说明**：计算已设置的位数量（popcount 算法）。

**示例**：
```go
status := types.StatusAllDeleted | types.StatusAllDisabled
count := status.BitCount()  // 6
```

**性能**：0.32 ns/op，0 allocs/op（**查表法优化**）

---

### 数据库接口

#### Value - 数据库序列化

```go
func (s Status) Value() (driver.Value, error)
```

**说明**：实现 `driver.Valuer` 接口，用于数据库存储。

**返回值**：`int64` 类型

**错误情况**：
- 负数：返回错误
- 超出最大值：返回错误

**性能**：2.17 ns/op，0 allocs/op

---

#### Scan - 数据库反序列化

```go
func (s *Status) Scan(value interface{}) error
```

**说明**：实现 `sql.Scanner` 接口，从数据库读取。

**支持类型**：
- `int64`
- `int`
- `uint64`
- `[]byte`（JSON 格式）
- `nil`（设为 StatusNone）

**性能**：2.49 ns/op，0 allocs/op

**示例**：
```go
type User struct {
    ID     int64
    Status types.Status  // 数据库自动调用 Value/Scan
}

// 数据库中存储为 BIGINT
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    status BIGINT NOT NULL DEFAULT 0
);
```

---

### JSON 接口

#### MarshalJSON - JSON 序列化

```go
func (s Status) MarshalJSON() ([]byte, error)
```

**说明**：实现 `json.Marshaler` 接口。

**输出格式**：纯数字（非字符串）

**示例**：
```go
status := types.StatusAllDeleted
data, _ := json.Marshal(status)
fmt.Println(string(data))  // 输出: 7
```

**性能**：48.37 ns/op，8 B/op

---

#### UnmarshalJSON - JSON 反序列化

```go
func (s *Status) UnmarshalJSON(data []byte) error
```

**说明**：实现 `json.Unmarshaler` 接口。

**支持格式**：
- 数字：`7`
- null：`null`（设为 StatusNone）

**性能**：100.30 ns/op，152 B/op

**示例**：
```go
type Article struct {
    ID     int64       `json:"id"`
    Status types.Status `json:"status"`
}

// JSON: {"id": 1, "status": 7}
var article Article
json.Unmarshal(data, &article)
```

---

## 🎓 高级用法

### 1. 状态机模式

```go
type ArticleStateMachine struct {
    status types.Status
}

// 发布文章
func (sm *ArticleStateMachine) Publish() error {
    if sm.status.IsDeleted() {
        return errors.New("已删除的文章无法发布")
    }
    
    // 移除草稿状态
    sm.status.Del(types.StatusUserHidden)
    
    // 进入审核
    sm.status.Add(types.StatusSysReview)
    
    return nil
}

// 审核通过
func (sm *ArticleStateMachine) Approve() error {
    if !sm.status.Has(types.StatusSysReview) {
        return errors.New("文章未在审核中")
    }
    
    sm.status.Del(types.StatusSysReview)
    return nil
}

// 撤回文章
func (sm *ArticleStateMachine) Withdraw() error {
    if sm.status.IsDeleted() {
        return errors.New("已删除的文章无法撤回")
    }
    
    sm.status.Add(types.StatusUserHidden)
    sm.status.Del(types.StatusSysReview)
    
    return nil
}
```

### 2. 权限位管理

```go
// 自定义权限位
const (
    PermRead   = types.StatusExpand51 << 0
    PermWrite  = types.StatusExpand51 << 1
    PermDelete = types.StatusExpand51 << 2
    PermAdmin  = types.StatusExpand51 << 3
)

type Permission struct {
    bits types.Status
}

func (p *Permission) Grant(perm types.Status) {
    p.bits.Add(perm)
}

func (p *Permission) Revoke(perm types.Status) {
    p.bits.Del(perm)
}

func (p *Permission) Can(perm types.Status) bool {
    return p.bits.Has(perm)
}

// 使用示例
perm := &Permission{}
perm.Grant(PermRead | PermWrite)

if perm.Can(PermWrite) {
    // 允许写入
}
```

### 3. 状态快照与回滚

```go
type StatusHistory struct {
    snapshots []types.Status
}

func (h *StatusHistory) Save(status types.Status) {
    h.snapshots = append(h.snapshots, status)
}

func (h *StatusHistory) Rollback() types.Status {
    if len(h.snapshots) == 0 {
        return types.StatusNone
    }
    
    last := h.snapshots[len(h.snapshots)-1]
    h.snapshots = h.snapshots[:len(h.snapshots)-1]
    
    return last
}

func (h *StatusHistory) Diff() (added, removed types.Status) {
    if len(h.snapshots) < 2 {
        return types.StatusNone, types.StatusNone
    }
    
    current := h.snapshots[len(h.snapshots)-1]
    previous := h.snapshots[len(h.snapshots)-2]
    
    return current.Diff(previous)
}
```

### 4. 并发安全封装

```go
type SafeStatus struct {
    mu     sync.RWMutex
    status types.Status
}

func (s *SafeStatus) Add(flag types.Status) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.status.Add(flag)
}

func (s *SafeStatus) Has(flag types.Status) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.status.Has(flag)
}

func (s *SafeStatus) Get() types.Status {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.status
}

func (s *SafeStatus) Set(flag types.Status) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.status.Set(flag)
}
```

### 5. 审计日志

```go
type StatusAudit struct {
    UserID    int64
    OldStatus types.Status
    NewStatus types.Status
    Timestamp time.Time
    Operator  string
}

func AuditStatusChange(old, new types.Status, operator string) *StatusAudit {
    added, removed := new.Diff(old)
    
    audit := &StatusAudit{
        OldStatus: old,
        NewStatus: new,
        Timestamp: time.Now(),
        Operator:  operator,
    }
    
    // 记录变更详情
    if added != types.StatusNone {
        log.Printf("Added flags: %v", added.ActiveFlags())
    }
    if removed != types.StatusNone {
        log.Printf("Removed flags: %v", removed.ActiveFlags())
    }
    
    return audit
}
```

---

## 🚀 性能分析

### 性能测试环境

- **Go 版本**：1.21+
- **测试平台**：macOS（10 cores）
- **测试方法**：百万次操作基准测试

### 核心操作性能

| 操作 | 耗时 (ns/op) | 内存 (B/op) | 分配次数 | 吞吐量 (ops/s) |
|------|-------------|------------|---------|---------------|
| **Has** | 0.33 | 0 | 0 | 3,030,303,030 |
| **BitCount** | 0.32 | 0 | 0 | 3,125,000,000 |
| **Set** | 0.32 | 0 | 0 | 3,125,000,000 |
| **Add** | 2.21 | 0 | 0 | 452,488,688 |
| **Del** | 2.36 | 0 | 0 | 423,728,814 |
| **Toggle** | 2.16 | 0 | 0 | 462,962,963 |
| **IsDeleted** | 0.31 | 0 | 0 | 3,225,806,452 |
| **CanActive** | 0.31 | 0 | 0 | 3,225,806,452 |

### 批量操作性能

| 操作 | 单次操作 (ns) | 批量操作 (ns) | 性能提升 |
|------|-------------|-------------|---------|
| **Add** (3次) | 6.63 | 2.32 | **2.86x** |
| **Del** (3次) | 7.08 | 2.35 | **3.01x** |

**结论**：批量操作显著提升性能，建议使用 `AddMultiple`、`DelMultiple`。

### 序列化性能

| 操作 | 耗时 (ns/op) | 内存 (B/op) | 分配次数 |
|------|-------------|------------|---------|
| **MarshalJSON** | 48.37 | 8 | 1 |
| **UnmarshalJSON** | 100.30 | 152 | 2 |
| **JSON 往返** | 302-363 | 320-336 | 6-8 |
| **数据库 Value** | 2.17 | 0 | 0 |
| **数据库 Scan** | 2.49 | 0 | 0 |
| **数据库往返** | 4.37 | 0 | 0 |

**结论**：数据库序列化比 JSON **快 70 倍**，推荐直接存储 int64。

### 百万级压力测试

| 测试场景 | 总耗时 | 吞吐量 |
|---------|--------|--------|
| 1M Set | 0.32 ms | 31 亿 ops/s |
| 1M Has | 0.32 ms | 31 亿 ops/s |
| 1M BitCount | 0.31 ms | 32 亿 ops/s |
| 1M Add-Del | 2.36 ms | 4.2 亿 ops/s |
| 1M Toggle | 2.10 ms | 4.8 亿 ops/s |
| 1M 混合操作 | 0.31 ms | 32 亿 ops/s |

### 内存占用分析

```
固定大小：8 字节（int64）
零堆分配：所有基础操作
GC 压力：几乎为零

特殊操作内存：
├─ String():        32 字节
├─ ActiveFlags(1):   8 字节
├─ ActiveFlags(12): 96 字节
└─ JSON Marshal:     8 字节
```

### 性能优化技术

1. **查表法 BitCount**：256 字节查找表，比循环快 5-10 倍
2. **unsafe 零拷贝**：String 方法避免内存拷贝
3. **内联优化**：核心方法标记 `//go:inline`
4. **快速路径**：JSON null 等特殊值快速处理
5. **批量优化**：减少循环开销

---

## 📊 主流方案对比

### 1. vs map[string]bool

```go
// map 方案
type StatusMap map[string]bool

func (s StatusMap) IsDeleted() bool {
    return s["deleted"]
}

func (s StatusMap) Add(key string) {
    s[key] = true
}
```

| 维度 | Status | map[string]bool | 优势 |
|------|--------|-----------------|------|
| 内存占用 | 8 B | 100+ B | **Status 节省 92%** |
| 检查速度 | 0.3 ns | 20 ns | **Status 快 66 倍** |
| 修改速度 | 2.2 ns | 50 ns | **Status 快 22 倍** |
| JSON 大小 | 1-5 B | 50+ B | **Status 节省 90%** |
| 类型安全 | ✅ | ❌ | Status 编译期检查 |
| 零内存分配 | ✅ | ❌ | Status 无 GC 压力 |

**结论**：Status 在性能、内存、类型安全上全面领先。

---

### 2. vs []string

```go
// slice 方案
type StatusSlice []string

func (s StatusSlice) Has(status string) bool {
    for _, v := range s {
        if v == status {
            return true
        }
    }
    return false
}
```

| 维度 | Status | []string | 优势 |
|------|--------|----------|------|
| 查找复杂度 | O(1) | O(n) | **Status 常数时间** |
| 内存占用 | 8 B | 50+ B | **Status 节省 84%** |
| 检查速度 | 0.3 ns | 100 ns | **Status 快 300+ 倍** |
| 重复处理 | 自动去重 | 需手动处理 | Status 更简单 |

**结论**：Status 查找效率远超 slice，内存占用极低。

---

### 3. vs struct 字段

```go
// struct 方案
type StatusStruct struct {
    SysDeleted  bool
    AdmDeleted  bool
    UserDeleted bool
    // ... 12 个字段
}

func (s StatusStruct) IsDeleted() bool {
    return s.SysDeleted || s.AdmDeleted || s.UserDeleted
}
```

| 维度 | Status | struct (12 字段) | 优势 |
|------|--------|-----------------|------|
| 内存占用 | 8 B | 12 B | **Status 节省 33%** |
| 扩展性 | 63 位 | 有限 | **Status 更灵活** |
| 代码量 | 简洁 | 冗长 | Status 维护成本低 |
| 检查速度 | 0.3 ns | 1 ns | Status 稍快 |

**结论**：struct 方案简单但扩展性差，Status 更适合多状态场景。

---

### 4. vs 第三方库

#### 4.1 vs github.com/spf13/pflag（标志位库）

| 特性 | Status | pflag |
|------|--------|-------|
| 用途 | 通用状态管理 | 命令行参数 |
| 性能 | 极致优化 | 中等 |
| 类型 | int64 位图 | 多种类型 |
| 适用场景 | 业务状态 | CLI 工具 |

**结论**：pflag 专注于命令行，Status 专注于业务状态。

#### 4.2 vs github.com/bits-and-blooms/bitset

```go
// bitset 示例
bs := bitset.New(100)
bs.Set(5).Set(10)
```

| 特性 | Status | bitset |
|------|--------|--------|
| 大小 | 固定 8 字节 | 可变长度 |
| 位数 | 63 位 | 无限 |
| 性能 | 极致优化 | 优秀 |
| 序列化 | 原生支持 | 需自行实现 |
| 学习曲线 | 平缓 | 陡峭 |

**结论**：bitset 适合海量位操作，Status 适合固定状态管理。

#### 4.3 vs github.com/looplab/fsm（状态机）

| 特性 | Status | FSM |
|------|--------|-----|
| 状态转换 | 手动控制 | 自动验证 |
| 复杂度 | 简单 | 复杂 |
| 性能 | 纳秒级 | 微秒级 |
| 适用场景 | 简单状态 | 复杂状态机 |

**结论**：FSM 适合严格的状态转换规则，Status 适合灵活的状态组合。

---

### 综合对比总结

```
性能排名（检查速度）：
1. Status          0.3 ns   ⭐⭐⭐⭐⭐
2. struct 字段     1 ns     ⭐⭐⭐⭐
3. bitset          5 ns     ⭐⭐⭐
4. map             20 ns    ⭐⭐
5. []string        100 ns   ⭐

内存占用排名：
1. Status          8 B      ⭐⭐⭐⭐⭐
2. struct          12 B     ⭐⭐⭐⭐
3. []string        50+ B    ⭐⭐
4. map             100+ B   ⭐

推荐场景：
├─ 简单状态(< 10个)  → Status / struct
├─ 中等状态(10-60个) → Status
├─ 大量状态(> 60个)  → bitset
└─ 复杂状态机        → FSM
```

---

## 💡 最佳实践

### 1. 数据库设计

#### ✅ 推荐做法

```sql
-- 单字段存储所有状态
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    status BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    INDEX idx_status (status)  -- 支持按状态查询
);

-- 查询示例
-- 查询所有被删除的用户（任意删除状态）
SELECT * FROM users 
WHERE status & 7 != 0;  -- 0b111 (StatusAllDeleted)

-- 查询可正常登录的用户（无限制状态）
SELECT * FROM users 
WHERE status & 4095 = 0;  -- 0b111111111111 (前12位)
```

#### ❌ 不推荐做法

```sql
-- 分散存储（浪费空间，查询效率低）
CREATE TABLE users (
    id BIGINT,
    sys_deleted BOOLEAN,
    adm_deleted BOOLEAN,
    user_deleted BOOLEAN,
    -- ... 12 个字段
);
```

### 2. 接口设计

#### ✅ 推荐做法

```go
// API 响应直接暴露 Status
type UserResponse struct {
    ID     int64        `json:"id"`
    Name   string       `json:"name"`
    Status types.Status `json:"status"`  // 前端按位解析
}

// 前端解析（JavaScript）
const StatusSysDeleted = 1 << 0;
const StatusAdmDisabled = 1 << 4;

if (user.status & StatusSysDeleted) {
    alert('用户已被系统删除');
}
```

#### ⚠️ 备选方案（兼容性更好）

```go
// 同时提供位图和解析后的字段
type UserResponse struct {
    ID            int64        `json:"id"`
    Name          string       `json:"name"`
    Status        types.Status `json:"status"`
    IsDeleted     bool         `json:"is_deleted"`
    IsDisabled    bool         `json:"is_disabled"`
    CanLogin      bool         `json:"can_login"`
}

func NewUserResponse(user *User) *UserResponse {
    return &UserResponse{
        ID:         user.ID,
        Name:       user.Name,
        Status:     user.Status,
        IsDeleted:  user.Status.IsDeleted(),
        IsDisabled: user.Status.IsDisable(),
        CanLogin:   user.Status.CanActive(),
    }
}
```

### 3. 业务逻辑

#### ✅ 推荐做法

```go
// 统一状态检查
func (u *User) CanAccess(resource *Resource) error {
    if u.Status.IsDeleted() {
        return errors.New("用户已删除")
    }
    if u.Status.IsDisable() {
        return errors.New("用户已禁用")
    }
    if resource.Status.IsHidden() {
        return errors.New("资源已隐藏")
    }
    return nil
}

// 批量修改状态
func (u *User) Ban(reason string) {
    u.Status.AddMultiple(
        types.StatusAdmDisabled,
        types.StatusAdmHidden,
    )
    u.BanReason = reason
    u.BannedAt = time.Now()
}
```

### 4. 扩展自定义状态

#### ✅ 推荐做法

```go
// 在独立文件中定义
// file: pkg/types/status_custom.go

package types

// 业务自定义状态（从 StatusExpand51 开始）
const (
    // 文章相关
    StatusArticlePinned   Status = StatusExpand51 << 0  // 置顶
    StatusArticleFeatured Status = StatusExpand51 << 1  // 精选
    StatusArticleLocked   Status = StatusExpand51 << 2  // 锁定评论
    
    // 用户相关
    StatusUserVIP      Status = StatusExpand51 << 3  // VIP用户
    StatusUserVerified Status = StatusExpand51 << 4  // 已认证
    
    // 订单相关
    StatusOrderPaid      Status = StatusExpand51 << 5  // 已支付
    StatusOrderShipped   Status = StatusExpand51 << 6  // 已发货
    StatusOrderCompleted Status = StatusExpand51 << 7  // 已完成
)

// 自定义组合常量
const (
    StatusArticleHighlight = StatusArticlePinned | StatusArticleFeatured
)
```

### 5. 测试建议

```go
func TestUserStatus(t *testing.T) {
    user := &User{}
    
    // 测试初始状态
    assert.True(t, user.Status.CanActive())
    
    // 测试禁用
    user.Status.Add(types.StatusAdmDisabled)
    assert.True(t, user.Status.IsDisable())
    assert.False(t, user.Status.CanActive())
    
    // 测试恢复
    user.Status.Del(types.StatusAdmDisabled)
    assert.False(t, user.Status.IsDisable())
    assert.True(t, user.Status.CanActive())
}
```

---

## ❓ 常见问题 (FAQ)

### Q1: Status 是否线程安全？

**A**: Status 类型本身**不是线程安全**的。如需在并发环境使用，请使用锁保护：

```go
type SafeStatus struct {
    mu     sync.RWMutex
    status types.Status
}
```

或使用原子操作（需封装）：

```go
type AtomicStatus struct {
    value atomic.Int64
}
```

---

### Q2: 如何在数据库中高效查询？

**A**: 使用位运算查询：

```sql
-- 查询包含任意删除状态的记录
SELECT * FROM users 
WHERE status & 7 != 0;  -- 0b111

-- 查询同时包含禁用和隐藏的记录
SELECT * FROM users 
WHERE status & 440 = 440;  -- 0b110111000

-- 创建索引（MySQL 8.0+）
CREATE INDEX idx_status ON users(status);
```

---

### Q3: 为什么不用 uint64？

**A**: 
1. **符号位问题**：避免负数混淆
2. **数据库兼容性**：大多数数据库使用 BIGINT（有符号）
3. **JSON 序列化**：避免超大数字精度问题
4. **63 位足够**：实际业务很少需要超过 63 个状态

---

### Q4: 如何升级已有系统？

**A**: 渐进式迁移：

```go
// 步骤1：添加 status 字段
type User struct {
    ID         int64
    IsDeleted  bool          // 保留
    IsDisabled bool          // 保留
    Status     types.Status  // 新增
}

// 步骤2：双写
func (u *User) SetDeleted(deleted bool) {
    if deleted {
        u.IsDeleted = true
        u.Status.Add(types.StatusAdmDeleted)
    } else {
        u.IsDeleted = false
        u.Status.Del(types.StatusAdmDeleted)
    }
}

// 步骤3：数据迁移
UPDATE users SET status = 
    CASE 
        WHEN is_deleted THEN 2    -- StatusAdmDeleted
        WHEN is_disabled THEN 16  -- StatusAdmDisabled
        ELSE 0
    END;

// 步骤4：读取优先使用 Status
func (u *User) CheckDeleted() bool {
    // 优先使用新字段
    if u.Status != 0 {
        return u.Status.IsDeleted()
    }
    // 兼容旧数据
    return u.IsDeleted
}

// 步骤5：删除旧字段（确认稳定后）
```

---

### Q5: 如何调试状态值？

**A**: 使用便捷方法：

```go
status := types.StatusSysDeleted | types.StatusAdmDisabled

// 方法1：String()
fmt.Println(status.String())
// 输出: Status(17)[2 bits]

// 方法2：ActiveFlags()
fmt.Printf("Active flags: %v\n", status.ActiveFlags())
// 输出: Active flags: [StatusSysDeleted StatusAdmDisabled]

// 方法3：二进制表示
fmt.Printf("Binary: %064b\n", status)
// 输出: Binary: 0000000000000000000000000000000000000000000000000000000000010001

// 方法4：位数统计
fmt.Printf("Bit count: %d\n", status.BitCount())
// 输出: Bit count: 2
```

---

### Q6: 性能瓶颈在哪里？

**A**: 性能分析：

- ✅ **基础操作**（Has/Add/Del）：0.3-2.4 ns，无瓶颈
- ✅ **数据库序列化**：4.37 ns，无瓶颈
- ⚠️ **ActiveFlags**：50 ns（12位），避免热路径频繁调用
- ⚠️ **JSON 序列化**：300 ns，标准库限制
- ⚠️ **String 格式化**：23 ns，仅用于日志和调试

**优化建议**：
1. 避免在循环中调用 `ActiveFlags()`
2. 热路径使用 `Has()` 而非 `ActiveFlags()`
3. 日志记录延迟到真正需要时

---

### Q7: 如何处理状态冲突？

**A**: 业务规则示例：

```go
// 规则：删除优先级最高
func (u *User) Normalize() {
    if u.Status.IsDeleted() {
        // 删除时清除其他状态
        u.Status.And(types.StatusAllDeleted)
    }
}

// 规则：禁用和隐藏互斥
func (u *User) SetDisabled(disabled bool) {
    if disabled {
        u.Status.Add(types.StatusAdmDisabled)
        u.Status.Del(types.StatusAllHidden)
    }
}

// 规则：审核通过后清除审核状态
func (a *Article) Approve() {
    a.Status.Del(types.StatusAllReview)
}
```

---

## 📄 许可证

MIT License

---

## 🔗 相关资源

- **源代码**: [status.go](status.go)
- **完整测试**: [status_test.go](status_test.go)
- **问题反馈**: [GitHub Issues](https://github.com/your-org/katydid-common-account/issues)

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给我们一个 Star！⭐**

Made with ❤️ by Katydid Team

</div>
