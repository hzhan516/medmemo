# MedMemo v1.1.10 Cross-Platform Packaging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or AgentSwarm to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix Windows missing-DLL distribution problems and deliver installer-based, architecture-aware release assets (Windows NSIS setup, Linux DEB/RPM/AppImage, macOS dual-arch DMG) with strict release-asset whitelisting and local CI pre-push validation.

**Architecture:** Keep Wails/GoReleaser-based release pipeline but tighten NSIS for per-user install, add nFPM-driven Linux packages, split macOS DMG builds by architecture via CI matrix, make updater match assets by platform/install-kind/architecture, and gate every push with a local validation script.

**Tech Stack:** Go 1.26, Wails v2, NSIS, GoReleaser + nFPM, GitHub Actions, Bash, PowerShell.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `build/windows/installer/project.nsi` | Per-user NSIS installer, creates `$INSTDIR\data`, writes `InstallPath` as directory, preserves data on uninstall. |
| `internal/infrastructure/updater/installer_windows.go` | Resolve Windows install directory from registry (directory, not exe), with backward compatibility. |
| `internal/infrastructure/updater/installer_windows_test.go` | Tests for Windows install-path resolution and `D` argument placement. |
| `internal/infrastructure/config/loader.go` | Wire platform-specific default data directory. |
| `internal/infrastructure/config/default_data_dir_windows.go` | Windows default data dir under install path. |
| `internal/infrastructure/config/default_data_dir_unix.go` | Unix default data dir `~/.medmemo/data`. |
| `internal/application/updater/service.go` | Adjust Windows update/backup directories under install dir. |
| `scripts/build/collect-windows-runtime-dlls.ps1` | Copy MinGW runtime DLLs into `build/bin` after Wails build. |
| `internal/adapters/updater/github.go` | Platform/arch/install-kind aware asset matching. |
| `internal/adapters/updater/github_test.go` | Tests for asset matching. |
| `build/package/linux/medmemo` | `/usr/bin/medmemo` wrapper for DEB/RPM. |
| `build/package/linux/medmemo.desktop` | Desktop entry for DEB/RPM. |
| `build/package/nfpm.yaml` | nFPM package definition. |
| `scripts/build/wails-build.sh` | Platform build orchestration (AppImage, DEB, RPM, DMG). |
| `build/package/build-dmg.sh` | DMG builder with architecture suffix. |
| `build/package/build-appimage.sh` | Existing AppImage builder. |
| `.github/workflows/build-and-release.yml` | Branch-aware version, dual macOS runners, asset whitelist, checksums. |
| `.github/workflows/release.yml` | Tag release with dual macOS artifacts and whitelist. |
| `.github/workflows/ci.yml` | Cross-platform build matrix with dual macOS. |
| `.goreleaser.yml` | Remove `archives`, collect DMG/AppImage/DEB/RPM/setup via `extra_files`. |
| `scripts/build/verify-release-artifact.sh` | Verify per-platform artifact exists. |
| `scripts/build/verify-release-assets.sh` | Whitelist final release assets. |
| `scripts/build/local-ci-check.sh` | Aggregate local CI validation. |
| `internal/infrastructure/updater/installer_linux.go` | Detect Linux install kind, return manual-install for DEB/RPM. |
| `internal/infrastructure/updater/installer_linux_test.go` | Tests for Linux installer behavior. |
| `pkg/resourcepath/resourcepath.go` | Add `/opt/medmemo/resources` candidate. |
| `docs/user-guide/installation.md` | English install guide with dual-arch macOS rows. |
| `docs/i18n/zh-Hans-CN/user-guide/installation.md` | Chinese install guide sync. |
| `internal/domain/entity/changelog/zh-Hans.json` | v1.1.10 packaging release notes. |

---

## Task 1: Windows InstallPath Normalization & Tests

**Files:**
- Modify: `internal/infrastructure/updater/installer_windows.go`
- Modify: `internal/infrastructure/updater/installer_windows_test.go`

- [ ] **Step 1.1:** Add `normalizeInstallPath` helper and update `resolveInstallDir` to prefer directory values with fallback order: HKCU matching current exe dir → HKLM matching current exe dir → HKCU → HKLM → current exe dir.

```go
func normalizeInstallPath(value string) string {
    value = strings.TrimSpace(value)
    if value == "" {
        return ""
    }
    if strings.EqualFold(filepath.Base(value), "MedMemo.exe") {
        return filepath.Dir(value)
    }
    return value
}
```

- [ ] **Step 1.2:** Add/update tests:
  - `TestResolveInstallDir_PerUserHKCUDirectory`
  - `TestResolveInstallDir_AllUsersHKLMDirectoryCompatibility`
  - `TestResolveInstallDir_LegacyRegistryValueExecutablePath`
  - `TestResolveInstallDir_BothPreferCurrentExecutablePath`
  - `TestResolveInstallDir_FallbackToCurrentDir`
  - `TestWindowsInstaller_DArgumentIsLast`

- [ ] **Step 1.3:** Run tests: `go test ./internal/infrastructure/updater/... -run 'TestResolveInstallDir|TestWindowsInstaller_DArgumentIsLast' -v`

- [ ] **Step 1.4:** Commit: `git add internal/infrastructure/updater/installer_windows.go internal/infrastructure/updater/installer_windows_test.go && git commit -m "fix(updater): treat Windows InstallPath as directory with fallback"`

---

## Task 2: Windows Default Data Directory

**Files:**
- Create: `internal/infrastructure/config/default_data_dir_windows.go`
- Create: `internal/infrastructure/config/default_data_dir_unix.go`
- Modify: `internal/infrastructure/config/loader.go`

- [ ] **Step 2.1:** Create `default_data_dir_unix.go`:

```go
//go:build !windows

package config

import "path/filepath"

func defaultDataDirPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".medmemo", "data")
}
```

- [ ] **Step 2.2:** Create `default_data_dir_windows.go`:

```go
//go:build windows

package config

import (
    "path/filepath"
    "strings"
    "golang.org/x/sys/windows/registry"
)

func defaultDataDirPath() string {
    k, err := registry.OpenKey(registry.CURRENT_USER, `Software\MedMemo`, registry.QUERY_VALUE)
    if err != nil {
        return fallbackDataDir()
    }
    defer k.Close()

    installPath, _, err := k.GetStringValue("InstallPath")
    if err != nil {
        return fallbackDataDir()
    }
    installPath = strings.TrimSpace(installPath)
    if strings.EqualFold(filepath.Base(installPath), "MedMemo.exe") {
        installPath = filepath.Dir(installPath)
    }
    if installPath == "" {
        return fallbackDataDir()
    }
    return filepath.Join(installPath, "data")
}

func fallbackDataDir() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".medmemo", "data")
}
```

- [ ] **Step 2.3:** Update `loader.go` to use `defaultDataDirPath()` after env `MEDMEMO_DATA_DIR` and config-file checks.

- [ ] **Step 2.4:** Add Windows-specific tests for registry mapping and env/config override.

- [ ] **Step 2.5:** Run tests: `go test ./internal/infrastructure/config/... -v`

- [ ] **Step 2.6:** Commit.

---

## Task 3: NSIS Per-User Installer

**Files:**
- Modify: `build/windows/installer/project.nsi`
- Modify: `wails.json` if `nsisType` needs per-user-only mode

- [ ] **Step 3.1:** Set `InstallDir "$LOCALAPPDATA\Programs\MedMemo"`.
- [ ] **Step 3.2:** After install files, add:

```nsis
CreateDirectory "$INSTDIR\data"
WriteRegStr HKCU "Software\MedMemo" "InstallPath" "$INSTDIR"
```

- [ ] **Step 3.3:** Replace `RMDir /r $INSTDIR` with explicit deletes:

```nsis
Delete "$INSTDIR\MedMemo.exe"
Delete "$INSTDIR\*.dll"
RMDir /r "$INSTDIR\resources"
Delete "$INSTDIR\uninstall.exe"
RMDir "$INSTDIR\data" ; keep data, only remove if empty
RMDir "$INSTDIR"
DeleteRegKey HKCU "Software\MedMemo"
```

- [ ] **Step 3.4:** Ensure NSIS includes runtime DLLs:

```nsis
SetOutPath $INSTDIR
File "..\..\bin\*.dll"
```

- [ ] **Step 3.5:** Commit.

---

## Task 4: Windows DLL Collection Script

**Files:**
- Create: `scripts/build/collect-windows-runtime-dlls.ps1`

- [ ] **Step 4.1:** Create PowerShell script that:
  - Reads `build/bin/MedMemo.exe` dependencies (e.g. via `dumpbin` or static list fallback).
  - Copies `libgcc_s_seh-1.dll`, `libstdc++-6.dll`, `libwinpthread-1.dll` from `C:\msys64\mingw64\bin` if present.
  - Never copies from `C:\Windows\System32`.
  - Fails if required DLLs missing.

- [ ] **Step 4.2:** Make executable and commit.

---

## Task 5: Remove Windows Bare Exe Fallback & Strict Updater

**Files:**
- Modify: `.github/workflows/build-and-release.yml`
- Modify: `.github/workflows/release.yml`
- Modify: `.goreleaser.yml`
- Modify: `scripts/build/verify-release-artifact.sh`
- Modify: `internal/adapters/updater/github.go`
- Modify: `internal/adapters/updater/github_test.go`

- [ ] **Step 5.1:** In workflows, fail Windows job if installer missing; remove copy of `MedMemo.exe` to `MedMemoSetup.exe`.
- [ ] **Step 5.2:** In `.goreleaser.yml`, remove `archives` section; keep `extra_files` glob that only matches allowed assets.
- [ ] **Step 5.3:** In `verify-release-artifact.sh`, require installer exe for Windows.
- [ ] **Step 5.4:** In `github.go`, remove fallback to arbitrary `.exe`; Windows matches only names containing `setup` or `installer`.
- [ ] **Step 5.5:** Add tests `TestFindTargetAsset_WindowsInstaller`, `TestFindTargetAsset_WindowsRejectsBareExe`.
- [ ] **Step 5.6:** Run updater tests.
- [ ] **Step 5.7:** Commit.

---

## Task 6: Linux DEB/RPM Packaging

**Files:**
- Create: `build/package/nfpm.yaml`
- Create: `build/package/linux/medmemo`
- Create: `build/package/linux/medmemo.desktop`
- Modify: `scripts/build/wails-build.sh`

- [ ] **Step 6.1:** Create `build/package/linux/medmemo` wrapper:

```bash
#!/bin/sh
export MEDMEMO_RESOURCE_DIR="${MEDMEMO_RESOURCE_DIR:-/opt/medmemo/resources}"
export MEDMEMO_INSTALL_KIND="${MEDMEMO_INSTALL_KIND:-package}"
exec /opt/medmemo/MedMemo "$@"
```

- [ ] **Step 6.2:** Create `build/package/linux/medmemo.desktop`.
- [ ] **Step 6.3:** Create `build/package/nfpm.yaml` with contents listed in source plan.
- [ ] **Step 6.4:** Update `wails-build.sh` Linux branch to build DEB/RPM after AppImage:

```bash
goreleaser release --snapshot --rm-dist --config build/package/nfpm.yaml
# or nfpm package -c build/package/nfpm.yaml -p deb/rpm
```

- [ ] **Step 6.5:** Add `/opt/medmemo/resources` candidate to `pkg/resourcepath/resourcepath.go`.
- [ ] **Step 6.6:** Run Linux build script smoke test where possible.
- [ ] **Step 6.7:** Commit.

---

## Task 7: Linux Update Behavior by Install Kind

**Files:**
- Modify: `internal/infrastructure/updater/installer_linux.go`
- Modify: `internal/infrastructure/updater/installer_linux_test.go`
- Modify: `internal/adapters/updater/github.go`
- Modify: `internal/adapters/updater/github_test.go`

- [ ] **Step 7.1:** Add install kind detection in `installer_linux.go`:
  - `.AppImage` suffix → `appimage`
  - `MEDMEMO_INSTALL_KIND=deb` or marker → `deb`
  - `MEDMEMO_INSTALL_KIND=rpm` or marker → `rpm`
  - else `unknown`

- [ ] **Step 7.2:** Add `ManualPackageInstallRequired` error type with PackagePath, Kind, Command.
- [ ] **Step 7.3:** `Install` returns manual-install error for DEB/RPM; replaces original for AppImage.
- [ ] **Step 7.4:** Update `github.go` to match `.deb`/`.rpm`/`.AppImage` based on install kind; unknown Linux falls back to AppImage.
- [ ] **Step 7.5:** Add tests listed in source plan.
- [ ] **Step 7.6:** Run tests.
- [ ] **Step 7.7:** Commit.

---

## Task 8: macOS Dual-Architecture DMG

**Files:**
- Modify: `build/package/build-dmg.sh`
- Modify: `scripts/build/wails-build.sh`

- [ ] **Step 8.1:** Update `build-dmg.sh` to accept `--arch x86_64|arm64` and output `MedMemo_${ARCH}.dmg`.
- [ ] **Step 8.2:** Update `wails-build.sh` darwin branch to derive `DMG_ARCH` from `DARWIN_PLATFORM` and call `./build/package/build-dmg.sh --arch "$DMG_ARCH"`.
- [ ] **Step 8.3:** Commit.

---

## Task 9: macOS Updater Architecture Matching

**Files:**
- Modify: `internal/adapters/updater/github.go`
- Modify: `internal/adapters/updater/github_test.go`

- [ ] **Step 9.1:** Add `matchArch(name, goarch)` helper mapping `amd64` ↔ `x86_64`.
- [ ] **Step 9.2:** Update `matchesPlatform` darwin branch:

```go
case "darwin":
    return strings.HasSuffix(name, ".dmg") && matchArch(name, goarch)
```

- [ ] **Step 9.3:** In `findTargetAsset`, add fallback to generic `.dmg` if no arch-specific DMG found.
- [ ] **Step 9.4:** Add tests from source plan.
- [ ] **Step 9.5:** Run tests.
- [ ] **Step 9.6:** Commit.

---

## Task 10: CI macOS Dual Runner Matrix

**Files:**
- Modify: `.github/workflows/build-and-release.yml`
- Modify: `.github/workflows/release.yml`
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 10.1:** Replace single `macos-latest` with matrix entries `macos-13` (darwin-amd64) and `macos-14` (darwin-arm64).
- [ ] **Step 10.2:** Ensure artifacts uploaded per platform key (`darwin-amd64`, `darwin-arm64`).
- [ ] **Step 10.3:** Update release job to download both macOS artifacts and copy both DMGs.
- [ ] **Step 10.4:** Commit.

---

## Task 11: Release Asset Whitelist & Checksums

**Files:**
- Create: `scripts/build/verify-release-assets.sh`
- Modify: `.github/workflows/build-and-release.yml`
- Modify: `.github/workflows/release.yml`

- [ ] **Step 11.1:** Create `verify-release-assets.sh` that checks only allowed patterns in given directory:

```text
MedMemoSetup.exe
MedMemo_x86_64.dmg
MedMemo_arm64.dmg
MedMemo-x86_64.AppImage
MedMemo_*.deb
MedMemo-*.rpm
checksums.txt
```

- [ ] **Step 11.2:** Add checksums generation step in release jobs:

```bash
cd release-assets
sha256sum * > checksums.txt
```

- [ ] **Step 11.3:** Run whitelist script against empty dir to verify it rejects unexpected files.
- [ ] **Step 11.4:** Commit.

---

## Task 12: Fix build-and-release.yml Version Generation

**Files:**
- Modify: `.github/workflows/build-and-release.yml`

- [ ] **Step 12.1:** Replace unconditional pre-release version logic with branch-aware logic:

```bash
BASE_VERSION=$(jq -r '.info.productVersion' wails.json)
REF="${GITHUB_REF_NAME}"

if [[ "$REF" == release/* ]]; then
  VERSION="v${BASE_VERSION}-Pre-release"
  PRERELEASE="true"
elif [[ "$REF" == "main" ]]; then
  VERSION="v${BASE_VERSION}"
  PRERELEASE="false"
else
  echo "Unsupported ref: $REF"
  exit 1
fi
```

- [ ] **Step 12.2:** Add release branch version consistency validation.
- [ ] **Step 12.3:** Set `draft: true` for pre-release, `draft: false` for stable in `softprops/action-gh-release`.
- [ ] **Step 12.4:** Commit.

---

## Task 13: Local CI Pre-Push Validation Script

**Files:**
- Create: `scripts/build/local-ci-check.sh`

- [ ] **Step 13.1:** Create script aggregating:
  - `make lint`
  - `cd web && npm run lint`
  - `cd web && npx tsc --noEmit`
  - `node scripts/validate-provider-templates.js`
  - `cd web && npm run build`
  - `make test`
  - `make test-integration`
  - `make test-e2e`
  - `make build`
  - `make wire && git diff --exit-code wire_gen.go`
  - `./scripts/build/verify-release-assets.sh release-assets` (when artifacts exist)
  - Cross-compilation checks
  - Platform-specific test compilation

- [ ] **Step 13.2:** On failure, write `.medmemo/review/local-ci-failure.log`.
- [ ] **Step 13.3:** Make executable and commit.

---

## Task 14: Documentation & Changelog

**Files:**
- Modify: `docs/user-guide/installation.md`
- Modify: `docs/i18n/zh-Hans-CN/user-guide/installation.md`
- Modify: `internal/domain/entity/changelog/zh-Hans.json`

- [ ] **Step 14.1:** Update English install guide with dual-arch macOS rows and six release assets.
- [ ] **Step 14.2:** Sync Chinese install guide.
- [ ] **Step 14.3:** Add v1.1.10 packaging notes to changelog.
- [ ] **Step 14.4:** Commit.

---

## Task 15: Final Verification & Push

- [ ] **Step 15.1:** Run `scripts/build/local-ci-check.sh` locally.
- [ ] **Step 15.2:** Fix any failures and re-run full script.
- [ ] **Step 15.3:** Run `git push -u origin hzhan516/feat/cross-platform-packaging-v1.1.10` only after all checks pass.

---

## Self-Review

**Spec coverage:** Each track in the source plan maps to one or more tasks above.
**Placeholder scan:** No TBD placeholders; each task names concrete files and behavior.
**Type consistency:** `InstallPath` treated as directory consistently across NSIS, Go updater, and config loader.
