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

## Three-Tier De-Identification Pipeline

```
User Input
  → L1 Rule-Based De-Identification Engine (<1ms, <1MB)
    → L2 NER De-Identification Model (20-50ms, ~50MB int8 quantized)
      → L3 Keyword Dictionary Matching (<5ms, ~5MB)
        → Safe Text Output
```

### L1: Rule-Based De-Identification Engine

- Aho-Corasick multi-pattern matching, O(n) time complexity
- Coverage: ID card numbers, phone numbers, bank card numbers, email, URL

### L2: NER De-Identification Model

- Hugot + ONNX Runtime BiLSTM-CRF model
- Coverage: Person names, organization names, disease names, drug names
- **ONNX Session is not thread-safe**; must be called serially through the 2-Worker pattern

### L3: Keyword Dictionary Matching

- Trie tree prefix matching
- Fallback strategy: professional terminology, drug aliases, organization abbreviations

### Sensitivity Level Classification

```go
P1Public     // Public information, no processing needed
P2Internal   // Internal information, soft replacement (reversible)
P3Confidential // Confidential information, hard replacement (irreversible)
```

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
