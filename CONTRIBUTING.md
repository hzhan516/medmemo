# Contributing Guide

> 🌐 [中文版本](./docs/i18n/zh-Hans-CN/CONTRIBUTING.md)

Thank you for your interest in MedMemo! Whether you are a Go developer, frontend engineer, medical professional, or coding beginner, we welcome your participation.

---

## Development Environment Setup

### Prerequisites

| Tool | Minimum Version | Installation |
|------|-----------------|--------------|
| Go | 1.22+ | [Official Download](https://go.dev/dl/) or `brew install go` |
| Node.js | 18.x+ | [Official Download](https://nodejs.org/) or `brew install node` |
| Wails CLI | v2.9+ | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| golangci-lint | latest | `make install-tools` |
| Wire | v0.6+ | `make install-tools` |

### Quick Start

```bash
git clone https://github.com/hzhan516/medmemo.git
cd medmemo
make install-tools
go mod download
cd web && npm install && cd ..
make dev
```

### Common Commands

```bash
make dev              # Development mode (hot reload)
make build            # Production build
make test             # Run unit tests
make test-integration # Run integration tests
make coverage         # Generate test coverage report
make lint             # Code linting
make wire             # Regenerate Wire dependency injection code
make fmt              # Format code
```

---

## Git Flow Branch Strategy

```
main (production branch, always releasable)
  ^
develop (integration branch)
  ^
feature/M<module>-<brief> (feature branch)
```

| Branch Type | Naming Convention | Merge Strategy |
|-------------|-------------------|----------------|
| `main` | `main` | Only accepts release/hotfix merges |
| `develop` | `develop` | Only accepts feature merges; requires all-green CI |
| `feature/*` | `feature/M<module>-<brief>` | Squash & Merge |
| `release/*` | `release/v<version>` | Squash & Merge |
| `hotfix/*` | `hotfix/<brief-description>` | Squash & Merge |

---

## Commit Convention (Conventional Commits)

```
<type>(<scope>): <subject>
```

| Type | Purpose | Scope Example |
|------|---------|---------------|
| `feat` | New feature | `feat(M03): add HNSW vector index` |
| `fix` | Bug fix | `fix(PER-03): reduce ONNX inference latency` |
| `perf` | Performance improvement | `perf(M06): optimize deidentify pipeline` |
| `refactor` | Refactoring (no functional change) | `refactor(domain): extract SensitivityLevel` |
| `test` | Testing related | `test(M01): add E2E test for conversation` |
| `docs` | Documentation update | `docs(adr): add ADR-006 for HNSW` |
| `chore` | Build / tooling | `chore(ci): add Windows build matrix` |
| `security` | Security fix | `security(M07): bump ONNX Runtime` |

**Scope reference**: `M01`–`M07` correspond to the 7 major functional modules; `ci`/`build`/`deps` correspond to engineering tasks.

---

## Code Review Process

1. All PRs must pass the CI pipeline (Lint / Unit Test / Integration Test / Build)
2. At least 1 maintainer Code Review Approval
3. Test coverage must not decrease (Codecov baseline check)
4. Architecture dependency check (`depguard`) must report 0 violations

---

## Coding Standards

### Go

- Format all code with `gofmt`
- Pass `golangci-lint` with 0 errors
- Error handling must wrap with `fmt.Errorf("...: %w", err)`
- **Domain layer zero external dependencies**: only standard library + `pkg/models/`
- Core domain logic uses **Chinese comments** (聚焦 Why, not What)

### Frontend

- TypeScript strict mode must remain enabled
- Components use PascalCase; Hooks use `use` prefix
- Prefer Tailwind CSS utility classes
- Colors follow UI spec: user message `#4F8CFF`, AI message white/`#2A2A2A`, system prompt `#F0F0F5`

---

## Reporting Issues

Submit via [GitHub Issues](https://github.com/hzhan516/medmemo/issues). Please include:
- Operating system version
- Go / Node.js version
- Reproduction steps
- Log output
- Expected vs. actual behavior

## Feature Requests

Search existing issues first to avoid duplicates. Use the Feature Request template and describe the use case and expected behavior.

## Security Disclosures

For security vulnerabilities, please email **doyle_zhang@outlook.com** instead of opening a public issue. We will respond within 48 hours.

---

## Code of Conduct

Participation in this project means you agree to treat every contributor with professionalism, respect, and inclusivity. Harassment, discrimination, or unfriendly behavior will not be tolerated.

---

*Last updated: 2026-05-19*
