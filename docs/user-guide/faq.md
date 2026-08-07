# Frequently Asked Questions

> 🌐 [中文版本](./../i18n/zh-Hans-CN/user-guide/faq.md)

> Answers to common questions about installing, configuring, and using MedMemo.

---

## Installation

### Q1: Which platforms does MedMemo support?

**A:** Windows 10+, macOS 12+, and Linux (Ubuntu 20.04+, Fedora 38+). Both x64 and ARM64 architectures are supported.

### Q2: Do I need an API key to use MedMemo?

**A:** Yes, unless you use a local model via Ollama. Cloud providers (Kimi, OpenAI, Alibaba) require an API key. You can get free-tier keys from most providers.

### Q3: How do I get a Kimi API key?

**A:** Visit [platform.moonshot.cn](https://platform.moonshot.cn), sign up, and create an API key in your dashboard.

### Q4: How do I get an OpenAI API key?

**A:** Visit [platform.openai.com](https://platform.openai.com), sign up, and go to **API Keys** to create one.

### Q5: Can I run MedMemo without internet?

**A:** Partially. You need internet for the initial download and if using cloud AI models. However, if you set up a local Ollama model, MedMemo works fully offline after installation.

### Q6: Why does macOS block MedMemo on first launch?

**A:** MedMemo is an open-source app and is not notarized by Apple. Go to **System Settings → Privacy & Security → Open Anyway** to allow it. See [Installation Guide → macOS](./installation.md#macos) for details.

### Q7: How much disk space does MedMemo need?

**A:** About 200 MB total — ~50 MB for the app binary, plus local database and optional AI models.

---

## Configuration

### Q8: How do I switch AI models?

**A:** Go to **Settings → Default Model** and click on your preferred provider. The change applies to new conversations.

### Q9: Can I use different models for different conversations?

**A:** Not in the current version. The default model applies to all new conversations. This feature is planned for a future release.

### Q10: Where is my API key stored?

**A:** In your operating system's secure keychain — macOS Keychain, Windows Credential Manager, or Linux Secret Service. It is never stored in plain text.

### Q11: How do I change the theme?

**A:** Go to **Settings → Appearance** and choose Light, Dark, or System. Changes apply immediately without restart.

### Q12: What is the "compliance bar" and should I turn it off?

**A:** The compliance bar is a blue info strip reminding you that AI output is for reference only, not medical advice. We recommend keeping it on **Always Show** for safety awareness. You can change it in **Settings → Compliance Bar**.

---

## Chat & Conversation

### Q13: Is there a limit to how long a conversation can be?

**A:** There is no hard limit, but very long conversations may be summarized to fit within the AI model's context window. Earlier messages are compressed while preserving key information.

### Q14: Can I export my conversations?

**A:** Export functionality (JSON/Markdown) is planned for a future release. For now, conversations are stored locally in an encrypted SQLite database.

### Q15: What happens if I close MedMemo mid-conversation?

**A:** Your conversation is automatically saved. When you reopen MedMemo, you'll see the conversation in the sidebar with all messages intact.

### Q16: Why did the AI stop responding halfway?

**A:** Possible reasons:
- You clicked the **stop button**
- Network timeout (check your connection)
- The AI provider's rate limit was hit
- A compliance L1 block was triggered (high-risk content detected)

### Q17: Can I edit a message after sending it?

**A:** You can press `↑` (Up Arrow) in an empty input box to recall your last message, edit it, and resend. There is no inline editing of past messages.

### Q18: What does `/new` do?

**A:** Typing `/new` in the input box and pressing `Enter` creates a new conversation. It's a quick alternative to the sidebar button or `Ctrl+N` shortcut.

---

## Privacy

### Q19: Does MedMemo upload my conversations to the cloud?

**A:** No. Conversations are stored locally in an encrypted database. Only the **desensitized** text is sent to AI providers, and sensitive information is replaced with placeholders before sending.

### Q20: What is "desensitization"?

**A:** It's an automatic process that replaces sensitive personal information (names, phone numbers, IDs, addresses) with placeholders like `<NAME_1>` or `<PHONE_1>` before sending to cloud AI. The original text is restored locally after receiving the response.

> **Important:** Desensitization protects only the content sent to cloud AI providers. The chat interface and local data always remain the original text.

### Q21: Can MedMemo staff read my conversations?

**A:** No. All data is stored locally on your device. There is no remote server or cloud storage for conversations.

### Q22: How do I completely delete all my data?

**A:** Uninstall MedMemo and delete the user data folder:
- **Windows**: see [Installation Guide → Where Your Windows Data Is Stored](./installation.md#where-your-windows-data-is-stored). Common locations are `%USERPROFILE%\.medmemo\data` or `<installDir>\data` (for example, `%LOCALAPPDATA%\Programs\MedMemo\data`).
- **macOS**: `~/.medmemo/data`
- **Linux**: `~/.medmemo/data`

### Q23: Why is my Windows data in a different folder after upgrading?

**A:** Starting with v1.1.10, MedMemo uses the following priority on Windows:
1. If `%USERPROFILE%\.medmemo\data\medmemo.db` exists, that legacy folder is used so your existing data stays visible.
2. Otherwise, if the install directory is writable, MedMemo uses `<installDir>\data`.
3. If the install directory is not writable, MedMemo falls back to `%USERPROFILE%\.medmemo\data`.

To force a specific folder, set `MEDMEMO_DATA_DIR` or `data_dir` in `config.yaml`.

### Q24: Does MedMemo collect analytics?

**A:** No telemetry or usage analytics are collected. The only "analytics" are local onboarding wizard completion stats (stored on your device only) to help improve the first-time experience.

---

## Troubleshooting

### Q25: The app won't start after installation. What should I do?

**A:** See [Troubleshooting → Installation Issues](./troubleshooting.md#installation-issues).

### Q26: Why do I see an orange warning on some AI responses?

**A:** The compliance engine detected potentially sensitive health content. The warning reminds you to consult a doctor for medical decisions.

### Q27: What should I do if I see a red emergency alert?

**A:** Take it seriously. A-level alerts indicate potentially life-threatening symptoms. Consider calling emergency services or visiting urgent care. You can also choose to continue the conversation if you believe it's a false positive.

### Q28: The AI response seems slow. How can I speed it up?

**A:** Try switching to a faster model (Kimi Lite or GPT-4o Mini). Local models (Ollama) depend on your hardware — a GPU significantly improves speed.

---

## Still Have Questions?

- [GitHub Issues](https://github.com/hzhan516/medmemo/issues) — Ask the community
- [Installation Guide](./installation.md) — Step-by-step setup
- [Getting Started](./getting-started.md) — Feature walkthrough
- [Privacy Policy](./privacy-policy.md) — Data handling details

---

*Last updated: 2026-08-05*
