# 开源合规报告

> 🌐 [English Version](../../OPEN_SOURCE_COMPLIANCE.md)

> **项目**：MedMemo — 开源桌面健康信息工具
> **报告日期**：2026-07-09
> **范围**：Go 后端依赖 + Node.js 前端依赖
> **主许可证**：MIT

---

## 摘要

本报告记录当前 `go.mod` 与 `web/package-lock.json` 的依赖盘点，替代 2026 年 5 月旧快照中 `typescript@5.6.2`、Tailwind CSS v4 等过期表述。

当前锁文件盘点：

| 生态 | 来源 | 当前盘点 |
|------|------|----------|
| Go modules | `go.mod` / `go.sum` | 208 个依赖模块，另有根模块（使用 Go 1.26.4 执行 `go list -m all`） |
| npm packages | `web/package-lock.json` | 排除根包后 521 个 package 条目 |

合规姿态保持保守：

| 发现 | 状态 |
|------|:----:|
| 项目许可证为 MIT | ✅ |
| 当前锁文件未发现已知 GPL/AGPL/SSPL 依赖 | ✅ |
| 高价值依赖仍处于宽松许可证生态 | ✅ |
| 发布前必须刷新完整许可证 notice | ⚠️ 必需 |

---

## 当前高价值依赖

| 依赖 | 版本 | 作用 | 许可证风险 |
|------|------|------|:----------:|
| `github.com/wailsapp/wails/v2` | `v2.12.0` | 桌面应用框架 | 低 |
| `github.com/knights-analytics/hugot` | `v0.7.4` | ONNX/Hugging Face 推理绑定 | 低 |
| `github.com/daulet/tokenizers` | `v1.27.0` | tokenizer 原生绑定 | 低 |
| `github.com/yalue/onnxruntime_go` | `v1.30.1` | ONNX Runtime 绑定 | 低 |
| `github.com/mutecomm/go-sqlcipher` | 以 `go.sum` 为准 | 加密 SQLite | 低 |
| `github.com/viant/sqlite-vec` | `v0.3.0` | SQLite 向量搜索 | 低 |
| `react` | `^18.2.0` | 前端 UI | 低 |
| `typescript` | `5.9.3` | 前端类型系统 | 低 |
| `tailwindcss` | `^3.4.1` | 样式系统 | 低 |
| `vite` | `6.4.3` | 前端构建工具 | 低 |
| `react-router-dom` | `^7.18.1` | 前端路由 | 低 |

---

## 发布前必须重扫

每次公开发布前运行以下命令；如输出变化，需要重新生成 `THIRD_PARTY_LICENSES.md`：

```bash
go install github.com/google/go-licenses@latest
go-licenses report ./...

cd web
npx license-checker --start . --json
```

发布负责人必须确认：

- 未引入 GPL、AGPL、SSPL 或专有许可证依赖。
- Apache-2.0、BSD、ISC、MIT、MPL-2.0、CC0、CC-BY 的义务已记录。
- MPL-2.0 依赖未被修改；若修改过，需按 MPL-2.0 提供对应源文件。
- `LICENSE` 与生成的第三方 notice 已随二进制安装包分发。

---

## 分发检查清单

- [ ] 包含 `LICENSE`。
- [ ] `THIRD_PARTY_LICENSES.md` 已刷新并随包分发。
- [ ] Go 与 npm 许可证扫描结果已归档到发布记录。
- [ ] 本报告中的依赖版本与 `go.mod`、`web/package.json` 一致。
- [ ] 中文翻译保持同步。

---

*最后更新：2026-07-09*
