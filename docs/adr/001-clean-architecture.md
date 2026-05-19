# ADR-001: 采用 Clean Architecture 四层目录模型

- **状态**: Accepted
- **日期**: 2026-05
- **决策者**: 后端技术负责人

## 背景（Context）

MedMemo 是一款桌面端健康咨询工具，核心诉求包括：
1. **长期可维护性**：健康数据模型（记忆、家族关系、会话）会随版本快速迭代，需要清晰的业务边界。
2. **可测试性**：医疗合规逻辑（脱敏、拦截、用词校验）必须可独立单元测试，不依赖外部框架。
3. **可替换性**：本地 AI 推理引擎（ONNX Runtime）、数据库（DuckDB/SQLite）、LLM 供应商均存在替换或降级风险。

在项目启动前，团队评估了三种后端架构风格：

| 架构风格                               | 优点                                     | 缺点                | 适用性                 |
|------------------------------------|----------------------------------------|-------------------|---------------------|
| **MVC/三层架构**                       | 简单直观，上手快                               | 业务逻辑与框架耦合，难以单元测试  | ❌ 不满足可测试性           |
| **Hexagonal Architecture（端口-适配器）** | 明确的输入/输出端口，适配器可替换                      | 目录命名不统一，新成员理解成本高  | ⚠️ 概念抽象，对 Go 生态支持一般 |
| **Clean Architecture（四层模型）**       | 依赖方向严格向内，domain 层可纯 Go 测试，与 Go 包机制天然契合 | 目录层级较多，小功能改动可能跨多层 | ✅ 最契合 MedMemo 需求    |

## 决策（Decision）

采用 **Robert C. Martin 的 Clean Architecture 四层模型**，映射到 Go 包结构如下：

```
internal/
├── domain/         # Entities 层 — 核心业务实体与规则
├── application/    # Use Cases 层 — 用例编排与端口定义
├── adapters/       # Interface Adapters 层 — 外部系统适配实现
└── infrastructure/ # Frameworks & Drivers 层 — 技术框架封装
```

### 关键约束

1. **domain 层零外部依赖**：仅允许导入 Go 标准库和 `pkg/models/`。违反此规则将导致 CI 的 depguard 检查阻断合并。
2. **application 层不直接调用 adapter 实现**：通过接口（Port）解耦，adapter 层负责实现 application 层定义的接口。
3. **infrastructure 层不知道业务存在**：仅封装第三方框架（Wails、DuckDB、ONNX Runtime），不导入任何业务包。
4. **依赖注入通过 Wire 编译期完成**：禁止运行时反射注入，确保冷启动速度和编译期安全。

## 后果（Consequences）

### 积极影响

- **domain 层 100% 可单元测试**：纯 Go 标准库，无需 Mock 外部依赖。
- **技术栈替换成本可控**：更换数据库或 AI 推理引擎时，只需重写 adapter/infrastructure 层，domain/application 层不受影响。
- **新成员 onboarding 有章可循**：四层目录本身就是一种架构文档，配合各层 README 可快速定位代码。

### 消极影响

- **小功能改动可能涉及 4+ 个文件**：新增一个实体需要修改 domain（实体定义）、application（用例）、adapter（仓库实现）、infrastructure（数据库表），开发效率略低于 MVC。
- **过度设计风险**：对于简单的 CRUD 操作，四层拆分显得冗余。缓解措施：允许 application 层直接透传简单查询，不强制每个操作都经过领域服务。

## 替代方案记录

| 替代方案 | 否决理由 |
|---------|---------|
| 传统三层架构（Controller-Service-DAO） | 业务逻辑与框架耦合，难以在桌面端离线场景下对合规引擎进行纯单元测试 |
| 按功能模块划分（feature-based） | MedMemo 的模块间存在强交叉（记忆池 ↔ 家族图谱 ↔ 合规引擎），按功能划分会导致循环依赖 |

## 相关文档

- [docs/DEVELOPMENT.md](../DEVELOPMENT.md) — 开发规范与依赖规则详解
- [internal/domain/README.md](../../internal/domain/README.md)
- [internal/application/README.md](../../internal/application/README.md)
- [internal/adapters/README.md](../../internal/adapters/README.md)
- [internal/infrastructure/README.md](../../internal/infrastructure/README.md)
