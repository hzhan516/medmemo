# MedMemo 术语表

> 🌐 [English Version](../../../glossary.md)

本文档定义 MedMemo 全文档使用的权威中英术语对照与含义。

| 英文术语 | 中文译名 | 说明 |
|---------|---------|------|
| Authentication | 认证 | 验证用户或 Provider 身份的过程（CLI Token、OAuth、API Key、本地模型）。 |
| Authorization | 授权 | 认证通过后授予访问权限；MedMemo 中主要由 OS keyring 与 Provider 凭据承担。 |
| Compliance | 合规 | 生成后拦截机制，用于执行医疗红线（L1 阻断、L2 警告、L3 提示）。 |
| Context compression | 上下文压缩 | 通过摘要或丢弃较早消息来减小 prompt 体积。 |
| Conversation | 会话 | 用户与助手之间的一次完整对话线程。 |
| De-identification | 脱敏 | 在向云端 Provider 发送数据前，去除或遮蔽个人可识别信息（PII）。 |
| Embedding | Embedding | 文本的稠密向量表示，用于语义记忆/知识库检索。 |
| Emergency detection | 紧急症状检测 | 基于本地规则的危急症状识别。 |
| Injection controls | 注入控制 | 决定检索到的记忆/事实是否插入 prompt 的设置。 |
| Memory tier | 记忆层级 | 工作记忆、短期归档、持久化语义记忆三层架构。 |
| Provider | Provider | AI 模型后端，如 Kimi、OpenAI、通义千问、Ollama 或本地端点。 |
| RAG | 检索增强生成 | 检索相关文档/记忆以增强 LLM 回复。 |
| Sensitive data | 敏感数据 | 受合规规则保护的个人身份信息、医疗标识符等数据。 |

## ADR 状态定义

| 英文状态 | 中文译名 |
|---------|---------|
| Accepted | 已采纳 (Accepted) |
| Superseded | 已替代 |
| Proposed | 提议中 |
| Rejected | 已拒绝 |

---

*最后更新：2026-07-14*
