# Knowledge API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/knowledge.md)

This document describes Wails bindings for local knowledge document import and management.

---

## Methods

### `SelectKnowledgeFile() (string, error)`

Opens a native file picker and returns the selected knowledge document path.

### `ImportKnowledgeFile(filePath string) (*ImportKnowledgeResponse, error)`

Starts importing a knowledge document from a local file path.

### `ListKnowledgeDocuments() ([]KnowledgeDocumentDTO, error)`

Returns imported knowledge documents.

### `DeleteKnowledgeDocument(id string) error`

Deletes an imported knowledge document and its indexed chunks.

### `GetKnowledgeImportJob(jobID string) (*ImportKnowledgeResponse, error)`

Returns the current status of a knowledge import job.

---

*Last updated: 2026-07-09*
