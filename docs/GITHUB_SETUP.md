# MedMemo GitHub 仓库生产级配置手册

> 本文档说明如何在 GitHub Web UI 中完成仓库的剩余手动配置。
> 配套文件已提交到 `.github/` 目录下。

---

## 1. 仓库基础设置（Settings → General）

### 1.1 Features
| 设置项 | 建议值 | 说明 |
|--------|--------|------|
| **Issues** | ✅ 开启 | 已配置 YAML 表单模板 |
| **Discussions** | ✅ 开启 | 用于 Q&A、Ideas、Show and tell |
| **Projects** | ✅ 开启 | 用于 Roadmap 管理 |
| **Wiki** | ❌ 关闭 | 文档统一放在 `docs/` 目录 |
| **Sponsorships** | 按需 | 如有赞助渠道可开启 |

### 1.2 Pull Requests
| 设置项 | 建议值 |
|--------|--------|
| Allow merge commits | ❌ 关闭 |
| Allow squash merging | ✅ 开启（默认） |
| Allow rebase merging | ❌ 关闭 |
| Always suggest updating pull request branches | ✅ 开启 |
| Automatically delete head branches | ✅ 开启 |

### 1.3 Actions
- **Actions permissions** → `Allow all actions and reusable workflows`
- **Fork pull request workflows** → `Require approval for first-time contributors`（防止恶意挖矿）

---

## 2. 分支保护规则（Settings → Branches）

### `main` 分支（生产分支）

点击 **Add rule**，Pattern 填 `main`，配置如下：

```
✅ Restrict deletions
✅ Require a pull request before merging
   └─ Require approvals: 1
   └─ Dismiss stale PR approvals when new commits are pushed: ✅
   └─ Require review from CODEOWNERS: ✅
   └─ Require conversation resolution before merging: ✅

✅ Require status checks to pass before merging
   └─ Require branches to be up to date before merging: ✅
   └─ Status checks that are required:
      - Lint
      - Unit Test
      - Integration Test
      - Build
      - Go Vulnerability Check

❌ Do not allow bypassing the above settings
   └─ 即使是管理员也要遵守规则

✅ Restrict pushes that create files larger than 100MB
```

### `develop` 分支（集成分支）

Pattern 填 `develop`，配置与 `main` 相同，或适当放宽 approvals 要求。

> ⚠️ **注意**：`Cross Platform Build` job 设置了 `continue-on-error: true`，**不要**将其设为 required status check。

---

## 3. Discussions 分类配置

开启 Discussions 后，前往 **Settings → Discussions → Categories**，建议保留/创建以下分类：

| 分类 | 用途 |
|------|------|
| 📣 Announcements | 维护者发布公告（只读） |
| 🙏 Q&A | 用户使用问题 |
| 💡 Ideas | 功能建议（轻量级，不直接开 Issue） |
| 🗣️ General | 社区闲聊、使用心得 |
| 🎉 Show and tell | 用户分享使用场景和反馈 |

---

## 4. 安全设置（Settings → Security）

### 4.1 Dependabot
- **Dependabot alerts**: ✅ 开启
- **Dependabot security updates**: ✅ 开启
- 已配置 `.github/dependabot.yml` 自动提交版本更新 PR

### 4.2 Secret scanning
- **Secret scanning**: ✅ 开启
- **Push protection**: ✅ 开启（阻止意外推送密钥）

---

## 5. 社交与展示

### 5.1 Social Preview
上传一张 1280×640 的 Open Graph 图片（Settings → Social preview）。

### 5.2 Topics
添加以下仓库标签：
```
health, desktop-app, wails, go, react, local-ai, privacy, onnx, medical-assistant, knowledge-graph
```

---

## 6. 发布流程检查清单

每次发版时，维护者按以下步骤操作：

1. 从 `develop` 创建 `release/vX.Y.Z` 分支
2. 更新版本号和 CHANGELOG
3. 合并 `release/vX.Y.Z` 到 `main`
4. 在 `main` 上打 tag：`git tag -a vX.Y.Z -m "Release vX.Y.Z"`
5. Push tag：`git push origin vX.Y.Z`
6. Release 工作流自动触发，生成跨平台构建产物
7. 在 GitHub Releases 页面检查草稿，补充发行说明后发布

---

*本手册与 `.github/` 目录下的自动化配置配套使用。*
