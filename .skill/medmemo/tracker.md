# 滚动更新追踪器

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
