# MedMemo User Guide

> 🌐 [中文版本](./../i18n/zh-Hans-CN/user-guide/README.md)

> Welcome to MedMemo — your private, open-source desktop health companion. This guide helps you get the most out of MedMemo, from installation to daily use.

---

## What MedMemo Does

MedMemo is an **open-source desktop health information tool** that helps you navigate health questions through natural conversation with AI. It is **not a medical device** and does not provide diagnoses, treatment advice, or prescriptions.

All your data stays on your device. Conversations are encrypted locally, and sensitive information is automatically anonymized before any cloud API calls.

---

## Quick Navigation

| Document                                | Description                                                           |
|-----------------------------------------|-----------------------------------------------------------------------|
| [Installation Guide](./installation.md) | Download and install MedMemo on Windows, macOS, or Linux              |
| [Getting Started](./getting-started.md) | Core features: chat, conversation management, settings, shortcuts     |
| [Privacy Policy](./privacy-policy.md)   | What data we collect, how it's stored, and your rights                |
| [FAQ](./faq.md)                         | Frequently asked questions (20+) covering installation, chat, privacy |
| [Troubleshooting](./troubleshooting.md) | Common issues and step-by-step solutions                              |

---

## First-Time Setup (3 Steps)

When you open MedMemo for the first time, you'll go through a quick onboarding:

1. **Disclaimer** — Review and accept the medical disclaimer (required to continue)
2. **Privacy Settings** — Choose your desensitization level and data retention period
3. **Model Setup** — Enter your API key for your preferred AI provider (Kimi, OpenAI, etc.)

> 💡 You can re-run the onboarding anytime from **Settings → Onboarding → Re-run**.

---

## Daily Workflow at a Glance

```
Open MedMemo → Select or create a conversation → Type your question →
Review AI response (with compliance notices) → Manage conversations in sidebar
```

---

## Key Concepts

### Conversation
A conversation is a single chat thread. You can create unlimited conversations, rename them, search through them, and delete them. Deleted conversations go to the **Trash** for 30 days before permanent removal.

### Compliance System
MedMemo has a built-in compliance engine that monitors both your inputs and AI outputs for potentially risky health content. Depending on the risk level, you may see:
- **Blue info bar** — General disclaimer on every new conversation
- **Orange warning** — Content that needs extra caution
- **Red alert banner** — Emergency symptoms detected (B-level)
- **Full-screen red dialog** — Critical emergency symptoms (A-level)

### Desensitization
Before sending your messages to cloud AI providers, MedMemo automatically replaces sensitive personal information (names, phone numbers, IDs) with placeholders. The original text is never sent over the network.

### Models
MedMemo supports multiple AI providers:
- **Kimi Lite** (Moonshot) — Fast, Chinese-optimized
- **GPT-4o Mini** (OpenAI) — Balanced performance
- **Tongyi Qwen Turbo** (Alibaba) — Chinese-optimized
- **Llama 3.1 8B** (Local via Ollama) — Runs entirely on your machine

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Enter` | Send message |
| `Shift + Enter` | Insert new line |
| `Escape` | Clear input |
| `↑` (empty input) | Edit last user message |
| `Ctrl / Cmd + N` | New conversation |
| `/new` | New conversation (typed in input) |
| `Ctrl + /` | Focus input box |

See [Getting Started → Keyboard Shortcuts](./getting-started.md#keyboard-shortcuts) for the full list.

---

## Getting Help

- **Documentation**: You're reading it! Check the navigation table above.
- **GitHub Issues**: [github.com/hzhan516/medmemo/issues](https://github.com/hzhan516/medmemo/issues)
- **Security**: See [docs/SECURITY.md](../SECURITY.md) for vulnerability disclosure

---

*Last updated: 2026-05-19*
