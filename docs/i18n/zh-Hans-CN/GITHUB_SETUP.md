# MedMemo GitHub 仓库生产级配置手册

> 🌐 [English Version](../../GITHUB_SETUP.md)

本文档说明如何在 GitHub Web UI 中完成仓库的剩余手动配置。自动化配置文件保存在 `.github/` 目录下。

---

## 1. 仓库基础设置

### Features

| 设置项 | 建议值 | 说明 |
|--------|--------|------|
| Issues | 开启 | 已配置 YAML Issue 表单。 |
| Discussions | 开启 | 用于 Q&A、Ideas、Show and tell。 |
| Projects | 开启 | 用于 Roadmap 管理。 |
| Wiki | 关闭 | 文档统一放在 `docs/` 目录。 |
| Sponsorships | 按需 | 有赞助渠道时再开启。 |

### Pull Requests

| 设置项 | 建议值 |
|--------|--------|
| Allow merge commits | 关闭 |
| Allow squash merging | 开启并设为默认 |
| Allow rebase merging | 关闭 |
| Always suggest updating pull request branches | 开启 |
| Automatically delete head branches | 开启 |

### Actions

- **Actions permissions**：`Allow all actions and reusable workflows`
- **Fork pull request workflows**：`Require approval for first-time contributors`

---

## 2. 分支保护规则

### `main`

为 `main` 创建分支规则：

```text
Restrict deletions
Require a pull request before merging
  Require approvals: 1
  Dismiss stale PR approvals when new commits are pushed
  Require review from CODEOWNERS
  Require conversation resolution before merging

Require status checks to pass before merging
  Require branches to be up to date before merging
  Required checks:
    - Lint
    - Unit Test
    - Integration Test
    - Build
    - Go Vulnerability Check

Do not allow bypassing the above settings
Restrict pushes that create files larger than 100MB
```

### `develop`

为 `develop` 创建与 `main` 相同的检查，或按维护者决策适当放宽审批要求。

当 `Cross Platform Build` job 配置了 `continue-on-error: true` 时，不要将它设为 required status check。

---

## 3. Discussions 分类

| 分类 | 用途 |
|------|------|
| Announcements | 维护者公告，尽量只读。 |
| Q&A | 用户使用问题。 |
| Ideas | 轻量级功能建议。 |
| General | 社区讨论。 |
| Show and tell | 用户场景和反馈。 |

---

## 4. 安全设置

### Dependabot

- 开启 Dependabot alerts。
- 开启 Dependabot security updates。
- 保持 `.github/dependabot.yml` 生效，用于自动提交依赖更新 PR。

### Secret Scanning

- 开启 secret scanning。
- 开启 push protection。

---

## 5. 社交展示

### Social Preview

在 Settings -> Social preview 上传 1280 x 640 Open Graph 图片。

### Topics

建议仓库标签：

```text
health, desktop-app, wails, go, react, local-ai, privacy, onnx, medical-assistant, knowledge-graph
```

---

## 6. 发布流程检查清单

1. 从 `develop` 创建 `release/vX.Y.Z`。
2. 更新版本元数据和 changelog。
3. 将 `release/vX.Y.Z` 合并到 `main`。
4. 在 `main` 打 tag：`git tag -a vX.Y.Z -m "Release vX.Y.Z"`。
5. 推送 tag：`git push origin vX.Y.Z`。
6. 等待 release workflow 生成跨平台产物。
7. 检查 GitHub Releases 草稿，补充发行说明后发布。

---

*最后更新：2026-07-09*
