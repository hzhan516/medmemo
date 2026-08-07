# 安全文档

> 🌐 [English Version](../../SECURITY.md)

> 本文档说明 MedMemo 的安全实践、漏洞披露流程与数据保护机制。

---

## 安全披露流程

如果你发现了安全漏洞，请通过以下方式负责任地披露：

1. **不要**在公开的 GitHub Issue 中披露漏洞详情
2. 发送邮件至 `doyle_zhang@outlook.com`，包含：
   - 漏洞描述与影响范围
   - 复现步骤（最小可复现示例）
   - 可能的修复建议
3. 维护团队将在 **72 小时内**确认收到报告
4. 修复完成后，我们将在公开披露前给予报告者合理的提前通知期

## 数据本地存储与加密

MedMemo 的核心设计原则是**数据本地优先**：

- **SQLCipher/SQLite**：对话记录、配置、抽取事实和审计数据使用 AES-256 加密后本地存储
- **sqlite-vec**：语义向量索引与 SQLite 数据一起保存在本地
- **DuckDB / Kùzǔ**：仅为 v2+ 规划存根，v1.x 运行时不启用
- **密钥管理**：API Key、加密密钥存储在平台密钥环中（macOS Keychain / Windows DPAPI / Linux Secret Service）

### 不收集的数据

MedMemo **不会**将以下数据上传到任何服务器：
- 用户对话内容
- 家族成员健康信息
- 个人身份识别信息（PII）
- 使用行为日志

### 可选的网络通信

仅在用户明确启用云端模型时，才会发生以下网络通信：
- 向选定的 LLM API 端点发送**已脱敏**的对话请求
- 模型可用性健康检查

所有网络请求均通过本地配置代理，不经过 MedMemo 控制的服务器。

### 本地/回环跳过脱敏的假设

MedMemo 对本地 provider（Ollama / llama.cpp）与回环端点（`localhost`、`127.0.0.1`、`::1`）跳过
出网脱敏，因为此类流量被假定**数据留在本机**。

**提示：** 若某进程监听回环地址但**将请求转发到云端服务**（localhost-to-cloud 代理），该假设即被
打破：从未脱敏的原文会离开设备。请将此类代理视同 `off` 脱敏级别的知情风险；除非确认端点数据不出
设备，否则不要将 MedMemo 指向回环地址。脱敏级别与严格级 NER 阈值理由参见 `docs/COMPLIANCE.md`。

## 依赖项安全扫描

项目使用以下工具进行依赖安全监控：

- **Dependabot**：自动检测 Go Modules 和 npm 依赖中的已知漏洞
- **govulncheck**：Go 官方漏洞扫描工具
- **npm audit**：Node.js 依赖安全审计

CI 流水线中集成安全扫描，高危漏洞阻断合并。

### npm 审计允许列表策略

前端门禁由 `scripts/check-npm-audit-policy.js` 执行，并使用 `scripts/npm-audit-allowlist.json` 作为已复核例外列表。该策略采用 fail-closed 设计：

- 任何 `critical` 严重级漏洞均阻断构建。
- 任何生产依赖中的 `high` 高危漏洞均阻断构建，除非存在标注 `scope: production` 且已复核的允许列表项。
- 开发依赖中的 `high` 高危漏洞仅在同时满足以下条件时才可列入允许列表：在正式分发的应用中不可达、已记录具体缓解措施、并设有失效日期。
- 生产域允许列表项仅在漏洞在已发布应用上下文中不可利用、已记录具体缓解措施、并设有失效日期时才被接受。
- 每条允许列表项须记录 advisory ID、包名、作用域、依据、缓解措施、目标复查版本和失效日期。过期、包名不匹配或不再存在的条目均阻断构建。

原始的 `npm audit` 输出保留供报告使用，但策略脚本才是最终门禁。

#### v1.1.10 已复核例外

| Advisory | 包 | 作用域 | 原因 | 复查目标版本 | 失效日期 |
|---|---|---|---|---|---|
| [GHSA-qwww-vcr4-c8h2](https://github.com/advisories/GHSA-qwww-vcr4-c8h2) | `react-router` | production | RSC/SSR CSRF 绕过；MedMemo 在桌面 Wails 壳内仅使用 HashRouter，无 RSC/SSR/服务器 action，因此该攻击向量不可达。 | `>=8.3.0` 或修复的 7.x patch | 2026-09-05 |
| [GHSA-qwww-vcr4-c8h2](https://github.com/advisories/GHSA-qwww-vcr4-c8h2) | `react-router-dom` | production | 同一底层 advisory；仅客户端 HashRouter，无 RSC/SSR action 执行路径。 | `>=8.3.0` 或修复的 7.x patch | 2026-09-05 |

## 构建安全

- 所有发布二进制通过 GitHub Actions 自动化构建，构建日志公开可审计
- 发布产物附带 SHA-256 校验和
- 鼓励用户从源码自行构建以验证二进制完整性

## 安全最佳实践（用户端）

1. 始终从官方渠道下载 MedMemo（GitHub Releases 或自行编译源码）
2. 定期更新到最新版本以获取安全补丁
3. 使用系统级别的全盘加密（BitLocker / FileVault / LUKS）增强数据保护
4. 妥善保管本地数据备份，避免未加密传输

---

*最后更新：2026-08-05*
