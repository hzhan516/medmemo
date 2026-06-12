# 滚动更新追踪器

> 🤖 **To Future AI Agents (记忆延续指令):**
> 每次你对本项目代码进行了实质性修改、重构、版本推进，或者通过阅读代码获得了新的深层理解后，**必须**在此文件顶部追加更新记录。
>
> **更新格式标准：**
> - **[Date/Version]**: 记录时间与当前操作的版本号（如 v1.1.2）。
> - **[Modified Areas]**: 变更了哪些核心文件、模块或配置文件。
> - **[Logic Evolution]**: 业务逻辑发生了什么演进？引入了什么新概念？
> - **[Checklist Status]**: 是否已写入 `/开发日志/`？是否已同步 `wails.json`, `package.json`, `changelog`（及大版本文档）？
> - **[Pending/Next Steps]**: 技术债或待办。

---

### [2026-06-11 / v1.1.4] — Skill 按 create-skill 规范重构

**[Modified Areas]**
- `.skill/medmemo/SKILL.md` — 精简入口（<200 行）
- 新增 `reference-architecture.md`, `reference-modules.md`, `reference-conventions.md`, `reference-release.md`, `tracker.md`

**[Logic Evolution]**
- Skill 采用渐进式披露：主文件仅保留工作流与检查清单，详情拆入 reference 文件
- 项目入口在仓库根（`main.go` + `wails_app.go`）；主存储 SQLCipher + sqlite-vec
- 记忆 v1.1 以 Fact + SemanticEmbedding 为主路径；DuckDB/Kùzǔ 为存根

**[Checklist Status]**
- N/A 无版本号变更

**[Pending/Next Steps]**
- `FamilyRepoKuzu` stub — Issue#016
- `DuckDBConnector` 未实现，确认移除或恢复
- v1.2.0 大版本需新建 `INTEGRATION_REPORT_v1.2.md`
