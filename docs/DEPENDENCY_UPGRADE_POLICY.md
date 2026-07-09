# Dependency Upgrade Policy

> 🌐 [中文版本](./i18n/zh-Hans-CN/DEPENDENCY_UPGRADE_POLICY.md)

> This document defines MedMemo's dependency version locking rules and the manual workflow for major-version upgrades.

---

## Purpose

MedMemo relies on a carefully selected technology stack with high-replacement-cost components (e.g., ONNX Runtime bindings, Wails desktop framework). Automated major-version upgrades from Dependabot have repeatedly introduced breaking changes that break CI, destabilize cross-platform builds, or violate compliance-related stack locks documented in `AGENTS.md`.

This policy establishes:
- Which dependencies are **locked** and why.
- How **patch/minor** updates are handled automatically.
- The **manual workflow** required for any major-version upgrade.

---

## Locked Dependency List

### Frontend (npm)

| Dependency | Current Version | Lock Reason |
|-----------|-----------------|-------------|
| `react` | `^18.2.0` | AGENTS.md locks React to 18+; major upgrades affect Wails frontend bindings and UI test suites. |
| `react-dom` | `^18.2.0` | Must stay in sync with `react`. |
| `@types/react` | `^18.2.55` | Must stay in sync with `react`. |
| `@types/react-dom` | `^18.2.19` | Must stay in sync with `react-dom`. |
| `typescript` | `5.9.3` | AGENTS.md locks TypeScript to 5.x+; strict mode and type behavior must not shift unexpectedly. |
| `@typescript-eslint/eslint-plugin` | `^8.62.1` | Tightly coupled to TypeScript version; major bumps may introduce incompatible lint rules. |
| `@typescript-eslint/parser` | `^8.63.0` | Must stay aligned with the ESLint plugin and TypeScript version. |
| `vite` | `6.4.3` | Build toolchain; major upgrades frequently break plugin APIs and cross-platform CGO builds. |
| `@vitejs/plugin-react` | `4.7.0` | Tightly coupled to Vite major version. |
| `tailwindcss` | `^3.4.1` | AGENTS.md locks Tailwind to 3.x+; major upgrades may break utility class generation. |
| `tailwindcss-animate` | `^1.0.7` | Tightly coupled to `tailwindcss` major version. |
| `eslint` | `^8.56.0` | Lint toolchain stability; major upgrades often require config migrations. |
| `eslint-plugin-react-refresh` | `^0.4.5` | Tightly coupled to React and ESLint major versions. |
| `zustand` | `^5.0.13` | State management recently upgraded to v5; major upgrades require store API validation. |
| `react-router-dom` | `^7.18.1` | Routing API changes can break deep-linking and Wails navigation integration. |
| `react-markdown` | `^10.1.0` | Markdown rendering engine; major upgrades may break `remark-gfm` plugin compatibility. |
| `vitest` | `^4.1.10` | Test framework; major upgrades may break test config, coverage reporters, and UI mode. |
| `@vitest/coverage-v8` | `^4.1.10` | Tightly coupled to `vitest` major version. |
| `@vitest/ui` | `^4.1.9` | Tightly coupled to `vitest` major version. |

### Backend (Go Modules)

| Dependency | Current Version | Lock Reason |
|-----------|-----------------|-------------|
| `github.com/knights-analytics/hugot` | `v0.7.4` | **High replacement cost**: ONNX Runtime Go binding. Major upgrades may change session APIs or break int8 quantized model loading. |
| `github.com/daulet/tokenizers` | `v1.27.0` | ONNX tokenizer binding; tightly coupled to `hugot` and ORT version. |
| `github.com/yalue/onnxruntime_go` | `v1.30.1` | Direct ONNX Runtime binding; must stay aligned with `hugot` ORT version. |
| `github.com/wailsapp/wails/v2` | `v2.12.0` | **High replacement cost**: Desktop application framework. Major upgrades affect CGO bindings, frontend bridge, and cross-platform packaging. |

---

## Auto-Update Policy

| Update Type | Policy | Tool |
|-------------|--------|------|
| **Patch** (e.g., `1.0.0` → `1.0.1`) | ✅ Auto-merge after CI passes | Dependabot |
| **Minor** (e.g., `1.0.0` → `1.1.0`) | ✅ Review and merge if CI passes | Dependabot |
| **Major** (e.g., `1.0.0` → `2.0.0`) | ❌ **Blocked** by `dependabot.yml` `ignore` rules | Manual feature branch only |

> **Note**: Go semantic versioning differs from npm. For Go modules at `v0.x`, a bump to `v0.y` is treated as *minor* by Dependabot, while `v0.x` → `v1.0.0` is *major*. The `ignore` rules above target `semver-major` only; `hugot` patch bumps (e.g., `v0.7.4` → `v0.7.5`) are still allowed but should be manually validated for ORT version changes.

---

## Manual Major-Version Upgrade Workflow

Any upgrade that crosses a major version boundary **must** follow this workflow.

### 1. Create a Feature Branch

```bash
git checkout develop
git pull origin develop
git checkout -b feature/upgrade-<dependency>-<new-major>
```

> Do **not** use `hotfix/*` for planned major upgrades. `hotfix` is reserved for urgent production fixes.

### 2. Apply the Upgrade

- Update the version in `package.json` (npm) or `go.mod` (Go).
- Run `npm install` or `go mod tidy` to update lock files.
- Address any immediate compilation or type errors.

### 3. Run Full CI Locally

```bash
# Go backend
make lint
make test

# Frontend
cd web && npm run lint && npm run test
```

### 4. ONNX-Specific Validation (Required for `hugot` / `tokenizers` / `onnxruntime_go`)

If the upgrade touches the ONNX stack:

1. Verify local NER inference still loads the int8 quantized model.
2. Verify embedding generation produces consistent vector dimensions.
3. Run a full end-to-end chat flow with de-identification pipeline enabled.
4. Check cross-platform build (`make build-windows`, `make build-macos`, `make build-linux`) if ORT native libraries changed.

### 5. Cross-Platform Build Check

Major frontend upgrades (React, Vite) must pass:

```bash
# Wails build smoke test
wails build -platform windows/amd64
wails build -platform darwin/amd64
wails build -platform linux/amd64
```

### 6. Code Review & Merge

- Open PR against `develop`.
- PR description must include:
  - Breaking changes summary from upstream release notes.
  - Local validation steps performed (especially ONNX if applicable).
  - Migration notes for any config or API changes.
- Require at least one approval before merge.
- After merging to `develop`, the change will ride the next release train to `main`.

---

## Post-Upgrade Checklist

- [ ] `AGENTS.md` technology stack table updated if version lock changed.
- [ ] `docs/DEPENDENCY_UPGRADE_POLICY.md` updated if new dependencies are added to the locked list.
- [ ] Chinese translation synchronized in `docs/i18n/zh-Hans-CN/DEPENDENCY_UPGRADE_POLICY.md`.
- [ ] `.github/dependabot.yml` updated if new `ignore` rules are needed.
- [ ] All CI checks (Build, Lint, Unit Test, Integration Test, E2E Test, Cross-Platform Build) pass.
- [ ] ONNX inference validated (if applicable).
- [ ] Wails cross-platform build validated (if frontend toolchain changed).

---

## Related Documents

- [`AGENTS.md`](../AGENTS.md) — Full development standards and stack version locks.
- [`.github/dependabot.yml`](../.github/dependabot.yml) — Dependabot configuration with `ignore` rules.
- [`docs/DEVELOPMENT.md`](./DEVELOPMENT.md) — Clean Architecture and coding conventions.

---

*Last updated: 2026-07-09*
