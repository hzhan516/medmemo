# 滚动更新追踪器

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