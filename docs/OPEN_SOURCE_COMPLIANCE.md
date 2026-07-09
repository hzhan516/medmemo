# Open Source Compliance Report

> 🌐 [中文版本](./i18n/zh-Hans-CN/OPEN_SOURCE_COMPLIANCE.md)

> **Project**: MedMemo — open-source desktop health information tool
> **Report Date**: 2026-07-09
> **Scope**: Go backend dependencies + Node.js frontend dependencies
> **Primary License**: MIT

---

## Executive Summary

This report records the current dependency inventory from `go.mod` and `web/package-lock.json`. It replaces the older May 2026 snapshot that referenced stale frontend versions such as `typescript@5.6.2` and Tailwind CSS v4.

Current lockfile inventory:

| Ecosystem | Source | Current Inventory |
|-----------|--------|-------------------|
| Go modules | `go.mod` / `go.sum` | 208 dependency modules plus the root module (`go list -m all`, Go 1.26.4) |
| npm packages | `web/package-lock.json` | 521 package entries excluding the root package |

Compliance posture remains conservative:

| Finding | Status |
|---------|:------:|
| Project license is MIT | ✅ |
| No known GPL/AGPL/SSPL dependency introduced by current lockfiles | ✅ |
| High-value dependencies remain permissive-license ecosystems | ✅ |
| Full generated license notices must be refreshed before release | ⚠️ Required |

---

## Current High-Value Dependencies

| Dependency | Version | Role | License Risk |
|------------|---------|------|:------------:|
| `github.com/wailsapp/wails/v2` | `v2.12.0` | Desktop application framework | Low |
| `github.com/knights-analytics/hugot` | `v0.7.4` | ONNX/Hugging Face inference binding | Low |
| `github.com/daulet/tokenizers` | `v1.27.0` | Tokenizer native binding | Low |
| `github.com/yalue/onnxruntime_go` | `v1.30.1` | ONNX Runtime binding | Low |
| `github.com/mutecomm/go-sqlcipher` | pinned in `go.sum` | Encrypted SQLite | Low |
| `github.com/viant/sqlite-vec` | `v0.3.0` | SQLite vector search | Low |
| `react` | `^18.2.0` | Frontend UI | Low |
| `typescript` | `5.9.3` | Frontend type system | Low |
| `tailwindcss` | `^3.4.1` | Styling | Low |
| `vite` | `6.4.3` | Frontend build tool | Low |
| `react-router-dom` | `^7.18.1` | Frontend routing | Low |

---

## Required Pre-Release Scan

Run these commands before any public release and regenerate `THIRD_PARTY_LICENSES.md` if output changes:

```bash
go install github.com/google/go-licenses@latest
go-licenses report ./...

cd web
npx license-checker --start . --json
```

The release owner must verify:

- No GPL, AGPL, SSPL, or proprietary dependency is introduced.
- Apache-2.0, BSD, ISC, MIT, MPL-2.0, CC0, and CC-BY obligations are documented.
- Any MPL-2.0 dependency is unmodified, or modified source files are provided under MPL-2.0.
- `LICENSE` and generated third-party notices are bundled with binary installers.

---

## Distribution Checklist

- [ ] `LICENSE` is included.
- [ ] `THIRD_PARTY_LICENSES.md` is refreshed and included.
- [ ] Go and npm license scans are archived for the release.
- [ ] Dependency versions in this report match `go.mod` and `web/package.json`.
- [ ] Chinese translation remains synchronized.

---

*Last updated: 2026-07-09*
