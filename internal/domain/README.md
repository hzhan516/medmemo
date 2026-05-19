# Domain Layer（实体层）

## 定位

Domain Layer 是 Clean Architecture 的最内层，承载 MedMemo 的核心业务实体与领域规则。

**核心原则**：该层对框架、数据库、UI、外部 API 一无所知。所有技术细节都被排除在外，确保业务逻辑的纯粹性与长期稳定性。

## 目录结构

```
internal/domain/
├── entity/       # 核心业务实体：Conversation, Memory, FamilyMember, HealthMemory...
├── repository/   # 仓库接口（Port）：MemoryRepository, FamilyRepository...
├── policy/       # 策略接口：脱敏策略、合规策略抽象
└── service/      # 领域服务接口：跨实体的复杂业务规则
```

## 导入约束（铁律）

| 允许导入 | 禁止导入 |
|---------|---------|
| Go 标准库 | `github.com/medmemo/medmemo/internal/**/*` |
| `github.com/medmemo/medmemo/pkg/models/` | `github.com/medmemo/medmemo/pkg/desensitizer/` |

> ⚠️ 违反上述规则将被 CI 的 `depguard` 检查阻断合并。

## 何时在此层添加代码

- 新增业务实体（如 `Conversation`、`HealthMemory`）
- 定义领域错误（如 `ErrRecordNotFound`、`ErrValidationFailed`）
- 定义仓库接口（由 adapter 层实现）
- 定义领域事件（如 `MemoryCreated`）

## 示例

```go
// entity/conversation.go
package entity

import "time"

type Conversation struct {
    ID        string
    Title     string
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (c *Conversation) Rename(title string) error {
    if title == "" {
        return ErrValidationFailed
    }
    c.Title = title
    c.UpdatedAt = time.Now()
    return nil
}
```
