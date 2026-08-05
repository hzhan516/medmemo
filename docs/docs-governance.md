# Documentation Governance

> 🌐 [中文版本](./i18n/zh-Hans-CN/docs-governance.md)

This document defines how MedMemo keeps its documentation accurate, consistent, and discoverable. It records the **single source of truth** for each class of information, the audit checks that guard against drift, and the process for closing documentation gaps.

---

## 1. Single Source of Truth

To prevent the "one change, multiple stale copies" problem, every major category of information has exactly one authoritative location. Other locations must link to it instead of duplicating it.

| Information Category | Single Source of Truth | Mirrors / Pointers |
|----------------------|------------------------|--------------------|
| Product version | `wails.json` → `info.productVersion` | `web/package.json`, `web/package-lock.json`, `internal/domain/entity/changelog/zh-Hans.json` |
| Layer READMEs (domain/application/adapters/infrastructure) | `internal/*/README.md` | `docs/internal/*/README.md` (pointer only) |
| API type/interface/binding reference | `pkg/models/*.go`, `internal/application/port/*.go`, `wails_app_*.go` | `docs/api/_generated/*.md` (auto-generated) |
| Architecture overview | `docs/ARCHITECTURE.md` | `.skill/medmemo/reference-architecture.md` (links to it) |
| API module guide | `docs/API.md` and `docs/api/*.md` | `.skill/medmemo/reference-modules.md` (links to it) |
| Third-party licenses | `THIRD_PARTY_LICENSES.md` (auto-generated) | Generated from `go.mod` / `web/package.json` |
| Compliance / privacy red lines | `AGENTS.md` § Compliance | `docs/COMPLIANCE.md` (expanded reference) |

Rules:
- Edit the authoritative file; never edit a generated file by hand.
- If you add a new duplicated category, either consolidate it or add it to this table.
- Pointer documents must not contain substantive content that can drift from the source.

---

## 2. Documentation Checks

All checks run in CI on every push/PR (`.github/workflows/ci.yml`) and are repeated by the monthly scheduled audit (`.github/workflows/docs-audit.yml`).

| Check | Tool / Script | Guards Against |
|-------|---------------|----------------|
| Broken Markdown links | `lychee` via `scripts/check-doc-links.sh` | Dead internal or external links |
| Missing Chinese mirrors | `scripts/check-doc-mirrors.js` | English docs without `docs/i18n/zh-Hans-CN/` counterparts |
| Terminology deviations | `scripts/check-terminology.js` | Inconsistent translations or project terms |
| Version drift | `scripts/check-version-consistency.js` | `wails.json` / `web/package.json` / docs out of sync |
| API doc drift | `make docs` + `git diff --exit-code docs/api/_generated/` | Source code changed without regenerating API docs |
| License drift | `make licenses` + `git diff --exit-code THIRD_PARTY_LICENSES.md` | Dependency changes without updating license list |

Local equivalents:

```bash
make docs-check   # link, mirror, terminology, version
make docs         # regenerate API docs
make licenses     # regenerate third-party licenses
```

---

## 3. Health Metrics

The scheduled audit produces a `docs-health-report.md` artifact with the following trendable metrics:

| Metric | Source | Target |
|--------|--------|--------|
| Broken link count | `scripts/check-doc-links.sh` | 0 |
| Missing Chinese mirror count | `scripts/check-doc-mirrors.js` | 0 |
| Terminology deviation count | `scripts/check-terminology.js` | 0 |
| Version drift count | `scripts/check-version-consistency.js` | 0 |
| API doc drift | `git diff --exit-code docs/api/_generated/` | No diff |
| License list drift | `git diff --exit-code THIRD_PARTY_LICENSES.md` | No diff |

A non-zero metric or any drift is treated as a documentation health failure and should be tracked to closure.

---

## 4. Issue Severity Classification (P0–P3)

Documentation gaps are classified by impact and urgency. Use these labels when opening or updating `medmemo/开发日志/issues.md`.

| Level | Definition | Examples | Response Time |
|-------|------------|----------|---------------|
| **P0** | Release blocker or user-facing incorrect instruction | Wrong build command, broken install steps, license omission | Fix immediately |
| **P1** | Factual inconsistency that misleads contributors | Outdated dependency version, phantom directory listed | Fix within the current sprint |
| **P2** | Incomplete or hard-to-maintain documentation | Missing API field table, duplicated content | Fix within the next release |
| **P3** | Process / governance / automation debt | Missing scheduled audit, no single-source table | Plan and implement via approved plan |

---

## 5. Closure Process

1. **Open**: Add a row to `medmemo/开发日志/issues.md` with the next global issue number, severity, and description.
2. **Fix**: Edit the authoritative source; regenerate any derived artifacts; run `make docs-check`, `make docs`, and `make licenses`.
3. **Verify**: Confirm CI passes and, for scheduled-audit findings, that the latest audit artifact shows the metric at target.
4. **Close**: Update the issue row in `medmemo/开发日志/issues.md`: set **完成** to `✅` and **状态** to `closed`.

---

## 6. Related Documents

- [`docs/ARCHITECTURE.md`](./ARCHITECTURE.md) — System architecture
- [`docs/API.md`](./API.md) — API reference entry point
- [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) — Per-PR documentation checks
- [`.github/workflows/docs-audit.yml`](../.github/workflows/docs-audit.yml) — Monthly scheduled audit

---

*Last updated: 2026-07-14*
