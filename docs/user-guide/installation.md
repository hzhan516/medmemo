# Installation Guide

> 🌐 [中文版本](./../i18n/zh-Hans-CN/user-guide/installation.md)

> This guide walks you through installing MedMemo on Windows, macOS, and Linux.

---

## System Requirements

| Platform | Minimum Version | Architecture | Disk Space |
|----------|----------------|--------------|------------|
| Windows | Windows 10 | x64, ARM64 | 250 MB |
| macOS | macOS 12 (Monterey) | Intel, Apple Silicon | 200 MB |
| Linux | Ubuntu 20.04+ / Fedora 38+ | x64, ARM64 | 200 MB |

> **Note for Windows users:** On first launch, MedMemo may automatically download ~55 MB of local AI runtime libraries (ONNX Runtime + Tokenizers) if they are not bundled with the installer. This requires an internet connection during the first run.

**Additional requirements:**
- Internet connection (for downloading AI models and cloud API access)
- An API key from at least one supported provider (Kimi, OpenAI, Alibaba, or a local Ollama instance)

---

## Download

Download the latest release from [GitHub Releases](https://github.com/hzhan516/medmemo/releases).

| Platform | File | Notes |
|----------|------|-------|
| Windows | `MedMemoSetup.exe` | NSIS per-user installer |
| macOS (Intel) | `MedMemo_x86_64.dmg` | Drag-and-drop installer for Intel Macs |
| macOS (Apple Silicon) | `MedMemo_arm64.dmg` | Drag-and-drop installer for M1/M2/M3 Macs |
| Linux (AppImage) | `MedMemo-x86_64.AppImage` | Portable, no installation needed |
| Linux (DEB) | `MedMemo_*.deb` | Debian/Ubuntu package |
| Linux (RPM) | `MedMemo-*.rpm` | Fedora/openSUSE package |

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

### Step 3: Local AI Model Setup (First Launch)

On first launch, MedMemo checks for local AI runtime libraries required for semantic memory retrieval and sensitive information detection:

- **If libraries are bundled** — Setup continues automatically.
- **If libraries need to be downloaded** — A brief download (~55 MB) will start automatically. This usually takes 10–30 seconds depending on your connection.

> 💡 The downloaded libraries are saved to `%LOCALAPPDATA%\medmemo\lib\` and reused on subsequent launches. No re-download is needed unless you update MedMemo.

### Step 4: Complete Onboarding

The first launch will show the onboarding wizard. See [First-Time Setup](./README.md#first-time-setup-3-steps).

### Uninstall

1. Open **Settings → Apps → Installed apps**
2. Find **MedMemo**
3. Click **Uninstall**
4. Optional: Delete user data folder at `%LOCALAPPDATA%\medmemo` (this also removes downloaded AI libraries)

---

## macOS

### Step 1: Open the DMG

1. Download the DMG matching your Mac:
   - **Intel Mac:** `MedMemo_x86_64.dmg`
   - **Apple Silicon Mac (M1/M2/M3):** `MedMemo_arm64.dmg`
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

1. Download `MedMemo-x86_64.AppImage`
2. Make it executable:
   ```bash
   chmod +x MedMemo-x86_64.AppImage
   ```
3. Run it:
   ```bash
   ./MedMemo-x86_64.AppImage
   ```

> **Fedora 43+ users**: If you encounter WebKit issues, ensure `webkit2gtk4.1` is installed:
> ```bash
> sudo dnf install webkit2gtk4.1
> ```

### DEB Package (Debian/Ubuntu)

1. Download `MedMemo_*.deb`
2. Install with your package manager:
   ```bash
   sudo dpkg -i MedMemo_*.deb
   sudo apt-get install -f  # if dependencies are missing
   ```
3. Launch MedMemo from the applications menu or run `medmemo` in a terminal.

### RPM Package (Fedora/openSUSE)

1. Download `MedMemo-*.rpm`
2. Install with your package manager:
   ```bash
   sudo dnf install MedMemo-*.rpm
   # or on openSUSE:
   sudo zypper install MedMemo-*.rpm
   ```
3. Launch MedMemo from the applications menu or run `medmemo` in a terminal.

### Updating DEB/RPM Installations

When MedMemo detects a newer version, it downloads the matching `.deb` or `.rpm` package and shows the downloaded file path. Install the update manually with the same command used for the initial installation.

### Build from Source

If you prefer to build from source:

```bash
# 1. Prerequisites
go version  # Requires Go 1.26.4+
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

- **AppImage:** Delete the AppImage file.
- **DEB:** `sudo dpkg -r medmemo`
- **RPM:** `sudo dnf remove medmemo`

(Optional) Remove user data at `~/.medmemo/data` after uninstalling any Linux package.

---

## Post-Installation Checklist

After installing, verify everything works:

- [ ] MedMemo launches without errors
- [ ] Onboarding wizard appears on first launch
- [ ] You can accept the disclaimer and reach the main chat screen
- [ ] Sidebar shows a "New Conversation" button
- [ ] You can type a message in the input box
- [ ] Settings page opens (gear icon in header)

### Windows-Specific Issues

#### "Failed to initialize local AI models" or download error

If the automatic library download fails on first launch:

1. **Check your internet connection** — The download requires a stable connection.
2. **Check Windows Defender / antivirus** — They may block the download. Temporarily disable or add MedMemo to the allowlist.
3. **Manual download** (advanced users):
   ```powershell
   # Run PowerShell as Administrator in the MedMemo installation directory
   .\scripts\build\download-onnx.ps1 -Platform windows
   .\scripts\build\download-tokenizers.ps1
   ```
4. **Firewall / corporate proxy** — If behind a restrictive proxy, set the proxy environment variable before launching:
   ```powershell
   $env:HTTP_PROXY = "http://proxy.company.com:8080"
   ```

#### Embedding features unavailable on Windows

If you see "semantic search unavailable" or "embedding engine not available":

- This means the ONNX Runtime or Tokenizers library failed to load.
- MedMemo will automatically fall back to keyword-based memory retrieval, which works without embedding.
- Core chat functionality is **not affected**.

If any other step fails, see [Troubleshooting](./troubleshooting.md).

---

## Updating MedMemo

MedMemo supports automatic update checks:

1. Go to **Settings → Auto-update**
2. Enable **"Check for updates automatically"**
3. Choose your channel:
   - **Stable** — Official releases only
   - **Beta** — Includes pre-releases with new features

When an update is available, a notification appears.

- **Windows / macOS / Linux AppImage:** The app downloads and installs the update automatically when possible.
- **Linux DEB / RPM:** The app downloads the matching package and shows the file path. Install it manually with the same command used for the initial installation.

> ⚠️ Security updates may be marked as **mandatory** and block usage until installed.

---

*Last updated: 2026-07-09*
