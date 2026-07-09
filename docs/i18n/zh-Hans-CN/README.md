# MedMemo — 你的私人健康记忆助手 🏥🧠

> 🌐 [English Version](../../../README.md)

> *一个越用越懂你的开源桌面健康咨询工具。分层记忆 × 多角色 Agent × 本地 AI，让每一次对话都沉淀为持久记忆。*

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./../../../LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue)](https://go.dev)
[![Wails v2](https://img.shields.io/badge/Wails-v2-green)](https://wails.io)

---

## 🚨 重要声明

**MedMemo 不是医疗器械，不提供医疗诊断、治疗建议或处方服务。**

MedMemo 是开源健康信息管理和咨询辅助工具，提供的所有信息、分析和建议**仅供参考，不能替代持有执照的执业医师的专业诊断、治疗建议或处方**。在做出任何医疗决定前，请务必咨询专业医疗机构和执业医师。

---

## ✨ 核心特性

### 🧠 分层长期记忆池

采用仿生记忆模型：工作记忆（L1 当前会话）、短期记忆（L2 近期归档）、长期记忆（L3 知识图谱 + 向量索引）。关键症状自动归档，再次提及相关话题时智能唤醒。

### 💬 Chatbox 式多模型自由切换

一键在 Kimi、GPT、通义、Ollama、llama.cpp 等模型间切换，无需重新加载上下文。会话级上下文独立存储，支持窗口截断与摘要压缩。

### 👨‍👩‍👧‍👦 家族关系网图谱

v2+ 规划功能。v1.x 中家族图谱仓库仍是冻结存根，不在产品运行时启用。

### 🗂️ 可视化记忆管理台

时间线视图 + 知识图谱视图双模式浏览，支持关键词搜索、标签筛选和导出。

### 🔒 隐私优先，本地运行

所有核心数据通过 SQLCipher/SQLite 与 sqlite-vec 本地存储，AI 推理本地完成（ONNX Runtime）。两级脱敏流水线确保敏感信息不会进入云端请求，除非用户明确选择关闭。

---

## 🛠 技术栈

| 层级     | 技术选型                                                |
|--------|-----------------------------------------------------|
| 桌面框架   | Wails v2（Go + React/TypeScript）                     |
| 架构     | Clean Architecture 四层模型                             |
| 依赖注入   | Google Wire（编译期）                                    |
| 本地 AI  | Hugot + ONNX Runtime（NER / 嵌入）                      |
| LLM 接入 | OpenAI-compatible API / Ollama / llama.cpp          |
| 数据库    | SQLCipher/SQLite + sqlite-vec；DuckDB/Kùzǔ 为 v2+ 规划存根 |
| 前端     | React 18 + TypeScript 严格模式 + Tailwind CSS + Zustand |

---

## 🚀 快速开始

### 环境要求

- **操作系统**：macOS 12+ / Windows 10+ / Linux (Ubuntu 20.04+)
- **Go**：1.26.4+
- **Node.js**：18.x+
- **npm**：9.x+

### 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails version
wails doctor  # 检查环境依赖
```

### 克隆与安装依赖

```bash
git clone https://github.com/hzhan516/medmemo.git
cd medmemo
go mod download
cd web && npm install && cd ..
```

### 开发模式

```bash
make dev      # 或 wails dev（热重载）
```

### 生产构建

```bash
make build    # 当前平台
make build-darwin   # macOS
make build-windows  # Windows
make build-linux    # Linux
```

产物位于 `build/bin/`：
- macOS: `MedMemo.app`
- Windows: `MedMemo.exe`
- Linux: `medmemo`

---

## 📁 项目结构

```
medmemo/
├── main.go                  # 应用入口
├── wails_app*.go            # Wails 前后端绑定
├── wire.go                  # Wire 注入蓝图
├── wire_gen.go              # Wire 生成文件，禁止手动编辑
├── internal/
│   ├── domain/              # [Entities层] 零外部依赖
│   │   ├── entity/          # Conversation, Memory, FamilyMember...
│   │   ├── repository/      # 仓库接口（Port）
│   │   ├── policy/          # 合规策略、敏感分级策略
│   │   └── service/         # 领域服务接口
│   ├── application/         # [Use Cases层]
│   │   ├── usecase/         # ChatOrchestrator, MemoryRetriever...
│   │   ├── port/            # LLMClient、仓库、检测器等端口
│   │   └── pipeline/        # 两级脱敏流水线编排
│   ├── adapters/            # [Interface Adapters层]
│   │   ├── ai/              # OpenAI/Kimi/Local 适配器
│   │   └── repository/      # SQLCipher/SQLite 仓库实现
│   └── infrastructure/      # [Frameworks & Drivers层]
│       ├── onnx/            # Hugot ONNX 推理运行时
│       ├── database/        # SQLCipher/SQLite + sqlite-vec 连接器
│       ├── config/          # 本地配置加载器
│       ├── secret/          # 系统密钥环封装
│       └── updater/         # 自动更新安装辅助
├── pkg/
│   ├── desensitizer/        # 脱敏算法工具包
│   └── models/              # 共享数据结构
├── web/                     # Wails 前端（React + TypeScript）
├── scripts/                 # 构建脚本、迁移、模型下载
├── build/                   # CI/CD、Docker、打包
├── docs/                    # 架构文档、API 文档、用户指南
├── resources/               # 运行时资源（模型/字典/规则）
├── go.mod
├── wails.json
└── Makefile
```

---

## 🤝 贡献指南

欢迎所有形式的贡献！请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解开发环境搭建、分支策略与提交规范。

### 快速提交

```bash
make install-tools   # 安装开发工具
git checkout -b feature/M01-your-feature
# 开发 → make dev 测试 → make lint → make test
git commit -m "feat(M01): add xxx"
git push origin feature/M01-your-feature
# 发起 Pull Request
```

---

## 📚 文档导航

| 文档                                                  | 内容                                          |
|-----------------------------------------------------|---------------------------------------------|
| [docs/ARCHITECTURE.md](./ARCHITECTURE.md)           | 系统架构总览、四层架构映射、数据流、模块依赖                      |
| [docs/DEVELOPMENT.md](./DEVELOPMENT.md)             | 开发规范、Clean Architecture 依赖规则、Wire 使用指南、测试策略 |
| [docs/API.md](./API.md)                             | 内部接口契约、Wails 前后端绑定说明、错误码定义                  |
| [docs/COMPLIANCE.md](./COMPLIANCE.md)               | 合规红线、两级脱敏流水线、四级拦截规则、紧急症状识别                  |
| [docs/SECURITY.md](./SECURITY.md)                   | 安全披露流程、数据加密说明、依赖项安全扫描                       |
| [docs/user-guide/README.md](./user-guide/README.md) | 用户指南：安装、快速入门、隐私政策、FAQ、故障排查                  |

---

## 🌐 代码仓库

- GitHub: [https://github.com/hzhan516/medmemo](https://github.com/hzhan516/medmemo)
- Gitee: [https://gitee.com/DoyleZhang/medmemo](https://gitee.com/DoyleZhang/medmemo)

---

## 📜 许可证

MedMemo 采用 [MIT License](../../../LICENSE)。可自由使用、修改和分发（含商业用途），须保留版权声明和许可证文本。

**使用本软件即表示您同意许可证中包含的医疗免责声明。**

---

<p align="center">
  Made with ❤️ by MedMemo Contributors
  <br>
  <em>你的健康，值得被记住</em>
</p>

---

*最后更新：2026-07-09*
