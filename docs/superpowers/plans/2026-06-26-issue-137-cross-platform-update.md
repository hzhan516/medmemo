# Issue #137 跨平台自动更新修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 MedMemo v1.1.8 在 Linux AppImage、macOS DMG、Windows NSIS 三种安装形态下的自动更新路径，确保更新后启动的是新版本，失败时给出明确回退提示，并补充完整测试文件与开发日志。

**Architecture:** 保持 Clean Architecture 四层边界：infrastructure 层负责平台安装器（AppImage/DMG/NSIS），application 层 `updater.Service` 统一编排下载/校验/安装，adapter 层 `InstallerAdapter` 将基础设施安装器注入服务，Wails 绑定层只负责调用安装结果并触发平台重启 helper。新增 OS-specific restart helper，避免 application 层依赖 Wails runtime。

**Tech Stack:** Go 1.26.4, Wails v2, React 18 + TypeScript + Vitest, NSIS (Windows), AppImage runtime env vars (Linux), hdiutil/osascript (macOS).

---

## File Structure

### Backend (Go)

| File | Responsibility |
|------|----------------|
| `internal/application/port/updater.go` | `Installer` 接口：`Install(assetPath string) (string, error)` / `Rollback() error` / `CurrentBinaryPath() string` |
| `internal/application/updater/service.go` | `ApplyUpdate` 改为返回 `(installedPath string, err error)`；安装失败时回滚 |
| `internal/application/updater/service_test.go` | 覆盖 `ApplyUpdate` 成功/失败/回滚失败路径 |
| `internal/infrastructure/updater/installer_linux.go` | AppImage 路径解析（ARGV0 / proc/cmdline / os.Executable fallback）、备份替换、回滚 |
| `internal/infrastructure/updater/installer_linux_test.go` | `resolveAppImagePath` 与 `LinuxInstaller` 边界测试 |
| `internal/infrastructure/updater/installer_darwin.go` | DMG 自动挂载/替换 `.app`、管理员授权复制、授权失败 fallback 到手动 DMG |
| `internal/infrastructure/updater/installer_darwin_test.go` | bundle 路径解析、DMG 内 app 查找、可写/授权失败/回滚测试 |
| `internal/infrastructure/updater/installer_windows.go` | 解析当前 exe、读取 HKCU/HKLM `Software\MedMemo\InstallPath`、按当前运行目录优先选择、启动 NSIS `/S /D=<dir>` |
| `internal/infrastructure/updater/installer_windows_test.go` | 注册表路径优先级、`/D` 参数位置、命令构建 |
| `internal/infrastructure/updater/client.go` | 通用 `getCurrentBinary()` 保留，Linux installer 不再直接使用 |
| `internal/adapters/updater/installer_adapter.go` | 透传 `Install` 新签名，无需改动 |
| `wails_app_update.go` | `ApplyUpdate` 调用 service 后执行平台 restart helper，失败不退出 |
| `wails_app_update_test.go` | 安装成功重启、重启失败、安装失败不退出 |
| `update_restart_linux.go` | Linux 平台 restart helper：执行新 AppImage 并 `runtime.Quit` |
| `update_restart_darwin.go` | macOS 平台 restart helper：`open` 新 `.app` 并 `runtime.Quit` |
| `update_restart_windows.go` | Windows 平台 restart helper：直接退出，由 NSIS 安装完成后自动启动 |

### Frontend (React/TypeScript)

| File | Responsibility |
|------|----------------|
| `web/src/hooks/useUpdate.ts` | 新增 `isRestarting` 状态；`doApply` 设置状态并抛出错误 |
| `web/src/components/UpdateModal.tsx` | 下载完成后显示“立即重启完成更新”；`isRestarting` 禁用按钮；macOS fallback 提示 |
| `web/src/components/UpdateModal.test.tsx` | 覆盖重启文案、禁用状态、手动安装 fallback |

### Build & Docs

| File | Responsibility |
|------|----------------|
| `build/windows/installer/project.nsi` | per-user 写 HKCU `InstallPath`；all-users 写 HKLM `InstallPath`；卸载清理；安装完成后启动 exe |
| `medmemo/开发日志/v1.1/v1.1.8.md` | 追加 Issue #137 修复阶段记录 |
| `internal/domain/entity/changelog/zh-Hans.json` | v1.1.8 条目 fixes 中追加跨平台自动更新修复说明 |

---

## Task 1: 更新 Service 与 Installer 接口签名

**Files:**
- Modify: `internal/application/port/updater.go`
- Modify: `internal/application/updater/service.go`
- Modify: `internal/application/updater/service_test.go`

- [ ] **Step 1: 修改 `Installer.Install` 返回安装后路径**

  在 `internal/application/port/updater.go` 中：
  ```go
  type Installer interface {
      Install(assetPath string) (string, error)
      Rollback() error
      CurrentBinaryPath() string
  }
  ```

- [ ] **Step 2: 修改 `Service.ApplyUpdate` 签名并透传安装路径**

  在 `internal/application/updater/service.go` 中：
  ```go
  func (s *Service) ApplyUpdate(assetPath string) (string, error) {
      installedPath, err := s.installer.Install(assetPath)
      if err != nil {
          _ = s.installer.Rollback()
          return "", fmt.Errorf("failed to install update: %w", err)
      }
      return installedPath, nil
  }
  ```

- [ ] **Step 3: 更新 `service_test.go` 中 `mockInstaller` 与断言**

  - `mockInstaller.Install` 返回 `(string, error)`。
  - 新增 `TestServiceApplyUpdate_ReturnsInstalledPath`。
  - 调整 `TestServiceApplyUpdate_RollbackError` 断言新签名。

- [ ] **Step 4: 运行测试**

  ```bash
  GOTOOLCHAIN=auto go test ./internal/application/updater -v
  ```

- [ ] **Step 5: 提交**

  ```bash
  git add internal/application/port/updater.go internal/application/updater/service.go internal/application/updater/service_test.go
  git commit -m "refactor(M01): ApplyUpdate returns installed path (Issue #137)"
  ```

---

## Task 2: Linux AppImage 自动替换

**Files:**
- Modify: `internal/infrastructure/updater/installer_linux.go`
- Modify: `internal/infrastructure/updater/installer_linux_test.go`

- [ ] **Step 1: 实现 `resolveAppImagePath`**

  新建私有函数：
  ```go
  func resolveAppImagePath() string {
      if argv0 := os.Getenv("ARGV0"); strings.HasSuffix(strings.ToLower(argv0), ".appimage") {
          if abs, err := filepath.Abs(argv0); err == nil {
              return abs
          }
          return argv0
      }
      if data, err := os.ReadFile("/proc/self/cmdline"); err == nil {
          args := strings.Split(string(bytes.TrimRight(data, "\x00")), "\x00")
          for _, arg := range args {
              if strings.HasSuffix(strings.ToLower(arg), ".appimage") {
                  if abs, err := filepath.Abs(arg); err == nil {
                      return abs
                  }
                  return arg
              }
          }
      }
      return getCurrentBinary()
  }
  ```

  并在 `newLinuxInstaller` 中使用：
  ```go
  func newLinuxInstaller() *LinuxInstaller {
      return &LinuxInstaller{currentPath: resolveAppImagePath()}
  }
  ```

- [ ] **Step 2: `Install` 增加 AppImage 识别与目录可写检查**

  - 若 `currentPath` 不是以 `.AppImage` 结尾，返回 `errors.New("manual update required: current binary is not an AppImage")`。
  - 若原 AppImage 所在目录不可写，返回明确错误。
  - 保留备份 → chmod → `os.Rename` 原子替换。

- [ ] **Step 3: 补充 Linux 测试**

  新增测试：
  - `TestResolveAppImagePath_ARGV0`
  - `TestResolveAppImagePath_ProcCmdline`
  - `TestResolveAppImagePath_FallbackExecutable`
  - `TestLinuxInstaller_Install_ReplacesOriginalAppImage`
  - `TestLinuxInstaller_Install_NotWritableFallback`
  - `TestLinuxInstaller_Install_NotAppImageReturnsManualError`
  - `TestLinuxInstaller_Rollback_RestoresBackup`

- [ ] **Step 4: 运行测试**

  ```bash
  GOTOOLCHAIN=auto go test ./internal/infrastructure/updater -v -run 'Linux|AppImage|Resolve'
  ```

- [ ] **Step 5: 提交**

  ```bash
  git add internal/infrastructure/updater/installer_linux.go internal/infrastructure/updater/installer_linux_test.go
  git commit -m "fix(M01): Linux AppImage resolves original path and replaces atomically (Issue #137)"
  ```

---

## Task 3: macOS DMG 自动安装与手动回退

**Files:**
- Modify: `internal/infrastructure/updater/installer_darwin.go`
- Create: `internal/infrastructure/updater/installer_darwin_test.go`

- [ ] **Step 1: 实现 `resolveAppBundlePath`**

  从当前二进制向上查找 `.app` 目录：
  ```go
  func resolveAppBundlePath() string {
      exe := getCurrentBinary()
      dir := filepath.Dir(exe)
      for {
          if strings.HasSuffix(filepath.Base(dir), ".app") {
              return dir
          }
          parent := filepath.Dir(dir)
          if parent == dir {
              return ""
          }
          dir = parent
      }
  }
  ```

- [ ] **Step 2: 实现 DMG 自动安装流程**

  `Install(dmgPath string) (string, error)`：
  1. 挂载 DMG：`hdiutil attach -nobrowse -readonly <dmg>`。
  2. 在挂载点查找 `MedMemo.app`。
  3. 确定目标目录：`/Applications/MedMemo.app` 或 `~/Applications/MedMemo.app`。
  4. 备份现有 `.app` 到 `~/medmemo_backup`。
  5. 若目标目录可写，直接 `cp -R` 替换；若不可写，使用 `osascript -e 'do shell script "cp -R ..." with administrator privileges'`。
  6. 若授权失败/用户取消：将 DMG 复制到 `~/Downloads/MedMemo-<version>.dmg`，返回 `ManualInstallRequired` 类型错误。
  7. 卸载 DMG。

- [ ] **Step 3: 定义 `ManualInstallRequired` 错误类型**

  在 `installer_darwin.go` 中：
  ```go
  type ManualInstallRequired struct {
      DMGPath string
  }
  func (e *ManualInstallRequired) Error() string {
      return fmt.Sprintf("manual install required: open %s", e.DMGPath)
  }
  ```

- [ ] **Step 4: 补充 macOS 测试**

  新增 `internal/infrastructure/updater/installer_darwin_test.go`（`//go:build darwin`）：
  - `TestResolveAppBundlePath`
  - `TestFindAppInMountedDMG`
  - `TestDarwinInstaller_WritableTargetAutoReplace`
  - `TestDarwinInstaller_AdminAuthorizationFailureFallback`
  - `TestDarwinInstaller_RollbackRestoresAppBundle`

  使用可注入的 `cmdRunner` 接口以 mock `hdiutil`/`osascript`/`cp` 命令。

- [ ] **Step 5: 运行测试**

  ```bash
  GOTOOLCHAIN=auto GOOS=darwin GOARCH=arm64 go test -c ./internal/infrastructure/updater
  GOTOOLCHAIN=auto go test ./internal/infrastructure/updater -v -run 'Darwin|Bundle|DMG'
  ```

- [ ] **Step 6: 提交**

  ```bash
  git add internal/infrastructure/updater/installer_darwin.go internal/infrastructure/updater/installer_darwin_test.go
  git commit -m "fix(M01): macOS auto-replace .app with manual DMG fallback (Issue #137)"
  ```

---

## Task 4: Windows per-user / all-users 安装升级

**Files:**
- Modify: `build/windows/installer/project.nsi`
- Modify: `internal/infrastructure/updater/installer_windows.go`
- Create: `internal/infrastructure/updater/installer_windows_test.go`

- [ ] **Step 1: 修改 NSIS 脚本写入安装路径注册表**

  在 `build/windows/installer/project.nsi` 的 `Section` 末尾、卸载区段增加：
  - per-user 模式（`SetShellVarContext current`）写入 `HKCU "Software\MedMemo" "InstallPath" $INSTDIR`。
  - all-users 模式（`SetShellVarContext all`）写入 `HKLM "Software\MedMemo" "InstallPath" $INSTDIR`。
  - 卸载时删除对应注册表项。
  - 安装完成后启动 `$INSTDIR\MedMemo.exe`。

- [ ] **Step 2: 实现 Windows 安装路径解析**

  在 `installer_windows.go` 中新增：
  ```go
  func resolveInstallDir(currentExe string) (string, error) {
      currentDir := filepath.Dir(currentExe)
      hkcu, _ := readInstallPath(registry.CURRENT_USER, `Software\MedMemo`)
      hklm, _ := readInstallPath(registry.LOCAL_MACHINE, `Software\MedMemo`)
      if hkcu != "" && filepath.Dir(hkcu) == currentDir {
          return currentDir, nil
      }
      if hklm != "" && filepath.Dir(hklm) == currentDir {
          return currentDir, nil
      }
      if hkcu != "" {
          return filepath.Dir(hkcu), nil
      }
      if hklm != "" {
          return filepath.Dir(hklm), nil
      }
      return currentDir, nil
  }
  ```

- [ ] **Step 3: 修改 `WindowsInstaller.Install` 使用解析目录启动静默安装**

  命令：
  ```go
  cmd := exec.Command(installerPath, "/S", "/D="+installDir)
  ```
  确保 `/D=` 为最后一个参数。

- [ ] **Step 4: 补充 Windows 测试**

  新增 `internal/infrastructure/updater/installer_windows_test.go`（`//go:build windows`）：
  - `TestResolveInstallDir_PerUserHKCU`
  - `TestResolveInstallDir_AllUsersHKLM`
  - `TestResolveInstallDir_BothPreferCurrentExecutablePath`
  - `TestWindowsInstaller_CommandUsesResolvedInstallDir`
  - `TestWindowsInstaller_DArgumentIsLast`

- [ ] **Step 5: 运行测试**

  ```bash
  GOTOOLCHAIN=auto GOOS=windows GOARCH=amd64 go test -c ./internal/infrastructure/updater
  GOTOOLCHAIN=auto go test ./internal/infrastructure/updater -v -run 'Windows|InstallDir'
  ```

- [ ] **Step 6: 提交**

  ```bash
  git add build/windows/installer/project.nsi internal/infrastructure/updater/installer_windows.go internal/infrastructure/updater/installer_windows_test.go
  git commit -m "fix(M01): Windows installer targets per-user/all-users install dir (Issue #137)"
  ```

---

## Task 5: 平台重启 helper 与 Wails 绑定更新

**Files:**
- Create: `update_restart_linux.go`
- Create: `update_restart_darwin.go`
- Create: `update_restart_windows.go`
- Modify: `wails_app_update.go`
- Modify: `wails_app_update_test.go`

- [ ] **Step 1: 定义 `restartAfterUpdate` 函数**

  每个平台文件实现：
  ```go
  func restartAfterUpdate(ctx context.Context, installedPath string) error { ... }
  ```

  - Linux：启动新 AppImage（`exec.Command(installedPath)`）后 `runtime.Quit(ctx)`。
  - macOS：`exec.Command("open", installedPath)` 后 `runtime.Quit(ctx)`。
  - Windows：直接 `runtime.Quit(ctx)`，由 NSIS 安装完成后启动。

- [ ] **Step 2: 修改 `WailsApp.ApplyUpdate`**

  在 `wails_app_update.go` 中：
  ```go
  func (a *WailsApp) ApplyUpdate(assetPath string) error {
      if a.updaterSvc == nil {
          return fmt.Errorf("updater service not initialized")
      }
      installedPath, err := a.updaterSvc.ApplyUpdate(assetPath)
      if err != nil {
          return fmt.Errorf("failed to apply update: %w", err)
      }
      if err := restartAfterUpdate(a.ctx, installedPath); err != nil {
          return fmt.Errorf("update installed but failed to restart: %w", err)
      }
      return nil
  }
  ```

- [ ] **Step 3: 补充 Wails 测试**

  在 `wails_app_update_test.go` 中新增：
  - `TestApplyUpdate_InstallSuccess_Restarts`
  - `TestApplyUpdate_RestartFails_ReturnsError`
  - `TestApplyUpdate_InstallFails_DoesNotRestart`

  由于 `runtime.Quit` 在测试中无法 mock，将 restart helper 设计为返回错误；测试通过构造文件让 helper 走到退出前一步骤。

- [ ] **Step 4: 运行测试**

  ```bash
  GOTOOLCHAIN=auto go test . -v -run 'ApplyUpdate'
  ```

- [ ] **Step 5: 提交**

  ```bash
  git add update_restart_linux.go update_restart_darwin.go update_restart_windows.go wails_app_update.go wails_app_update_test.go
  git commit -m "feat(M01): restart helper and Wails ApplyUpdate use installed path (Issue #137)"
  ```

---

## Task 6: 前端更新体验

**Files:**
- Modify: `web/src/hooks/useUpdate.ts`
- Modify: `web/src/components/UpdateModal.tsx`
- Create: `web/src/components/UpdateModal.test.tsx`

- [ ] **Step 1: `useUpdate` 新增 `isRestarting` 状态**

  ```ts
  const [isRestarting, setIsRestarting] = useState(false)
  ```

  `doApply` 改为：
  ```ts
  const doApply = useCallback(async () => {
      if (!downloadPath) return
      setIsRestarting(true)
      setError('')
      try {
          await applyUpdate(downloadPath)
      } catch (err) {
          setIsRestarting(false)
          const msg = err instanceof Error ? err.message : String(err)
          setError(msg)
      }
  }, [downloadPath, applyUpdate])
  ```

- [ ] **Step 2: `UpdateModal` 显示“立即重启完成更新”与禁用状态**

  下载完成后：
  ```tsx
  {downloadPath && !isDownloading && (
      <div className="flex items-center gap-2 rounded-md bg-green-50 p-3 text-sm text-green-800 ...">
          <CheckCircle2 size={16} />
          <span>下载完成，立即重启完成更新</span>
      </div>
  )}
  ```

  按钮禁用：
  ```tsx
  <Button size="sm" onClick={onApply} disabled={isRestarting}>
      {isRestarting ? <Loader2 className="animate-spin" /> : <CheckCircle2 />}
      {isRestarting ? '重启中...' : '立即重启完成更新'}
  </Button>
  ```

- [ ] **Step 3: macOS 手动安装 fallback 文案**

  当 `error` 包含 `"manual install"` 时显示：
  ```tsx
  <div>macOS 自动替换需要管理员授权。请打开下载的 DMG 文件手动拖拽 MedMemo.app 到 Applications。</div>
  ```

- [ ] **Step 4: 创建 `UpdateModal.test.tsx`**

  使用 `web/src/test/render.tsx` 与 `userEvent`。

  测试：
  - `shows restart prompt after download`
  - `disables apply button while restarting`
  - `shows manual install hint for macOS authorization failure`

- [ ] **Step 5: 运行前端测试**

  ```bash
  cd web && npm run test -- UpdateModal
  ```

- [ ] **Step 6: 提交**

  ```bash
  git add web/src/hooks/useUpdate.ts web/src/components/UpdateModal.tsx web/src/components/UpdateModal.test.tsx
  git commit -m "feat(M01): frontend restart flow and manual install fallback (Issue #137)"
  ```

---

## Task 7: 开发日志与 Changelog

**Files:**
- Modify: `medmemo/开发日志/v1.1/v1.1.8.md`
- Modify: `internal/domain/entity/changelog/zh-Hans.json`

- [ ] **Step 1: 在 `v1.1.8.md` 追加 Issue #137 阶段**

  新增阶段 7：跨平台自动更新修复，包含：
  - Linux AppImage 路径解析与原子替换
  - macOS DMG 自动替换 + 手动 fallback
  - Windows per-user/all-users 注册表路径识别
  - Service 返回安装路径 + 平台 restart helper
  - 前端“立即重启完成更新”与禁用状态
  - 测试文件清单
  - 手动验收结果（待测）

- [ ] **Step 2: 更新 `zh-Hans.json` v1.1.8 fixes**

  在 v1.1.8 条目的 `fixes` 数组追加：
  - "修复 Linux AppImage 自动更新误替换 /tmp/.mount_* 运行时文件的问题"
  - "修复 macOS DMG 更新仅提示下载、无法自动替换 .app 的问题，授权失败时回退手动安装"
  - "修复 Windows 安装程序 per-user 与 all-users 升级路径混淆的问题"

- [ ] **Step 3: 提交**

  ```bash
  git add medmemo/开发日志/v1.1/v1.1.8.md internal/domain/entity/changelog/zh-Hans.json
  git commit -m "docs(M01): log Issue #137 cross-platform update fixes (Issue #137)"
  ```

---

## Task 8: 全量验证

- [ ] **Step 1: Go 单元测试**

  ```bash
  GOTOOLCHAIN=auto go test ./internal/infrastructure/updater ./internal/application/updater ./internal/adapters/updater . -race
  ```

- [ ] **Step 2: 跨平台编译检查**

  ```bash
  GOTOOLCHAIN=auto GOOS=windows GOARCH=amd64 go test -c ./internal/infrastructure/updater
  GOTOOLCHAIN=auto GOOS=darwin GOARCH=arm64 go test -c ./internal/infrastructure/updater
  GOTOOLCHAIN=auto GOOS=linux GOARCH=amd64 go test -c ./internal/infrastructure/updater
  rm -f updater.test.exe updater.test
  ```

- [ ] **Step 3: 前端构建与测试**

  ```bash
  cd web && npm run build && npm run test
  ```

- [ ] **Step 4: 格式与静态检查**

  ```bash
  GOTOOLCHAIN=auto gofmt -w .
  GOTOOLCHAIN=auto go vet ./...
  git diff --check
  ```

- [ ] **Step 5: 提交验证结果（如全部通过则推送）**

  若发现失败，先修复再提交；全部通过后：
  ```bash
  git push origin hotfix/issue-137-cross-platform-update-v1.1.8
  ```

---

## Execution Options

**Plan complete and saved to `docs/superpowers/plans/2026-06-26-issue-137-cross-platform-update.md`.**

Two execution options:

1. **Subagent-Driven (recommended)** — Dispatch a fresh subagent per task, with spec + code-quality review between tasks.
2. **Inline Execution** — Execute tasks in this session using `executing-plans`, batch execution with checkpoints.

Which approach would you like? (Recommended: Subagent-Driven)
