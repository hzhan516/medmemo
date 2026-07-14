# 环境变量

> 🌐 [English Version](../environment.md)

> 本文档列出 MedMemo 生产代码读取的所有环境变量，按作用域分组。环境变量会覆盖对应配置文件中的值。

---

## 配置覆盖

这些变量覆盖 `config.yaml` / `config.json` 中的值（详见 [`docs/DEVELOPMENT.md`](../DEVELOPMENT.md)）。

| 变量 | 作用 | 默认值 | 示例 | 生效范围 |
|------|------|--------|------|----------|
| `MEDMEMO_DATA_DIR` | 本地加密数据根目录（数据库、日志、记忆） | `~/.medmemo/data`（平台相关） | `/home/user/.medmemo/data` | config / database |
| `MEDMEMO_DEFAULT_MODEL` | 新会话默认模型 ID | `kimi-lite` | `kimi-pro` | config |
| `MEDMEMO_PROVIDER_TYPE` | 默认厂商类型 | `kimi` | `openai` | config |
| `MEDMEMO_API_ENDPOINT` | 自定义 API 端点（覆盖厂商默认） | *(空)* | `https://api.openai.com/v1` | config |
| `MEDMEMO_API_KEY_FILE` | 存放 API Key 的文件路径 | *(空)* | `/run/secrets/medmemo_key` | config |
| `MEDMEMO_MODEL_DIR` | 本地 ONNX / 嵌入模型目录 | `resources/models/distilbert-ner` | `/opt/medmemo/models` | config |
| `MEDMEMO_UPDATE_CHECK` | 是否启用自动更新检查 | `true` | `false` | config |
| `MEDMEMO_UPDATE_CHANNEL` | 自动更新使用的通道 | 由构建版本推导（异常兜底 `beta`） | `stable` | config |
| `MEDMEMO_DESENSITIZATION_LEVEL` | 云端请求前的脱敏级别 | `standard` | `off` / `strict` | config |
| `MEDMEMO_DATA_RETENTION_DAYS` | 会话 / 记忆数据保留天数 | `30` | `90` | config |
| `MEDMEMO_EMBEDDING_MODEL_DOWNLOAD_URL` | 首次运行时下载嵌入模型的 URL | *(空)* | `https://example.com/model.onnx` | config |

## 认证

OAuth Device Flow 需要为每个厂商配置 `client_id`。注册前提条件见 [`docs/api/auth.md`](../api/auth.md)。

| 变量 | 作用 | 默认值 | 示例 | 生效范围 |
|------|------|--------|------|----------|
| `MEDMEMO_KIMI_CLIENT_ID` | Kimi OAuth client ID | *(空)* | `kimi_xxx` | auth |
| `MEDMEMO_GEMINI_CLIENT_ID` | Gemini OAuth client ID | *(空)* | `gemini_xxx` | auth |
| `MEDMEMO_MICROSOFT_CLIENT_ID` | Microsoft OAuth client ID | *(空)* | `ms_xxx` | auth |
| `MEDMEMO_GITHUB_CLIENT_ID` | GitHub OAuth client ID | *(空)* | `github_xxx` | auth |

## 更新器

| 变量 | 作用 | 默认值 | 示例 | 生效范围 |
|------|------|--------|------|----------|
| `MEDMEMO_INSTALL_KIND` | Linux 包格式（更新器据此选择安装包） | `unknown` | `deb` / `rpm` | updater |

---

## 说明

- 布尔值接受 `true` / `1`（真）或 `false` / `0`（假）。
- `MEDMEMO_DATA_DIR` 同时被配置加载器和 SQLCipher 连接器读取。
- 若配置文件与环境变量同时设置，环境变量优先级更高。

---

*最后更新：2026-07-14*
