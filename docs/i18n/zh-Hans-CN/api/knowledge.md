# 知识库 API

> 🌐 [English Version](../../../api/knowledge.md)

本文档描述本地知识文档导入与管理相关的 Wails 绑定方法。

---

## 方法

### `SelectKnowledgeFile() (string, error)`

打开原生文件选择器，并返回用户选择的知识文档路径。

### `ImportKnowledgeFile(filePath string) (*ImportKnowledgeResponse, error)`

从本地文件路径开始导入知识文档。

#### `ImportKnowledgeResponse`

| 字段 | 类型 | 说明 |
|------|------|:----|
| `job_id` | `string` | 导入任务 ID |
| `status` | `string` | 任务状态：`pending`、`running`、`done`、`failed` |
| `total` | `int` | 待导入的记录/分块总数 |
| `processed` | `int` | 已处理数量 |
| `error` | `string` | 失败时的错误信息 |

---

### `ListKnowledgeDocuments() ([]KnowledgeDocumentDTO, error)`

返回已导入的知识文档列表。

#### `KnowledgeDocumentDTO`

| 字段 | 类型 | 说明 |
|------|------|:----|
| `id` | `string` | 文档 ID |
| `title` | `string` | 文档标题 |
| `source` | `string` | 来源类型 |
| `citation` | `string` | 引用字符串 |
| `url` | `string` | 可选来源 URL |
| `language` | `string` | 文档语言 |
| `checksum` | `string` | 内容校验和 |
| `created_at` | `int64` | 创建时间戳（毫秒） |
| `updated_at` | `int64` | 最后更新时间戳（毫秒） |

---

### `DeleteKnowledgeDocument(id string) error`

删除已导入知识文档及其索引分块。

### `GetKnowledgeImportJob(jobID string) (*ImportKnowledgeResponse, error)`

返回知识导入任务的当前状态。

---

*最后更新：2026-07-14*
