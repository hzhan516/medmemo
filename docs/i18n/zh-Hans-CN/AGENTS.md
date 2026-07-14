# MedMemo 代理指南

> 本文档是面向在 MedMemo 上工作的 AI 编码代理的主要入职与参考文档。在编写或修改代码前请完整阅读，并在项目约定发生变化时保持其准确性。

> 🌐 [English Version](../../../AGENTS.md)

---

## AI 代理系统指令

每次代码变更前，请自我检查以下约束。不要将它们视为可选项。

1. **`internal/domain/` 零依赖规则** —— 领域层只能导入 Go 标准库和 `pkg/models/`。任何其他导入都是依赖规则违规。
2. **合规红线自检** —— 在编写 AI 提示词、聊天逻辑或 UI 文案前，对照下文的 [合规、隐私与安全红线](#合规隐私与安全红线) 进行自我检查。绝不输出确定性诊断、药物剂量或治疗方案。
3. **前端类型安全** —— 每个 React 组件都必须声明 TypeScript `interface Props`（或等价形式）。生产代码中禁止 `any` 逃逸。必须保持 TypeScript `strict` 模式启用。
4. **仅通过 Wire 进行依赖注入** —— 所有新依赖都必须通过仓库根目录的 `wire.go` 注入。运行 `wire .` 重新生成 `wire_gen.go`。**永远不要手动编辑 `wire_gen.go`。**
5. **错误包装** —— 禁止裸 `return err`。使用 `fmt.Errorf("...: %w", err)` 包装上下文。
6. **ONNX 并发安全** —— ONNX Runtime `Session.Run` 不是线程安全的。推理通过固定工作池（当前 2 个 worker）分发；每个 worker 通过各自的互斥锁串行化调用。不要在没有序列化的情况下跨 goroutine 共享 session。
7. **触发必需 Skill** —— 编写或更新文档时，调用 `codebase-documenter` Skill，然后调用 `submission-checker` Skill。在编写、修改、审查或重构代码注释时，调用 `code-comment` Skill。
8. **同步问题跟踪** —— 关闭 TODO/FIXME/HACK/BUG/XXX 问题时，在 `medmemo/开发日志/issues.md` 中更新对应行：将 **完成** 设为 `✅`，**状态** 设为 `closed`。
9. **不要实现冻结占位** —— `DuckDBConnector` 和 `FamilyRepoKuzu` 是 v2+ 规划存根且故意冻结。未经维护者明确批准，不要填充它们。
10. **主文档使用英文** —— 所有顶层文档（包括本文件）均使用英文编写。简体中文翻译位于 `docs/i18n/zh-Hans-CN/`。内容变更时请同时更新两者。
11. **版本单一来源** —— 产品版本从 `wails.json`（`info.productVersion`）读取。升级版本时，还需更新 `web/package.json`、刷新 `web/package-lock.json`，并在 `internal/domain/entity/changelog/zh-Hans.json` 中添加变更日志条目。

---

## 项目概览

MedMemo 是一款**开源桌面健康信息助手**。定位为医院分诊与健康咨询信息工具，**明确不是医疗器械**。它不提供诊断、处方或治疗建议。

当前产品版本：**1.1.10**（取自 `wails.json`）。

核心能力：

- 多模型聊天（Kimi、OpenAI、Qwen、SiliconFlow、Ollama、llama.cpp），支持会话级上下文、标题生成和流式响应。
- 分层长期记忆：工作记忆（当前会话）、短期归档、由本地向量搜索（`sqlite-vec`）支持的持久语义记忆，以及异步事实抽取。
- 家族健康关系图谱（v2+ 规划；当前通过 `FamilyRepoKuzu` 占位）。
- 医学知识库本地 RAG（sqlite-vec 关键词 + 向量召回）。
- 本地优先、隐私优先：数据保留在设备上，云端调用可选，且仅在去标识化后发送。
- 四级合规拦截与紧急症状检测。
- 由 GitHub Release 驱动的应用内自动更新器。
- Provider 健康检查与 OAuth / CLI token Provider 的自动 token 刷新。

---

## 技术栈

版本取自实际配置文件（`go.mod`、`wails.json`、`web/package.json`）。不要在没有检查兼容性的情况下假设更新版本。

### 后端

| 组件 | 版本 / 选择 | 用途 |
|-----------|------------------|---------|
| Go | 1.26.4 | 后端语言 |
| Wails v2 | 2.12.0 | 桌面应用框架（Go + React/TypeScript） |
| Google Wire | 0.7.0 | 编译期依赖注入 |
| ONNX Runtime | 1.26.0 | 通过 CGO 进行本地 NER / 嵌入推理 |
| Hugot | 0.7.4 | 基于 ONNX Runtime 的 Hugging Face transformers Go 绑定 |
| SQLCipher | 通过 `mutecomm/go-sqlcipher` | AES-256 加密 SQLite |
| modernc.org/sqlite | 1.53.0 | 普通 SQLite 降级方案 |
| sqlite-vec (`viant/sqlite-vec`) | 0.3.0 | SQLite 向量相似度扩展 |
| 99designs/keyring | 1.2.2 | 操作系统密钥环抽象 |
| testify | 1.11.1 | 单元测试断言 |

### 前端

| 组件 | 版本 / 选择 | 用途 |
|-----------|------------------|---------|
| React | 18.2.0 | UI 框架 |
| TypeScript | 5.9.3 | 类型系统（严格模式） |
| Vite | 6.4.3 | 构建工具 |
| Tailwind CSS | 3.4.1 | 样式 |
| shadcn/ui primitives | 通过本地 `web/src/components/ui/` | 组件基础 |
| react-markdown + remark-gfm | 10.1.0 / 4.0.0 | Markdown 渲染 |
| Zustand | 5.0.13 | 全局状态 |
| Vitest | 4.1.8 | 单元测试 |
| React Router DOM | 7.17.0 | 基于 HashRouter 的导航 |
| React Hook Form + Zod | 7.76.0 / 4.4.3 | 表单与校验 |

---

## 仓库布局

```
medmemo/
├── main.go                    # 应用入口（依赖组装 + 生命周期）
├── main_linux.go              # Linux 特定入口辅助
├── wails_app.go               # 暴露给前端的 Wails 绑定层
├── wails_app_*.go             # 其他绑定 / 测试文件
├── wire.go                    # Wire 注入蓝图（//go:build wireinject）
├── wire_gen.go                # Wire 生成 —— 不要手动编辑
├── cgo_ort_libs_*.go          # 平台特定 ONNX Runtime CGO 指令
├── wails.json                 # Wails 应用元数据；版本单一来源
├── go.mod / go.sum            # Go 模块定义
├── Makefile                   # 主要构建 / 测试 /  lint 命令
├── web/                       # React + TypeScript 前端
│   ├── package.json
│   ├── package-lock.json
│   ├── tsconfig.json
│   ├── .eslintrc.cjs
│   ├── vite.config.ts
│   ├── src/
│   │   ├── components/        # React 组件（聊天、provider、引导、ui 等）
│   │   ├── data/              # Provider 模板与静态数据
│   │   ├── hooks/             # 自定义 hooks
│   │   ├── lib/               # 工具与 Wails 辅助
│   │   ├── pages/             # 页面级组件
│   │   ├── stores/            # Zustand 状态
│   │   ├── types/             # 共享 TypeScript 类型
│   │   ├── utils/             # 辅助工具
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── test/                  # 前端测试辅助
│   └── wailsjs/               # Wails 生成的绑定
├── internal/
│   ├── domain/                # 实体、仓库接口、策略、领域服务
│   │   ├── entity/            # Conversation、Message、HealthMemory、FamilyMember、AppConfig ...
│   │   ├── repository/        # 仓库接口（从领域视角声明的端口）
│   │   └── policy/            # 合规与敏感数据策略
│   ├── application/           # 用例、端口、流水线、流、合规、紧急、健康检查、更新器、反馈
│   │   ├── usecase/           # ChatOrchestrator、MemoryRetriever、TitleGenerator、EmbeddingMigrator ...
│   │   ├── port/              # LLMClient、仓库端口、检测器端口等
│   │   ├── pipeline/          # 去标识化流水线编排
│   │   ├── stream/            # 流式响应的 Wails 事件代理
│   │   ├── feedback/
│   │   ├── healthcheck/
│   │   ├── updater/
│   │   ├── compliance_interceptor.go
│   │   ├── compliance_logger.go
│   │   └── emergency_detector.go
│   ├── adapters/              # 具体适配器
│   │   ├── ai/                # LLM 客户端适配器（OpenAI 兼容、Ollama、本地）
│   │   ├── auth/              # OAuth device flow、CLI token、token 刷新
│   │   ├── detector/          # 基于规则与 ONNX NER 的检测器
│   │   ├── repository/        # SQLCipher / SQLite 仓库实现
│   │   └── updater/           # GitHub Release 适配器
│   └── infrastructure/        # 框架包装
│       ├── config/            # 配置加载器
│       ├── database/          # SQLCipher / SQLite 连接器
│       ├── onnx/              # ONNX Runtime 引擎包装
│       ├── secret/            # 密钥环存储包装
│       └── updater/           # 各平台自动更新安装器
├── pkg/                       # 公共库
│   ├── desensitizer/          # 去标识化工具
│   ├── models/                # 共享数据结构（允许在领域层使用）
│   └── resourcepath/          # 运行时资源路径解析
├── resources/
│   ├── lib/                   # 平台特定 ONNX / tokenizer 本地库
│   ├── models/                # 捆绑的 ONNX 模型（例如 all-MiniLM-L6-v2）
│   └── rules/                 # 合规规则 JSON
├── scripts/                   # 构建、下载与校验脚本
│   ├── build/                 # Wails/GoReleaser 构建辅助与本地库下载器
│   └── validate-provider-templates.js
├── build/                     # CI / 打包脚本
├── docs/                      # 架构、API、用户指南、ADR 文档
│   └── i18n/zh-Hans-CN/       # 简体中文翻译
├── e2e/go/                    # 端到端 Go 测试（//go:build e2e）
├── internal/benchmark/        # 基准测试（//go:build benchmark）
└── medmemo/开发日志/          # 开发日志与问题跟踪
    └── issues.md              # TODO/FIXME/HACK/BUG/XXX 跟踪
```

**重要：** 应用入口与 Wire 文件位于仓库根目录，而不是旧的 `cmd/` 应用目录。`DuckDBConnector` 和 `FamilyRepoKuzu` 是 v2+ 规划存根；活跃存储后端是 SQLCipher/SQLite，向量搜索使用 `sqlite-vec`。

---

## 架构与依赖规则

MedMemo 遵循 Clean Architecture，共四层。依赖方向始终向内。

| 层级 | 包 | 可导入 | 禁止导入 |
|-------|---------|------------|-----------------|
| Domain | `internal/domain/*` | Go 标准库、`pkg/models/` | 任何 `internal/` 子包、`pkg/desensitizer/` |
| Application | `internal/application/*` | `internal/domain/*`、`pkg/models/`、标准库 | `internal/adapters/*`、`internal/infrastructure/*` |
| Adapters | `internal/adapters/*` | `internal/domain/*`、`internal/infrastructure/*`、`pkg/models/`、标准库 | `internal/application/*`、`cmd/*` |
| Infrastructure | `internal/infrastructure/*` | 标准库、第三方框架 | `internal/domain/*`、`internal/application/*`、`internal/adapters/*` |
| 公共包 | `pkg/*` | 标准库 | 任何 `internal/` 子包 |

导入规则也在 `.golangci.yml` 的 `linters-settings.depguard` 中配置。注意 `depguard` linter 当前在启用列表中被禁用，其 v2 配置正在适配；在此之前请手动遵守分层规则。

- 仓库接口从消费侧声明：应用层端口位于 `internal/application/port/`，领域层仓库接口位于 `internal/domain/repository/`。
- DTO 转换应委托给构造函数（如 `domain.NewXxx()`）进行领域校验。

---

## 依赖注入（Wire）

使用 Google Wire 进行编译期依赖注入。

1. 每一层暴露一个 `ProviderSet` 变量（例如 `usecase.ApplicationSet`、`ai.ProviderSet`、`repository.RepositorySet`）。
2. 顶层 `wire.go` 在 `InitializeApp()` 中组装所有 provider。
3. Provider 函数返回**具体类型**，而不是接口。Wire 按返回类型匹配，并通过 `wire.Bind` 绑定接口。
4. 添加依赖的步骤：
   - 编写返回具体类型的构造函数。
   - 将其加入合适的 `ProviderSet`。
   - 如需新绑定，更新 `wire.go`。
   - 在仓库根目录运行 `wire .`。
   - 提交重新生成的 `wire_gen.go`。

**永远不要手动编辑 `wire_gen.go`。**

---

## 构建、测试与开发命令

主要入口是 `Makefile`。版本从 `wails.json`（`info.productVersion`）读取，并在链接时注入。

```bash
# 开发（热重载）。Fedora 43+ / 新版 Ubuntu 使用 webkit2gtk-4.1。
make dev

# 当前平台生产构建
make build

# 跨平台构建
make build-linux      # linux/amd64，tags webkit2_41,ORT
make build-darwin     # darwin/arm64（或 darwin/universal），tags ORT
make build-windows    # windows/amd64，tags ORT

# 测试
make test             # 带 race detector 与覆盖率的单元测试
make test-integration # 集成测试（-tags=integration,ORT）
make test-e2e         # 端到端测试（-tags=e2e）

# 覆盖率报告
make coverage         # 从 coverage.out 生成 coverage.html

# Lint 与格式化
make lint             # golangci-lint run ./...
make fmt              # gofmt + npm run lint -- --fix

# 依赖注入
make wire             # wire .

# 安装工具
make install-tools    # wire、golangci-lint、mockery

# 本地发布快照
make release-local    # 通过 scripts/build/wails-build.sh 构建当前 OS 包
make release-dry-run  # goreleaser snapshot，不发布
```

### 前端专用命令

```bash
cd web && npm install
cd web && npm run dev          # Vite 开发服务器
cd web && npm run build        # tsc + vite build
cd web && npm run lint         # ESLint
cd web && npm run test         # Vitest run
cd web && npm run test:coverage
```

### 构建标签

| 标签 | 含义 |
|-----|---------|
| `ORT` | 启用 ONNX Runtime CGO 绑定（`cgo_ort_libs_*.go`） |
| `webkit2_41` | 在提供 webkit2gtk-4.1 的 Linux 发行版上需要 |
| `integration` | 门控集成测试代码路径 |
| `e2e` | `e2e/go/` 中的门控端到端测试 |
| `benchmark` | `internal/benchmark/` 中的门控基准测试 |

### 本地库

ONNX Runtime 与 tokenizer 本地库由 `scripts/build/` 中的脚本下载到 `resources/lib/<platform>/`：

- `download-onnx.sh` / `download-onnx.ps1`
- `download-tokenizers.sh` / `download-tokenizers.ps1`
- `download-model.sh` / `download-model.ps1`（捆绑的 ONNX 模型）

构建时通过 `CGO_LDFLAGS` 指向这些目录。

### 开发环境搭建

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
go mod download
cd web && npm install && cd ..
make dev
```

Linux 上可能需要系统依赖：

```bash
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev libjavascriptcoregtk-4.1-dev libsqlcipher-dev
```

---

## 编码约定

### Go

- **格式化**：强制使用 `gofmt`。CI 运行 `gofmt`、`goimports`、`go vet` 和 `golangci-lint`。
- **注释**：领域逻辑、合规规则、业务意图优先使用中文注释。纯算法或技术细节可使用英文。注释必须解释 **为什么**，而非重复 **做了什么**。
- **错误处理**：包装每个错误：

  ```go
  // 禁止
  return err

  // 必须
  return fmt.Errorf("failed to retrieve family member %s: %w", id, err)
  ```

- **TODO 格式**：`// TODO(author): 具体描述 [Issue#NNN]`。每个 TODO/FIXME/HACK/BUG/XXX 必须有唯一 issue 编号，并在 `medmemo/开发日志/issues.md` 中跟踪。
- **Context**：所有 I/O 操作接受 `context.Context`。会话级超时使用 `context.WithTimeout(ctx, 30*time.Second)`。
- **并发**：
  - ONNX：推理通过固定工作池分发；每个 worker 串行化调用，因为 `Run()` 不是线程安全的。
  - 数据库写入：在需要时通过单一 goroutine 串行化。
  - 云端 HTTP：信号量限制为最多 4 个并发请求。

### 前端

- **TypeScript**：`strict: true` 不可协商。`tsconfig.json` 还启用 `noUnusedLocals`、`noUnusedParameters` 和 `noFallthroughCasesInSwitch`。
- **组件**：PascalCase（`ComplianceBar.tsx`）。每个组件声明 `interface Props`；生产代码中禁止 `any`。
- **Hooks**：camelCase 前缀 `use`（`useConversation.ts`）。
- **样式**：优先 Tailwind CSS；通过 CSS 变量进行自定义主题。
- **路径别名**：`web/tsconfig.json` 定义 `@/*` → `src/*`，`@wails/*` → `wailsjs/*`。
- **颜色规范**：
  - 用户消息气泡：`#4F8CFF` → `#3B7AF7` 渐变，白色文字。
  - AI 消息气泡：浅色模式白色，深色模式 `#2A2A2A`，文字 `#333333`（浅色）/ `#E5E5E5`（深色）。
  - 系统提示：`#F0F0F5` / `#FFF3E0` / `#E3F2FD`。
- **路由**：使用 HashRouter，因为桌面包中没有服务器。

---

## 测试策略

| 测试类型 | 命令 | 说明 |
|-----------|---------|-------|
| 单元 | `make test` 或 `go test -race -coverprofile=coverage.out ./...` | Makefile 启用 race detector。 |
| 集成 | `make test-integration` | 使用 `-tags=integration,ORT`。 |
| E2E | `make test-e2e` | `-tags=e2e ./e2e/go/...` |
| 基准 | `go test -tags=benchmark ./internal/benchmark/...` | 需要 ONNX 模型存在。 |
| 前端 | `cd web && npm run test` | Vitest。 |

- 目标覆盖率：领域层 100%，整体单元测试行覆盖率 ≥ 70%。
- PR 中覆盖率不得下降（Codecov 基线检查）。
- 必须测试的关键路径：合规引擎（全部四个风险等级）、紧急症状关键词、去标识化往返、会话生命周期、模型切换、离线降级。

---

## 合规、隐私与安全红线

这些是发布阻塞项。所有代码、提示词和 UI 字符串都必须遵守。

| 类别 | 禁止内容 | 后果 |
|----------|------------|-------------|
| 诊断 | 确定性结论，如“你得了 X 病” | 发布阻塞 |
| 处方 | 具体药物、剂量或化验单 | 发布阻塞 |
| 治疗 | 治疗方案或手术建议 | 发布阻塞 |
| AI 身份 | “AI 医生”、“智能诊断”、“数字医生”等 | 发布阻塞 |
| 数据商业化 | 广告、保险定向或出售健康数据 | 发布阻塞 |
| 紧急处理 | 对紧急症状未触发强制医疗提醒 | 发布阻塞 |

### 安全 vs 禁止用语

| 场景 | 安全 | 禁止 |
|----------|------|------------|
| 症状关联 | “可能与……有关”、“常见于……”、“建议关注” | “诊断为”、“确诊”、“患有” |
| 医疗建议 | “建议咨询”、“建议就诊”、“可以考虑” | “必须立即”、“肯定需要”（非紧急情况下） |
| 检查 | “医生可能会建议……”、“常规评估可能包括……” | “你需要做……检查”、“必须做……化验” |
| 治疗 / 用药 | “具体治疗方案需就诊后由医生确定”、“请遵医嘱” | “治疗方案”、“建议服用……”、“用……可治愈” |
| 风险评估 | “危险因素包括……”、“家族史可能增加关注必要性” | “你的风险是……%”、“肯定会/不会……” |

### 二级去标识化流水线

任何云端请求前，用户输入都会先去标识化：

1. **L1 规则引擎** —— Aho-Corasick 匹配；覆盖身份证、手机、银行卡、邮箱、URL；<1 ms。
2. **L2 NER 模型** —— Hugot + ONNX Runtime DistilBERT-ONNX token classification；覆盖人名（PER）、地点（LOC）、机构名（ORG）；20–50 ms。

敏感度等级：`P1Public`、`P2Internal`（软 / 可逆替换）、`P3Confidential`（硬 / 不可逆替换）。

本地模型（Ollama / llama.cpp）跳过去标识化，因为数据不会离开设备。

### 四级合规拦截

LLM 生成后、展示前：

| 级别 | 触发条件 | 响应 |
|-------|---------|----------|
| L1 阻断 | 确定性诊断、剂量、手术 | 阻断并替换为标准提示 |
| L2 警告 | 暗示性诊断、OTC 药品建议、检查建议 | 橙色警告 + 免责声明 |
| L3 提示 | 关于严重疾病的健康教育 | 追加蓝色免责声明条 |
| L4 正常 | 一般健康 / 生活方式建议 | 正常展示 |

流式响应按句子缓冲，通过检测后推送到前端。L1 命中立即中断流。

### 紧急症状检测

每次用户输入都在本地运行，独立于 LLM 路径。

- **A 级**（立即就医）：全屏红色遮罩，提供“拨打 120”、“查找附近急诊”和“继续咨询”选项。
- **B 级**（尽快就医）：红色警告横幅，用户必须确认后才能继续。

目标延迟 < 5 ms。

### 数据保护

- 所有核心数据本地存储在 `~/.medmemo/data`（或 `MEDMEMO_DATA_DIR`）。
- 数据库为 SQLCipher，采用 AES-256 页级加密。
- API key 和数据库加密密钥存储在操作系统密钥环中（macOS Keychain、Windows DPAPI、Linux Secret Service）。
- 仅在启用云端 provider 时才产生网络流量，且仅在去标识化之后。不会向 MedMemo 控制的服务器发送会话内容、家族数据、PII 或行为日志。

---

## CI/CD 与发布

GitHub Actions 工作流位于 `.github/workflows/`：

| 工作流 | 触发条件 | 用途 |
|----------|---------|--------|
| `ci.yml` | push/PR 到 `main`/`develop` | Lint、前端类型检查、单元测试、集成测试、E2E 测试、Linux 构建、跨平台构建 |
| `build-and-release.yml` | push 到 `main` 或 `release/**` | 跨平台 Wails 构建、Linux 冒烟测试、起草/发布 GitHub Release |
| `release.yml` | 标签 `v*` | 跨平台构建 + GoReleaser 发布并生成校验和 |
| `security-scan.yml` | push/PR 到 `main`/`develop`，每周定时 | `govulncheck`、`npm audit`、TruffleHog 密钥扫描 |
| `stale.yml` | 定时 | 过期 issue 管理 |

发布打包由 `scripts/build/wails-build.sh` 和 GoReleaser（`.goreleaser.yml`）处理。发布产物包括平台安装包（`.dmg`、`.exe`、`.AppImage`）和 SHA-256 `checksums.txt`。Linux 二进制在 CI 中强制执行 150 MB 大小门限。

---

## Git 与提交约定

分支模型：

- `main` —— 生产分支，始终可发布。
- `develop` —— 集成分支。
- `feature/M<module>-<brief>` —— 功能分支。
- `release/v<version>` —— 发布分支。
- `hotfix/<brief>` —— 热修分支。

功能分支在 CI 通过后通过 Squash & Merge 合并到 `develop`。

提交信息遵循 Conventional Commits：

```
<type>(<scope>): <subject>
```

| 类型 | 用途 | 示例 |
|------|-----|---------|
| `feat` | 新功能 | `feat(M03): add semantic memory search` |
| `fix` | Bug 修复 | `fix(M01): repair stream buffer flush` |
| `perf` | 性能 | `perf(M06): reduce deidentify latency` |
| `refactor` | 无功能变化 | `refactor(domain): extract confidence rules` |
| `test` | 测试 | `test(M07): add compliance L1 cases` |
| `docs` | 文档 | `docs(adr): update storage ADR` |
| `chore` | 工具 / 构建 | `chore(ci): update Wails version` |
| `security` | 安全修复 | `security(M07): bump ONNX Runtime` |

作用域 `M01`–`M07` 映射到项目文档中定义的七个功能模块：

- `M01` —— Chatbox 对话引擎
- `M02` —— 多模型切换
- `M03` —— 分层长期记忆
- `M04` —— 家族健康图谱
- `M05` —— 可视化记忆控制台
- `M06` —— 边云/本地 AI 协同
- `M07` —— 合规与隐私保护

---

## 文档与国际化

- 所有主文档（`docs/`、根目录 `README.md`、`CONTRIBUTING.md`、`AGENTS.md` 等）均使用英文编写。
- 每份英文主文档顶部必须使用相对 Markdown 链接指向其简体中文翻译。
- 简体中文翻译位于 `docs/i18n/zh-Hans-CN/`，镜像原始相对路径。
- 更新英文文档时，请在同一 PR 中或尽快在后续 PR 中更新中文翻译。
- 架构决策记录（ADR）位于 `docs/adr/`。

---

## 问题跟踪

代码中所有 `TODO/FIXME/HACK/BUG/XXX` 标记都跟踪在 `medmemo/开发日志/issues.md` 中。

规则：

- Issue 编号全局唯一且单调递增。
- 新 issue 编号 = 当前最大值 + 1。
- 添加 TODO 时，在 `issues.md` 中添加一行。
- 关闭 TODO 时，更新该行：**完成** → `✅`，**状态** → `closed`。
- PR 作者必须在请求审查前自我检查 `issues.md` 一致性。

---

## 有用资源

- `README.md` —— 产品概览与快速开始。
- `docs/DEVELOPMENT.md` —— 开发工作流与详细约定。
- `docs/ARCHITECTURE.md` —— 系统架构与数据流。
- `docs/API.md` —— 内部接口契约与 Wails 绑定。
- `docs/COMPLIANCE.md` —— 完整的合规、去标识化与紧急检测参考。
- `docs/SECURITY.md` —— 安全披露、数据本地优先设计与依赖扫描。
- `CONTRIBUTING.md` —— 环境搭建、分支策略与代码审查流程。
- `.skill/medmemo/SKILL.md` —— 针对在本仓库工作的 AI 代理的项目特定 skill。
- `wails.json` —— 应用版本的单一来源。

---

*最后更新：2026-07-09*
