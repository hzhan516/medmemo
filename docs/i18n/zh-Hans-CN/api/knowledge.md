# 知识库 API

> 🌐 [English Version](../../../api/knowledge.md)

本文档描述本地知识文档导入与管理相关的 Wails 绑定方法。

---

## 方法

### `SelectKnowledgeFile() (string, error)`

打开原生文件选择器，并返回用户选择的知识文档路径。

### `ImportKnowledgeFile(filePath string) (*ImportKnowledgeResponse, error)`

从本地文件路径开始导入知识文档。

### `ListKnowledgeDocuments() ([]KnowledgeDocumentDTO, error)`

返回已导入的知识文档列表。

### `DeleteKnowledgeDocument(id string) error`

删除已导入知识文档及其索引分块。

### `GetKnowledgeImportJob(jobID string) (*ImportKnowledgeResponse, error)`

返回知识导入任务的当前状态。

---

*最后更新：2026-07-09*
