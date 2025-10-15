# int64 类型用于位运算的安全性说明

## 问题：int64 为负数时会影响位运算吗？

### 简短回答
**在我们的使用场景下，完全不会有问题。**

### 详细说明

#### 1. int64 的位结构
- int64 有 64 位（0-63）
- 第 63 位是**符号位**
- 0-62 位可以安全使用而不会变成负数

#### 2. 我们当前的使用情况
```go
const (
    StatusNone      Status = 0         // 第0位 = 0
    StatusEnabled   Status = 1 << iota // 第1位 = 2
    StatusVisible                      // 第2位 = 4
    StatusLocked                       // 第3位 = 8
    StatusDeleted                      // 第4位 = 16
    StatusActive                       // 第5位 = 32
    StatusVerified                     // 第6位 = 64
    StatusPublished                    // 第7位 = 128
    StatusArchived                     // 第8位 = 256
    StatusFeatured                     // 第9位 = 512
    StatusPinned                       // 第10位 = 1024
    StatusHidden                       // 第11位 = 2048
    StatusSuspended                    // 第12位 = 4096
    StatusPending                      // 第13位 = 8192
    StatusApproved                     // 第14位 = 16384
    StatusRejected                     // 第15位 = 32768
    StatusDraft                        // 第16位 = 65536
)
```

**最高使用位：第 16 位**  
**安全范围：0-62 位**  
**结论：我们还有 46 位的扩展空间！**

#### 3. 什么情况下会变成负数？

只有当你使用**第 63 位（符号位）**时才会变成负数：

```go
var safe Status = 1 << 62    // 正数，安全
var negative Status = 1 << 63 // 负数，这是符号位
```

#### 4. 负数对位运算的影响

即使误设置为负数，位运算**仍然正常工作**：

```go
var s Status = -1  // 所有64位都是1（二进制补码）

// 位运算完全正常
s.Has(StatusEnabled)   // true（包含该位）
s.Unset(StatusEnabled) // 正常取消该位
s.Set(StatusVisible)   // 正常设置该位
```

**为什么？** 因为位运算（&, |, ^, &^）对有符号和无符号整数的底层操作是一样的。

#### 5. 实际使用保证

在正常使用中，Status 值**永远不会变成负数**，因为：

1. ✅ 所有预定义常量都是正数（使用 0-16 位）
2. ✅ `Set()` 方法只会设置指定的位，不会触碰符号位
3. ✅ `Unset()` 方法只会清除指定的位
4. ✅ 组合状态也都是正数

#### 6. 数据库存储

```go
// 存入数据库时，转为 int64
func (s Status) Value() (driver.Value, error) {
    return int64(s), nil  // 始终是正数
}

// 从数据库读取
func (s *Status) Scan(value interface{}) error {
    // 数据库中存储的也是正数
    *s = Status(v)
}
```

所有主流数据库的 BIGINT 类型都支持 int64 的正数范围（0 到 2^63-1）。

#### 7. 对比 uint64

| 特性 | int64 | uint64 |
|------|-------|--------|
| 可用位数 | 0-62 位（63位）| 0-63 位（64位）|
| 数据库兼容性 | ✅ 所有数据库 | ❌ 大部分不支持 |
| 我们的需求 | 仅用了 17 位 | 仅用了 17 位 |
| 扩展空间 | 46 位可用 | 47 位可用 |
| 位运算 | ✅ 完全正常 | ✅ 完全正常 |

**结论：int64 完全够用，且兼容性更好！**

#### 8. 安全建议

✅ **推荐做法：**
- 继续使用 0-16 位定义状态
- 扩展时使用 17-62 位
- 永远不要使用第 63 位

❌ **不要这样做：**
```go
// 不要直接赋值负数
status := Status(-1)

// 不要使用符号位
const StatusBad Status = 1 << 63  // 这会导致负数
```

### 总结

✅ **使用 int64 作为 Status 类型是完全安全的**  
✅ **正常使用场景下不会出现负数**  
✅ **即使出现负数，位运算仍然正常工作**  
✅ **我们只用了 17 位，还有 46 位扩展空间**  
✅ **int64 在所有主流数据库中都有完美支持**

**放心使用吧！** 👍

