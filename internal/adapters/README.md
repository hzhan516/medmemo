# Adapters Layer（适配器层）

## 定位

Adapters Layer 是 Clean Architecture 的第三层，负责将外部系统的数据格式和协议转换为应用层可理解的形式。

该层实现 application/port 中定义的接口，连接领域核心与外部世界。

## 目录结构

```
internal/adapters/
├── ai/           # AI 模型客户端适配器簇：OpenAI, Kimi, Ollama, Local...
├── repository/   # 数据持久化适配器：DuckDB 实现、SQLite 实现、Kùzǔ 实现...
├── detector/     # 敏感检测适配器：规则引擎、NER 模型...
└── dto/          # 数据传输对象转换层：外部格式 ↔ 领域格式
```

## 导入约束

| 允许导入 | 禁止导入 |
|---------|---------|
| `github.com/medmemo/medmemo/internal/domain/*` | `github.com/medmemo/medmemo/internal/application/*` |
| `github.com/medmemo/medmemo/internal/infrastructure/*` | `github.com/medmemo/medmemo/cmd/*` |
| `github.com/medmemo/medmemo/pkg/models/` | — |

## 核心职责

1. **接口实现**：实现 application/port 中定义的接口（如 `LLMClient`、`MemoryRepository`）
2. **数据转换**：将外部 API 响应、数据库记录转换为领域实体（通过 DTO 层）
3. **错误映射**：将外部错误（HTTP 超时、数据库连接失败）映射为领域错误

## 设计原则

- **一个外部系统一个适配器**：OpenAI 有独立适配器，Kimi 也有独立适配器，避免硬编码差异
- **DTO 转换纯函数**：`dto/` 中的转换函数无状态、无副作用，返回 `error` 而非 panic
- **可降级**：适配器应实现 `CheckAvailability()`，在不可用时有明确的降级策略

## 示例

```go
// ai/openai_adapter.go
package ai

import "github.com/medmemo/medmemo/internal/application/port"

type OpenAIAdapter struct {
    client *http.Client
    apiKey string
}

func (a *OpenAIAdapter) Chat(messages []port.Message) (string, error) {
    // 实现 OpenAI API 调用
}

// 通过 Wire 注册到 ProviderSet
var ProviderSet = wire.NewSet(
    NewOpenAIAdapter,
    wire.Bind(new(port.LLMClient), new(*OpenAIAdapter)),
)
```
