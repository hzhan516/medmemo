# Open Source Compliance Report

> **Project**: MedMemo — Open-source desktop health information tool
> **Report Date**: 2026-05-19
> **Scope**: Go backend dependencies + Node.js frontend dependencies
> **Primary License**: MIT

---

## Executive Summary

MedMemo has completed a comprehensive open source license compliance audit. The audit covered all direct and transitive dependencies in both the Go backend (37 packages) and the Node.js frontend (462 packages).

**Key Findings**:

| Finding | Status |
|:--------|:------:|
| No GPL/AGPL/SSPL dependencies detected | ✅ Pass |
| All dependency licenses compatible with MIT | ✅ Pass |
| License declaration files complete | ✅ Pass |
| Patent risk assessment — Low | ✅ Pass |
| Document-level license consistency (all docs → MIT) | ✅ Pass |

**Overall Compliance Status**: ✅ **COMPLIANT** — The project is ready for open source distribution under the MIT License with no blocking license conflicts.

---

## 1. Scan Methodology

### 1.1 Tools

| Tool | Version | Scope | Command |
|:-----|:--------|:------|:--------|
| `go-licenses` | v1.6.0 | Go modules | `go-licenses report .` |
| `license-checker` | v25.0.1 | Node.js packages | `npx license-checker --start . --json` |

### 1.2 Scan Coverage

- **Go**: All modules imported via `go.mod` (direct + transitive)
- **Node.js**: All packages in `web/node_modules/` (direct + transitive)
- **Excluded**: `web/node_modules/` dev-only packages are included in the scan but clearly identified

### 1.3 Manual Verification

Packages with automatically undetected licenses were manually inspected:

| Package | Auto-Detected | Manual Finding | Verification Method |
|:--------|:-------------|:---------------|:--------------------|
| `modernc.org/mathutil` | Unknown | BSD-3-Clause | Direct inspection of package license file |

---

## 2. Go Dependency Audit

### 2.1 License Breakdown

| License | Count | Percentage |
|:--------|:-----:|:----------:|
| MIT | 14 | 37.8% |
| Apache-2.0 | 9 | 24.3% |
| BSD-3-Clause | 12 | 32.4% |
| BSD-2-Clause | 2 | 5.4% |
| **Total** | **37** | **100%** |

### 2.2 High-Value Dependencies

| Dependency | License | Role | Risk |
|:-----------|:--------|:-----|:----:|
| `github.com/wailsapp/wails/v2` | MIT | Desktop application framework | 🟢 Low |
| `github.com/knights-analytics/hugot` | Apache-2.0 | ONNX Runtime Go bindings (local AI inference) | 🟢 Low |
| `github.com/gomlx/gomlx` | Apache-2.0 | Go ML framework | 🟢 Low |
| `github.com/google/wire` | Apache-2.0 | Compile-time dependency injection | 🟢 Low |
| `github.com/mutecomm/go-sqlcipher` | MIT | AES-256 encrypted SQLite | 🟢 Low |
| `modernc.org/sqlite` | BSD-3-Clause | Pure-Go SQLite driver | 🟢 Low |
| `github.com/99designs/keyring` | MIT | Cross-platform secret storage | 🟢 Low |

### 2.3 Go Copyleft Assessment

**Result**: ✅ No GPL-family licenses found in the Go dependency tree.

The most restrictive license in the Go tree is **Apache-2.0**, which is permissive and compatible with MIT for distribution. Apache-2.0's explicit patent grant (Section 3) is a risk-reducing feature, not a restriction.

---

## 3. Node.js Dependency Audit

### 3.1 License Breakdown

| License | Count | Percentage | Compatibility |
|:--------|:-----:|:----------:|:-------------:|
| MIT | 396 | 85.7% | ✅ |
| ISC | 26 | 5.6% | ✅ |
| Apache-2.0 | 13 | 2.8% | ✅ |
| BSD-2-Clause | 10 | 2.2% | ✅ |
| BSD-3-Clause | 7 | 1.5% | ✅ |
| MPL-2.0 | 3 | 0.6% | ⚠️ Compatible (file-level copyleft) |
| Other (CC-BY, Python-2.0, BlueOak, CC0) | 6 | 1.3% | ✅ |
| **Total** | **462** | **100%** | — |

### 3.2 Notable Non-MIT Dependencies

#### MPL-2.0: lightningcss

`lightningcss` (v1.32.0) and its platform-specific binaries are used by Tailwind CSS v4 for CSS parsing and transformation. MPL-2.0 is a **file-level copyleft** license:

- ✅ Compatible with MIT for linking and distribution
- ✅ Does not "infect" the entire project
- ⚠️ If MedMemo modifies `lightningcss` source files, those modifications must remain under MPL-2.0
- **Current Status**: MedMemo uses `lightningcss` as an unmodified dependency. No action required.

#### Apache-2.0: TypeScript & ESLint Internals

TypeScript (`typescript@5.6.2`) and ESLint internal packages (`@humanwhocodes/*`) are Apache-2.0 licensed. These are development/build-time dependencies. Apache-2.0 is fully compatible with MIT.

#### CC-BY-4.0: caniuse-lite

`caniuse-lite` provides browser compatibility data under CC-BY-4.0. This is a data/content license, not a software license. Attribution of the data source is sufficient for compliance.

### 3.3 Node.js Copyleft Assessment

**Result**: ✅ No GPL/AGPL/SSPL/LGPL dependencies found in the Node.js dependency tree.

---

## 4. Compatibility Matrix

### 4.1 MIT Project + Dependency Licenses

| Dependency License | Compatibility | Obligations for Distributors |
|:-------------------|:-------------:|:-----------------------------|
| MIT | ✅ Full | Preserve copyright + license text |
| Apache-2.0 | ✅ Full | Preserve copyright + license text + NOTICE files |
| BSD-2-Clause | ✅ Full | Preserve copyright + disclaimer |
| BSD-3-Clause | ✅ Full | Preserve copyright + disclaimer + no endorsement clause |
| ISC | ✅ Full | Preserve copyright + license text |
| MPL-2.0 | ⚠️ Full | Preserve copyright + provide MPL file sources if modified |
| CC-BY-4.0 | ✅ Full | Attribute the data source |
| Python-2.0 | ✅ Full | Preserve copyright + license text |
| BlueOak-1.0.0 | ✅ Full | Preserve copyright + license text |
| CC0-1.0 | ✅ Full | None (public domain dedication) |

### 4.2 Incompatible Licenses (Not Present)

The following licenses are **NOT** present in the dependency tree and would require evaluation if introduced:

| License | Risk Level | Concern |
|:--------|:----------:|:--------|
| GPL-2.0 | 🔴 High | Strong copyleft — entire project may need to be GPL |
| GPL-3.0 | 🔴 High | Strong copyleft + anti-tivoization |
| LGPL-2.1 / LGPL-3.0 | 🟡 Medium | Weak copyleft — linking restrictions apply |
| AGPL-3.0 | 🔴 High | Network service copyleft |
| SSPL | 🔴 High | Source-available, not OSI-approved |
| Proprietary / Commercial | 🟡 Medium | Usage restrictions may apply |

---

## 5. Patent Risk Assessment

### 5.1 Methodology

Patent risk was assessed through:
1. **License-based analysis**: Apache-2.0 dependencies provide explicit patent grants
2. **Public records search**: Web search for patent litigation involving key dependencies
3. **Project governance review**: Organizational backing (Linux Foundation, Google, etc.)

### 5.2 Key Dependencies — Patent Risk

| Dependency | Backing Org | License | Patent Grant | Litigation History | Risk |
|:-----------|:------------|:--------|:------------:|:-------------------|:----:|
| Wails v2 | Community (lead: Lea Anthony) | MIT | ❌ | None | 🟢 Low |
| ONNX Runtime | LF AI & Data (Linux Foundation) | Apache-2.0 | ✅ | None³ | 🟢 Low |
| Hugot | Knights Analytics | Apache-2.0 | ✅ | None | 🟢 Low |
| GoMLX | Community (lead: Jan Pfeifer) | Apache-2.0 | ✅ | None | 🟢 Low |
| Google Wire | Google | Apache-2.0 | ✅ | None | 🟢 Low |
| React | Meta | MIT | ❌⁴ | None | 🟢 Low |
| Tailwind CSS | Tailwind Labs | MIT | ❌ | None | 🟢 Low |
| Vite | Vite Team / Evan You | MIT | ❌ | None | 🟢 Low |
| SQLite / SQLCipher | SQLite Consortium / Community | Public Domain / MIT | ❌ | None | 🟢 Low |

> ³ A 2024 case involved third-party phishing kits misusing the "ONNX" trademark. This was a **trademark dispute**, not patent litigation. No technology patents are asserted against ONNX Runtime.
> ⁴ Facebook's React patent clause (2015–2017) was removed in September 2017 when React re-licensed under MIT.

### 5.3 Risk Mitigation

| Risk Factor | Mitigation |
|:------------|:-----------|
| Non-contributor patent holders | Apache-2.0 components provide contributor patent grants; no guarantee against non-contributors |
| Future patent assertions | Monitor dependency security advisories; maintain upgrade path |
| Jurisdiction differences | MIT and Apache-2.0 are globally recognized; no known jurisdiction-specific risks |

### 5.4 Overall Patent Risk

**🟢 LOW** — No known patent controversies exist among MedMemo's dependencies. The presence of Apache-2.0 licensed core components (ONNX Runtime, Hugot, GoMLX, Wire) provides explicit patent grants, which is a risk-reducing factor.

---

## 6. Document-Level License Consistency

All project documentation was scanned for license declarations to ensure consistency with the project's MIT License.

### 6.1 Scan Results

| File | License Declaration | Status |
|:-----|:--------------------|:------:|
| `LICENSE` | MIT | ✅ |
| `README.md` | MIT | ✅ |
| `docs/i18n/zh-Hans-CN/README.md` | MIT | ✅ |
| `CONTRIBUTING.md` | — | ✅ (no conflicting declaration) |
| `AGENTS.md` | — | ✅ (no conflicting declaration) |
| All internal `README.md` files | — | ✅ (no conflicting declaration) |
| All `docs/user-guide/*.md` | — | ✅ (no conflicting declaration) |

### 6.2 Action Taken

- **No modifications required** — All existing license declarations are already consistent with MIT.
- No document incorrectly declares Apache-2.0, GPL, or any other license as the project license.

---

## 7. Compliance Checklist

| # | Requirement | Status | Evidence |
|:-:|:------------|:------:|:---------|
| 1 | Dependency license scan completed | ✅ | `go-licenses` + `license-checker` reports |
| 2 | All licenses compatible with MIT | ✅ | Compatibility matrix (Section 4) |
| 3 | `LICENSE` file present and correct | ✅ | `LICENSE` (MIT) verified |
| 4 | `THIRD_PARTY_LICENSES.md` generated | ✅ | See `THIRD_PARTY_LICENSES.md` |
| 5 | No GPL/AGPL/SSPL dependencies | ✅ | Copyleft assessment (Sections 2.3, 3.3) |
| 6 | MPL-2.0 dependencies assessed | ✅ | `lightningcss` — file-level copyleft, no action required |
| 7 | Patent risk assessed | ✅ | Low risk, no known controversies |
| 8 | Document-level license consistency verified | ✅ | All docs declare or are consistent with MIT |
| 9 | Scan methodology documented | ✅ | This report (Section 1) |
| 10 | Attribution requirements documented | ✅ | `THIRD_PARTY_LICENSES.md` Section "Attribution Requirements" |

---

## 8. Recommendations

### 8.1 Ongoing Compliance

1. **Pre-release scan**: Run `go-licenses` and `license-checker` before each release to catch new dependency license changes.
2. **Dependency pin review**: When upgrading dependencies, verify the new version's license has not changed.
3. **CI integration**: Consider adding a license check step to the GitHub Actions CI pipeline (e.g., `fossa-cli` or `go-licenses` in CI).

### 8.2 Risk Monitoring

1. **Security advisories**: Subscribe to security advisories for key dependencies (Wails, ONNX Runtime, SQLite).
2. **License change alerts**: Monitor dependencies for license changes (e.g., `elastic/elasticsearch` changed from Apache-2.0 to SSPL in 2021).

### 8.3 Distribution Compliance

When distributing MedMemo binaries or source code:

1. Include `LICENSE` and `THIRD_PARTY_LICENSES.md` in the distribution
2. For installer packages (`.exe`, `.dmg`, `.AppImage`), embed or bundle these files
3. Preserve Apache-2.0 NOTICE files if any dependency includes them

---

*Report generated: 2026-05-19*
*Tools: go-licenses v1.6.0, license-checker v25.0.1*
*Auditor: Development Team*
