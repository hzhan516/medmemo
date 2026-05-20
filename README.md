# MedMemo — Your Private Health Memory Assistant 🏥🧠

> 🌐 [中文版本](./docs/i18n/zh-Hans-CN/README.md)

> *An open-source desktop health companion that learns you better over time. Layered memory × multi-model agents × local AI — every conversation becomes lasting knowledge.*

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue)](https://go.dev)
[![Wails v2](https://img.shields.io/badge/Wails-v2-green)](https://wails.io)

---

## 🚨 Important Disclaimer

**MedMemo is not a medical device. It does not provide medical diagnoses, treatment advice, or prescriptions.**

MedMemo is an open-source health information management and consultation assistance tool. All information, analysis, and suggestions provided are **for reference only and cannot replace professional diagnosis, treatment advice, or prescriptions from a licensed physician**. Always consult professional medical institutions and licensed physicians before making any medical decisions.

---

## ✨ Core Features

### 🧠 Layered Long-Term Memory Pool

Inspired by biological memory models: Working Memory (L1, current session), Short-Term Memory (L2, recent archive), and Long-Term Memory (L3, knowledge graph + vector index). Key symptoms are automatically archived and intelligently recalled when related topics are mentioned again.

### 💬 Chatbox-Style Multi-Model Switching

Switch between Kimi, GPT, Qwen, Ollama, and llama.cpp with one click — no context reloading required. Session-level context is stored independently, with window truncation and summary compression support.

### 👨‍👩‍👧‍👦 Family Health Relationship Graph

Visualize your family's health tree with Cypher query support. The system automatically analyzes disease clustering patterns and intelligently alerts you to potential hereditary risks.

### 🗂️ Visual Memory Management Console

Browse via timeline view and knowledge graph view. Supports keyword search, tag filtering, and export.

### 🔒 Privacy-First, Local-First

All core data is stored locally (SQLite + DuckDB + Kùzǔ). AI inference runs on-device via ONNX Runtime. A three-stage de-identification pipeline ensures sensitive information never leaves your device.

---

## 🛠 Tech Stack

| Layer | Technology |
|-------|------------|
| Desktop Framework | Wails v2 (Go + React/TypeScript) |
| Architecture | Clean Architecture (4-layer model) |
| Dependency Injection | Google Wire (compile-time) |
| Local AI | Hugot + ONNX Runtime (NER / embeddings) |
| LLM Access | OpenAI-compatible API / Ollama / llama.cpp |
| Database | DuckDB (analytics) + SQLite/SQLCipher (transactions) + Kùzǔ (graph) |
| Frontend | React 18 + TypeScript Strict Mode + Tailwind CSS + Zustand |

---

## 🚀 Quick Start

### Requirements

- **OS**: macOS 12+ / Windows 10+ / Linux (Ubuntu 20.04+)
- **Go**: 1.22+
- **Node.js**: 18.x+
- **npm**: 9.x+

### Install Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails version
wails doctor  # Check environment dependencies
```

### Clone & Install Dependencies

```bash
git clone https://github.com/hzhan516/medmemo.git
cd medmemo
go mod download
cd web && npm install && cd ..
```

### Development Mode

```bash
make dev      # Or wails dev (hot reload)
```

### Production Build

```bash
make build          # Current platform
make build-darwin   # macOS
make build-windows  # Windows
make build-linux    # Linux
```

Build artifacts are located in `build/bin/`:
- macOS: `MedMemo.app`
- Windows: `MedMemo.exe`
- Linux: `medmemo`

---

## 📁 Project Structure

```
medmemo/
├── cmd/health-assistant/    # Application entry point + Wire injection blueprint
├── internal/
│   ├── domain/              # [Entities Layer] Zero external dependencies
│   │   ├── entity/          # Conversation, Memory, FamilyMember...
│   │   ├── repository/      # Repository interfaces (Ports)
│   │   ├── policy/          # Compliance and sensitivity policies
│   │   └── service/         # Domain service interfaces
│   ├── application/         # [Use Cases Layer]
│   │   ├── usecase/         # ChatOrchestrator, MemoryRetriever...
│   │   ├── port/            # LLMClient, RecordStore...
│   │   └── pipeline/        # Three-stage de-identification orchestration
│   ├── adapters/            # [Interface Adapters Layer]
│   │   ├── ai/              # OpenAI/Kimi/Local adapters
│   │   ├── repository/      # DuckDB/Kùzǔ repository implementations
│   │   └── dto/             # Data transfer object transformations
│   └── infrastructure/      # [Frameworks & Drivers Layer]
│       ├── onnx/            # Hugot ONNX inference runtime
│       ├── database/        # DuckDB/SQLite connection pools
│       ├── config/          # Viper configuration loading
│       ├── secret/          # System keychain wrapper
│       └── network/         # HTTP client (retry/timeout/circuit breaker)
├── pkg/
│   ├── desensitizer/        # De-identification algorithm toolkit
│   └── models/              # Shared data structures
├── web/                     # Wails frontend (React + TypeScript)
├── scripts/                 # Build scripts, migrations, model downloads
├── build/                   # CI/CD, Docker, packaging
├── docs/                    # Architecture docs, API docs, user guides
├── resources/               # Runtime resources (models/dictionaries/rules)
├── go.mod
├── wails.json
└── Makefile
```

---

## 🤝 Contributing

All forms of contribution are welcome! Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for development environment setup, branch strategy, and commit conventions.

### Quick Contribution Workflow

```bash
make install-tools   # Install development tools
git checkout -b feature/M01-your-feature
# Develop → make dev test → make lint → make test
git commit -m "feat(M01): add xxx"
git push origin feature/M01-your-feature
# Open a Pull Request
```

---

## 📚 Documentation

| Document                                                 | Content                                                                                                                  |
|----------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)           | System architecture overview, four-layer mapping, data flow, module dependencies                                         |
| [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md)             | Development standards, Clean Architecture dependency rules, Wire guide, testing strategy                                 |
| [docs/API.md](./docs/API.md)                             | Internal interface contracts, Wails frontend-backend binding, error code definitions                                     |
| [docs/COMPLIANCE.md](./docs/COMPLIANCE.md)               | Compliance red lines, three-stage de-identification pipeline, four-level interception rules, emergency symptom detection |
| [docs/SECURITY.md](./docs/SECURITY.md)                   | Security disclosure process, data encryption details, dependency security scanning                                       |
| [docs/user-guide/README.md](./docs/user-guide/README.md) | End-user guide: installation, getting started, privacy policy, FAQ, troubleshooting                                      |

---

## 🌐 Repositories

- GitHub: [https://github.com/hzhan516/medmemo](https://github.com/hzhan516/medmemo)
- Gitee: [https://gitee.com/DoyleZhang/medmemo](https://gitee.com/DoyleZhang/medmemo)

---

## 📜 License

MedMemo is licensed under [MIT License](./LICENSE). Free to use, modify, and distribute (including commercially), provided the copyright notice and license text are retained.

**Using this software means you agree to the medical disclaimer contained in the license.**

---

<p align="center">
  Made with ❤️ by MedMemo Contributors
  <br>
  <em>Your health deserves to be remembered</em>
</p>

---

*Last updated: 2026-05-19*
