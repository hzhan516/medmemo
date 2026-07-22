# MedMemo Issue 追踪清单

> **作用范围**：代码中所有 `TODO/FIXME/HACK/BUG/XXX` 标记的集中管理表
> **更新规则**：新增/修改/关闭 issue 时必须同步更新本文件
> **编号规则**：新 Issue 编号必须 = 当前最大编号 + 1，禁止复用或跳号
> **最后更新**：2026-07-14

## 格式说明

| 字段 | 说明 |
|------|------|
| 编号 | 唯一 Issue#，全局递增，不可复用 |
| 状态 | `open` / `closed` / `in_progress` |
| 完成 | `✅` 已关闭 / `⬜` 未完成 |
| 优先级 | `P0` 阻塞发布 / `P1` 核心功能 / `P2` 优化改进 |
| 文件位置 | 代码中 TODO 所在的具体文件与行号 |
| 描述 | Issue 具体事项 |
| 关联模块 | M01-M07 功能模块编号 |

---

## M01 — Chatbox 对话引擎

| 编号 | 状态 | 完成 | 优先级 | 文件位置 | 描述 | 关联模块 |
|------|------|------|--------|----------|------|----------|
| #003 | closed | ✅ | P0 | `internal/application/usecase/chat.go:53` | 实现完整对话流水线：输入脱敏→记忆检索→上下文组装→LLM调用→输出还原→合规检测 | M01/M06/M07 |
| #027 | closed | ✅ | P1 | `wails_app.go:95` | 持久化到 ConversationRepository（CreateConversation 中调用仓库保存） | M01 |
| #028 | closed | ✅ | P0 | `wails_app.go:263` | 接入紧急症状检测引擎（AGENTS.md 7.3 节 A级/B级实时检测） | M01/M07 |
| #031 | closed | ✅ | P1 | `wails_app.go:88` | 接入 ConversationRepository（GetConversations 从仓库读取会话列表） | M01 |
| #032 | closed | ✅ | P2 | `web/src/components/chat/ChatInput.tsx:14` | 空输入框时按 Up Arrow 可编辑上一条已发送的消息 | M01 |

---

## M02 — 多模型自由切换

| 编号 | 状态 | 完成 | 优先级 | 文件位置 | 描述 | 关联模块 |
|------|------|------|--------|----------|------|----------|
| #008 | closed | ✅ | P0 | `internal/adapters/ai/openai_adapter.go` | 实现 OpenAI-compatible API 非流式调用（含 Kimi/Qwen/SiliconFlow） | M02/M06 |
| #009 | closed | ✅ | P0 | `internal/adapters/ai/openai_adapter.go` | 实现 SSE 流式解析与分句缓冲合规检测（AGENTS.md 7.2.1 节） | M02/M06/M07 |
| #010 | closed | ✅ | P1 | `internal/adapters/ai/local_adapter.go:65` | 实现 Ollama API 调用（非流式） | M02/M06 |
| #011 | closed | ✅ | P1 | `internal/adapters/ai/local_adapter.go:109` | 实现本地流式推理（llama.cpp / Ollama SSE） | M02/M06 |
| #012 | closed | ✅ | P1 | `internal/adapters/ai/local_adapter.go:167` | 健康检查端点探测（CheckAvailability 实现） | M02/M06 |
| #013 | closed | ✅ | P0 | `internal/adapters/ai/provider.go` | 基于配置路由到具体适配器（ProviderFactory 根据配置返回对应 LLMClient） | M02 |

---

## M03 — 分层长期记忆池

| 编号 | 状态 | 完成 | 优先级 | 文件位置 | 描述 | 关联模块 |
|------|------|------|--------|----------|------|----------|
| #004 | open | ⬜ | P1 | `internal/application/usecase/memory.go:34` | 实现对话摘要与实体提取后的记忆归档（L2/L3 升级） | M03 |
| #014 | closed | ✅ | P1 | `internal/adapters/repository/memory_repo.go:42` | 实现 INSERT OR REPLACE，SQLite 降级完成基本 CRUD | M03 |
| #015 | closed | ✅ | P1 | `internal/adapters/repository/memory_repo.go:42` | 接入 DuckDB vss 扩展 HNSW 向量检索（语义搜索 Top-K）—— 已降级为 sqlite-vec，不再需要 DuckDB vss | M03 |
| #025 | closed | ✅ | P1 | `internal/adapters/repository/memory_repo.go:26` | 替换为 SQLiteConnector 降级实现（DuckDB 驱动待引入） | M03 |
| #035 | open | ⬜ | P2 | `internal/application/usecase/memory.go:347` | 通过 raw_dialogues 关联实现会话间隙记忆召回（当前为 `_ = sessionID` 占位） | M03 |

---

## M04 — 家族关系网图谱

| 编号 | 状态 | 完成 | 优先级 | 文件位置 | 描述 | 关联模块 |
|------|------|------|--------|----------|------|----------|
| #002 | open | ⬜ | P1 | `internal/domain/entity/family.go:69` | 实现血缘关系环检测（AddRelation 中校验无环） | M04 |
| #016 | open | ⬜ | P1 | `internal/adapters/repository/family_repo.go:13` | 接入 Kùzǔ Go 绑定（嵌入式图数据库） | M04 |
| #019 | open | ⬜ | P1 | `internal/adapters/repository/family_repo.go:48` | Cypher 查询实现（FindByDisease 等图查询） | M04 |

---

## M05 — 可视化记忆管理台

> 当前无独立 Issue，M03 的记忆存储与检索完成后自动支撑 M05 前端页面。

---

## M06 — 端云协同

| 编号 | 状态 | 完成 | 优先级 | 文件位置 | 描述 | 关联模块 |
|------|------|------|--------|----------|------|----------|
| #018 | closed | ✅ | P1 | `internal/infrastructure/onnx/runtime.go:16` | 替换为 hugot 实际 Session 类型（`any` → `*hugot.Session`） | M06 |
| #026 | closed | ✅ | P1 | `internal/infrastructure/onnx/runtime.go:21` | 加载 DistilBERT-ONNX 模型框架就绪（`NewEngine` 支持模型路径注入，缺失时降级） | M06 |
| #021 | closed | ✅ | P1 | `internal/infrastructure/onnx/runtime.go:27` | 调用 Hugot `RunPipeline` 并解析 BIO 标签为 `EntitySpan` | M06 |
| #020 | closed | ✅ | P1 | `internal/infrastructure/database/duckdb.go:14` | 接入 DuckDB Go 驱动（github.com/marcboeker/go-duckdb）—— 已降级为 modernc.org/sqlite，不再需要 DuckDB 驱动 | M06 |
| #023 | closed | ✅ | P1 | `internal/infrastructure/database/duckdb.go:29` | 执行 schema 迁移 — DuckDB 已降级为 SQLite，`migrateSQLiteSchema` 已完成并接入 `SQLiteConnector`/`SQLCipherConnector` | M06 |
| #022 | closed | ✅ | P1 | `internal/infrastructure/config/loader.go:48` | 接入标准库 YAML/JSON 配置加载（Viper 待后续引入） | M06 |
| #024 | closed | ✅ | P1 | `internal/infrastructure/secret/store.go:22` | 接入 99designs/keyring 或平台原生 API（macOS/Windows/Linux） | M06 |
| #034 | closed | ✅ | P1 | `internal/infrastructure/network/client.go:58` | 实现指数退避重试 + semaphore 并发限制（最大 4 并发） | M06 |

---

## M07 — 合规与隐私保护

| 编号 | 状态 | 完成 | 优先级 | 文件位置 | 描述 | 关联模块 |
|------|------|------|--------|----------|------|----------|
| #001 | closed | ✅ | P1 | `pkg/desensitizer/desensitizer.go:58` | 补充完整的正则规则与占位符映射（身份证号、手机号、银行卡、邮箱、URL） | M07 |
| #029 | closed | ✅ | P1 | `pkg/desensitizer/desensitizer.go:85` | 实现 regexp 批量替换（Aho-Corasick 待后续优化） | M07 |
| #005 | closed | ✅ | P1 | `internal/application/pipeline/deidentify.go:60` | 接入 pkg/desensitizer 规则引擎（L1RuleStage） | M07 |
| #006 | closed | ✅ | P1 | `internal/application/pipeline/deidentify.go:74` | 调用 ONNX NER 模型检测并替换实体（L2NERStage） | M07 |
| #007 | closed | ✅ | P1 | `internal/application/pipeline/deidentify.go:82` | 移除 no-op L3KeywordStage stub；Trie 树前缀匹配字典非 v1.1.7 实现范围，若未来需要请另建新 issue | M07 |
| #030 | closed | ✅ | P1 | `internal/adapters/detector/rule_detector.go:22` | 接入 pkg/desensitizer 规则引擎（RuleDetector.Detect 实现） | M07 |
| #036 | closed | ✅ | P1 | `internal/application/usecase/chat.go` | strict 脱敏兜底策略已实现（Phase 1：L1.5 兜底规则 + 全消息脱敏 + 严格级 NER 0.5 阈值；Phase 2：脱敏失败 fail-closed，降级 L1-only/完全遮蔽并需用户确认，绝不发送原文） | M07 |
| #040 | open | ⬜ | P2 | `internal/application/usecase/sensitive_detector.go` | M04 医学敏感展示策略（医学关键词列表留给 M04 复用） | M07/M04 |
| #041 | open | ⬜ | P2 | `internal/application/usecase/title_generator.go` | 云端标题生成恢复方案（需随会话 provider 判定 + 脱敏后再恢复） | M07/M01 |

---

## 未分配模块（基础设施通用）

| 编号 | 状态 | 完成 | 优先级 | 文件位置 | 描述 | 关联模块 |
|------|------|------|--------|----------|------|----------|
| #033 | open | ⬜ | P2 | `internal/infrastructure/updater/installer_darwin.go:17` | macOS Sparkle 框架 CGO 绑定集成，替换当前浏览器引导下载方案 | 基础设施 |
| #037 | closed | ✅ | P3 | `docs/internal/*/README.md` | 将 docs/internal 各层 README 改为指向 internal/*/README.md 的指针文档，消除与源码侧 README 的内容重复 | 文档治理 |
| #038 | closed | ✅ | P3 | `scripts/api-docs/generate-api-docs.go` | 从源码自动生成 API 类型 / 端口 / Wails 绑定文档到 docs/api/_generated/，并在 CI 中校验漂移 | 文档治理 |
| #039 | closed | ✅ | P3 | `.github/workflows/docs-audit.yml` | 新增每月定时文档健康审计工作流，产出量化健康度报告 artifact | 文档治理 |

---

## 变更记录

| 日期         | 操作      | 说明                                                                                                                                           |
|------------|---------|----------------------------------------------------------------------------------------------------------------------------------------------|
| 2026-05-18 | 创建      | 初始汇总代码库中全部 TODO，统一编号并解决冲突（原 Issue#016/017/023 各出现 2 次，已拆分）                                                                                   |
| 2026-05-18 | 关闭      | v0.2 基础框架层修复：关闭 Issue#001/#005/#010/#011/#012/#014/#022/#024/#025/#028/#029/#030                                                             |
| 2026-05-18 | 完成      | TASK-025 SQLCipher 数据库加密集成完成                                                                                                                 |
| 2026-05-18 | 关闭      | v0.4 本地 AI 层：关闭 Issue#003（完整对话流水线）                                                                                                           |
| 2026-05-19 | 新增      | TASK-028 自动更新完成，新增 Issue#033（macOS Sparkle 预留接口）                                                                                             |
| 2026-05-19 | 关闭      | TASK-031 Bug 修复冲刺：关闭 Issue#032（ChatInput Up Arrow 编辑上一条消息）                                                                                   |
| 2026-05-22 | 修正      | 同步代码 TODO 与 issues.md：修正 #017→#019、#021→#023 编号不匹配；将重复 #024(network/client) 修正为 #034；wails_app.go 清理 debug 代码；memory_repo.go 补充 #015 TODO 标记 |
| 2026-05-25 | 关闭      | v1.1 补做：关闭 Issue#015（DuckDB vss 已降级为 sqlite-vec）、Issue#020（DuckDB 驱动已降级为 modernc.org/sqlite）                                                 |
| 2026-05-27 | 关闭      | DuckDB 降级后 schema 迁移已完成：关闭 Issue#023（`migrateSQLiteSchema` 已接入 SQLiteConnector/SQLCipherConnector）                                           |
| 2026-05-27 | 新增      | 同步代码 TODO：新增 Issue#035（memory.go:347 会话间隙记忆召回占位代码）                                                                                           |
| 2026-06-25 | 关闭      | v1.1.7 PONYTAIL cleanup：关闭 Issue#007（移除 no-op L3KeywordStage stub）                                                                           |
| 2026-07-07 | 新增      | 脱敏级别打通：新增 Issue#036（strict 兜底策略待实现）                                                                                                          |
| 2026-07-07 | 关闭      | strict 脱敏兜底落地：关闭 Issue#036（Phase 1 L1.5 兜底规则 + 全消息脱敏 + 严格级 NER 0.5 阈值；Phase 2 fail-closed 确认流）                                               |
| 2026-07-14 | 新增 / 关闭 | v1.1.10 文档修复收尾：新增并关闭 Issue#037/#038/#039（单一真源、API 文档自动生成、定期文档审计）                                                                             |
| 2026-07-22 | 新增      | 新增 Issue#040（M04 医学敏感展示策略，医学关键词列表留给 M04 复用）                                                                                                  |
| 2026-07-22 | 新增      | 新增 Issue#041（云端标题生成恢复方案，需随会话 provider 判定 + 脱敏后再恢复）                                                                                        |
