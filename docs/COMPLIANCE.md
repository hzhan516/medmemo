# Compliance and Privacy Protection

> 🌐 [中文版本](./i18n/zh-Hans-CN/COMPLIANCE.md)

> This document serves as the developer reference for MedMemo's compliance system, covering red-line policies, the de-identification pipeline, interception rules, and emergency symptom detection.

---

## Compliance Red Lines (Non-Negotiable)

| Red Line Category | Prohibited Behavior | Consequence of Violation |
|:------------------|:--------------------|:-------------------------|
| **Diagnostic Red Line** | Outputting definitive diagnostic conclusions (e.g., "You have XX disease") | Compliance fatal, blocks release |
| **Prescription Red Line** | Recommending specific drugs/dosages or ordering tests | Compliance fatal, blocks release |
| **Treatment Red Line** | Outputting treatment plans or surgical recommendations | Compliance fatal, blocks release |
| **AI Identity Red Line** | Using terms like "AI doctor," "smart diagnosis," or "digital doctor" | Compliance fatal |
| **Data Commercialization Red Line** | Commercializing user health data (ads/insurance targeting) | Trust fatal |
| **Emergency Scenario Red Line** | Failing to trigger mandatory medical reminders for emergency symptoms | Safety fatal |

### Recommended Terminology vs. Prohibited Terms

| Scenario | Safe Wording | Prohibited Wording |
|:---------|:-------------|:-------------------|
| Symptom association | "May be related to...", "Commonly seen in...", "Suggested attention" | "Diagnosed as", "Confirmed", "Screening results", "Suffering from" |
| Medical advice | "Recommended consultation", "Suggest visiting", "May consider" | "Must immediately", "Definitely need to" (non-emergency) |
| Test recommendations | "Doctor may suggest...", "Routine evaluation may include..." | "You need to do... test", "Must do... lab work" |
| Treatment/medication | "Treatment plan to be determined by doctor after visit", "Please follow doctor's advice" | "Treatment", "Recommended to take...", "Can be cured with..." |
| Risk assessment | "Risk factors include...", "Family history may increase attention necessity" | "Your risk is...%", "Definitely will/won't..." |

---

## Two-Level De-Identification Pipeline

```
User Input
  → L1 Rule-Based De-Identification Engine (<1ms, <1MB)
    → L2 NER De-Identification Model (20-50ms, ~50MB int8 quantized)
      → Safe Text Output
```

### L1: Rule-Based De-Identification Engine

- Aho-Corasick multi-pattern matching, O(n) time complexity
- Coverage: ID card numbers, phone numbers, bank card numbers, email, URL

### L2: NER De-Identification Model

- Hugot + ONNX Runtime DistilBERT-ONNX token-classification model
- Coverage: Person names (PER), locations (LOC), organization names (ORG). Disease and medication names are NOT de-identified by the L2 NER stage in v1.1.10 — they are only handled by L1 rules where a rule exists.
- **ONNX Session is not thread-safe**; must be called serially through the 2-Worker pattern

### Sensitivity Level Classification

```go
P1Public     // Public information, no processing needed
P2Internal   // Internal information, soft replacement (reversible)
P3Confidential // Confidential information, hard replacement (irreversible)
```

### De-Identification Levels (standard / strict / off)

The de-identification strength is user-configurable and applied only to cloud requests:

| Level | Behavior |
|:------|:---------|
| **standard** | De-identifies the last user message with L1 rules + L2 NER at the default confidence threshold (0.75). |
| **strict** | De-identifies **all** user messages; adds an L1.5 deterministic fallback stage (birth dates, age+name, addresses, medical/record numbers, license plates, passport numbers, IP addresses); and runs L2 NER at a **lower confidence threshold of 0.5** for higher recall. |
| **off** | Skips outbound de-identification entirely. Only for users who explicitly accept the informed risk. |

**Strict NER 0.5 threshold — rationale and trade-off.** Strict mode prioritizes recall to minimize
outbound PII: lowering the threshold from 0.75 to 0.5 captures more low-confidence candidate entities.
The trade-off is reduced precision and possible over-masking. Because L2/strict masking uses
**P3 (irreversible)** replacement, over-masked spans cannot be restored, and prompt fidelity may be
slightly reduced. This trade-off is acceptable for strict-mode users, who opt into stronger privacy at
the cost of some prompt detail.

**Strict fail-closed.** If de-identification fails for a message under strict mode, the system never
sends the raw text. It degrades to L1-only de-identification (or fully masks the message if L1 also
fails) and requires explicit user confirmation before sending the degraded content.

### Local / Loopback Skip Assumption (Security Caveat)

Local providers (Ollama / llama.cpp) and loopback endpoints (`localhost`, `127.0.0.1`, `::1`) skip
outbound de-identification because data is assumed to **stay on the device**. This assumption breaks if
a process listening on a loopback address acts as a **proxy that forwards requests to a cloud service**:
in that case raw, un-de-identified text would leave the device. Treat a localhost proxy that forwards to
the cloud with the same informed-risk posture as the `off` level. Do not point MedMemo at a loopback
address unless you are certain the endpoint keeps data on-device.

---

## Four-Level Compliance Message Interception

The interception layer sits between AI response generation and user display, using a dual-layer detection of "rule matching + local lightweight model classification."

| Risk Level | Trigger Condition | System Response |
|:-----------|:------------------|:----------------|
| **L1-Block** | Definitive disease diagnosis, specific drug dosage prescriptions, surgical recommendations | Block display, replace with standard prompt |
| **L2-Warning** | Implied diagnosis, OTC drug recommendations, test suggestions | Allow display but append orange highlighted warning + disclaimer |
| **L3-Notice** | Health education content involving severe diseases | Append standard disclaimer at message bottom (blue notice bar) |
| **L4-Normal** | General health education, lifestyle advice | Normal display |

### Streaming Output Interception Strategy

1. **Punctuation-based sentence buffering**: Split by Chinese/English punctuation marks; perform detection immediately after each complete sentence is formed.
2. **Push only after detection passes**: Only sentences that pass L1~L4 detection are pushed to the frontend via Wails Events.
3. **L1 blockword instant interruption**: If an L1-level blockword is hit during buffering, **immediately interrupt the Stream**, discard subsequent content, and replace with a standard prompt.
4. **Performance budget**: Sentence buffering + single-sentence detection latency < 20ms.

---

## Emergency Symptom Detection

Independent of the AI reply flow; performs local real-time detection at the user input stage.

| Level | Trigger Conditions | Response Behavior |
|:------|:-------------------|:------------------|
| **Level A — Seek Immediate Care** | Chest pain with difficulty breathing, loss of consciousness, severe bleeding, severe allergy, etc. (~200 rules) | Full-screen red overlay modal (cannot be dismissed); offers "Call 120", "Find Nearby ER", "Continue Consulting" |
| **Level B — Seek Care Soon** | Persistent high fever >3 days, severe abdominal pain, blood in urine, sudden vision loss, etc. (~50 rules) | Red warning banner above input area; user must click "I Understand" before continuing |

**Performance requirement**: Detection latency < 5ms, independent of AI response or network.

---

## Informed Consent and Disclaimer

### First Launch

Non-skippable three-step process:
1. **Core Information Display** — Product nature, service boundaries, data usage, risk notices
2. **Comprehension Check** — Two test questions; must answer correctly to proceed
3. **Active Consent** — Check consent box and click confirm; record encrypted and saved locally

### Every Session

A persistent compliance notice bar at the top of the conversation interface (height <= 40px):
> "This tool provides health information for reference only. It does not diagnose or treat. For emergencies, please call 120."

---

*Last updated: 2026-07-09*
