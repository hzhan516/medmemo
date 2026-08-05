# Troubleshooting

> 🌐 [中文版本](./../i18n/zh-Hans-CN/user-guide/troubleshooting.md)

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

## Database Issues

### "database key verification failed"

**Symptoms:** MedMemo fails to start and logs `database key verification failed`.

**Root cause:** The OS keyring entry for the SQLCipher database key is missing, corrupted, or was created by a different user profile.

**Solutions:**
1. Do not delete or modify files in the data directory manually.
2. If you recently reinstalled the OS or cleared the system keyring, the database key is unrecoverable — restore from a backup if available.
3. As a last resort, move the old data directory aside and let MedMemo create a new one:
   - Windows: `%USERPROFILE%\.medmemo\data` (or `<installDir>\data` if you configured a custom install location)
   - macOS: `~/.medmemo/data`
   - Linux: `~/.medmemo/data`

### "failed to open sqlcipher database"

**Symptoms:** Startup fails with `failed to open sqlcipher database`.

**Root cause:** The data directory is not writable, the disk is full, or the database file is locked by another process.

**Solutions:**
1. Check disk space.
2. Ensure the user running MedMemo has write access to `MEDMEMO_DATA_DIR`.
3. Close any other MedMemo instance that may be holding the database lock.
4. Set `MEDMEMO_DATA_DIR` to a writable path if the default location is restricted.

On Windows, MedMemo prefers the legacy directory `%USERPROFILE%\.medmemo\data` when a previous database exists; otherwise it uses `<installDir>\data` if writable, or falls back to `%USERPROFILE%\.medmemo\data`. An unwritable install directory (for example, `Program Files`) is a common cause of this error on new installs.

### "failed to migrate plaintext database"

**Symptoms:** First launch after an older version shows a migration error.

**Root cause:** MedMemo is upgrading an unencrypted SQLite database to SQLCipher.

**Solutions:**
1. Make a backup of the data directory before upgrading.
2. Ensure the system keyring is accessible (Linux: Secret Service / GNOME Keyring is running).
3. If migration keeps failing, restore the backup and [open an issue](https://github.com/hzhan516/medmemo/issues).

---

## ONNX / Embedding Issues

### "ONNX Runtime library not found"

**Symptoms:** Local NER or embedding is unavailable and the log shows `ONNX Runtime library not found`.

**Root cause:** The ONNX Runtime native libraries were not downloaded or are not in `resources/lib/<platform>/`.

**Solutions:**
1. Run `make download-resources` (or `scripts/build/download-onnx.sh` on Linux/macOS, `.ps1` on Windows).
2. Verify that `resources/lib/linux/libonnxruntime.so`, `resources/lib/darwin/libonnxruntime.dylib`, or `resources/lib/windows/onnxruntime.dll` exists.
3. On Linux, ensure `LD_LIBRARY_PATH` includes `resources/lib/linux` if you are running from source.

### "embedding model dir not found" / "NER model dir not found"

**Symptoms:** Log shows `embedding model dir not found: ...` or `NER model dir not found: ...`.

**Root cause:** The bundled DistilBERT NER model is missing from `resources/models/distilbert-ner/`.

**Solutions:**
1. Run `make download-resources` to fetch the model.
2. Verify that `resources/models/distilbert-ner/model.onnx` exists.
3. If you moved the model, set `MEDMEMO_MODEL_DIR` to the new directory.

### "embedding pipeline creation failed"

**Symptoms:** The embedding feature fails with `embedding pipeline creation failed`.

**Root cause:** The ONNX model is incompatible with the ONNX Runtime library, or the tokenizer static library is missing.

**Solutions:**
1. Run `make download-resources` to ensure the tokenizer static library is present.
2. Check that the ONNX Runtime version matches the Hugot / ortgenai version in `go.mod`.
3. Delete `resources/lib/` and `resources/models/` and re-run `make download-resources`.

---

## Authentication Issues

### "需要配置 OAuth client_id"

**Symptoms:** Starting OAuth Device Flow returns an error containing `需要配置 OAuth client_id`.

**Root cause:** OAuth Device Flow requires a per-provider `client_id` registered by you. MedMemo ships without pre-registered OAuth clients.

**Solutions:**
1. Register an OAuth application with the provider.
2. Set the corresponding environment variable:
   - Kimi: `MEDMEMO_KIMI_CLIENT_ID`
   - Gemini: `MEDMEMO_GEMINI_CLIENT_ID`
   - Microsoft: `MEDMEMO_MICROSOFT_CLIENT_ID`
   - GitHub: `MEDMEMO_GITHUB_CLIENT_ID`
3. Restart MedMemo and try again.

See [`docs/api/auth.md`](../api/auth.md) for detailed prerequisites.

### CLI Token Not Detected

**Symptoms:** `DetectAuthMethods` reports `cli_token` as unavailable, or `BuildCLIProvider` fails.

**Root cause:** The provider CLI credentials file does not exist, is empty, or cannot be parsed.

**Solutions:**
1. Log in with the provider CLI (e.g., `kimi auth login`, `gcloud auth login`).
2. Check the credential file path reported by the error.
3. Common error strings:
   - `credential file is empty` — re-run the provider login command.
   - `failed to parse credential` — the credential file format may have changed; update the provider CLI.
   - `unsupported cli provider type` — the provider is not supported for CLI token detection.

### Service Account JSON Parse Error

**Symptoms:** `ParseServiceAccountJSON` returns `failed to parse service account JSON` or `invalid service account type`.

**Root cause:** The Google Service Account JSON is malformed or was downloaded for the wrong credential type.

**Solutions:**
1. Download the JSON key from Google Cloud Console → IAM & Admin → Service Accounts.
2. Ensure the file contains `"type": "service_account"`.
3. Verify that `project_id`, `client_email`, and `private_key` are present.

---

## Ollama Issues

### "ollama not reachable"

**Symptoms:** Local model detection reports `ollama not reachable`.

**Root cause:** The Ollama server is not running or is not listening on the expected URL.

**Solutions:**
1. Start Ollama: `ollama serve`
2. Verify the server URL in Settings (default: `http://localhost:11434`).
3. Check that no firewall is blocking `localhost:11434`.

### "ollama server did not become ready within ..."

**Symptoms:** MedMemo tried to start `ollama serve` automatically but timed out.

**Root cause:** Ollama is installed but takes longer than the check timeout to initialize.

**Solutions:**
1. Start Ollama manually: `ollama serve`
2. Wait until `ollama list` works, then retry in MedMemo.
3. If Ollama is not installed, download it from [ollama.com](https://ollama.com).

### "ollama pull ... failed"

**Symptoms:** Downloading a model through MedMemo fails with `ollama pull ... failed`.

**Root cause:** Network issue, disk space, or the model name is invalid.

**Solutions:**
1. Pull the model manually: `ollama pull llama3.1:8b`
2. Ensure sufficient disk space (8B models need ~6 GB, 70B models need ~40 GB).
3. Check that the model name is valid in Ollama's library.

### "ollama returned HTTP 404"

**Symptoms:** Chat with a local model returns `ollama returned HTTP 404`.

**Root cause:** The requested model is not downloaded locally.

**Solutions:**
1. Run `ollama list` to see available models.
2. Pull the missing model: `ollama pull <model>`.
3. Select a model that exists in `ollama list` in MedMemo Settings.

---

## Streaming Issues

### "stream execution failed"

**Symptoms:** A streaming response stops abruptly and the UI shows a red error bar, or logs show `stream execution failed`.

**Root cause:** Network interruption, provider error, or a panic during stream processing.

**Solutions:**
1. Check your internet connection.
2. Switch to a different model or provider in Settings.
3. If using a local model, ensure Ollama is still running.
4. Restart MedMemo and retry the conversation.

### Empty or Truncated Stream Response

**Symptoms:** The AI message bubble appears empty or only contains part of the expected text.

**Root cause:** The provider returned an empty stream, or compliance/emergency interception replaced the content.

**Solutions:**
1. Check the chat area for system notices (orange warning / blue disclaimer bars).
2. Look at the application logs for `chat:stream:compliance` events.
3. If the input triggered an emergency symptom, review the warning and decide whether to continue.

### Repeated Panic During Streaming

**Symptoms:** MedMemo crashes or becomes unresponsive during a long streaming response.

**Root cause:** A runtime panic in the stream callback path (e.g., UI race) was recovered, but the app state may be inconsistent.

**Solutions:**
1. Save any important context and restart MedMemo.
2. If reproducible, note the model, message length, and any compliance warning, then [open an issue](https://github.com/hzhan516/medmemo/issues).

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
- **Windows**: see [Installation Guide → Where Your Windows Data Is Stored](./installation.md#where-your-windows-data-is-stored). The most common locations are:
  - Legacy or fallback: `%USERPROFILE%\.medmemo\data`
  - Per-user install: `%LOCALAPPDATA%\Programs\MedMemo\data`
- **macOS**: `~/.medmemo/data`
- **Linux**: `~/.medmemo/data`

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
2. Close unused conversations (they stay in the sidebar but release rendering memory)
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

### Linux Update Shows Release Page Instead of Install Command

**Symptoms:** On Linux, the update dialog shows "Please download the package manually" and a link to the Release page instead of a `dpkg`/`rpm` command.

**Root cause:** MedMemo could not determine whether the current binary was installed from a DEB, RPM, or AppImage package. This can happen for portable builds, custom install paths, or when both `dpkg` and `rpm` are unavailable.

**Solutions:**
1. Download the correct package for your system from the Release page (DEB for Debian/Ubuntu, RPM for Fedora/openSUSE, AppImage for portable use).
2. To avoid this in the future, set the environment variable before launching MedMemo:
   ```bash
   export MEDMEMO_INSTALL_KIND=deb   # or rpm
   ```
3. Ensure the `.install_kind` marker file shipped with the DEB/RPM package is present next to the `medmemo` binary.

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

*Last updated: 2026-08-05*
