# MedMemo Agent Guide

> 🌐 [中文版本](./docs/i18n/zh-Hans-CN/AGENTS.md)

> This file is the primary onboarding and reference document for AI coding agents working on MedMemo. Read it in full before writing or modifying code, and keep it accurate when project conventions change.

---

## AI Agent System Directives

Before every code change, self-check the following constraints. Do not treat them as optional.

1. **`internal/domain/` zero-dependency rule** — The domain layer may only import the Go standard library and `pkg/models/`. Any other import is a dependency-rule violation.
2. **Compliance red-line self-check** — Before writing AI prompts, chat logic, or UI copy, verify against the diagnostic / prescription / treatment red lines in [Compliance, Privacy & Security Red Lines](#compliance-privacy--security-red-lines). Never output definitive diagnoses, drug dosages, or treatment plans.
3. **Frontend type safety** — Every React component must declare a TypeScript `interface Props` (or equivalent). `any` escapes are prohibited in production code. TypeScript `strict` mode must remain enabled.
4. **Wire-only dependency injection** — All new dependencies must be wired through `wire.go` at the repository root. Run `wire .` to regenerate `wire_gen.go`. **Never edit `wire_gen.go` by hand.**
5. **Error wrapping** — Bare `return err` is prohibited. Wrap context with `fmt.Errorf("...: %w", err)`.
6. **ONNX concurrency safety** — ONNX Runtime `Session.Run` is not thread-safe. Inference is dispatched through a fixed worker pool (currently 2 workers); each worker serializes calls via its own mutex. Do not share a session across goroutines without serialization.
7. **Trigger required skills** — When writing or updating documentation, invoke the `codebase-documenter` Skill and then the `submission-checker` Skill. When writing, modifying, reviewing, or refactoring code comments, invoke the `code-comment` Skill.
8. **Sync issue tracking** — When closing a TODO/FIXME/HACK/BUG/XXX issue, update the corresponding row in `medmemo/开发日志/issues.md`: set **完成** to `✅` and **状态** to `closed`.
9. **Do not implement frozen stubs** — `DuckDBConnector` and `FamilyRepoKuzu` are v2+ planning stubs and intentionally frozen. Do not fill them in without explicit maintainer approval.
10. **Main docs in English** — All top-level documents (including this file) are authored in English. A Simplified Chinese translation lives under `docs/i18n/zh-Hans-CN/`. Update both when content changes.
11. **Version source of truth** — Product version is read from `wails.json` (`info.productVersion`). When bumping the version, also update `web/package.json`, refresh `web/package-lock.json`, and add a changelog entry in `internal/domain/entity/changelog/zh-Hans.json`.

---

## Project Overview

MedMemo is an **open-source desktop health-information assistant**. It is positioned as a hospital triage and health-consultation information tool and is **explicitly not a medical device**. It does not diagnose, prescribe, or recommend treatments.

Current product version: **1.1.10** (taken from `wails.json`).

Core capabilities:

- Multi-model chat (Kimi, OpenAI, Qwen, SiliconFlow, Ollama, llama.cpp) with session-level context, title generation, and streaming responses.
- Layered long-term memory: working memory (current session), short-term archive, persistent semantic memory backed by local vector search (`sqlite-vec`), and asynchronous fact extraction.
- Family health relationship graph (v2+ planned; currently stubbed via `FamilyRepoKuzu`).
- Medical knowledge base with local RAG (sqlite-vec keyword + vector recall).
- Local-first, privacy-first: data stays on device, cloud calls are optional and only sent after de-identification.
- Four-level compliance interception and emergency-symptom detection.
- In-app auto-updater driven by GitHub releases.
- Provider health checks and automatic token refresh for OAuth / CLI-token providers.

---

## Technology Stack

Versions are taken from the actual configuration files (`go.mod`, `wails.json`, `web/package.json`). Do not assume newer versions without checking compatibility.

### Backend

| Component | Version / Choice | Purpose |
|-----------|------------------|---------|
| Go | 1.26.4 | Backend language |
| Wails v2 | 2.12.0 | Desktop application framework (Go + React/TypeScript) |
| Google Wire | 0.7.0 | Compile-time dependency injection |
| ONNX Runtime | 1.26.0 | Local NER / embedding inference via CGO |
| Hugot | 0.7.4 | Go binding for Hugging Face transformers on ONNX Runtime |
| SQLCipher | via `mutecomm/go-sqlcipher` | AES-256 encrypted SQLite |
| modernc.org/sqlite | 1.53.0 | Plain SQLite fallback |
| sqlite-vec (`viant/sqlite-vec`) | 0.3.0 | Vector similarity extension for SQLite |
| 99designs/keyring | 1.2.2 | OS keyring abstraction |
| testify | 1.11.1 | Unit-test assertions |

### Frontend

| Component | Version / Choice | Purpose |
|-----------|------------------|---------|
| React | 18.2.0 | UI framework |
| TypeScript | 5.9.3 | Type system (strict mode) |
| Vite | 6.4.3 | Build tooling |
| Tailwind CSS | 3.4.1 | Styling |
| shadcn/ui primitives | via local `web/src/components/ui/` | Component base |
| react-markdown + remark-gfm | 10.1.0 / 4.0.0 | Markdown rendering |
| Zustand | 5.0.13 | Global state |
| Vitest | 4.1.8 | Unit testing |
| React Router DOM | 7.17.0 | HashRouter-based navigation |
| React Hook Form + Zod | 7.76.0 / 4.4.3 | Forms and validation |

---

## Repository Layout

```
medmemo/
├── main.go                    # Application entry point (dependency assembly + lifecycle)
├── main_linux.go              # Linux-specific entry helpers
├── wails_app.go               # Wails binding surface exposed to the frontend
├── wails_app_*.go             # Additional binding / test files
├── wire.go                    # Wire injection blueprint (//go:build wireinject)
├── wire_gen.go                # Generated by Wire — DO NOT EDIT
├── cgo_ort_libs_*.go          # Platform-specific ONNX Runtime CGO directives
├── wails.json                 # Wails application metadata; version source of truth
├── go.mod / go.sum            # Go module definition
├── Makefile                   # Primary build / test / lint commands
├── web/                       # React + TypeScript frontend
│   ├── package.json
│   ├── package-lock.json
│   ├── tsconfig.json
│   ├── .eslintrc.cjs
│   ├── vite.config.ts
│   ├── src/
│   │   ├── components/        # React components (chat, provider, onboarding, ui, ...)
│   │   ├── data/              # Provider templates and static data
│   │   ├── hooks/             # Custom hooks
│   │   ├── lib/               # Utilities and Wails helpers
│   │   ├── pages/             # Page-level components
│   │   ├── stores/            # Zustand stores
│   │   ├── types/             # Shared TypeScript types
│   │   ├── utils/             # Helper utilities
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── test/                  # Frontend test helpers
│   └── wailsjs/               # Wails-generated bindings
├── internal/
│   ├── domain/                # Entities, repository interfaces, and policies
│   │   ├── entity/            # Conversation, Message, HealthMemory, FamilyMember, AppConfig, ...
│   │   ├── repository/        # Repository interfaces (ports declared from domain perspective)
│   │   └── policy/            # Compliance and sensitive-data policies
│   ├── application/           # Use cases, ports, pipeline, stream, compliance, emergency, healthcheck, updater, feedback
│   │   ├── usecase/           # ChatOrchestrator, MemoryRetriever, TitleGenerator, EmbeddingMigrator, ...
│   │   ├── port/              # LLMClient, repository ports, detector ports, etc.
│   │   ├── pipeline/          # De-identification pipeline orchestration
│   │   ├── stream/            # Wails event broker for streaming responses
│   │   ├── feedback/
│   │   ├── healthcheck/
│   │   ├── updater/
│   │   ├── compliance_interceptor.go
│   │   ├── compliance_logger.go
│   │   └── emergency_detector.go
│   ├── adapters/              # Concrete adapters
│   │   ├── ai/                # LLM client adapters (OpenAI-compatible, Ollama, local)
│   │   ├── auth/              # OAuth device flow, CLI token, token refresh
│   │   ├── detector/          # Rule-based and ONNX NER detectors
│   │   ├── repository/        # SQLCipher / SQLite repository implementations
│   │   └── updater/           # GitHub release adapter
│   └── infrastructure/        # Framework wrappers
│       ├── config/            # Configuration loader
│       ├── database/          # SQLCipher / SQLite connectors
│       ├── onnx/              # ONNX Runtime engine wrapper
│       ├── secret/            # Keyring store wrapper
│       └── updater/           # Auto-update installer per platform
├── pkg/                       # Public libraries
│   ├── desensitizer/          # De-identification utilities
│   ├── models/                # Shared data structures (allowed in domain layer)
│   └── resourcepath/          # Runtime resource path resolution
├── resources/
│   ├── lib/                   # Platform-specific ONNX / tokenizer native libraries
│   ├── models/                # Bundled ONNX models (e.g., all-MiniLM-L6-v2)
│   └── rules/                 # Compliance rule JSON
├── scripts/                   # Build, download, and validation scripts
│   ├── build/                 # Wails/GoReleaser build helpers and native-library downloaders
│   └── validate-provider-templates.js
├── build/                     # CI / packaging scripts
├── docs/                      # Architecture, API, user-guide, ADR documentation
│   └── i18n/zh-Hans-CN/       # Simplified Chinese translations
├── e2e/go/                    # End-to-end Go tests (//go:build e2e)
├── internal/benchmark/        # Benchmark tests (//go:build benchmark)
└── medmemo/开发日志/          # Development logs and issue tracker
    └── issues.md              # TODO/FIXME/HACK/BUG/XXX tracking
```

**Important:** The application entry point and Wire files are at the repository root, not under a legacy `cmd/` application directory. `DuckDBConnector` and `FamilyRepoKuzu` are v2+ planning stubs; the active storage backend is SQLCipher/SQLite with `sqlite-vec` for vector search.

---

## Architecture & Dependency Rules

MedMemo follows Clean Architecture with four layers. Dependency direction always points inward.

| Layer | Package | May Import | Must Not Import |
|-------|---------|------------|-----------------|
| Domain | `internal/domain/*` | Go standard library, `pkg/models/` | Any `internal/` subpackage, `pkg/desensitizer/` |
| Application | `internal/application/*` | `internal/domain/*`, `pkg/models/`, standard library | `internal/adapters/*`, `internal/infrastructure/*` |
| Adapters | `internal/adapters/*` | `internal/domain/*`, `internal/infrastructure/*`, `pkg/models/`, standard library | `internal/application/*`, `cmd/*` |
| Infrastructure | `internal/infrastructure/*` | Standard library, third-party frameworks | `internal/domain/*`, `internal/application/*`, `internal/adapters/*` |
| Public packages | `pkg/*` | Standard library | Any `internal/` subpackage |

The import rules are also configured in `.golangci.yml` under `linters-settings.depguard`. Note that the `depguard` linter itself is currently disabled in the enabled-linters list while its v2 configuration is being adapted; follow the layer rules manually until it is re-enabled.

- Repository interfaces are declared from the consumer side: application-layer ports live in `internal/application/port/`, domain-layer repository interfaces live in `internal/domain/repository/`.
- DTO conversions should delegate domain validation to constructor functions such as `domain.NewXxx()`.

---

## Dependency Injection (Wire)

Google Wire is used for compile-time dependency injection.

1. Each layer exposes a `ProviderSet` variable (e.g., `usecase.ApplicationSet`, `ai.ProviderSet`, `repository.RepositorySet`).
2. The top-level `wire.go` assembles all providers in `InitializeApp()`.
3. Provider functions return **concrete types**, not interfaces. Wire matches by return type and binds interfaces with `wire.Bind`.
4. To add a dependency:
   - Write a constructor returning the concrete type.
   - Add it to the appropriate `ProviderSet`.
   - Update `wire.go` if new bindings are needed.
   - Run `wire .` from the repository root.
   - Commit the regenerated `wire_gen.go`.

**Never edit `wire_gen.go` manually.**

---

## Build, Test & Development Commands

The primary interface is `Makefile`. Version is read from `wails.json` (`info.productVersion`) and injected at link time.

```bash
# Development (hot reload). On Fedora 43+ / modern Ubuntu uses webkit2gtk-4.1.
make dev

# Production build for the current platform
make build

# Cross-platform builds
make build-linux      # linux/amd64, tags webkit2_41,ORT
make build-darwin     # darwin/arm64 (or darwin/universal), tags ORT
make build-windows    # windows/amd64, tags ORT

# Tests
make test             # Unit tests with race detector and coverage
make test-integration # Integration tests (-tags=integration,ORT)
make test-e2e         # E2E tests (-tags=e2e)

# Coverage report
make coverage         # Generates coverage.html from coverage.out

# Lint & format
make lint             # golangci-lint run ./...
make fmt              # gofmt + npm run lint -- --fix

# Dependency injection
make wire             # wire .

# Install tooling
make install-tools    # wire, golangci-lint, mockery

# Local release snapshot
make release-local    # builds package for current OS via scripts/build/wails-build.sh
make release-dry-run  # goreleaser snapshot, no publish
```

### Frontend-specific commands

```bash
cd web && npm install
cd web && npm run dev          # Vite dev server
cd web && npm run build        # tsc + vite build
cd web && npm run lint         # ESLint
cd web && npm run test         # Vitest run
cd web && npm run test:coverage
```

### Build Tags

| Tag | Meaning |
|-----|---------|
| `ORT` | Enables ONNX Runtime CGO bindings (`cgo_ort_libs_*.go`) |
| `webkit2_41` | Required on Linux distributions shipping webkit2gtk-4.1 |
| `integration` | Gated integration-test code paths |
| `e2e` | Gated end-to-end tests in `e2e/go/` |
| `benchmark` | Gated benchmarks in `internal/benchmark/` |

### Native Libraries

ONNX Runtime and tokenizer native libraries are downloaded into `resources/lib/<platform>/` by the scripts in `scripts/build/`:

- `download-onnx.sh` / `download-onnx.ps1`
- `download-tokenizers.sh` / `download-tokenizers.ps1`
- `download-model.sh` / `download-model.ps1` (bundled ONNX model)

The build sets `CGO_LDFLAGS` to point to these directories.

### Development Environment Setup

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
go mod download
cd web && npm install && cd ..
make dev
```

On Linux you may need system dependencies:

```bash
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev libjavascriptcoregtk-4.1-dev libsqlcipher-dev
```

---

## Coding Conventions

### Go

- **Formatting**: `gofmt` is mandatory. CI runs `gofmt`, `goimports`, `go vet`, and `golangci-lint`.
- **Comments**: Prefer Chinese comments for domain logic, compliance rules, and business intent. English is acceptable for pure algorithmic or technical details. Comments must explain **why**, not restate **what**.
- **Error handling**: Wrap every error:

  ```go
  // Prohibited
  return err

  // Required
  return fmt.Errorf("failed to retrieve family member %s: %w", id, err)
  ```

- **TODO format**: `// TODO(author): concrete description [Issue#NNN]`. Every TODO/FIXME/HACK/BUG/XXX must have a unique issue number tracked in `medmemo/开发日志/issues.md`.
- **Context**: All I/O operations accept `context.Context`. Use `context.WithTimeout(ctx, 30*time.Second)` for session-level timeouts.
- **Concurrency**:
  - ONNX: inference is dispatched through a fixed worker pool; each worker serializes calls because `Run()` is not thread-safe.
  - Database writes: serialized through a single goroutine where required.
  - Cloud HTTP: semaphore limits to 4 concurrent requests.

### Frontend

- **TypeScript**: `strict: true` is non-negotiable. `tsconfig.json` also enables `noUnusedLocals`, `noUnusedParameters`, and `noFallthroughCasesInSwitch`.
- **Components**: PascalCase (`ComplianceBar.tsx`). Declare `interface Props` for every component; no `any` in production code.
- **Hooks**: camelCase prefixed with `use` (`useConversation.ts`).
- **Styling**: Tailwind CSS first; custom theming through CSS variables.
- **Path aliases**: `web/tsconfig.json` defines `@/*` → `src/*` and `@wails/*` → `wailsjs/*`.
- **Color specification**:
  - User message bubble: `#4F8CFF` → `#3B7AF7` gradient, white text.
  - AI message bubble: white in light mode, `#2A2A2A` in dark mode, `#333333` text (light) / `#E5E5E5` (dark).
  - System notices: `#F0F0F5` / `#FFF3E0` / `#E3F2FD`.
- **Routing**: HashRouter is used because there is no server in the desktop bundle.

---

## Testing Strategy

| Test Type | Command | Notes |
|-----------|---------|-------|
| Unit | `make test` or `go test -race -coverprofile=coverage.out ./...` | Race detector enabled in Makefile. |
| Integration | `make test-integration` | Uses `-tags=integration,ORT`. |
| E2E | `make test-e2e` | `-tags=e2e ./e2e/go/...` |
| Benchmark | `go test -tags=benchmark ./internal/benchmark/...` | Requires ONNX model present. |
| Frontend | `cd web && npm run test` | Vitest. |

- Target coverage: domain layer 100%, overall unit-test line coverage ≥ 70%.
- Coverage must not decrease in PRs (Codecov baseline check).
- Critical paths that must have tests: compliance engine (all four risk levels), emergency symptom keywords, de-identification round-trip, conversation lifecycle, model switching, offline fallback.

---

## Compliance, Privacy & Security Red Lines

These are release blockers. Every code, prompt, and UI string must respect them.

| Category | Prohibited | Consequence |
|----------|------------|-------------|
| Diagnosis | Definitive conclusions such as "You have X disease" | Release blocker |
| Prescription | Specific drugs, dosages, or lab orders | Release blocker |
| Treatment | Treatment plans or surgery recommendations | Release blocker |
| AI identity | Terms like "AI doctor", "smart diagnosis", "digital doctor" | Release blocker |
| Data commercialization | Ads, insurance targeting, or selling health data | Release blocker |
| Emergency handling | Failing to trigger mandatory medical reminders for emergency symptoms | Release blocker |

### Safe vs. Prohibited Wording

| Scenario | Safe | Prohibited |
|----------|------|------------|
| Symptom association | "may be related to...", "commonly seen in...", "suggested attention" | "diagnosed as", "confirmed", "suffering from" |
| Medical advice | "recommended consultation", "suggest visiting", "may consider" | "must immediately", "definitely need to" (non-emergency) |
| Tests | "doctor may suggest...", "routine evaluation may include..." | "you need to do... test", "must do... lab work" |
| Treatment / medication | "treatment plan to be determined by doctor after visit", "please follow doctor's advice" | "treatment", "recommended to take...", "can be cured with..." |
| Risk assessment | "risk factors include...", "family history may increase attention necessity" | "your risk is...%", "definitely will/won't..." |

### Two-Tier De-Identification Pipeline

User input is de-identified before any cloud request:

1. **L1 Rule-based engine** — Aho-Corasick matching; covers ID cards, phones, bank cards, emails, URLs; <1 ms.
2. **L2 NER model** — Hugot + ONNX Runtime DistilBERT-ONNX token classification; covers person names (PER), locations (LOC), organizations (ORG); 20–50 ms.

Sensitivity levels: `P1Public`, `P2Internal` (soft / reversible replacement), `P3Confidential` (hard / irreversible replacement).

Local models (Ollama / llama.cpp) skip de-identification because data never leaves the device.

### Four-Level Compliance Interception

After LLM generation and before display:

| Level | Trigger | Response |
|-------|---------|----------|
| L1 Block | Definitive diagnosis, dosage, surgery | Block and replace with standard prompt |
| L2 Warning | Implied diagnosis, OTC drug suggestion, test suggestion | Show with orange warning + disclaimer |
| L3 Notice | Health education about severe disease | Append blue disclaimer bar |
| L4 Normal | General health / lifestyle advice | Normal display |

Streaming responses are buffered by sentence and pushed to the frontend only after passing detection. An L1 hit immediately interrupts the stream.

### Emergency Symptom Detection

Runs locally on every user input, independent of the LLM path.

- **Level A** (immediate care): full-screen red overlay with "Call 120", "Find Nearby ER", and "Continue Consulting" options.
- **Level B** (seek care soon): red warning banner; user must acknowledge before continuing.

Target latency < 5 ms.

### Data Protection

- All core data is stored locally in `~/.medmemo/data` (or `MEDMEMO_DATA_DIR`).
- The database is SQLCipher with AES-256 page-level encryption.
- API keys and the database encryption key are stored in the OS keyring (macOS Keychain, Windows DPAPI, Linux Secret Service).
- Network traffic only occurs when a cloud provider is enabled, and only after de-identification. No conversation content, family data, PII, or behavior logs are sent to MedMemo-controlled servers.

---

## CI/CD & Release

GitHub Actions workflows live in `.github/workflows/`:

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | push/PR to `main`/`develop` | Lint, frontend typecheck, unit tests, integration tests, E2E tests, Linux build, cross-platform build |
| `build-and-release.yml` | push to `main` or `release/**` | Multi-platform Wails build, Linux smoke test, draft/publish GitHub release |
| `release.yml` | tag `v*` | Multi-platform build + GoReleaser release with checksums |
| `security-scan.yml` | push/PR to `main`/`develop`, weekly cron | `govulncheck`, `npm audit`, TruffleHog secret scanning |
| `stale.yml` | scheduled | Stale issue management |

Release packaging is handled by `scripts/build/wails-build.sh` and GoReleaser (`.goreleaser.yml`). Release artifacts include platform installers (`.dmg`, `.exe`, `.AppImage`) and a SHA-256 `checksums.txt`. The Linux binary has a 150 MB size gate enforced in CI.

---

## Git & Commit Conventions

Branch model:

- `main` — production branch, always releasable.
- `develop` — integration branch.
- `feature/M<module>-<brief>` — feature branches.
- `release/v<version>` — release branches.
- `hotfix/<brief>` — hotfix branches.

Feature branches merge into `develop` with Squash & Merge after CI is green.

Commit messages follow Conventional Commits:

```
<type>(<scope>): <subject>
```

| Type | Use | Example |
|------|-----|---------|
| `feat` | New feature | `feat(M03): add semantic memory search` |
| `fix` | Bug fix | `fix(M01): repair stream buffer flush` |
| `perf` | Performance | `perf(M06): reduce deidentify latency` |
| `refactor` | No functional change | `refactor(domain): extract confidence rules` |
| `test` | Tests | `test(M07): add compliance L1 cases` |
| `docs` | Documentation | `docs(adr): update storage ADR` |
| `chore` | Tooling / build | `chore(ci): update Wails version` |
| `security` | Security fix | `security(M07): bump ONNX Runtime` |

Scopes `M01`–`M07` map to the seven functional modules defined in the project documentation:

- `M01` — Chatbox conversation engine
- `M02` — Multi-model switching
- `M03` — Layered long-term memory
- `M04` — Family health graph
- `M05` — Visual memory console
- `M06` — Edge-cloud/local AI collaboration
- `M07` — Compliance & privacy protection

---

## Documentation & Internationalization

- All main documents (`docs/`, root `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, etc.) are written in English.
- Each English main document must link to its Simplified Chinese translation at the top using a relative Markdown link.
- Chinese translations live in `docs/i18n/zh-Hans-CN/` mirroring the original relative path.
- When updating an English document, update the Chinese translation in the same PR or promptly in a follow-up.
- Architecture Decision Records (ADRs) are in `docs/adr/`.

---

## Issue Tracking

All `TODO/FIXME/HACK/BUG/XXX` markers in code are tracked in `medmemo/开发日志/issues.md`.

Rules:

- Issue numbers are globally unique and monotonically increasing.
- A new issue number = current maximum + 1.
- When adding a TODO, add a row to `issues.md`.
- When closing a TODO, update the row: **完成** → `✅`, **状态** → `closed`.
- PR authors must self-check `issues.md` consistency before requesting review.

---

## Useful Resources

- `README.md` — Product overview and quick start.
- `docs/DEVELOPMENT.md` — Developer workflow and detailed conventions.
- `docs/ARCHITECTURE.md` — System architecture and data flows.
- `docs/API.md` — Internal interface contracts and Wails bindings.
- `docs/COMPLIANCE.md` — Full compliance, de-identification, and emergency-detection reference.
- `docs/SECURITY.md` — Security disclosure, data-local-first design, and dependency scanning.
- `CONTRIBUTING.md` — Environment setup, branch strategy, and code-review process.
- `.skill/medmemo/SKILL.md` — Project-specific skill for AI agents working inside this repository.
- `wails.json` — Single source of truth for application version.

---

*Last updated: 2026-07-09*
