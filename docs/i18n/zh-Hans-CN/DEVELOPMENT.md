# 开发指南

> 🌐 [English Version](../../DEVELOPMENT.md)

> 本文档面向 MedMemo 开发者，说明如何在本项目中编写符合规范的代码。

---

## Clean Architecture 四层依赖规则

MedMemo 严格遵循 Clean Architecture 四层模型，依赖方向始终向内指向领域核心。

```
┌──────────────────────────────────────┐
│    Infrastructure Layer              │  ← Frameworks & Drivers
│  (ONNX/SQLCipher/SQLite/sqlite-vec/Wails) │
├──────────────────────────────────────┤
│    Adapters Layer                    │  ← Interface Adapters
│  (AI适配器/仓库实现)                 │
├──────────────────────────────────────┤
│    Application Layer                 │  ← Use Cases
│  (用例编排/端口定义/流水线)           │
├──────────────────────────────────────┤
│    Domain Layer                      │  ← Entities
│  (实体/领域服务/策略接口)             │
└──────────────────────────────────────┘
```

### 包导入白名单

| 源目录                         | 允许导入                                                              | 禁止导入                                                                   |
|-----------------------------|-------------------------------------------------------------------|------------------------------------------------------------------------|
| `internal/domain/*`         | 标准库 + `pkg/models/`                                               | 任何 `internal/` 子包                                                      |
| `internal/application/*`    | `internal/domain/*` + `pkg/models/` + 标准库                         | `internal/adapters/*` + `internal/infrastructure/*`                    |
| `internal/adapters/*`       | `internal/domain/*` + `internal/infrastructure/*` + `pkg/models/` | `internal/application/*`                                               |
| `internal/infrastructure/*` | 标准库 + 第三方框架库                                                      | 任何 `internal/domain/` / `internal/application/` / `internal/adapters/` |

**核心铁律**：`internal/domain/` 零外部依赖。CI 的 depguard 会阻断任何违规导入。

---

## Wire 依赖注入使用指南

MedMemo 使用 Google Wire 进行**编译期**依赖注入，禁止运行时反射注入。

### 添加新依赖的步骤

1. 在对应包内编写返回**具体类型**的 Provider 函数
2. 在包的 `ProviderSet` 变量中注册（如 `ApplicationSet = wire.NewSet(...)`）
3. 修改仓库根 `wire.go` 的 `InitializeApp` 函数，加入新的 ProviderSet
4. 运行 `make wire` 重新生成 `wire_gen.go`

**绝对禁止**手动修改 `wire_gen.go`。

### Provider 函数签名规范

```go
// 正确：返回具体类型
func NewChatOrchestrator(llm port.LLMClient) *ChatOrchestrator

// 错误：Wire 通过返回值匹配需求类型，不应返回接口
func NewChatOrchestrator(llm port.LLMClient) port.UseCase
```

---

## 错误处理规范

禁止裸返回原始错误，必须包装上下文：

```go
// 禁止：
return err

// 必须：
return fmt.Errorf("failed to retrieve family member %s: %w", id, err)

// 适配器层外部错误映射：
if err != nil {
    return nil, fmt.Errorf("sqlite query failed: %w", domain.ErrRecordNotFound)
}
```

---

## 并发安全规范

### ONNX 推理

- 固定 **2 个推理 Worker**，每个持有独立 ONNX Session
- 任务通过有缓冲 channel（容量 16）派发
- **不可共享 Session 并发调用**——`Run()` 非线程安全

### SQLCipher/SQLite 写入

- 可能发生冲突的数据库写入在应用层串行化
- 加密 SQLite 与 sqlite-vec 操作使用保守的连接池设置

### HTTP 请求

- semaphore 限制最大 4 个并发云端请求

---

## 前端开发规范

### TypeScript 严格模式

`tsconfig.json` 中 `"strict": true` 不可关闭。

### 组件规范

- 命名：PascalCase（如 `ComplianceBar.tsx`）
- Props：必须编写完整的 TypeScript 接口定义，禁止 `any` 逃逸
- Hooks：camelCase 前缀 `use`（如 `useConversation.ts`）

### UI 颜色规范

| 元素      | 亮色模式                              | 暗色模式      |
|---------|-----------------------------------|-----------|
| 用户消息背景  | `#4F8CFF` → `#3B7AF7` 渐变          | 同左        |
| 用户消息文字  | 白色                                | 白色        |
| AI 消息背景 | `#FFFFFF`                         | `#2A2A2A` |
| AI 消息文字 | `#333333`                         | `#E5E5E5` |
| 系统提示背景  | `#F0F0F5` / `#FFF3E0` / `#E3F2FD` | 同左        |

### CSS

优先使用 Tailwind CSS 工具类，自定义样式通过 CSS 变量实现主题切换。

### Provider 模板文件

`web/src/data/provider-templates.json` 是 Provider 模板的**唯一真源**。构建时被打包，并由 `APIKeyPanel`、`OAuthDevicePanel`、`ProviderTemplateList` 在运行时引入。

新增或修改 provider 模板时，请只修改 `web/src/data/provider-templates.json`。提交 provider 模板变更前请先运行 `node scripts/validate-provider-templates.js`，该脚本现在校验的是打包源文件。

---

## 测试策略

### 测试金字塔

```
      /\
     /  \  E2E (5%)  — Wails 集成 / Playwright
    /____\
   /      \
  /        \ 集成测试 (25%) — go test + SQLite 内存模式
 /__________\
/            \
/______________\ 单元测试 (70%) — go test + testify + mockery
```

### 覆盖率门禁

- 单元测试行覆盖率 >= 70%
- `domain` 层覆盖率 100%
- 测试覆盖率不可下降（Codecov 基线检查）

### 关键测试场景

1. 合规引擎：全部四层风险等级用例，覆盖率 >= 80%
2. 紧急症状：A级/B级关键词 100% 触发测试
3. 脱敏流水线：PII 输入 → 占位符替换 → 云端响应回填
4. 会话生命周期：新建 → 发送 → 重命名 → 重启 → 数据完整
5. 模型切换：上下文继承、窗口截断、超时降级
6. 离线降级：网络不可用时的本地模板响应

---

## Context 使用规范

所有 I/O 操作必须接收 `context.Context`：

```go
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
defer cancel()
```

优雅关闭顺序遵循依赖倒置：先关闭前端桥接，再停止推理 Worker，最后关闭数据库连接。

---

## 构建标签与命令

| 标签 / 命令 | 用途 |
|-------------|------|
| `ORT` | 启用 ONNX Runtime CGO 绑定，用于本地 NER 与 embedding 推理。 |
| `webkit2_41` | 面向使用 webkit2gtk-4.1 的 Linux 发行版构建。 |
| `make test` | 运行带 race detector 与覆盖率的单元测试。 |
| `make test-integration` | 使用 `integration,ORT` 标签运行集成测试。 |
| `make lint` | 运行 Go 与前端 lint 检查。 |

---

*最后更新：2026-07-09*
