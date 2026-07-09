# Adapters Layer（适配器层）

> 🌐 [English Version](../../../../internal/adapters/README.md)

## 定位

Adapters Layer 是 Clean Architecture 的第三层，负责将外部系统的数据格式和协议转换为应用层可理解的形式。

该层实现 application/port 中定义的接口，连接领域核心与外部世界。

## 目录结构

```
internal/adapters/
├── ai/           # LLM 适配器：OpenAI-compatible API 与本地端点
├── auth/         # OAuth、CLI token 与 token 刷新适配器
├── detector/     # 规则检测与 ONNX 检测适配器
├── repository/   # SQLCipher/SQLite 仓库实现
└── updater/      # GitHub Release 适配器
```

## 导入约束

| 允许导入                                                   | 禁止导入                                                |
|--------------------------------------------------------|-----------------------------------------------------|
| `github.com/hzhan516/medmemo/internal/domain/*`         | `github.com/hzhan516/medmemo/internal/application/*` |
| `github.com/hzhan516/medmemo/internal/infrastructure/*` | 仓库根应用装配代码                                  |
| `github.com/hzhan516/medmemo/pkg/models/`               | —                                                   |

## 核心职责

1. **接口实现**：实现 application/port 中定义的接口（如 `LLMClient`、仓库、健康检查适配器）
2. **数据转换**：将外部 API 响应、数据库记录转换为领域实体或共享模型
3. **错误映射**：将外部错误（HTTP 超时、数据库连接失败）映射为领域错误

## 设计原则

- **按协议复用适配器**：OpenAI-compatible 云端点使用 `OpenAIAdapter`，本地端点使用 `LocalAdapter`
- **转换逻辑贴近边界**：外部格式到领域模型的转换放在对应适配器附近
- **可降级**：适配器应实现 `CheckAvailability()`，在不可用时有明确的降级策略

## 示例

```go
// ai/openai_adapter.go
package ai

import "github.com/hzhan516/medmemo/internal/application/port"

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

---

*最后更新：2026-07-09*
