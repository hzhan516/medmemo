# 开发约定参考

## 架构铁律

1. `internal/domain/` 零外部依赖 — 仅标准库 + `pkg/models/`
2. 接口由消费者声明 — `port/` 或 `domain/repository/`
3. 错误必须 `fmt.Errorf("...: %w", err)` 包装
4. Wire 是唯一 DI — 改 `wire.go`，运行 `wire .`，禁止手改 `wire_gen.go`
5. 入口在仓库根目录，非 `cmd/health-assistant/`

## 强制触发 Skill

- 写/改文档时：先触发 `codebase-documenter`，完成后触发 `submission-checker`
- 写/改/审/重构代码注释时：触发 `code-comment`

## 文档与国际化

- 顶层文档（`docs/`、根目录 `.md`）使用英文撰写
- 每个英文主文档顶部须链接中文版本：`[中文版本](<relative-path>)`
- 中文翻译位于 `docs/i18n/zh-Hans-CN/`，结构与英文原文镜像
- 修改英文文档时须同步更新中文翻译（同 PR 或尽快跟进）

## 命名

| 类别 | 规范 | 示例 |
|------|------|------|
| Go 包 | 小写，按层分目录 | `usecase`, `entity` |
| 接口 | 名词 + 角色后缀 | `LLMClient`, `MemoryRepository` |
| SQLite 实现 | 技术后缀 | `ConversationRepoSQLite` |
| ProviderSet | `XxxSet` | `ApplicationSet`, `ONNXSet` |
| React 组件 | PascalCase | `ChatContainer.tsx` |
| Hooks | `use` 前缀 | `useWails` |
| Stores | `xxxStore.ts` | `chatStore.ts` |
| TODO | `// TODO(作者): 描述 [Issue#N]` | 同步 `medmemo/开发日志/issues.md` |

## 并发

- ONNX Session / Embedding：**非线程安全**，mutex 串行
- `WailsApp.activeStreams` + `streamMu` 管理流式取消
- `ollamaMu` 防 Ollama 并发启动冲突
- SQLCipher 单连接池，写入应用层串行

## Context

- 所有 I/O 接收 `context.Context`
- 启动迁移：`context.WithTimeout(ctx, 30*time.Second)`
- 流式超时：`WailsApp.streamTimeout(providerID)`

## 前端

- TypeScript `strict: true` 不可关闭
- Wails 调用统一走 `useWails()`
- Provider 类型转换：`web/src/utils/providerAdapter.ts`
- 路由：`HashRouter`（桌面 file:// 场景）
- UI 颜色：用户消息 `#4F8CFF` 渐变；AI 消息白/`#2A2A2A`；系统提示浅色背景

## 合规红线

| 禁止 | 安全替代 |
|------|----------|
| 确诊（"你患有XX"） | "可能与...有关""建议关注" |
| 药品剂量推荐 | "请遵医嘱用药" |
| 治疗方案/手术建议 | "建议咨询专业医生" |
| "AI医生"等称谓 | "健康信息助手" |
| 紧急症状不提醒 | A 级全屏弹窗 / B 级横幅需确认 |

## 隐性规则

- 本地模型（Ollama/Local）对话不走脱敏
- 云端必须经过 `DeidentifyPipeline`，响应还原 P2 占位符
- 流式通过 Wails Events 推送（`chat:stream:chunk` 等）
- 版本运行时：`wails.json` embed → `main.version`；发布 `-ldflags -X main.version=...`
- 修改 Provider 逻辑须同步模板 JSON + 校验脚本

## 测试

```bash
go test -race ./...
go test -tags=integration ./...
cd web && npm run test
```

## 提交规范（Conventional Commits）

```
<type>(<scope>): <subject>
```

Type：`feat` `fix` `perf` `refactor` `test` `docs` `chore` `security`
Scope：`M01`~`M07` 或 `ci`/`build`/`deps`
