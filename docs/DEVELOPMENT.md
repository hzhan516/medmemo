# Development Guide

> 🌐 [中文版本](./i18n/zh-Hans-CN/DEVELOPMENT.md)

> This document is intended for MedMemo developers, explaining how to write code that conforms to project standards.

---

## Development Environment Setup

### Prerequisites

- **Go** `1.26.4` (the `go.mod` toolchain directive enforces this)
- **Node.js** `18+` and `npm`
- **Wails v2 CLI** `2.12.0`:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

### Linux System Dependencies

On Debian / Ubuntu:

```bash
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev libjavascriptcoregtk-4.1-dev libsqlcipher-dev
```

On Fedora 43+:

```bash
sudo dnf install gtk3-devel webkit2gtk4.1-devel libsoup3-devel javascriptcoregtk4.1-devel sqlcipher-devel
```

> Ubuntu 22.04+ is required for `webkit2gtk-4.1` / `libsoup-3.0`.

### Download Runtime Resources

MedMemo needs ONNX Runtime native libraries, the tokenizer static library, and the bundled DistilBERT NER model:

```bash
make download-resources
```

This runs the following scripts in order:

1. `scripts/build/download-onnx.sh` — ONNX Runtime native libraries
2. `scripts/build/download-tokenizers.sh` — tokenizer static library
3. `scripts/build/download-model.sh` — DistilBERT NER ONNX model

### First Build

```bash
cd web && npm install && cd ..
make build
```

---

## Clean Architecture Four-Layer Dependency Rules

MedMemo strictly follows the Clean Architecture four-layer model; dependency direction always points inward toward the domain core.

```
┌──────────────────────────────────────┐
│    Infrastructure Layer              │  ← Frameworks & Drivers
│  (ONNX/SQLCipher/SQLite/sqlite-vec/Wails) │
├──────────────────────────────────────┤
│    Adapters Layer                    │  ← Interface Adapters
│  (AI Adapter / Repository)          │
├──────────────────────────────────────┤
│    Application Layer                 │  ← Use Cases
│  (Use Case Orchestration / Ports)   │
├──────────────────────────────────────┤
│    Domain Layer                      │  ← Entities
│  (Entities / Domain Services)       │
└──────────────────────────────────────┘
```

### Package Import Allowlist

| Source Directory | Allowed Imports | Prohibited Imports |
|-----------------|-----------------|-------------------|
| `internal/domain/*` | Standard library + `pkg/models/` | Any `internal/` subpackage |
| `internal/application/*` | `internal/domain/*` + `pkg/models/` + standard library | `internal/adapters/*` + `internal/infrastructure/*` |
| `internal/adapters/*` | `internal/domain/*` + `internal/infrastructure/*` + `pkg/models/` | `internal/application/*` |
| `internal/infrastructure/*` | Standard library + third-party framework libraries | Any `internal/domain/` / `internal/application/` / `internal/adapters/` |

**Core Iron Rule**: `internal/domain/` has zero external dependencies. CI depguard will block any violating imports.

---

## Wire Dependency Injection Guide

MedMemo uses Google Wire for **compile-time** dependency injection; runtime reflection injection is prohibited.

### Steps to Add a New Dependency

1. Write a Provider function in the corresponding package that returns a **concrete type**.
2. Register it in the package's `ProviderSet` variable (e.g., `ApplicationSet = wire.NewSet(...)`).
3. Modify the `InitializeApp` function in the repository-root `wire.go` to include the new ProviderSet.
4. Run `make wire` to regenerate `wire_gen.go`.

**Absolutely prohibited** to manually edit `wire_gen.go`.

### Provider Function Signature Convention

```go
// Correct: returns concrete type
func NewChatOrchestrator(llm port.LLMClient) *ChatOrchestrator

// Incorrect: Wire matches by return type; should not return an interface
func NewChatOrchestrator(llm port.LLMClient) port.UseCase
```

---

## Error Handling Convention

Bare returns of raw errors are prohibited; context must be wrapped:

```go
// Prohibited:
return err

// Required:
return fmt.Errorf("failed to retrieve family member %s: %w", id, err)

// Adapter-layer external error mapping:
if err != nil {
    return nil, fmt.Errorf("sqlite query failed: %w", domain.ErrRecordNotFound)
}
```

---

## Concurrency Safety Convention

### ONNX Inference

- Fixed **2 inference workers**, each holding an independent ONNX Session.
- Tasks are dispatched through a buffered channel (capacity 16).
- **Session sharing for concurrent calls is prohibited** — `Run()` is not thread-safe.

### SQLCipher/SQLite Writes

- Database writes that can conflict are serialized at the application layer.
- Keep connection-pool settings conservative for encrypted SQLite and sqlite-vec operations.

### HTTP Requests

- Semaphore limits maximum 4 concurrent cloud requests.

---

## Frontend Development Convention

### TypeScript Strict Mode

`"strict": true` in `tsconfig.json` must not be disabled.

### Component Convention

- Naming: PascalCase (e.g., `ComplianceBar.tsx`)
- Props: Must write complete TypeScript interface definitions; `any` escape is prohibited.
- Hooks: camelCase prefix `use` (e.g., `useConversation.ts`)

### UI Color Specification

| Element | Light Mode | Dark Mode |
|---------|-----------|-----------|
| User message background | `#4F8CFF` → `#3B7AF7` gradient | Same as left |
| User message text | White | White |
| AI message background | `#FFFFFF` | `#2A2A2A` |
| AI message text | `#333333` | `#E5E5E5` |
| System notice background | `#F0F0F5` / `#FFF3E0` / `#E3F2FD` | Same as left |

### CSS

Prioritize Tailwind CSS utility classes; custom styles should use CSS variables for theme switching.

### Provider Template Files

`web/src/data/provider-templates.json` is the **single source of truth** for provider templates. It is bundled by the build and imported at runtime by `APIKeyPanel`, `OAuthDevicePanel`, and `ProviderTemplateList`.

When adding or editing provider templates, modify only `web/src/data/provider-templates.json`. Run `node scripts/validate-provider-templates.js` before submitting provider-template changes; the validator checks the bundled source file.

---

## Testing Strategy

### Test Pyramid

```
      /\
     /  \  E2E (5%)  — Wails Integration / Playwright
    /____\
   /      \
  /        \ Integration Tests (25%) — go test + SQLite in-memory mode
 /__________\
/            \
/______________\ Unit Tests (70%) — go test + testify + mockery
```

### Coverage Gate

- Unit test line coverage >= 70%
- `domain` layer coverage 100%
- Test coverage must not decrease (Codecov baseline check)

### Key Test Scenarios

1. Compliance engine: all four risk levels, coverage >= 80%
2. Emergency symptoms: 100% trigger test for Level A/B keywords
3. De-identification pipeline: PII input → placeholder replacement → cloud response backfill
4. Conversation lifecycle: create → send → rename → restart → data integrity
5. Model switching: context inheritance, window truncation, timeout downgrade
6. Offline downgrade: local template response when network is unavailable

---

## Context Usage Convention

All I/O operations must accept `context.Context`:

```go
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
defer cancel()
```

Graceful shutdown order follows dependency inversion: close frontend bridge first, then stop inference workers, and finally close database connections.

---

## Build Tags and Commands

| Tag / Command | Purpose |
|---------------|---------|
| `ORT` | Enables ONNX Runtime CGO bindings for local NER and embedding inference. |
| `webkit2_41` | Linux build tag for distributions using webkit2gtk-4.1. |
| `make test` | Runs unit tests with race detector and coverage. |
| `make test-integration` | Runs integration tests with `integration,ORT` tags. |
| `make lint` | Runs Go and frontend lint checks. |

---

## Make Targets

| Target | Purpose | Common Parameters |
|--------|---------|-------------------|
| `make dev` | Start Wails development mode with hot reload | — |
| `make build` | Production build for the current platform | — |
| `make build-linux` | Cross-compile for `linux/amd64` | — |
| `make build-darwin` | Cross-compile for macOS | `DARWIN_PLATFORM=darwin/arm64` or `darwin/universal` |
| `make build-windows` | Cross-compile for `windows/amd64` | — |
| `make test` | Run unit tests with race detector and coverage | — |
| `make test-integration` | Run integration tests | — |
| `make test-e2e` | Run end-to-end tests | — |
| `make coverage` | Generate `coverage.html` from `coverage.out` | — |
| `make lint` | Run `golangci-lint` on the whole project | — |
| `make fmt` | Format Go code and auto-fix frontend lint | — |
| `make wire` | Regenerate `wire_gen.go` | — |
| `make download-resources` | Download ONNX / tokenizer / model resources | — |
| `make install-tools` | Install `wire`, `golangci-lint`, `mockery` | — |
| `make clean` | Remove build artifacts and coverage files | — |
| `make release-local` | Build a local release package | — |
| `make release-dry-run` | Validate GoReleaser config without publishing | — |

For the full and authoritative list, see [`Makefile`](../Makefile).

---

*Last updated: 2026-07-14*
