# Privacy Policy

> 🌐 [中文版本](./../i18n/zh-Hans-CN/user-guide/privacy-policy.md)

> This document explains how MedMemo handles your data. MedMemo is designed with a **privacy-first, local-first** philosophy.

---

## Our Promise

**Your health data never leaves your device unless you explicitly choose to use a cloud AI model — and even then, sensitive information is automatically removed first.**

MedMemo does not:
- Collect usage analytics or telemetry
- Store conversations on remote servers
- Share your data with third parties
- Use your data for advertising or insurance profiling
- Track your behavior across sessions

---

## What Data We Collect

### Data Stored Locally

| Data Type | Purpose | Storage Location |
|-----------|---------|-----------------|
| **Conversations** | Chat history and messages | Local SQLite database (encrypted) |
| **Session Titles** | Organize your chat threads | Local SQLite database (encrypted) |
| **Settings** | Theme, model preference, compliance mode | Local SQLite database (encrypted) |
| **API Keys** | Authentication with AI providers | OS keychain (macOS Keychain / Windows Credential Manager / Linux Secret Service) |
| **Disclaimers** | Acceptance records with timestamps | Local SQLite database (encrypted) |
| **Onboarding Analytics** | Wizard completion stats (local only) | Local storage |

### Data Sent to Cloud (Only When Using Cloud Models)

When you use a cloud AI provider (Kimi, OpenAI, Alibaba):

| What Is Sent | What Is NOT Sent |
|-------------|------------------|
| Your **desensitized** conversation text | Original names, phone numbers, IDs, addresses |
| Model type and API key (for authentication) | Family health information |
| | Usage behavior logs |

**Desensitization process:**
```
Your input
  → L1 Rule Engine (regex patterns: IDs, phones, emails)
    → L2 NER Model (names, locations, organizations)
      → Safe outbound text
        → Safe text sent to cloud API
```

The original text is restored locally after receiving the AI response. The mapping never leaves your device.

---

## How Your Data Is Protected

### Encryption

| Layer | Method | Details |
|-------|--------|---------|
| **Database** | AES-256-GCM | Full SQLite database encryption via SQLCipher |
| **API Keys** | OS Keychain | Platform-native secure storage |
| **Encryption Key** | CSPRNG + Keychain | 256-bit random key stored in OS keychain |

### Access Control

- All data files are stored in your user profile directory
- No other applications or users can access MedMemo's data without your OS credentials
- MedMemo does not run as administrator/root

### Local-Only Operations

The following operations happen entirely offline on your device:
- Emergency symptom detection
- Compliance content filtering
- Conversation management (CRUD)
- Theme switching
- Settings storage
- Markdown rendering

---

## Your Rights

As a MedMemo user, you have complete control over your data:

| Right | How To Exercise |
|-------|----------------|
| **Access** | All your data is visible in the app. Database files are in your local storage. |
| **Delete** | Delete individual conversations (Trash → permanent after 30 days) or clear all data by uninstalling |
| **Export** | Future versions will support exporting conversations as JSON/Markdown |
| **Restrict** | Set data retention period in **Settings → Privacy → Data Retention** |
| **Withdraw Consent** | Decline the disclaimer at any time (app will close). Re-open to re-accept. |

---

## Third-Party Services

MedMemo integrates with the following external services **only when you configure them:**

| Service | Data Sent | Privacy Policy |
|---------|-----------|---------------|
| Moonshot (Kimi) | Desensitized text | [moonshot.cn](https://moonshot.cn) |
| OpenAI (GPT) | Desensitized text | [openai.com/privacy](https://openai.com/privacy) |
| Alibaba (Qwen) | Desensitized text | [alibabacloud.com](https://www.alibabacloud.com) |
| GitHub Releases | Version check only | [github.com/privacy](https://github.com/privacy) |

> **Ollama (local models)**: No data is sent anywhere. Everything runs on your machine.

---

## Data Retention

You control how long MedMemo keeps your data:

| Setting | Behavior |
|---------|----------|
| **7 days** | Conversations older than 7 days are automatically deleted |
| **30 days** | Conversations older than 30 days are automatically deleted |
| **90 days** | Conversations older than 90 days are automatically deleted |
| **1 year** | Conversations older than 1 year are automatically deleted |
| **Forever** | Data is kept indefinitely until you manually delete it |

Change this in **Settings → Privacy → Data Retention**.

---

## Children's Privacy

MedMemo is not intended for use by children under 13. We do not knowingly collect data from children.

---

## Changes to This Policy

When the privacy policy is updated, you will be prompted to review and re-accept it on the next app launch.

---

## Contact

For privacy-related questions or concerns:

- Open a GitHub issue (general questions): [github.com/hzhan516/medmemo/issues](https://github.com/hzhan516/medmemo/issues)
- Email (security/privacy): doyle_zhang@outlook.com

---

*Last updated: 2026-07-09*
