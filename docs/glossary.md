# MedMemo Glossary

> 🌐 [中文版本](./i18n/zh-Hans-CN/glossary.md)

This document defines authoritative English/Chinese translations and meanings for terms used across MedMemo documentation.

| Term | Chinese | Definition |
|------|---------|------------|
| Authentication | 认证 | Verifying user or provider identity (CLI token, OAuth, API key, local model). |
| Authorization | 授权 | Granting access after authentication; in MedMemo this is mostly handled by the OS keyring and provider credentials. |
| Compliance | 合规 | Post-generation interception that enforces medical red lines (L1 block, L2 warning, L3 notice). |
| Context compression | 上下文压缩 | Reducing prompt size by summarizing or dropping older messages. |
| Conversation | 会话 | A single chat thread between the user and the assistant. |
| De-identification | 脱敏 | Removing or masking personally identifiable information before sending data to cloud providers. |
| Embedding | Embedding | Dense vector representation of text used for semantic memory/knowledge retrieval. |
| Emergency detection | 紧急症状检测 | Local rule-based detection of urgent symptoms. |
| Injection controls | 注入控制 | Settings that decide whether retrieved memories/facts are inserted into the prompt. |
| Memory tier | 记忆层级 | Working, short-term archive, and persistent semantic memory layers. |
| Provider | Provider | An AI model backend such as Kimi, OpenAI, Qwen, Ollama, or a local endpoint. |
| RAG | 检索增强生成 | Retrieving relevant documents/memories to augment LLM replies. |
| Sensitive data | 敏感数据 | PII, medical identifiers, and other data covered by compliance rules. |

## ADR Status Definitions

| Status | Chinese |
|--------|---------|
| Accepted | 已采纳 (Accepted) |
| Superseded | 已替代 |
| Proposed | 提议中 |
| Rejected | 已拒绝 |

---

*Last updated: 2026-07-14*
