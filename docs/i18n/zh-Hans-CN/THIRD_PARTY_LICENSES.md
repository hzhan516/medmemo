# 第三方许可证

> 🌐 [English Version](../../../THIRD_PARTY_LICENSES.md)

> **项目**：MedMemo — 开源桌面端健康信息工具
> **项目许可证**：[MIT License](../../../LICENSE)
> **最后更新**：2026-05-19

---

## 概览

MedMemo 采用 MIT 许可证。本文档列出项目使用的所有第三方依赖及其 respective 许可证，并评估其与 MIT 许可证的兼容性。

### 兼容性图例

| 符号 | 含义 |
|:----:|:-----|
| ✅ | 与 MIT 完全兼容 — 无额外义务 |
| ⚠️ | 与 MIT 兼容 — 有轻微义务（署名、文件级 copyleft） |
| ❓ | 许可证未自动检测 — 经人工确认为兼容 |

### 汇总统计

| 生态系统 | 依赖总数 | MIT | Apache-2.0 | BSD-* | ISC | MPL-2.0 | 其他 |
|:---------|:--------:|:---:|:----------:|:-----:|:---:|:-------:|:----:|
| Go | 37 | 14 | 9 | 14 | 0 | 0 | 0 |
| Node.js | 462 | 396 | 13 | 17 | 26 | 3 | 7 |

**总体评估**：✅ 所有依赖均与 MIT 许可证兼容。未检测到 GPL/AGPL/SSPL 依赖。

---

## Go 依赖

### MIT 许可证

| 包名 | 版本 | 许可证链接 | 兼容性 |
|:-----|:-----|:-----------|:------:|
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
| github.com/wailsapp/wails/v2 | v2.12.0 | [MIT](https://github.com/wailsapp/wails/blob/v2.12.0/v2/LICENSE) | ✅ |
| github.com/x448/float16 | v0.8.4 | [MIT](https://github.com/x448/float16/blob/v0.8.4/LICENSE) | ✅ |
| gopkg.in/yaml.v3 | v3.0.1 | [MIT](https://github.com/go-yaml/yaml/blob/v3.0.1/LICENSE) | ✅ |

### Apache-2.0 许可证

| 包名 | 版本 | 许可证链接 | 兼容性 |
|:-----|:-----|:-----------|:------:|
| github.com/go-logr/logr | v1.4.3 | [Apache-2.0](https://github.com/go-logr/logr/blob/v1.4.3/LICENSE) | ✅ |
| github.com/gomlx/exceptions | v0.0.3 | [Apache-2.0](https://github.com/gomlx/exceptions/blob/v0.0.3/LICENSE) | ✅ |
| github.com/gomlx/go-huggingface | v0.3.5 | [Apache-2.0](https://github.com/gomlx/go-huggingface/blob/v0.3.5/LICENSE) | ✅ |
| github.com/gomlx/gomlx | v0.27.3 | [Apache-2.0](https://github.com/gomlx/gomlx/blob/v0.27.3/LICENSE) | ✅ |
| github.com/gomlx/onnx-gomlx | v0.4.2 | [Apache-2.0](https://github.com/gomlx/onnx-gomlx/blob/v0.4.2/LICENSE) | ✅ |
| github.com/google/wire | v0.7.0 | [Apache-2.0](https://github.com/google/wire/blob/v0.7.0/LICENSE) | ✅ |
| github.com/knights-analytics/hugot | v0.7.2 | [Apache-2.0](https://github.com/knights-analytics/hugot/blob/v0.7.2/LICENSE) | ✅ |
| github.com/viant/afs | v1.30.0 | [Apache-2.0](https://github.com/viant/afs/blob/v1.30.0/LICENSE.txt) | ✅ |
| k8s.io/klog/v2 | v2.140.0 | [Apache-2.0](https://github.com/kubernetes/klog/blob/v2.140.0/LICENSE) | ✅ |

> **说明**：Apache-2.0 与 MIT 分发兼容。两种许可证均要求保留版权声明和许可证文本。无额外 copyleft 义务。

### BSD 许可证

| 包名 | 版本 | 许可证 | 许可证链接 | 兼容性 |
|:-----|:-----|:-------|:-----------|:------:|
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
| modernc.org/mathutil | v1.7.1 | BSD-3-Clause* | 人工确认 | ✅ |
| modernc.org/memory | v1.11.0 | BSD-3-Clause | [LICENSE-GO](https://gitlab.com/cznic/memory/blob/v1.11.0/LICENSE-GO) | ✅ |
| modernc.org/sqlite | v1.50.1 | BSD-3-Clause | [LICENSE](https://gitlab.com/cznic/sqlite/blob/v1.50.1/LICENSE) | ✅ |

> **说明**：BSD 许可证（2-Clause 和 3-Clause）为宽松许可证，与 MIT 完全兼容。唯一额外要求是保留版权声明和免责声明文本。
>
> \* `modernc.org/mathutil` 未被 `go-licenses` 自动检测，但其包内许可证文件经人工检查确认为 BSD-3-Clause。

---

## Node.js 依赖（前端）

### 顶层生产依赖

前端（`web/`）使用以下顶层生产依赖：

| 包名 | 版本 | 许可证 | 兼容性 |
|:-----|:-----|:-------|:------:|
| react | ^18.3.1 | MIT | ✅ |
| react-dom | ^18.3.1 | MIT | ✅ |
| typescript | ^5.6.2 | Apache-2.0 | ✅ |
| tailwindcss | ^4.1.3 | MIT | ✅ |
| vite | ^6.3.4 | MIT | ✅ |
| @tailwindcss/vite | ^4.1.3 | MIT | ✅ |
| @types/react | ^18.3.12 | MIT | ✅ |
| @types/react-dom | ^18.3.1 | MIT | ✅ |
| @vitejs/plugin-react | ^4.3.5 | MIT | ✅ |

### 许可证分布（全部 462 个包）

| 许可证 | 数量 | 兼容性评估 |
|:-------|:----:|:-----------|
| MIT | 396 | ✅ 完全兼容 |
| ISC | 26 | ✅ 完全兼容 |
| Apache-2.0 | 13 | ✅ 兼容（专利授权 + 署名） |
| BSD-2-Clause | 10 | ✅ 完全兼容 |
| BSD-3-Clause | 7 | ✅ 完全兼容 |
| MPL-2.0 | 3 | ⚠️ 兼容（文件级 copyleft，MPL 文件需开源） |
| MIT-0 | 2 | ✅ 完全兼容（MIT 无需署名） |
| CC-BY-4.0 | 1 | ✅ 兼容（数据/内容许可证） |
| Python-2.0 | 1 | ✅ 兼容 |
| BlueOak-1.0.0 | 1 | ✅ 兼容（宽松许可证，类 MIT） |
| CC0-1.0 | 1 | ✅ 完全兼容（公有领域 dedication） |
| (MIT OR CC0-1.0) | 1 | ✅ 双许可证，两者均兼容 |

### 值得关注的非 MIT 依赖

#### MPL-2.0 — lightningcss

| 包名 | 版本 | 说明 |
|:-----|:-----|:-----|
| lightningcss | 1.32.0 | Tailwind CSS v4 使用的 CSS 解析器和转换器 |
| lightningcss-linux-x64-gnu | 1.32.0 | 平台特定原生二进制 |
| lightningcss-linux-x64-musl | 1.32.0 | 平台特定原生二进制 |

> **评估**：MPL-2.0 与 MIT 兼容。它是文件级 copyleft 许可证 — MPL 许可文件的修改必须保留在 MPL-2.0 下，但不会"感染"整个项目。MedMemo 未修改 lightningcss 源代码，无需额外操作。

#### Apache-2.0 — 关键依赖

| 包名 | 版本 | 用途 |
|:-----|:-----|:-----|
| @humanwhocodes/config-array | 0.13.0 | ESLint 内部依赖 |
| @humanwhocodes/module-importer | 1.0.1 | ESLint 内部依赖 |
| typescript | 5.6.2 | TypeScript 编译器 |

> **评估**：Apache-2.0 提供明确的专利授权（第 3 条），对下游用户有利。与 MIT 分发兼容。如有 NOTICE 文件需保留。

#### 其他许可证

| 包名 | 版本 | 许可证 | 说明 |
|:-----|:-----|:-------|:-----|
| argparse | 2.0.1 | Python-2.0 | CLI 参数解析工具；Python-2.0 经 OSI 批准，与 MIT 兼容 |
| caniuse-lite | 1.0.30001793 | CC-BY-4.0 | 浏览器兼容性数据；CC-BY-4.0 要求数据集署名 |
| lru-cache | 11.4.0 | BlueOak-1.0.0 | 缓存工具；BlueOak 为现代宽松许可证，与 MIT 兼容 |

---

## Copyleft / GPL 评估

### 结果：✅ 未检测到 GPL/AGPL/SSPL 依赖

| 许可证类型 | Go | Node.js | 需采取的操作 |
|:-----------|:--:|:-------:|:-------------|
| GPL-2.0 | 0 | 0 | 无 |
| GPL-3.0 | 0 | 0 | 无 |
| LGPL-2.1 | 0 | 0 | 无 |
| LGPL-3.0 | 0 | 0 | 无 |
| AGPL-3.0 | 0 | 0 | 无 |
| SSPL | 0 | 0 | 无 |
| MPL-2.0 | 0 | 3 | 无（文件级 copyleft，兼容） |

### 影响

由于依赖树中不存在 GPL 家族许可证：

- **无源代码披露义务**被触发，超出 MIT/Apache/BSD 的署名要求。
- **无链接限制**适用 — 项目可静态或动态链接而无需担忧。
- **无 copyleft "感染"风险**存在于专有下游使用。

---

## 专利风险评估

### 方法论

专利风险通过以下方式评估：
1. 审查许可证条款中的明确专利授权（Apache-2.0 提供第 3 条专利授权）
2. 网络搜索关键依赖的已知专利诉讼
3. 审查项目治理（ONNX 由 LF AI & Data Foundation 管理，Wire 由 Google 管理等）

### 关键依赖评估

| 依赖 | 许可证 | 专利授权 | 已知争议 | 风险等级 |
|:-----|:-------|:--------:|:---------|:--------:|
| Wails v2 | MIT | ❌ 无 | 未发现 | 🟢 低 |
| ONNX Runtime (via Hugot) | Apache-2.0 | ✅ 第 3 条 | 无（仅商标纠纷¹） | 🟢 低 |
| Hugot | Apache-2.0 | ✅ 第 3 条 | 未发现 | 🟢 低 |
| GoMLX | Apache-2.0 | ✅ 第 3 条 | 未发现 | 🟢 低 |
| Google Wire | Apache-2.0 | ✅ 第 3 条 | 未发现 | 🟢 低 |
| modernc.org/sqlite | BSD-3-Clause | ❌ 无 | 未发现 | 🟢 低 |
| React | MIT | ❌ 无 | 无（React 专利条款已于 2017 年退役²） | 🟢 低 |
| Tailwind CSS | MIT | ❌ 无 | 未发现 | 🟢 低 |
| Vite | MIT | ❌ 无 | 未发现 | 🟢 低 |

> ¹ 2024 年一起商标纠纷涉及第三方滥用 "ONNX" 品牌进行钓鱼攻击。这是**商标侵权问题**，非技术专利争议。未发现 ONNX Runtime 的专利诉讼。
> ² Facebook 的 React 专利条款（2015–2017）在 React 于 2017 年 9 月重新以 MIT 许可时已被移除。

### 总体专利风险结论

**🟢 低风险** — MedMemo 的关键依赖中不存在已知专利争议。Apache-2.0 许可组件（Hugot、GoMLX、Wire、ONNX Runtime）的存在为下游用户提供了来自贡献者的明确专利授权，降低了风险。

---

## 署名要求

为遵守所有第三方许可证，分发 MedMemo 时必须包含：

1. **本文件**（`THIRD_PARTY_LICENSES.md`）— 汇总所有依赖及其许可证
2. **项目 `LICENSE` 文件** — MedMemo 的 MIT 许可证
3. **依赖许可证文本** — 对于 Apache-2.0 依赖，保留任何 NOTICE 文件

Wails 构建过程（`wails build`）将前端资源嵌入 Go 二进制。编译后的二进制本身是 MIT 条款下的"实质性部分"，因此二进制分发必须保留 LICENSE 和 THIRD_PARTY_LICENSES 文件，与二进制一并提供或包含在安装包内。

---

## 使用的工具

| 工具 | 版本 | 用途 |
|:-----|:-----|:-----|
| `go-licenses` | v1.6.0 | Go 模块依赖许可证扫描 |
| `license-checker` | v25.0.1 | Node.js 依赖许可证扫描 |

### 扫描命令

```bash
# Go 依赖
go-licenses report .

# Node.js 依赖
cd web && npx license-checker --start . --json
```

---

*本文档由许可证扫描工具自动生成，并经过人工审核以确保准确性。最后更新：2026-05-19。*
