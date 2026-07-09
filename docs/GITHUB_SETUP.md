# MedMemo GitHub Repository Production Setup

> 🌐 [中文版本](./i18n/zh-Hans-CN/GITHUB_SETUP.md)

This document lists the remaining manual GitHub settings for the MedMemo repository. Automated configuration files are kept under `.github/`.

---

## 1. Repository Basics

### Features

| Setting | Recommended Value | Notes |
|---------|-------------------|-------|
| Issues | Enabled | YAML issue forms are configured. |
| Discussions | Enabled | Use for Q&A, ideas, and show-and-tell. |
| Projects | Enabled | Use for roadmap tracking. |
| Wiki | Disabled | Keep documentation in `docs/`. |
| Sponsorships | As needed | Enable only when a sponsorship channel exists. |

### Pull Requests

| Setting | Recommended Value |
|---------|-------------------|
| Allow merge commits | Disabled |
| Allow squash merging | Enabled and default |
| Allow rebase merging | Disabled |
| Always suggest updating pull request branches | Enabled |
| Automatically delete head branches | Enabled |

### Actions

- **Actions permissions**: `Allow all actions and reusable workflows`
- **Fork pull request workflows**: `Require approval for first-time contributors`

---

## 2. Branch Protection

### `main`

Create a branch rule for `main`:

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

Create a branch rule for `develop` with the same checks as `main`, or slightly relaxed approval requirements if maintainers choose.

Do not make `Cross Platform Build` a required status check while that job is configured with `continue-on-error: true`.

---

## 3. Discussions Categories

| Category | Purpose |
|----------|---------|
| Announcements | Maintainer announcements, read-only if possible. |
| Q&A | User questions. |
| Ideas | Lightweight feature suggestions. |
| General | Community discussion. |
| Show and tell | User scenarios and feedback. |

---

## 4. Security Settings

### Dependabot

- Enable Dependabot alerts.
- Enable Dependabot security updates.
- Keep `.github/dependabot.yml` active for automated update PRs.

### Secret Scanning

- Enable secret scanning.
- Enable push protection.

---

## 5. Social Metadata

### Social Preview

Upload a 1280 x 640 Open Graph image in Settings -> Social preview.

### Topics

Suggested repository topics:

```text
health, desktop-app, wails, go, react, local-ai, privacy, onnx, medical-assistant, knowledge-graph
```

---

## 6. Release Checklist

1. Create `release/vX.Y.Z` from `develop`.
2. Update version metadata and changelog.
3. Merge `release/vX.Y.Z` into `main`.
4. Tag on `main`: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
5. Push the tag: `git push origin vX.Y.Z`.
6. Let the release workflow generate cross-platform artifacts.
7. Review the GitHub Releases draft, add release notes, and publish.

---

*Last updated: 2026-07-09*
