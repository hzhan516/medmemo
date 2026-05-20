# Getting Started

> 🌐 [中文版本](./../i18n/zh-Hans-CN/user-guide/getting-started.md)

> Learn the core features of MedMemo: chatting, managing conversations, configuring settings, and using keyboard shortcuts.

---

## The Main Interface

MedMemo uses a split-pane layout:

```
┌─────────────────┬────────────────────────────────────┐
│                 │  Header (title + settings + theme) │
│   Sidebar       ├────────────────────────────────────┤
│  (conversation  │  Compliance Bar (blue info strip)  │
│    list,        ├────────────────────────────────────┤
│   searchable)   │                                    │
│                 │         Chat Messages              │
│   200-400px     │         (scrollable)               │
│   (resizable)   │                                    │
│                 │                                    │
│                 ├────────────────────────────────────┤
│                 │  Input Box + Send Button           │
│                 │  (shortcut hints below)            │
└─────────────────┴────────────────────────────────────┘
```

- **Sidebar** (left): Conversation list, search, trash
- **Header** (top): App title, theme toggle, settings link
- **Compliance Bar**: Blue info strip reminding you that AI output is for reference only
- **Chat Area**: Message bubbles — blue for you, white/gray for AI
- **Input Box** (bottom): Type your questions here

---

## Starting a Conversation

### Method 1: New Conversation Button
Click **"New Conversation"** at the top of the sidebar (or press `Ctrl/Cmd + N`).

### Method 2: Slash Command
Type `/new` in the input box and press `Enter`.

### Method 3: Empty State
When no conversation exists, the chat area shows a welcome screen with a **"Start Chatting"** button.

---

## Sending Messages

1. Click the input box at the bottom (or press `Ctrl + /` to focus)
2. Type your health-related question
3. Press `Enter` to send

> 💡 **Tip**: MedMemo supports multi-line messages. Press `Shift + Enter` to insert a new line without sending.

### While AI Is Responding

- The send button becomes a **stop button** (square icon). Click it or press `Enter` to stop generation.
- Text appears word-by-word (streaming) for a natural reading experience.
- If the response is interrupted, the message is marked **"[Interrupted by user]"** but the partial content is preserved.

### Editing Your Last Message

If your input box is empty, press the `↑` (Up Arrow) key to fill it with your last sent message. Edit and resend as needed.

---

## Conversation Management

### Sidebar Features

| Feature | How To |
|---------|--------|
| **Search** | Type in the search box above the conversation list |
| **Rename** | Right-click a conversation → **Rename** (max 16 characters) |
| **Delete** | Right-click a conversation → **Delete** (moves to Trash) |
| **Undo Delete** | A 5-second toast appears after deletion — click **Undo** |
| **Trash** | Click the **Trash** icon at the bottom of the sidebar to view deleted conversations |
| **Resize** | Drag the handle between sidebar and chat area (200–400px) |
| **Collapse** | Click the collapse arrow (auto-collapses below 768px width) |

### Time Grouping

Conversations are automatically grouped by time:
- **Pinned** (if any)
- **Today**
- **Yesterday**
- **Last 7 Days**
- **Earlier**

---

## Settings

Open settings by clicking the **gear icon** in the header.

### Appearance

| Theme | Description |
|-------|-------------|
| **Light** | Bright background, dark text |
| **Dark** | Dark background, light text |
| **System** | Follows your OS theme automatically |

### Default Model

Choose your preferred AI provider:

| Model | Provider | Location |
|-------|----------|----------|
| Kimi Lite | Moonshot | Cloud |
| GPT-4o Mini | OpenAI | Cloud |
| Tongyi Qwen Turbo | Alibaba | Cloud |
| Llama 3.1 8B | Ollama | **Your machine** |

> **Local models** (Ollama) require you to install and run Ollama separately. No API key needed, and no data leaves your device.

### Compliance Bar Mode

| Mode | Behavior |
|------|----------|
| **Always Show** | Blue info bar appears on every new conversation (can be closed per session) |
| **First Time Only** | Shown once per conversation, then hidden |
| **Off** | Never shown |

> ⚠️ We recommend keeping it on **Always Show** for safety awareness.

### Privacy Settings

**Desensitization Level:**
- **Standard** — Rule-based + NER model (balanced)
- **Strict** — Triple-layer protection (maximum privacy)
- **Off** — No desensitization (not recommended)

**Data Retention:**
- Options: 7 days / 30 days / 90 days / 1 year / Forever
- Old data is automatically purged according to this setting

### Auto-Update

- Enable/disable automatic update checks
- Choose channel: **Stable** or **Beta**

---

## Keyboard Shortcuts

### Input Box

| Shortcut | Action |
|----------|--------|
| `Enter` | Send message |
| `Shift + Enter` | Insert new line |
| `Ctrl / Cmd + Enter` | Force send (ignores typing state) |
| `Escape` | Clear input |
| `↑` (when empty) | Fill with last user message |
| `Ctrl + /` | Focus input box |

### Global

| Shortcut | Action |
|----------|--------|
| `Ctrl / Cmd + N` | New conversation |

### Sidebar

| Shortcut | Action |
|----------|--------|
| `Enter` | Confirm rename |
| `Escape` | Cancel rename |

---

## Understanding AI Responses

MedMemo's AI responses may include special visual indicators:

| Indicator | Meaning |
|-----------|---------|
| **Blue info bar** at top | General disclaimer — "AI output is for reference only" |
| **Orange warning box** | Content flagged by compliance engine — read carefully |
| **Red alert banner** | Emergency symptom detected (B-level) — acknowledge to continue |
| **Full-screen red dialog** | Critical emergency (A-level) — consider calling emergency services |

> 🚨 **If you see an A-level emergency alert**, take it seriously. The alert provides options to call emergency services, find nearby urgent care, or continue chatting.

---

## Markdown Support

AI responses support rich formatting:

- **Bold**, *italic*, `inline code`
- Code blocks with syntax highlighting
- Tables
- Bullet and numbered lists
- Links (clickable)

Medical terms may appear with a **dashed underline** — hover to see a tooltip definition.

---

## Best Practices

1. **Be specific** — Instead of "I feel bad", describe symptoms, duration, and severity
2. **One topic per conversation** — Keeps context focused and responses more relevant
3. **Review compliance notices** — They exist to keep information safe and accurate
4. **Never ignore A-level alerts** — They detect potentially life-threatening symptoms
5. **Verify with a doctor** — MedMemo provides information, not medical advice

---

*Last updated: 2026-05-19*
