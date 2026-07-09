# 滚动更新追踪器

### [2026-07-09 / v1.1.10] — Skill 文档修订计划（skill_plan）落地

**[Modified Areas]**
- `.skill/medmemo/tracker.md` — 纠偏 v1.1.9 条目，使其与当前磁盘状态一致
- `.skill/medmemo/SKILL.md` — 新增 docs 滞后提示、入口速查表、权威顺序、实测构建命令；统一 DuckDB/Kùzǔ 为 v2+ 规划组件口径；补充 DistilBERT-ONNX token-classification 与 daulet/tokenizers v1.27.0
- `.skill/medmemo/reference-architecture.md` — 删除 v1.1.9 硬编码，新增 daulet/tokenizers 1.27.0，改述「已冻结组件」为「v2+ 规划组件」，添加版本仅供参考声明
- `.skill/medmemo/reference-release.md` — changelog 模板版本改为占位符 `vX.Y.Z`；补充 `web/package-lock.json` 刷新项
- `.skill/medmemo/reference-modules.md` — 按 `wails_app_*.go` 实际公开方法重建 Wails 绑定表；删除已不存在的记忆方法；补充 `EmbeddingService` / `ComplianceChecker` / `CompressionService` 关键接口签名

**[Logic Evolution]**
- Skill 文档与源码事实来源（`wails.json` → `info.productVersion`）保持一致，消除硬编码版本号
- 明确文档冲突时的权威顺序：源码 > Skill > AGENTS.md > docs/
- 将 DuckDB / Kùzǔ 统一描述为 v2+ 规划组件，避免 AI 被误导为当前活跃后端
- 绑定表与源码逐一核对，覆盖 13 个域共 86 个公开方法

**[Checklist Status]**
- 分支基于当前工作分支
- ✅ 批次 A：tracker 纠偏完成，无与磁盘不符的「已完成」声明
- ✅ 批次 B：硬编码版本号清理、tokenizers 补充、占位符修正、绑定表重建完成
- ✅ 批次 C：入口速查表、权威顺序、实测构建命令、版本集中化标注、v2+ 规划清单完成
- ✅ 所有改动仅落在 `.skill/medmemo/`
- ✅ `git diff --check` 通过

**[Pending/Next Steps]**
- 本次仅更新 Skill 文档，未修改源码；后续继续通过 `docs/kb/` 滚动更新知识库

---

### [2026-07-06 / v1.1.10] — M01 会话压缩服务接入与 CompressSession 绑定实现

**[Modified Areas]**
- `internal/application/usecase/chat.go` — `ApplicationSet` 增加 `NewCompressionService`
- `wails_app.go` — `WailsApp` 新增 `compressionService` 字段与构造函数注入
- `wails_app_context.go` — `CompressSession` 从占位实现改为真实调用 `CompressionService.Compress`
- `wire.go` / `wire_gen.go` — 通过 `wire .` 重新生成依赖注入代码

**[Logic Evolution]**
- 前端触发压缩后，后端使用默认策略 `StrategySummarizeAndReplace`（anchor=1, recent=6）压缩指定会话
- 压缩成功后通过 `context:usage_refresh` 事件通知前端刷新上下文用量显示
- 复用已有的 `ContextEstimator`、`LLMClientFactory`、`ProviderStore`、`MessageRepository` 依赖，未引入新基础设施

**[Checklist Status]**
- 分支 `feature/M01-context-usage-session-compression`
- ✅ `wire .` 重新生成 `wire_gen.go`
- ✅ `gofmt -w` 已执行
- ✅ `GOTOOLCHAIN=auto CGO_LDFLAGS="..." go build -tags "webkit2_41,ORT" -o /tmp/medmemo .` 通过
- ✅ Wails JS 绑定已包含 `CompressSession`，无需重新生成

**[Pending/Next Steps]**
- 前端可在压缩按钮调用 `CompressSession` 并监听 `context:usage_refresh` 刷新用量
- 后续可考虑将压缩策略、anchor/recent 数量外置为配置或请求参数

---

### [2026-07-06 / v1.1.9] — Skill 与知识库对齐、顶层文档与当前版本统一

**[Modified Areas]**
- `.skill/medmemo/SKILL.md` — 增加 `codebase-documenter` / `submission-checker` / `code-comment` Skill 触发要求；补充双语文档规则；更新脱敏管线为 L1→L2
- `.skill/medmemo/reference-*.md` / `tracker.md` — 依赖版本、架构白名单、发布检查清单同步到 `wails.json` 的 `info.productVersion`（当前 1.1.10）；统一 DuckDB/Kùzǔ 为 v2+ 规划组件口径；补充 daulet/tokenizers v1.27.0
- `docs/kb/` — 新建项目 LLM Wiki 知识库（Home、Status Board、Feature Map、Architecture Wiki、Roadmap、Modules、Templates 等）及其中文翻译
- `README.md` / `docs/API.md` / `docs/ARCHITECTURE.md` / `docs/DEVELOPMENT.md` 及其中文翻译 — 修正版本号来源、入口路径、存储栈、DuckDB/Kùzǔ 为 v2+ 规划组件/存根、API 实现列表

**[Logic Evolution]**
- 项目 Skill 与 AGENTS.md 最新约束对齐：文档/注释变更必须触发对应 Skill
- 在仓库内建立可持续更新的 Markdown 知识库体系，避免个人笔记与项目源码分离
- 顶层文档统一以 `wails.json` 的 `info.productVersion`（当前 1.1.10）为基准；DuckDB/Kùzǔ 仅作为 v2+ 规划组件/存根描述，不作为当前活跃后端

**[Checklist Status]**
- 分支 `feature/M01-M07-update-skill-and-knowledge-base`
- ✅ 版本号统一以 `wails.json` → `info.productVersion`（当前 1.1.10）为准
- ✅ `web/package.json` / `web/package-lock.json` 同步刷新
- ✅ `docs/kb/` 知识库初始化
- ✅ `README.md` / `docs/API.md` / `docs/ARCHITECTURE.md` / `docs/DEVELOPMENT.md` 及其中文翻译同步修正
- ✅ `medmemo/开发日志/v1.1/v1.1.9.md` → 创建
- ✅ `go test ./...`、`go vet ./...`、`git diff --check` 通过

**[Pending/Next Steps]**
- `AGENTS.md` 与其中文翻译处于 `.gitignore`，本次修改为本地参考，不影响提交
- 后续继续通过 `docs/kb/` 滚动更新知识库，定期回刷 `Source Index` 中的过时标记

---

### [2026-06-29 / v1.1.8] — 跨平台更新器热修复与版本通道统一

**[Modified Areas]**
- `internal/adapters/updater/github.go` / `internal/infrastructure/updater/` — 跨平台更新器下载、校验、安装路径修复
- `wails_app.go` — 新增 `GetVersionInfo` 绑定，统一返回 productVersion / buildChannel / goVersion / commit
- `web/src/components/UpdateModal.tsx` / `web/src/pages/SettingsPage.tsx` — 更新弹窗与 About 面板字段与后端元数据对齐
- `scripts/build/wails-build.sh` / `.goreleaser.yml` / CI — `-ldflags` 注入版本、commit、channel
- `go.mod` — `golang.org/x/sys` 升级至 v0.45.0

**[Logic Evolution]**
- 默认更新通道从 `stable` 收敛为 `release`，与 CI 产物命名一致
- 版本号/通道/commit 改为构建时注入，避免开发包与发布包版本不一致
- Linux AppImage 更新不再误替换 `/tmp/.mount_*` 运行时文件
- macOS DMG 更新支持自动替换 `.app`，授权失败时回退手动安装
- Windows 安装程序区分 per-user 与 all-users 升级路径

**[Checklist Status]**
- 分支 `hotfix/issue-137-cross-platform-update-v1.1.8`
- ✅ `wails.json` → v1.1.8
- ✅ `web/package.json` → v1.1.8
- ✅ `internal/domain/entity/changelog/zh-Hans.json` → 末尾新增 v1.1.8
- ✅ `go test ./...`、`go vet ./...`、`git diff --check` 通过

**[Pending/Next Steps]**
- `DuckDBConnector` / `FamilyRepoKuzu` 仍为冻结 stub
- 后续可考虑将版本信息 API 暴露到设置页诊断导出

---

### [2026-06-25 / v1.1.7 第二轮 cleanup] — PONYTAIL dead code 清理

**[Modified Areas]**
- `internal/adapters/ai/compat_client.go` / `compat_types.go` / `compat_client_test.go` — 删除孤立 OpenAICompatibleClient 旧实现
- `pkg/desensitizer/desensitizer.go` — 删除未使用的 `Stage` / `Pipeline` / `NewPipeline` 抽象
- `internal/application/port/port_test.go` — 删除空测试
- `internal/application/pipeline/deidentify.go` — 移除 no-op `L3KeywordStage`，`NewDefaultDeidentifyPipeline` 改为 L1→L2 二级
- `internal/application/usecase/local_answer.go` / `chat.go` — 删除 `FactView` 接口与 Null Object，直接传入 `*entity.ExtractedFact`
- `pkg/models/app_config.go` / `internal/infrastructure/config/loader.go` — 移除 `EnableAnalytics` 配置项
- `internal/adapters/auth/token_refresh_service.go` — 统一为 `NewTokenRefreshService(store, opts ...)`，新增 HTTPClient / OnDegraded option
- `wails_app_embedding.go` — 删除 `embeddingAvailabilityReporter` duck-type，直接调用 `EmbeddingService.IsAvailable()`

**[Logic Evolution]**
- 生产路径仅保留实际运行的 L1 规则引擎 + L2 NER 模型脱敏
- `TokenRefreshService` 通过 option 模式配置，Wire 注入使用空 option slice
- `LocalAnswerService.Format` 恢复为直接依赖 `*entity.ExtractedFact`，nil/object guard 内聚在函数内部

**[Checklist Status]**
- 分支 `refactor/code-cleanup-dead-code-audit`，本轮 7 个独立 commit
- ✅ `internal/domain/entity/changelog/zh-Hans.json` → v1.1.7 features 追加
- ✅ `medmemo/开发日志/v1.1/v1.1.7.md` → 新增 Stage 7
- ✅ `medmemo/开发日志/issues.md` → Issue#007 关闭
- ✅ `wire .` 重新生成 `wire_gen.go`
- ✅ `go test ./...`、`go vet ./...`、`git diff --check` 通过

**[Pending/Next Steps]**
- `DBConnector` 暂未删除，影响所有 repository 构造函数，建议单独 PR
- `LLMClientFactory` / `ProviderFactory` 暂未删除，分别被 `ChatOrchestrator` 与 `TitleGenerator` 依赖

---

> 🤖 **To Future AI Agents (记忆延续指令):**
> 每次你对本项目代码进行了实质性修改、重构、版本推进，或者通过阅读代码获得了新的深层理解后，**必须**在此文件顶部追加更新记录。
>
> **更新格式标准：**
> - **[Date/Version]**: 记录时间与当前操作的版本号（如 v1.1.2）。
> - **[Modified Areas]**: 变更了哪些核心文件、模块或配置文件。
> - **[Logic Evolution]**: 业务逻辑发生了什么演进？引入了什么新概念？
> - **[Checklist Status]**: 是否已写入 `/开发日志/`？是否已同步 `wails.json`, `package.json`, `changelog`（及大版本文档）？
> - **[Pending/Next Steps]**: 技术债或待办。

---

### [2026-06-22 / v1.1.7] — 死代码清理、本地回答配置化与检索性能优化

**[Modified Areas]**
- `internal/application/usecase/local_answer_config.go` — 新增本地回答业务配置（subject / 人称 / intent → 模板）
- `internal/application/usecase/local_answer.go` — `LocalAnswerService` 改为配置驱动，引入 `FactView` Null Object
- `internal/application/usecase/chat.go` — 新增 `ChatOrchestratorDeps`，`NewChatOrchestrator` 从 11 参数改为单参数
- `internal/application/usecase/memory.go` + `internal/adapters/repository/fact_repo.go` — `FindByIDs` 批量查询，消除 `recallByVector` N+1
- `internal/adapters/repository/embedding_repo.go` — 统一 `searchSimilarInGo` fallback helper
- `internal/adapters/updater/github.go` — 移除冗余 `strings.ToLower`
- `web/src/pages/MemoryPage.tsx` / `web/src/components/UpdateModal.tsx` — 并行 I/O、模块级常量/函数
- 删除废弃包/文件：`internal/adapters/dto/`、`internal/infrastructure/network/`、`accuracy_tracker`、domain 层多个空接口等

**[Logic Evolution]**
- 本地回答模板从 `switch` 硬编码改为 `map` + `strings.ReplaceAll` 数据驱动
- 事实查询 subject 从硬编码 `"用户"` 改为 `LocalAnswerConfig.Subject`
- `ChatOrchestrator` 依赖按字段分组注入，构造函数可扩展性提升
- 向量召回候选 fact 一次性批量加载，保留原有 approved / confidence / decay / 排序逻辑

**[Checklist Status]**
- 分支 `refactor/code-cleanup-dead-code-audit`，6 commits
- ✅ `wails.json` → v1.1.7
- ✅ `web/package.json` → v1.1.7
- ✅ `npm install` → package-lock.json 刷新
- ✅ `internal/domain/entity/changelog/zh-Hans.json` → 末尾新增 v1.1.7
- ✅ `medmemo/开发日志/v1.1/v1.1.7.md` → 创建
- ✅ `go vet`、`go build -tags webkit2_41,ORT ./...`、`make test`、`make test-integration`、`make test-e2e`、`wire .` 全部通过

**[Pending/Next Steps]**
- `DuckDBConnector` / `FamilyRepoKuzu` 仍为冻结 stub，等待维护者决策
- 后续可考虑将 `IntentResolver` 的 `intentAliasConfig` 也外置到 `LocalAnswerConfig` 族配置

---

### [2026-06-12 / v1.1.5] — M03 混合检索多路召回管线实现

**[Modified Areas]**
- `internal/application/usecase/memory.go` — MemoryRetriever 重构为四路并行召回管线
  - 新增：`recallByIntent` / `recallByKeyword` / `recallByVector` / `recallRecentSameIntent` / `mergeCandidates` / `rerank` / `retrieveWithDiagnostics` / `logDiagnostics` / `applyTokenBudgetToCandidates`
  - 新增字段：`intentResolver *IntentResolver` / `expansionSvc *QueryExpansionService`
  - 保留兼容：`detectEntityMentions` / `mergeMemories` / `applyTokenBudget`（现有测试依赖）
- `internal/application/usecase/memory_retrieval_types.go` — 新建，统一召回类型模型
  - `RetrievalRequest` / `RetrievalCandidate` / `RetrievalDiagnostics` / `PathStatus` / `BuildExpandedQuery`
- `internal/domain/repository/memory_v2.go` — FactRepository 接口新增 `FindApprovedByPredicates`
- `internal/adapters/repository/fact_repo.go` — `FindApprovedByPredicates` SQL 实现
- `internal/application/usecase/memory_test.go` — 新增 9 个管线测试 + stub 增强
- `wire_gen.go` — `wire .` 重新生成

**[Logic Evolution]**
- 旧两路径（detectEntityMentions + semanticSearch）替换为四路并行召回：intent / keyword / vector / recent
- 重排优先级：intent_level → keyword_score → recency_score → vector_similarity → confidence → created_at
- Recent boost：ConfidenceHigh 个人属性时优先推 recent path 候选
- mergeCandidates 按 fact_id 去重合并 matched_paths / reasons / 最高分
- retrieveWithDiagnostics 作为可测试的内部入口，返回完整诊断信息
- 旧 mergeMemories / applyTokenBudget 保留为兼容桩

**[Checklist Status]**
- 分支 `feature/M03-multipath-retrieval-rerank`，6 commits
- `go test ./internal/...` 全部通过
- ✅ `wails.json` → v1.1.5
- ✅ `web/package.json` → v1.1.5
- ✅ `npm install` → package-lock.json 刷新
- ✅ `changelog/zh-Hans.json` → 末尾新增 v1.1.5
- ✅ `medmemo/开发日志/v1.1/v1.1.5.md` → 创建

**[Pending/Next Steps]**
- M03 跨路召回需要更全面的 E2E 测试覆盖
- 考虑将 `recallByVector` 内的 category 同义词扩展与其他路径对齐

---

### [2026-06-11 / v1.1.4] — Skill 按 create-skill 规范重构

**[Modified Areas]**
- `.skill/medmemo/SKILL.md` — 精简入口（<200 行）
- 新增 `reference-architecture.md`, `reference-modules.md`, `reference-conventions.md`, `reference-release.md`, `tracker.md`

**[Logic Evolution]**
- Skill 采用渐进式披露：主文件仅保留工作流与检查清单，详情拆入 reference 文件
- 项目入口在仓库根（`main.go` + `wails_app.go`）；主存储 SQLCipher + sqlite-vec
- 记忆 v1.1 以 Fact + SemanticEmbedding 为主路径；DuckDB/Kùzǔ 为存根

**[Checklist Status]**
- N/A 无版本号变更

**[Pending/Next Steps]**
- `FamilyRepoKuzu` stub — Issue#016
- `DuckDBConnector` 未实现，确认移除或恢复
- v1.2.0 大版本需新建 `INTEGRATION_REPORT_v1.2.md`
