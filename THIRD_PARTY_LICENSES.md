# Third-Party Licenses

> 🌐 [中文版本](./docs/i18n/zh-Hans-CN/THIRD_PARTY_LICENSES.md)

> **Project**: MedMemo — Open-source desktop health information tool
> **Project License**: [MIT License](./LICENSE)
> **Last updated**: 2026-07-14

---

## Overview

MedMemo is licensed under the MIT License. This document lists all third-party dependencies used by the project, along with their respective licenses and compatibility assessment against the MIT License.

### Compatibility Legend

| Symbol | Meaning |
|:------:|:--------|
| ✅ | Fully compatible with MIT — no additional obligations |
| ⚠️ | Compatible with MIT — minor obligations (attribution, file-level copyleft) |
| ❓ | License not automatically detected — manually verified compatible |

### Summary Statistics

| Ecosystem | Total Dependencies | MIT | Apache-2.0 | BSD-* | ISC | MPL-2.0 | Other |
|:----------|:------------------:|:---:|:----------:|:-----:|:---:|:-------:|:-----:|
| Go | 37 | 15 | 11 | 14 | 0 | 0 | 0 |
| Node.js | 462 | 396 | 13 | 17 | 26 | 3 | 7 |

**Overall Assessment**: ✅ All dependencies are compatible with the MIT License. No GPL/AGPL/SSPL dependencies detected.

---

## Go Dependencies

### MIT License

| Package | Version | License URL | Compatibility |
|:--------|:--------|:------------|:-------------:|
| github.com/99designs/keyring | v1.2.2 | [MIT](https://github.com/99designs/keyring/blob/v1.2.2/LICENSE) | ✅ |
| github.com/dustin/go-humanize | v1.0.1 | [MIT](https://github.com/dustin/go-humanize/blob/v1.0.1/LICENSE) | ✅ |
| github.com/dvsekhvalnov/jose2go | v1.5.0 | [MIT](https://github.com/dvsekhvalnov/jose2go/blob/v1.5.0/LICENSE) | ✅ |
| github.com/go-errors/errors | v1.5.1 | [MIT](https://github.com/go-errors/errors/blob/v1.5.1/LICENSE.MIT) | ✅ |
| github.com/gsterjov/go-libsecret | — | [MIT](https://github.com/gsterjov/go-libsecret/blob/a6f4afe4910c/LICENSE) | ✅ |
| github.com/leaanthony/go-ansi-parser | v1.6.1 | [MIT](https://github.com/leaanthony/go-ansi-parser/blob/v1.6.1/LICENSE) | ✅ |
| github.com/leaanthony/slicer | v1.6.0 | [MIT](https://github.com/leaanthony/slicer/blob/v1.6.0/LICENSE) | ✅ |
| github.com/leaanthony/u | v1.1.1 | [MIT](https://github.com/leaanthony/u/blob/v1.1.1/LICENSE) | ✅ |
| github.com/mtibben/percent | v0.2.1 | [MIT](https://github.com/mtibben/percent/blob/v0.2.1/LICENSE) | ✅ |
| github.com/mutecomm/go-sqlcipher | — | [MIT](https://github.com/mutecomm/go-sqlcipher/blob/55dbde17881f/LICENSE) | ✅ |
| github.com/rivo/uniseg | v0.4.7 | [MIT](https://github.com/rivo/uniseg/blob/v0.4.7/LICENSE.txt) | ✅ |
| github.com/stretchr/testify | v1.11.1 | [MIT](https://github.com/stretchr/testify/blob/v1.11.1/LICENSE) | ✅ |
| github.com/wailsapp/wails/v2 | v2.12.0 | [MIT](https://github.com/wailsapp/wails/blob/v2.12.0/v2/LICENSE) | ✅ |
| github.com/x448/float16 | v0.8.4 | [MIT](https://github.com/x448/float16/blob/v0.8.4/LICENSE) | ✅ |
| gopkg.in/yaml.v3 | v3.0.1 | [MIT](https://github.com/go-yaml/yaml/blob/v3.0.1/LICENSE) | ✅ |

### Apache-2.0 License

| Package | Version | License URL | Compatibility |
|:--------|:--------|:------------|:-------------:|
| github.com/daulet/tokenizers | v1.27.0 | [Apache-2.0](https://github.com/daulet/tokenizers/blob/v1.27.0/LICENSE) | ✅ |
| github.com/go-logr/logr | v1.4.3 | [Apache-2.0](https://github.com/go-logr/logr/blob/v1.4.3/LICENSE) | ✅ |
| github.com/gomlx/exceptions | v0.0.3 | [Apache-2.0](https://github.com/gomlx/exceptions/blob/v0.0.3/LICENSE) | ✅ |
| github.com/gomlx/go-huggingface | v0.3.5 | [Apache-2.0](https://github.com/gomlx/go-huggingface/blob/v0.3.5/LICENSE) | ✅ |
| github.com/gomlx/gomlx | v0.27.3 | [Apache-2.0](https://github.com/gomlx/gomlx/blob/v0.27.3/LICENSE) | ✅ |
| github.com/gomlx/onnx-gomlx | v0.4.2 | [Apache-2.0](https://github.com/gomlx/onnx-gomlx/blob/v0.4.2/LICENSE) | ✅ |
| github.com/google/wire | v0.7.0 | [Apache-2.0](https://github.com/google/wire/blob/v0.7.0/LICENSE) | ✅ |
| github.com/knights-analytics/hugot | v0.7.4 | [Apache-2.0](https://github.com/knights-analytics/hugot/blob/v0.7.4/LICENSE) | ✅ |
| github.com/viant/afs | v1.30.0 | [Apache-2.0](https://github.com/viant/afs/blob/v1.30.0/LICENSE.txt) | ✅ |
| github.com/viant/sqlite-vec | v0.3.0 | [Apache-2.0](https://github.com/viant/sqlite-vec/blob/v0.3.0/LICENSE) | ✅ |
| k8s.io/klog/v2 | v2.140.0 | [Apache-2.0](https://github.com/kubernetes/klog/blob/v2.140.0/LICENSE) | ✅ |

> **Note**: Apache-2.0 is compatible with MIT for distribution. Both licenses require preservation of copyright notices and license texts. No additional copyleft obligations apply.

### BSD License

| Package | Version | License | License URL | Compatibility |
|:--------|:--------|:--------|:------------|:-------------:|
| github.com/godbus/dbus | — | BSD-2-Clause | [LICENSE](https://github.com/godbus/dbus/blob/4481cbc300e2/LICENSE) | ✅ |
| github.com/gofrs/flock | v0.13.0 | BSD-3-Clause | [LICENSE](https://github.com/gofrs/flock/blob/v0.13.0/LICENSE) | ✅ |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause | [LICENSE](https://github.com/google/uuid/blob/v1.6.0/LICENSE) | ✅ |
| github.com/pkg/errors | v0.9.1 | BSD-2-Clause | [LICENSE](https://github.com/pkg/errors/blob/v0.9.1/LICENSE) | ✅ |
| github.com/remyoudompheng/bigfft | — | BSD-3-Clause | [LICENSE](https://github.com/remyoudompheng/bigfft/blob/24d4a6f8daec/LICENSE) | ✅ |
| golang.org/x/crypto | v0.50.0 | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/crypto/+/v0.50.0:LICENSE) | ✅ |
| golang.org/x/exp/constraints | — | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/exp/+/746e56fc:LICENSE) | ✅ |
| golang.org/x/image | v0.39.0 | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/image/+/v0.39.0:LICENSE) | ✅ |
| golang.org/x/net/context | v0.52.0 | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/net/+/v0.52.0:LICENSE) | ✅ |
| golang.org/x/sync/errgroup | v0.20.0 | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/sync/+/v0.20.0:LICENSE) | ✅ |
| golang.org/x/sys/unix | v0.43.0 | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/sys/+/v0.43.0:LICENSE) | ✅ |
| golang.org/x/term | v0.42.0 | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/term/+/v0.42.0:LICENSE) | ✅ |
| golang.org/x/text | v0.36.0 | BSD-3-Clause | [LICENSE](https://cs.opensource.google/go/x/text/+/v0.36.0:LICENSE) | ✅ |
| google.golang.org/protobuf | v1.36.11 | BSD-3-Clause | [LICENSE](https://github.com/protocolbuffers/protobuf-go/blob/v1.36.11/LICENSE) | ✅ |
| modernc.org/libc | v1.72.3 | MIT* | [LICENSE-3RD-PARTY.md](https://gitlab.com/cznic/libc/blob/v1.72.3/LICENSE-3RD-PARTY.md) | ✅ |
| modernc.org/mathutil | v1.7.1 | BSD-3-Clause* | Manually verified | ✅ |
| modernc.org/memory | v1.11.0 | BSD-3-Clause | [LICENSE-GO](https://gitlab.com/cznic/memory/blob/v1.11.0/LICENSE-GO) | ✅ |
| modernc.org/sqlite | v1.53.0 | BSD-3-Clause | [LICENSE](https://gitlab.com/cznic/sqlite/blob/v1.53.0/LICENSE) | ✅ |

> **Note**: BSD licenses (2-Clause and 3-Clause) are permissive and fully compatible with MIT. The only additional requirement is preservation of copyright notices and disclaimer text.
> \* `modernc.org/mathutil` was not automatically detected by `go-licenses`, but its license file was manually verified to be BSD-3-Clause.

---

## Node.js Dependencies (Frontend)

### Top-Level Production Dependencies

The frontend (`web/`) uses the following top-level production dependencies:

| Package | Version | License | Compatibility |
|:--------|:--------|:--------|:-------------:|
| react | ^18.3.1 | MIT | ✅ |
| react-dom | ^18.3.1 | MIT | ✅ |
| typescript | 5.9.3 | Apache-2.0 | ✅ |
| tailwindcss | ^3.4.1 | MIT | ✅ |
| vite | ^6.3.4 | MIT | ✅ |
| @types/react | ^18.3.12 | MIT | ✅ |
| @types/react-dom | ^18.3.1 | MIT | ✅ |
| @vitejs/plugin-react | ^4.3.5 | MIT | ✅ |

### License Distribution (All 462 packages)

| License | Count | Compatibility Assessment |
|:--------|:-----:|:-------------------------|
| MIT | 396 | ✅ Fully compatible |
| ISC | 26 | ✅ Fully compatible |
| Apache-2.0 | 13 | ✅ Compatible (patent grant + attribution) |
| BSD-2-Clause | 10 | ✅ Fully compatible |
| BSD-3-Clause | 7 | ✅ Fully compatible |
| MPL-2.0 | 3 | ⚠️ Compatible (file-level copyleft, source disclosure for MPL files) |
| MIT-0 | 2 | ✅ Fully compatible (MIT without attribution) |
| CC-BY-4.0 | 1 | ✅ Compatible (data/content license) |
| Python-2.0 | 1 | ✅ Compatible |
| BlueOak-1.0.0 | 1 | ✅ Compatible (permissive, MIT-like) |
| CC0-1.0 | 1 | ✅ Fully compatible (public domain dedication) |
| (MIT OR CC0-1.0) | 1 | ✅ Dual-licensed, both compatible |

### Notable Non-MIT Dependencies

#### MPL-2.0 — lightningcss

| Package | Version | Notes |
|:--------|:--------|:------|
| lightningcss | 1.32.0 | CSS parser and transformer used by Tailwind CSS v3 |
| lightningcss-linux-x64-gnu | 1.32.0 | Platform-specific native binary |
| lightningcss-linux-x64-musl | 1.32.0 | Platform-specific native binary |

> **Assessment**: MPL-2.0 is compatible with MIT. It is a file-level copyleft license — modifications to MPL-licensed files must remain under MPL-2.0, but the license does not "infect" the entire project. No action required for MedMemo as we do not modify lightningcss source code.

#### Apache-2.0 — Key Dependencies

| Package | Version | Role |
|:--------|:--------|:-----|
| @humanwhocodes/config-array | 0.13.0 | ESLint internal dependency |
| @humanwhocodes/module-importer | 1.0.1 | ESLint internal dependency |
| typescript | 5.9.3 | TypeScript compiler |

> **Assessment**: Apache-2.0 provides an explicit patent grant (Section 3), which is beneficial for downstream users. Compatible with MIT for distribution. Requires preservation of NOTICE files if present.

#### Other Licenses

| Package | Version | License | Notes |
|:--------|:--------|:--------|:------|
| argparse | 2.0.1 | Python-2.0 | CLI argument parsing utility; Python-2.0 is OSI-approved and MIT-compatible |
| caniuse-lite | 1.0.30001793 | CC-BY-4.0 | Browser compatibility data; CC-BY-4.0 requires attribution for the data set |
| lru-cache | 11.4.0 | BlueOak-1.0.0 | Cache utility; BlueOak is a modern permissive license, MIT-compatible |

---

## Copyleft / GPL Assessment

### Result: ✅ No GPL/AGPL/SSPL Dependencies Detected

| License Type | Go | Node.js | Action Required |
|:-------------|:--:|:-------:|:----------------|
| GPL-2.0 | 0 | 0 | None |
| GPL-3.0 | 0 | 0 | None |
| LGPL-2.1 | 0 | 0 | None |
| LGPL-3.0 | 0 | 0 | None |
| AGPL-3.0 | 0 | 0 | None |
| SSPL | 0 | 0 | None |
| MPL-2.0 | 0 | 3 | None (file-level copyleft, compatible) |

### Implications

Since no GPL-family licenses are present in the dependency tree:

- **No source code disclosure obligations** are triggered beyond MIT/Apache/BSD attribution requirements.
- **No linking restrictions** apply — the project can be statically or dynamically linked without concern.
- **No copyleft "infection"** risk exists for proprietary downstream usage.

---

## Patent Risk Assessment

### Methodology

Patent risk was assessed via:
1. Review of license terms for explicit patent grants (Apache-2.0 provides Section 3 patent grants)
2. Web search for known patent litigation involving key dependencies
3. Review of project governance (LF AI & Data Foundation for ONNX, Google for Wire, etc.)

### Key Dependencies Assessment

| Dependency | License | Patent Grant | Known Controversies | Risk Level |
|:-----------|:--------|:------------:|:--------------------|:----------:|
| Wails v2 | MIT | ❌ None | None found | 🟢 Low |
| ONNX Runtime (via Hugot) | Apache-2.0 | ✅ Section 3 | None (trademark dispute only¹) | 🟢 Low |
| Hugot | Apache-2.0 | ✅ Section 3 | None found | 🟢 Low |
| GoMLX | Apache-2.0 | ✅ Section 3 | None found | 🟢 Low |
| Google Wire | Apache-2.0 | ✅ Section 3 | None found | 🟢 Low |
| modernc.org/sqlite | BSD-3-Clause | ❌ None | None found | 🟢 Low |
| React | MIT | ❌ None | None (React patent clause retired in 2017²) | 🟢 Low |
| Tailwind CSS | MIT | ❌ None | None found | 🟢 Low |
| Vite | MIT | ❌ None | None found | 🟢 Low |

> ¹ A 2024 trademark dispute involved third parties misusing the "ONNX" brand for phishing kits. This is a **trademark infringement issue**, not a technology patent dispute. No patent litigation involving ONNX Runtime has been identified.
> ² Facebook's React patent clause (2015–2017) was removed when React re-licensed under MIT in September 2017.

### Overall Patent Risk Conclusion

**🟢 Low Risk** — No known patent controversies exist among MedMemo's key dependencies. The presence of Apache-2.0 licensed components (Hugot, GoMLX, Wire, ONNX Runtime) provides explicit patent grants from contributors, which reduces risk for downstream users.

---

## Attribution Requirements

To comply with all third-party licenses, distributions of MedMemo must include:

1. **This file** (`THIRD_PARTY_LICENSES.md`) — summarizing all dependencies and their licenses
2. **The project `LICENSE` file** — MedMemo's MIT License
3. **Dependency license texts** — for Apache-2.0 dependencies, preserve any NOTICE files

The Wails build process (`wails build`) embeds frontend assets into the Go binary. The compiled binary itself is a "substantial portion" under MIT terms, so binary distributions must retain the LICENSE and THIRD_PARTY_LICENSES files alongside the binary or within the installer package.

---

## Tools Used

| Tool | Version | Purpose |
|:-----|:--------|:--------|
| `go-licenses` | v1.6.0 | Go module dependency license scanning |
| `license-checker` | v25.0.1 | Node.js dependency license scanning |

### Scan Commands

```bash
# Go dependencies
go-licenses report .

# Node.js dependencies
cd web && npx license-checker --start . --json
```

---

*This document was generated automatically by license scanning tools and manually reviewed for accuracy. Last updated: 2026-07-14.*
