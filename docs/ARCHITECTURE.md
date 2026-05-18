# 架构文档

> 本文档说明 MedMemo 的系统架构设计、模块划分与关键决策。面向开发团队、技术评审者和开源贡献者。

---

## 系统架构总览

```mermaid
flowchart TB
    subgraph FW["Frameworks & Drivers (表示层)"]
        WAILS["Wails v2 Runtime"]
        TRAY["System Tray"]
        WEBVIEW["WebView2 (UI)"]
    end

    subgraph ADPT["Interface Adapters (适配层)"]
        HTTP["HTTP Client<br/>云端API适配"]
        DUCK["DuckDB Repository"]
        KUZU["Kùzǔ Graph DB"]
        ONNX["ONNX Runtime<br/>Hugot适配器"]
        CFG["Config/Secret Loader"]
    end

    subgraph APP["Application Core (用例层)"]
        UC_CHAT["ChatOrchestrator<br/>对话编排"]
        UC_MEM["MemoryRetriever<br/>记忆检索"]
        PIPE["DeidentifyPipeline<br/>三级脱敏流水线"]
        PORT_LLM["Port: LLMClient"]
        PORT_DB["Port: RecordStore"]
        PORT_DET["Port: SensitiveDetector"]
    end

    subgraph DOM["Domain (领域层)"]
        ENT["Entity: Conversation<br/>HealthMemory, FamilyMember"]
        POLICY["Policy: Compliance<br/>SensitiveData"]
        SVC["Service: MemoryConsolidator<br/>FamilyGraphAnalyzer"]
    end

    FW --> ADPT
    ADPT --> APP
    APP --> DOM

    HTTP -. implements .-> PORT_LLM
    DUCK -. implements .-> PORT_DB
    ONNX -. implements .-> PORT_DET

    style DOM fill:#e8f4e8,stroke:#2d5a2d,stroke-width:2px
    style APP fill:#e8ecf4,stroke:#2d3a5a,stroke-width:2px
    style ADPT fill:#f4ece8,stroke:#5a3a2d,stroke-width:2px
    style FW fill:#f0e8f4,stroke:#4a2d5a,stroke-width:2px
```

实线箭头 = 编译期依赖方向（外层→内层）。虚线箭头 = 接口实现关系（适配器实现应用层端口）。

---

## 四层架构映射

| 层级 | 职责 | 典型包路径 | 编译依赖原则 |
|------|------|-----------|------------|
| **Frameworks & Drivers** | Wails v2 前端桥接、系统托盘、窗口管理 | `cmd/health-assistant/` | 仅允许依赖内层 interface |
| **Interface Adapters** | HTTP 客户端、DuckDB/Kùzǔ 仓库、ONNX 推理适配 | `internal/adapters/*` | 依赖 Application 层定义的接口 |
| **Application Core** | 用例编排、脱敏流水线、端口定义 | `internal/application/*` | 不依赖任何外层，不依赖框架 |
| **Domain** | 实体定义、领域服务、策略抽象 | `internal/domain/*` | **零外部依赖**，纯 Go 标准库 |

---

## 核心数据流

### 用户对话请求流

```
用户输入
  → Wails Frontend (React)
    → 后端 HTTP Handler (Wails Bindings)
      → ChatOrchestrator (用例编排)
        → DeidentifyPipeline (L1规则 → L2 NER → L3关键词)
          → 记忆检索 (DuckDB/SQLite 关键词 + 向量搜索)
            → 上下文组装 (L1工作记忆 + L2/L3检索结果)
              → LLMClient (OpenAI/Kimi/Ollama)
                → 流式响应分句缓冲
                  → 合规检测 (L1~L4)
                    → 输出还原 (P2占位符回填)
                      → Wails Events 推送前端
                        → 前端打字机效果渲染
```

### 记忆写入流

```
对话结束/归档触发
  → MemoryRetriever.ArchiveConversation
    → 对话摘要生成 (LLM / 规则模板)
      → 实体提取 (ONNX NER)
        → 冲突检测 (MemoryConsolidator)
          → [无冲突] 直接写入 DuckDB/SQLite
          → [有冲突] 高亮冲突 → 请求用户确认 → 写入
            → 知识图谱更新 (Kùzǔ)
              → 向量索引更新 (DuckDB vss)
```

---

## 模块依赖矩阵

```
                    domain   application   adapters   infrastructure   pkg
                  ┌────────┬─────────────┬──────────┬────────────────┬─────┐
domain            │   -    │      -      │    -     │       -        │  √  │
application       │   √    │      -      │    -     │       -        │  √  |
adapters          │   √    │      √      │    -     │       √        │  √  |
infrastructure    │   -    │      -      │    -     │       -        │  -  |
pkg               │   -    │      -      │    -     │       -        │  -  |
                  └────────┴─────────────┴──────────┴────────────────┴─────┘
```

`√` = 允许导入，`-` = 禁止导入。

---

## 关键设计决策（ADR 索引）

| ADR | 主题 | 决策 | 状态 |
|-----|------|------|------|
| ADR-001 | 桌面框架选型 | Wails v2（Go + React） | 已采纳 |
| ADR-002 | 架构模式 | Clean Architecture 四层模型 | 已采纳 |
| ADR-003 | 依赖注入 | Google Wire（编译期） | 已采纳 |
| ADR-004 | 本地 AI 推理 | Hugot + ONNX Runtime | 已采纳 |
| ADR-005 | 嵌入式数据库 | DuckDB（分析）+ SQLite（事务）+ Kùzǔ（图） | 已采纳 |
| ADR-006 | 向量检索 | DuckDB `vss` 扩展 + HNSW 索引 | 已采纳 |
| ADR-007 | 脱敏架构 | L1规则 + L2 NER ONNX + L3关键词字典 | 已采纳 |
| ADR-008 | 流式合规检测 | 标点分句缓冲 + 检测通过后再推送 | 已采纳 |

完整 ADR 文档位于 `docs/adr/` 目录。

---

## 编译产物形态

| 组件 | 体积预算 | 说明 |
|------|---------|------|
| 主二进制 | ~50MB | Wails Runtime + DuckDB 引擎 + Go 运行时 + 业务逻辑 |
| 资源目录 | ~100MB | ONNX 模型 ~50MB + Kùzǔ 库 ~15MB + 字典/规则 ~10MB + 前端资源 |
| **总计** | **< 150MB** | 单二进制 + 资源目录 |

支持平台：Windows 10+ / macOS 12+ / Linux (Ubuntu 20.04+)，架构 x64 与 ARM64。
