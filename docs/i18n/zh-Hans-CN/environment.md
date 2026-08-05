# 环境变量

> 🌐 [English Version](../../environment.md)

> 本文档列出 MedMemo 生产代码读取的所有环境变量，按作用域分组。环境变量会覆盖对应配置文件中的值。

---

## 配置覆盖

这些变量覆盖 `config.yaml` / `config.json` 中的值（详见 [`docs/DEVELOPMENT.md`](../../DEVELOPMENT.md)）。

| 变量 | 作用 | 默认值 | 示例 | 生效范围 |
|------|------|--------|------|----------|
| `MEDMEMO_DATA_DIR` | 本地加密数据根目录（数据库、日志、记忆、embedding 模型） | macOS/Linux 默认 `~/.medmemo/data`；Windows 选择规则见[下文](#windows-数据目录选择) | `/home/user/.medmemo/data` | config / database |
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

OAuth Device Flow 需要为每个厂商配置 `client_id`。注册前提条件见 [`docs/api/auth.md`](../../api/auth.md)。

| 变量 | 作用 | 默认值 | 示例 | 生效范围 |
|------|------|--------|------|----------|
| `MEDMEMO_KIMI_CLIENT_ID` | Kimi OAuth client ID | *(空)* | `kimi_xxx` | auth |
| `MEDMEMO_GEMINI_CLIENT_ID` | Gemini OAuth client ID | *(空)* | `gemini_xxx` | auth |
| `MEDMEMO_MICROSOFT_CLIENT_ID` | Microsoft OAuth client ID | *(空)* | `ms_xxx` | auth |
| `MEDMEMO_GITHUB_CLIENT_ID` | GitHub OAuth client ID | *(空)* | `github_xxx` | auth |

## 更新器

| 变量 | 作用 | 默认值 | 示例 | 生效范围 |
|------|------|--------|------|----------|
| `MEDMEMO_INSTALL_KIND` | Linux 包格式（覆盖标记文件与 legacy `dpkg`/`rpm` 探测） | `unknown` | `deb` / `rpm` | updater |

---

## Windows 数据目录选择

在 Windows 上，当未显式设置 `MEDMEMO_DATA_DIR` 或 `config.yaml` 中的 `data_dir` 时，MedMemo 按以下优先级选择默认数据目录：

1. **旧库优先** — 若 `%USERPROFILE%\.medmemo\data\medmemo.db` 存在，则使用 `%USERPROFILE%\.medmemo\data`，保证从 v1.1.9 及更早版本升级的用户历史数据可见。
2. **可写的安装目录** — 若注册表中记录了安装目录（例如 `%LOCALAPPDATA%\Programs\MedMemo`），且其 `data` 子目录可写，则使用 `<installDir>\data`。
3. **用户目录兜底** — 若安装目录缺失或不可写（例如只读的 `Program Files` 安装），则回退到 `%USERPROFILE%\.medmemo\data`。

MedMemo 不会在这些位置之间自动移动、复制或合并数据库。如需强制指定位置，请设置 `MEDMEMO_DATA_DIR` 或 `config.yaml` 中的 `data_dir`。

---

## 说明

- 布尔值接受 `true` / `1`（真）或 `false` / `0`（假）。
- `MEDMEMO_DATA_DIR` 同时被配置加载器和 SQLCipher 连接器读取。
- 若配置文件与环境变量同时设置，环境变量优先级更高。
- 在 Windows 上，`MEDMEMO_DATA_DIR` 或 `config.yaml` 中的 `data_dir` 会覆盖上述自动选择。

---

*最后更新：2026-08-05*
