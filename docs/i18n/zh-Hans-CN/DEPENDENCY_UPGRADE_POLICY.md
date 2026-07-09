# 依赖升级策略

> 🌐 [English Version](../../DEPENDENCY_UPGRADE_POLICY.md)

> 本文档定义 MedMemo 的依赖版本锁定规则以及 Major 版本升级的手动工作流程。

---

## 目的

MedMemo 依赖于经过精心筛选的技术栈，其中包含多个高替换成本组件（例如 ONNX Runtime 绑定、Wails 桌面框架）。Dependabot 自动发起的 Major 版本升级曾多次引入破坏性变更，导致 CI 失败、跨平台构建不稳定，或违反 `AGENTS.md` 中记录的合规相关技术栈锁定。

本策略确立以下内容：
- 哪些依赖被**锁定**及其原因。
- **Patch/Minor** 更新如何自动处理。
- 任何 **Major 版本升级** 所需的**手动工作流程**。

---

## 锁定依赖清单

### 前端（npm）

| 依赖项 | 当前版本 | 锁定原因 |
|--------|---------|---------|
| `react` | `^18.2.0` | AGENTS.md 锁定 React 为 18+；Major 升级会影响 Wails 前端绑定和 UI 测试套件。 |
| `react-dom` | `^18.2.0` | 必须与 `react` 保持同步。 |
| `@types/react` | `^18.2.55` | 必须与 `react` 保持同步。 |
| `@types/react-dom` | `^18.2.19` | 必须与 `react-dom` 保持同步。 |
| `typescript` | `5.9.3` | AGENTS.md 锁定 TypeScript 为 5.x+；严格模式和类型行为不可意外变动。 |
| `@typescript-eslint/eslint-plugin` | `^8.62.1` | 与 TypeScript 版本强耦合；Major 升级可能引入不兼容的 Lint 规则。 |
| `@typescript-eslint/parser` | `^8.63.0` | 必须与 ESLint plugin 和 TypeScript 版本保持一致。 |
| `vite` | `6.4.3` | 构建工具链；Major 升级经常破坏插件 API 和跨平台 CGO 构建。 |
| `@vitejs/plugin-react` | `4.7.0` | 与 Vite Major 版本强耦合。 |
| `tailwindcss` | `^3.4.1` | AGENTS.md 锁定 Tailwind 为 3.x+；Major 升级可能破坏工具类生成。 |
| `tailwindcss-animate` | `^1.0.7` | 与 `tailwindcss` Major 版本强耦合。 |
| `eslint` | `^8.56.0` | Lint 工具链稳定性；Major 升级通常需要配置迁移。 |
| `eslint-plugin-react-refresh` | `^0.4.5` | 与 React 和 ESLint Major 版本强耦合。 |
| `zustand` | `^5.0.13` | 状态管理近期刚升级到 v5；Major 升级需要验证 Store API。 |
| `react-router-dom` | `^7.18.1` | 路由 API 变更可能破坏深层链接和 Wails 导航集成。 |
| `react-markdown` | `^10.1.0` | Markdown 渲染引擎；Major 升级可能破坏 `remark-gfm` 插件兼容性。 |
| `vitest` | `^4.1.10` | 测试框架；Major 升级可能破坏测试配置、覆盖率报告器和 UI 模式。 |
| `@vitest/coverage-v8` | `^4.1.10` | 与 `vitest` Major 版本强耦合。 |
| `@vitest/ui` | `^4.1.9` | 与 `vitest` Major 版本强耦合。 |

### 后端（Go Modules）

| 依赖项 | 当前版本 | 锁定原因 |
|--------|---------|---------|
| `github.com/knights-analytics/hugot` | `v0.7.4` | **高替换成本**：ONNX Runtime Go 绑定。Major 升级可能改变 Session API 或破坏 int8 量化模型加载。 |
| `github.com/daulet/tokenizers` | `v1.27.0` | ONNX Tokenizer 绑定；与 `hugot` 和 ORT 版本强耦合。 |
| `github.com/yalue/onnxruntime_go` | `v1.30.1` | 直接 ONNX Runtime 绑定；必须与 `hugot` 的 ORT 版本保持一致。 |
| `github.com/wailsapp/wails/v2` | `v2.12.0` | **高替换成本**：桌面应用框架。Major 升级会影响 CGO 绑定、前端桥接和跨平台打包。 |

---

## 自动更新策略

| 更新类型 | 策略 | 工具 |
|---------|------|------|
| **Patch**（如 `1.0.0` → `1.0.1`） | ✅ CI 通过后自动合并 | Dependabot |
| **Minor**（如 `1.0.0` → `1.1.0`） | ✅ CI 通过后审查并合并 | Dependabot |
| **Major**（如 `1.0.0` → `2.0.0`） | ❌ **已阻断**，由 `dependabot.yml` 的 `ignore` 规则拦截 | 仅允许手动 Feature 分支 |

> **注意**：Go 的语义化版本与 npm 不同。对于 `v0.x` 的 Go 模块，`v0.y` 的升级被 Dependabot 视为 *Minor*，而 `v0.x` → `v1.0.0` 才是 *Major*。上述 `ignore` 规则仅针对 `semver-major`；`hugot` 的 Patch 升级（如 `v0.7.4` → `v0.7.5`）仍然允许，但涉及 ORT 版本变更时应进行手动验证。

---

## 手动 Major 版本升级工作流程

任何跨越 Major 版本边界的升级**必须**遵循以下工作流程。

### 1. 创建 Feature 分支

```bash
git checkout develop
git pull origin develop
git checkout -b feature/upgrade-<dependency>-<new-major>
```

> **不要**将 `hotfix/*` 用于计划内的 Major 升级。`hotfix` 仅用于紧急生产修复。

### 2. 应用升级

- 在 `package.json`（npm）或 `go.mod`（Go）中更新版本。
- 运行 `npm install` 或 `go mod tidy` 更新锁定文件。
- 处理任何即时编译或类型错误。

### 3. 本地运行完整 CI

```bash
# Go 后端
make lint
make test

# 前端
cd web && npm run lint && npm run test
```

### 4. ONNX 专项验证（`hugot` / `tokenizers` / `onnxruntime_go` 升级时必需）

如果升级触及 ONNX 技术栈：

1. 验证本地 NER 推理仍能加载 int8 量化模型。
2. 验证嵌入生成仍产生一致的向量维度。
3. 运行完整的端到端对话流程，确保脱敏流水线正常工作。
4. 如果 ORT 原生库发生变更，检查跨平台构建（`make build-windows`、`make build-macos`、`make build-linux`）。

### 5. 跨平台构建检查

前端工具链（React、Vite）的 Major 升级必须通过：

```bash
# Wails 构建冒烟测试
wails build -platform windows/amd64
wails build -platform darwin/amd64
wails build -platform linux/amd64
```

### 6. 代码审查与合并

- 针对 `develop` 创建 PR。
- PR 描述必须包含：
  - 上游发布说明中的破坏性变更摘要。
  - 已执行的本地验证步骤（如涉及 ONNX 需特别说明）。
  - 任何配置或 API 变更的迁移说明。
- 合并前至少需要一次审批。
- 合并到 `develop` 后，变更将随下一次发布列车进入 `main`。

---

## 升级后检查清单

- [ ] 如果版本锁定发生变化，更新 `AGENTS.md` 技术栈表格。
- [ ] 如果有新依赖加入锁定清单，更新 `docs/DEPENDENCY_UPGRADE_POLICY.md`。
- [ ] 中文翻译已在 `docs/i18n/zh-Hans-CN/DEPENDENCY_UPGRADE_POLICY.md` 中同步。
- [ ] 如果需要新的 `ignore` 规则，更新 `.github/dependabot.yml`。
- [ ] 所有 CI 检查（Build、Lint、Unit Test、Integration Test、E2E Test、Cross-Platform Build）通过。
- [ ] ONNX 推理已验证（如适用）。
- [ ] Wails 跨平台构建已验证（如前端工具链发生变更）。

---

## 相关文档

- [`AGENTS.md`](../../../AGENTS.md) — 完整的开发标准和技术栈版本锁定。
- [`.github/dependabot.yml`](../../../.github/dependabot.yml) — 包含 `ignore` 规则的 Dependabot 配置。
- [`docs/DEVELOPMENT.md`](../../DEVELOPMENT.md) — Clean Architecture 和编码规范。

---

*最后更新：2026-07-09*
