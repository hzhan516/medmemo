# Development Guide

> 🌐 [中文版本](./i18n/zh-Hans-CN/DEVELOPMENT.md)

> This document is intended for MedMemo developers, explaining how to write code that conforms to project standards.

---

## Clean Architecture Four-Layer Dependency Rules

MedMemo strictly follows the Clean Architecture four-layer model; dependency direction always points inward toward the domain core.

```
┌──────────────────────────────────────┐
│    Infrastructure Layer              │  ← Frameworks & Drivers
│  (ONNX/DuckDB/SQLite/Wails/Viper)   │
├──────────────────────────────────────┤
│    Adapters Layer                    │  ← Interface Adapters
│  (AI Adapter / Repository / DTO)    │
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
3. Modify the `InitializeApp` function in `cmd/health-assistant/wire.go` to include the new ProviderSet.
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
    return nil, fmt.Errorf("duckdb query failed: %w", domain.ErrRecordNotFound)
}
```

---

## Concurrency Safety Convention

### ONNX Inference

- Fixed **2 inference workers**, each holding an independent ONNX Session.
- Tasks are dispatched through a buffered channel (capacity 16).
- **Session sharing for concurrent calls is prohibited** — `Run()` is not thread-safe.

### DuckDB Writes

- Single goroutine executes writes.
- MVCC guarantees read concurrency safety.

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

---

## Testing Strategy

### Test Pyramid

```
      /\
     /  \  E2E (5%)  — Wails Integration / Playwright
    /____\
   /      \
  /        \ Integration Tests (25%) — go test + DuckDB in-memory mode
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
