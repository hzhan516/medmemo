# Installation Guide

> 🌐 [中文版本](./../i18n/zh-Hans-CN/user-guide/installation.md)

> This guide walks you through installing MedMemo on Windows, macOS, and Linux.

---

## System Requirements

| Platform | Minimum Version | Architecture | Disk Space |
|----------|----------------|--------------|------------|
| Windows | Windows 10 | x64, ARM64 | 200 MB |
| macOS | macOS 12 (Monterey) | Intel, Apple Silicon | 200 MB |
| Linux | Ubuntu 20.04+ / Fedora 38+ | x64, ARM64 | 200 MB |

**Additional requirements:**
- Internet connection (for downloading AI models and cloud API access)
- An API key from at least one supported provider (Kimi, OpenAI, Alibaba, or a local Ollama instance)

---

## Download

Download the latest release from [GitHub Releases](https://github.com/hzhan516/medmemo/releases).

| Platform | File | Notes |
|----------|------|-------|
| Windows | `MedMemoSetup.exe` | NSIS installer |
| macOS | `MedMemo.dmg` | Drag-and-drop installer |
| Linux | `MedMemo.AppImage` | Portable, no installation needed |

---

## Windows

### Step 1: Run the Installer

1. Download `MedMemoSetup.exe`
2. Double-click to run
3. If Windows SmartScreen appears, click **More info → Run anyway**
4. Follow the setup wizard
5. Choose whether to create a Start Menu shortcut and desktop icon

### Step 2: Launch MedMemo

- From the Start Menu: **MedMemo**
- From the desktop shortcut (if selected during install)

### Step 3: Complete Onboarding

The first launch will show the onboarding wizard. See [First-Time Setup](./README.md#first-time-setup-3-steps).

### Uninstall

1. Open **Settings → Apps → Installed apps**
2. Find **MedMemo**
3. Click **Uninstall**
4. Optional: Delete user data folder at `%LOCALAPPDATA%\medmemo`

---

## macOS

### Step 1: Open the DMG

1. Download `MedMemo.dmg`
2. Double-click to mount the disk image
3. Drag **MedMemo.app** into your **Applications** folder

### Step 2: First Launch

macOS may block the app on first launch because it is not notarized (expected for open-source projects).

**Option A — System Settings:**
1. Try to open MedMemo (it will be blocked)
2. Open **System Settings → Privacy & Security**
3. Scroll down and click **Open Anyway** next to the MedMemo block message

**Option B — Terminal:**
```bash
xattr -d com.apple.quarantine /Applications/MedMemo.app
```

### Step 3: Complete Onboarding

See [First-Time Setup](./README.md#first-time-setup-3-steps).

### Uninstall

1. Drag **MedMemo.app** from Applications to Trash
2. Optional: Delete user data at `~/Library/Application Support/medmemo`

---

## Linux

### AppImage (Recommended)

1. Download `MedMemo.AppImage`
2. Make it executable:
   ```bash
   chmod +x MedMemo.AppImage
   ```
3. Run it:
   ```bash
   ./MedMemo.AppImage
   ```

> **Fedora 43+ users**: If you encounter WebKit issues, ensure `webkit2gtk4.1` is installed:
> ```bash
> sudo dnf install webkit2gtk4.1
> ```

### Build from Source

If you prefer to build from source:

```bash
# 1. Prerequisites
go version  # Requires Go 1.22+
node --version  # Requires Node.js 18+

# 2. Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 3. Clone and build
git clone https://github.com/hzhan516/medmemo.git
cd medmemo
go mod download
cd web && npm install && cd ..
make build
```

The binary will be at `build/bin/medmemo`.

> **Fedora 43+**: Add `-tags webkit2_41` to the build flags.

### Uninstall

1. Delete the AppImage file
2. (Optional) Remove user data at `~/.medmemo/data`

---

## Post-Installation Checklist

After installing, verify everything works:

- [ ] MedMemo launches without errors
- [ ] Onboarding wizard appears on first launch
- [ ] You can accept the disclaimer and reach the main chat screen
- [ ] Sidebar shows a "New Conversation" button
- [ ] You can type a message in the input box
- [ ] Settings page opens (gear icon in header)

If any step fails, see [Troubleshooting](./troubleshooting.md).

---

## Updating MedMemo

MedMemo supports automatic update checks:

1. Go to **Settings → Auto-update**
2. Enable **"Check for updates automatically"**
3. Choose your channel:
   - **Stable** — Official releases only
   - **Beta** — Includes pre-releases with new features

When an update is available, a notification appears. You can download and install directly from the app.

> ⚠️ Security updates may be marked as **mandatory** and block usage until installed.

---

*Last updated: 2026-05-19*
