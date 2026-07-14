# Knowledge API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/knowledge.md)

This document describes Wails bindings for local knowledge document import and management.

---

## Methods

### `SelectKnowledgeFile() (string, error)`

Opens a native file picker and returns the selected knowledge document path.

### `ImportKnowledgeFile(filePath string) (*ImportKnowledgeResponse, error)`

Starts importing a knowledge document from a local file path.

#### `ImportKnowledgeResponse`

| Field | Type | Description |
|-------|------|:------------|
| `job_id` | `string` | Import job ID |
| `status` | `string` | Job status: `pending`, `running`, `done`, `failed` |
| `total` | `int` | Total records/chunks to import |
| `processed` | `int` | Processed count |
| `error` | `string` | Error message if failed |

---

### `ListKnowledgeDocuments() ([]KnowledgeDocumentDTO, error)`

Returns imported knowledge documents.

#### `KnowledgeDocumentDTO`

| Field | Type | Description |
|-------|------|:------------|
| `id` | `string` | Document ID |
| `title` | `string` | Document title |
| `source` | `string` | Source type |
| `citation` | `string` | Citation string |
| `url` | `string` | Optional source URL |
| `language` | `string` | Document language |
| `checksum` | `string` | Content checksum |
| `created_at` | `int64` | Creation timestamp (ms) |
| `updated_at` | `int64` | Last update timestamp (ms) |

---

### `DeleteKnowledgeDocument(id string) error`

Deletes an imported knowledge document and its indexed chunks.

### `GetKnowledgeImportJob(jobID string) (*ImportKnowledgeResponse, error)`

Returns the current status of a knowledge import job.

---

*Last updated: 2026-07-09*
