# Troubleshooting

> 🌐 [中文版本](./../../i18n/zh-Hans-CN/user-guide/troubleshooting.md)

> Common issues and their solutions. If your problem isn't listed here, please [open a GitHub issue](https://github.com/hzhan516/medmemo/issues).

---

## Installation Issues

### App Won't Start (Windows)

**Symptoms:** Double-clicking the installer or app does nothing, or an error dialog appears.

**Solutions:**
1. Check that your Windows version is 10 or later (`Win + R`, type `winver`)
2. Try running as regular user (not Administrator)
3. If SmartScreen blocks it: click **More info → Run anyway**
4. Check Windows Defender exclusions — add MedMemo if falsely flagged
5. Re-download the installer from GitHub Releases (file may be corrupted)

### App Won't Start (macOS)

**Symptoms:** "MedMemo cannot be opened because the developer cannot be verified."

**Solutions:**
1. Open **System Settings → Privacy & Security**
2. Scroll down and click **Open Anyway**
3. If that doesn't work, run in Terminal:
   ```bash
   xattr -d com.apple.quarantine /Applications/MedMemo.app
   ```
4. Ensure macOS 12 (Monterey) or later

### App Won't Start (Linux)

**Symptoms:** AppImage doesn't launch, or shows library errors.

**Solutions:**
1. Make it executable: `chmod +x MedMemo.AppImage`
2. Install missing dependencies:
   - Ubuntu/Debian: `sudo apt install libwebkit2gtk-4.1-0`
   - Fedora: `sudo dnf install webkit2gtk4.1`
3. If using Wayland, try launching with: `./MedMemo.AppImage --enable-features=UseOzonePlatform --ozone-platform=wayland`
4. Check AppImage compatibility with your distribution version

---

## AI Model Issues

### "Failed to send message" Error

**Symptoms:** Red error bar appears when sending a message.

**Solutions:**
1. Check your internet connection
2. Verify your API key is correct in **Settings** (re-enter if unsure)
3. Check the AI provider's status page (e.g., [OpenAI Status](https://status.openai.com))
4. Try switching to a different model in **Settings → Default Model**
5. If using Ollama, ensure it's running: `ollama serve`

### AI Responses Are Very Slow

**Symptoms:** Long delay before first word appears, or streaming is choppy.

**Solutions:**
1. Switch to a faster model (Kimi Lite is typically fastest for Chinese)
2. Check your network speed (cloud models need stable internet)
3. If using Ollama locally:
   - Ensure your machine has sufficient RAM (8GB+ recommended)
   - A GPU significantly speeds up local inference
   - Try a smaller model (e.g., Llama 3.1 8B instead of 70B)
4. Close other resource-heavy applications

### "Model Not Available" Error

**Symptoms:** Error saying the selected model is unavailable.

**Solutions:**
1. For cloud models: verify API key and internet connection
2. For Ollama: ensure the model is downloaded (`ollama list`) and the server is running
3. Check if the provider has deprecated the model (check their documentation)

---

## Privacy & Security Issues

### Desensitization Seems Not Working

**Symptoms:** You suspect sensitive info is being sent without replacement.

**How to verify:**
1. Check **Settings → Privacy → Desensitization Level** — ensure it's not set to "Off"
2. The desensitization process happens automatically and silently
3. To confirm it's active: send a message with a fake phone number like `13800138000` — the AI response should not contain this number if it references your input

### Forgot Where My Data Is Stored

**Locations:**
- **Windows**: `%LOCALAPPDATA%\medmemo\`
- **macOS**: `~/Library/Application Support/medmemo/`
- **Linux**: `~/.local/share/medmemo/`

> ⚠️ These folders contain encrypted data. Do not manually edit them.

---

## UI & Interface Issues

### Sidebar Disappeared

**Symptoms:** No conversation list visible.

**Solutions:**
1. If the window is narrow (< 768px), the sidebar auto-collapses to icons. Click the **panel icon** to expand.
2. If fully collapsed, click the **expand arrow** at the top-left of the collapsed sidebar.
3. Resize the window wider to restore the full sidebar.

### Dark Mode Not Working Correctly

**Symptoms:** Some UI elements look wrong in dark mode.

**Solutions:**
1. Toggle theme in **Settings → Appearance** and toggle back
2. Restart MedMemo
3. If using Linux with a custom theme, MedMemo may inherit system colors unexpectedly

### Text Is Too Small / Too Large

**Symptoms:** Difficulty reading chat text.

**Solutions:**
1. Use your OS-level zoom/scaling (MedMemo respects OS display scaling)
2. Resize the window — the chat area adapts
3. Future versions will include in-app font size controls

---

## Performance Issues

### High Memory Usage

**Symptoms:** MedMemo uses 500MB+ of RAM.

**Expected behavior:**
- Base app: ~100-200 MB
- With ONNX model loaded: +100-200 MB
- Long conversations: gradual increase (capped by cleanup)

**If usage is excessive:**
1. Restart MedMemo
2. Close unused conversations (they stay in sidebar but release rendering memory)
3. If using local Ollama models, memory depends on model size (8B ≈ 6GB, 70B ≈ 40GB)

### App Feels Sluggish

**Symptoms:** Slow UI response, laggy scrolling.

**Solutions:**
1. Reduce the number of visible messages (scroll to load more)
2. Check if your system is under memory pressure
3. Disable unnecessary browser extensions (MedMemo uses a WebView)
4. On Linux, ensure GPU acceleration is available for WebKit

---

## Update Issues

### Update Check Fails

**Symptoms:** "Failed to check for updates" notification.

**Solutions:**
1. Check internet connection
2. GitHub may be rate-limiting — try again later
3. If behind a corporate proxy, configure system proxy settings
4. You can always manually download the latest release from GitHub

### Update Download Interrupted

**Symptoms:** Download stops halfway and won't resume.

**Solutions:**
1. The update will retry automatically on next launch
2. Or manually download from GitHub Releases and reinstall

---

## Reporting Issues

If none of the above solutions work:

1. **Check existing issues**: [github.com/hzhan516/medmemo/issues](https://github.com/hzhan516/medmemo/issues)
2. **Open a new issue** with:
   - MedMemo version (from **Settings** or `Help → About`)
   - Operating system and version
   - Steps to reproduce
   - Expected vs actual behavior
   - Screenshots (if UI-related)

---

*Last updated: 2026-05-19*
