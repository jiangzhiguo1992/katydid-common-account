# v5_refactored 文档索引

欢迎使用 v5_refactored 企业级验证器框架！本目录包含完整的文档和代码实现。

---

## 📚 文档列表

### 中文文档

1. **[架构重构总结.md](架构重构总结.md)** ⭐ 推荐首先阅读
   - SOLID 原则详细讲解
   - 设计模式应用说明
   - 性能优化措施
   - 迁移指南

### 英文文档

2. **[ARCHITECTURE.md](ARCHITECTURE.md)** - 架构设计文档
   - 完整的架构设计
   - 分层架构说明
   - 组件交互图
   - 设计模式详解

3. **[README.md](README.md)** - 使用文档
   - 快速开始
   - 使用示例
   - API 文档
   - 配置建议

4. **[REFACTOR_SUMMARY.md](REFACTOR_SUMMARY.md)** - 重构总结
   - 核心改进点
   - 代码对比
   - 质量提升
   - 迁移建议

5. **[COMPARISON.md](COMPARISON.md)** - v5 vs v5_refactored 对比
   - 详细对比表
   - 使用场景建议
   - 迁移成本分析

---

## 💻 代码文件

### 核心接口

- **[interface.go](interface.go)** (~280 行)
  - 所有接口定义
  - 验证器接口
  - 策略接口
  - 组件接口

### 基础类型

- **[types.go](types.go)** (~150 行)
  - Scene 场景定义
  - FieldError 字段错误
  - ValidationError 验证错误
  - TypeInfo 类型信息

### 验证上下文

- **[context.go](context.go)** (~150 行)
  - ValidationContext 定义
  - 对象池实现
  - 元数据管理

### 核心组件

- **[error_collector.go](error_collector.go)** (~200 行)
  - DefaultErrorCollector
  - ConcurrentErrorCollector
  - ErrorCollectorFactory

- **[event_bus.go](event_bus.go)** (~250 行)
  - SyncEventBus 同步事件总线
  - AsyncEventBus 异步事件总线
  - NoOpEventBus 空事件总线

- **[hook_manager.go](hook_manager.go)** (~100 行)
  - DefaultHookManager
  - NoOpHookManager

- **[pipeline.go](pipeline.go)** (~200 行)
  - DefaultPipelineExecutor
  - ConcurrentPipelineExecutor

- **[registry.go](registry.go)** (~200 行)
  - DefaultTypeRegistry
  - MultiLevelTypeRegistry

### 验证引擎

- **[engine.go](engine.go)** (~180 行)
  - ValidatorEngine 核心实现
  - 默认实例
  - 便捷函数

### 辅助组件

- **[formatter.go](formatter.go)** (~100 行)
  - DefaultErrorFormatter
  - ChineseErrorFormatter
  - JSONErrorFormatter

- **[builder.go](builder.go)** (~120 行)
  - ValidatorBuilder 建造者
  - ValidatorFactory 工厂

### 示例代码

- **[example_test.go](example_test.go)** (~150 行)
  - 完整的使用示例
  - 各种场景演示
  - 可直接运行

---

## 🚀 快速开始

### 1. 阅读文档

**推荐阅读顺序**：

1. [架构重构总结.md](架构重构总结.md) - 了解设计思想（中文）
2. [README.md](README.md) - 学习如何使用
3. [COMPARISON.md](COMPARISON.md) - 对比 v5 和 v5_refactored
4. [example_test.go](example_test.go) - 查看示例代码

### 2. 运行示例

```bash
cd pkg/validator/v5_refactored
go run example_test.go
```

### 3. 开始使用

```go
import v5 "your-project/pkg/validator/v5_refactored"

// 基础使用
err := v5.Validate(user, v5.SceneCreate)

// 高级配置
validator := v5.NewBuilder().
    WithEventBus(v5.NewAsyncEventBus(4, 100)).
    WithRegistry(v5.NewMultiLevelTypeRegistry(100)).
    Build()
```

---

## 📊 文件统计

| 类型 | 数量 | 总行数 |
|------|-----|-------|
| 文档文件 | 5 | ~3000 行 |
| 代码文件 | 11 | ~1930 行 |
| 示例文件 | 1 | ~150 行 |
| **总计** | **17** | **~5080 行** |

---

## 🎯 核心特性

### ✅ SOLID 原则

- **单一职责**：5 个独立组件
- **开放封闭**：通过接口扩展
- **里氏替换**：所有实现可互换
- **接口隔离**：细粒度接口
- **依赖倒置**：完全依赖接口

### ✅ 设计模式

- 策略模式
- 观察者模式
- 责任链模式
- 工厂模式
- 建造者模式
- 对象池模式

### ✅ 架构质量

- 高内聚低耦合
- 可测试可维护
- 可扩展可复用
- 性能优化
- 生产就绪

---

## 📖 深度阅读

### 架构设计

- [ARCHITECTURE.md](ARCHITECTURE.md) - 完整架构设计
  - 分层架构
  - 组件职责
  - 交互流程
  - 扩展点设计

### 技术细节

- [interface.go](interface.go) - 接口定义
- [engine.go](engine.go) - 核心实现
- [pipeline.go](pipeline.go) - 策略执行
- [event_bus.go](event_bus.go) - 事件系统

---

## 🔧 配置参考

### 开发环境

```go
validator := v5.NewBuilder().
    WithEventBus(v5.NewSyncEventBus()).
    WithErrorFormatter(v5.NewChineseErrorFormatter()).
    Build()
```

### 生产环境

```go
validator := v5.NewBuilder().
    WithPipeline(v5.NewConcurrentPipelineExecutor(8)).
    WithEventBus(v5.NewAsyncEventBus(4, 1000)).
    WithRegistry(v5.NewMultiLevelTypeRegistry(200)).
    WithMaxErrors(100).
    Build()
```

---

## 🤝 贡献指南

欢迎贡献！请：

1. Fork 项目
2. 创建功能分支
3. 提交代码
4. 发起 Pull Request

---

## 📄 许可证

MIT License

---

## 📮 联系方式

如有问题或建议，请：

- 提交 Issue
- 发起 Discussion
- 联系维护者

---

**最后更新**：2025-10-28  
**版本**：v5_refactored 1.0.0

