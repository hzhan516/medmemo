# 版本管理与发布规范

> 🤖 **版本变更与日志操作强制指南:**
>
> **1. 日常开发日志 (Daily Dev Logs)**
> - 任何功能开发、Debug或代码修改，必须在 `/home/doyle/Codes/medmemo/medmemo/开发日志/` 目录中，根据当前开发版本编号新建或更新对应的日志文件，详细记录开发过程与思考。
>
> **2. 常规版本更新 (Minor/Patch Update, 例如 v1.1.1 -> v1.1.2)**
> 只要涉及新增版本内容或推进版本号，必须**实时同步**修改以下文件：
> - 📄 `wails.json`：作为应用版本号的**唯一事实来源**，更新 `info.productVersion` 字段（例如 `"1.1.4"` → `"1.1.5"`）。
> - 📄 `web/package.json`：前端 npm 包版本，更新 `version` 字段，**必须**与 `wails.json` 保持绝对一致。
> - 📄 `internal/domain/entity/changelog/zh-Hans.json`：应用内「更新提示 / What's New」弹窗的数据源。必须在 JSON 数组**末尾**新增一个版本条目，结构必须包含：`version`, `title`, `features`, `fixes`。
>
> **3. 大版本更新 (Major Update, 例如 v1.1.x -> v1.2.0)**
> 发生大版本迭代时，**在完成上述第 2 点所有操作的基础上**，必须同步更新以下文档：
> - 📄 `README.md`：项目首页。更新底部的 `Last updated` 日期；如有功能变更，同步更新特性描述。
> - 📄 `docs/i18n/zh-Hans-CN/README.md`：README 的中文翻译。必须与英文 `README.md` 同步更新。
> - 📄 `docs/INTEGRATION_REPORT_v{大版本号}.md`（如 `docs/INTEGRATION_REPORT_v1.2.md`）：大版本发布时**新建**本版本的集成测试报告。
> - 📄 `docs/i18n/zh-Hans-CN/INTEGRATION_REPORT_v{大版本号}.md`：新建对应的集成测试报告中文翻译文档，与英文保持同步。

## changelog 条目模板

```json
{
  "version": "vX.Y.Z",
  "title": "简短版本标题",
  "features": ["功能 1", "功能 2"],
  "fixes": ["修复 1"]
}
```

## 完整检查清单

```
常规 (Minor/Patch):
□ wails.json → info.productVersion
□ web/package.json → version（无 v 前缀，与上一致）
□ web/package-lock.json → 执行 `cd web && npm install` 刷新
□ internal/domain/entity/changelog/zh-Hans.json → 末尾新增
□ medmemo/开发日志/ → 版本日志
□ .skill/medmemo/tracker.md → 滚动记录
□ 若涉及顶层文档变更：同步更新 `docs/i18n/zh-Hans-CN/` 中文翻译

大版本额外:
□ README.md + docs/i18n/zh-Hans-CN/README.md
□ docs/INTEGRATION_REPORT_vX.Y.md + 中文翻译
```

## 构建与发布

- 开发：`wails dev`
- 构建：`wails build`（`//go:embed all:web/dist`）
- Wire：`wire .`
- CI：`.github/workflows/build-and-release.yml` + GoReleaser
- 自动更新：`internal/adapters/updater/github.go` → GitHub Releases API
