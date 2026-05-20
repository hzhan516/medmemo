# Application Layer（用例层）

> 🌐 [English Version](../../../../internal/application/README.md)

## 定位

Application Layer 是 Clean Architecture 的第二层，负责**编排领域对象完成具体用户用例**。

该层定义系统能做什么（Use Cases），以及系统需要什么外部能力（Ports）。它不知道这些能力由谁提供——数据库操作、HTTP 请求、AI 推理都由 adapter 层实现。

## 目录结构

```
internal/application/
├── usecase/    # 用例实现：ChatOrchestrator, MemoryRetriever, TitleGenerator...
├── port/       # 端口定义：LLMClient, MemoryRepository, SensitiveDetector, ComplianceChecker...
└── pipeline/   # 脱敏流水线编排器：协调 L1/L2/L3 三级脱敏
```

## 导入约束

| 允许导入                                           | 禁止导入                                                   |
|------------------------------------------------|--------------------------------------------------------|
| `github.com/hzhan516/medmemo/internal/domain/*` | `github.com/hzhan516/medmemo/internal/adapters/*`       |
| `github.com/hzhan516/medmemo/pkg/models/`       | `github.com/hzhan516/medmemo/internal/infrastructure/*` |
| Go 标准库                                         | —                                                      |

## 核心职责

1. **用例编排**：接收输入 → 调用领域对象 → 协调适配器 → 返回输出
2. **事务边界**：定义一个用例的原子性边界（如"发送消息"包含脱敏、API调用、合规检测、持久化）
3. **端口定义**：通过 Go Interface 声明系统需要的外部能力，由 adapter 层注入实现

## 示例

```go
// port/llm.go
package port

type LLMClient interface {
	Chat(messages []Message) (string, error)
	StreamChat(messages []Message, callback func(chunk string))
	CheckAvailability() (bool, string)
}

// usecase/chat.go
package usecase

func (o *ChatOrchestrator) Execute(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// 1. 紧急症状检测
	// 2. 三级脱敏流水线
	// 3. LLM 调用
	// 4. 合规拦截
	// 5. 消息持久化
}
```

---

*最后更新：2026-05-19*
