# 贡献指南

感谢你对 MedMemo 的兴趣！无论你是 Go 开发者、前端工程师、医学专业人士还是编程新手，我们都欢迎你的参与。

## 开发环境搭建

### 前置依赖

| 工具 | 最低版本 | 安装方式 |
|------|---------|---------|
| Go | 1.22+ | [官方下载](https://go.dev/dl/) 或 `brew install go` |
| Node.js | 18.x+ | [官方下载](https://nodejs.org/) 或 `brew install node` |
| Wails CLI | v2.9+ | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| golangci-lint | latest | `make install-tools` |
| Wire | v0.6+ | `make install-tools` |

### 快速开始

```bash
git clone https://github.com/medmemo/medmemo.git
cd medmemo
make install-tools
go mod download
cd web && npm install && cd ..
make dev
```

### 常用命令

```bash
make dev              # 开发模式（热重载）
make build            # 生产构建
make test             # 运行单元测试
make test-integration # 运行集成测试
make coverage         # 生成测试覆盖率报告
make lint             # 代码检查
make wire             # 重新生成 Wire 依赖注入代码
make fmt              # 格式化代码
```

## Git Flow 分支策略

```
main (生产分支，永远可发布)
  ^
develop (集成分支)
  ^
feature/M<模块号>-<简述> (功能分支)
```

| 分支类型 | 命名规范 | 合并策略 |
|----------|----------|---------|
| `main` | `main` | 仅接受 release/hotfix 合并 |
| `develop` | `develop` | 仅接受 feature 合并，CI 全绿方可合并 |
| `feature/*` | `feature/M<模块号>-<简述>` | Squash & Merge |
| `release/*` | `release/v<版本号>` | Squash & Merge |
| `hotfix/*` | `hotfix/<问题简述>` | Squash & Merge |

## 提交规范（Conventional Commits）

```
<type>(<scope>): <subject>
```

| Type | 用途 | Scope 示例 |
|------|------|-----------|
| `feat` | 新功能 | `feat(M03): add HNSW vector index` |
| `fix` | Bug 修复 | `fix(PER-03): reduce ONNX inference latency` |
| `perf` | 性能优化 | `perf(M06): optimize deidentify pipeline` |
| `refactor` | 重构（无功能变更） | `refactor(domain): extract SensitivityLevel` |
| `test` | 测试相关 | `test(M01): add E2E test for conversation` |
| `docs` | 文档更新 | `docs(adr): add ADR-006 for HNSW` |
| `chore` | 构建/工具 | `chore(ci): add Windows build matrix` |
| `security` | 安全修复 | `security(M07): bump ONNX Runtime` |

**Scope 对照表**：`M01`-`M07` 对应 7 大功能模块，`ci`/`build`/`deps` 对应工程化。

## 代码审查流程

1. 所有 PR 必须通过 CI 流水线（Lint / Unit Test / Integration Test / Build）
2. 至少 1 名维护者 Code Review Approve
3. 测试覆盖率不可下降（Codecov 基线检查）
4. 架构依赖检查（depguard）0 违规

## 开发规范

### Go 编码

- `gofmt` 格式化所有代码
- `golangci-lint` 0 错误通过
- 错误处理必须使用 `fmt.Errorf("...: %w", err)` 包装
- **domain 层零外部依赖**：仅允许标准库 + `pkg/models/`
- 核心领域逻辑使用**中文注释**，聚焦 Why 而非 What

### 前端编码

- TypeScript 严格模式不可关闭
- 组件使用 PascalCase，Hooks 使用 `use` 前缀
- 优先使用 Tailwind CSS 工具类
- 颜色遵循 UI 规范：用户消息 `#4F8CFF`、AI 消息白色/`#2A2A2A`、系统提示 `#F0F0F5`

## 问题报告

通过 [GitHub Issues](https://github.com/medmemo/medmemo/issues) 提交，请包含：
- 操作系统版本
- Go / Node.js 版本
- 复现步骤
- 日志输出
- 预期 vs 实际行为

## 功能请求

先搜索现有 Issues 避免重复，使用 Feature Request 模板，描述应用场景和预期行为。

## 行为准则

参与本项目即表示你同意以专业、尊重和包容的态度对待每一位贡献者。任何骚扰、歧视或不友善行为都将不被容忍。
