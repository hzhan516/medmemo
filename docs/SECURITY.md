# Security

> 🌐 [中文版本](./i18n/zh-Hans-CN/SECURITY.md)

> This document describes MedMemo's security practices, vulnerability disclosure process, and data protection mechanisms.

---

## Security Disclosure Process

If you discover a security vulnerability, please disclose it responsibly through the following channels:

1. **Do not** disclose vulnerability details in a public GitHub Issue.
2. Send an email to `doyle_zhang@outlook.com` including:
   - Vulnerability description and impact scope
   - Reproduction steps (minimal reproducible example)
   - Suggested fix if available
3. The maintainers will acknowledge receipt within **72 hours**.
4. After the fix is complete, we will provide the reporter with a reasonable advance notice period before public disclosure.

## Data Local Storage and Encryption

MedMemo's core design principle is **data-local-first**:

- **SQLCipher/SQLite**: Conversation records, configurations, extracted facts, and audit data are stored locally with AES-256 encryption.
- **sqlite-vec**: Semantic vector indexes are stored locally alongside SQLite data.
- **DuckDB / Kùzǔ**: v2+ planning stubs only; these stores are not active in v1.x runtime.
- **Key Management**: API Keys and encryption keys are stored in the platform keyring (macOS Keychain / Windows DPAPI / Linux Secret Service).

### Data We Do Not Collect

MedMemo **will never** upload the following data to any server:
- User conversation content
- Family member health information
- Personally identifiable information (PII)
- Usage behavior logs

### Optional Network Communication

Network communication only occurs when the user explicitly enables a cloud-based model:
- **De-identified** conversation requests are sent to the selected LLM API endpoint.
- Model availability health checks.

All network requests are routed through a locally configured proxy and do not pass through MedMemo-controlled servers.

### Local / Loopback De-Identification Skip Assumption

MedMemo skips outbound de-identification for local providers (Ollama / llama.cpp) and loopback
endpoints (`localhost`, `127.0.0.1`, `::1`), because such traffic is assumed to **stay on the device**.

**Caveat:** this assumption is broken by a process that listens on a loopback address but **forwards
requests to a cloud service** (a localhost-to-cloud proxy). In that scenario, raw text that was never
de-identified leaves the device. Treat such a proxy with the same informed-risk posture as the `off`
de-identification level, and only point MedMemo at a loopback endpoint when you are certain it keeps
data on-device. See `docs/COMPLIANCE.md` for the de-identification levels and the strict-mode NER
threshold rationale.

## Dependency Security Scanning

The project uses the following tools for dependency security monitoring:

- **Dependabot**: Automatically detects known vulnerabilities in Go Modules and npm dependencies.
- **govulncheck**: Official Go vulnerability scanning tool.
- **npm audit**: Node.js dependency security audit.

Security scanning is integrated into the CI pipeline; high-severity vulnerabilities block merges.

### npm Audit Allowlist Policy

The frontend gate is implemented by `scripts/check-npm-audit-policy.js` using the reviewed allowlist in `scripts/npm-audit-allowlist.json`. The policy is fail-closed:

- Any `critical` severity vulnerability blocks the build.
- Any `high` severity vulnerability in production dependencies blocks the build, unless it is covered by a reviewed allowlist entry marked with `scope: production`.
- `high` severity vulnerabilities in development dependencies may be allowlisted only when they are not reachable in the shipped application, a concrete mitigation is documented, and an expiry date is set.
- Production-scope allowlist entries are only accepted when the vulnerability is not exploitable in the shipped application context, a concrete mitigation is documented, and an expiry date is set.
- Each allowlist entry records the advisory ID, package name, scope, justification, mitigation, target review version, and expiration date. Expired, mismatched, or stale entries block the build.

The raw `npm audit` output is retained for reporting, but the policy script is the authoritative gate.

#### Reviewed Exceptions for v1.1.10

| Advisory | Package | Scope | Reason | Review Target | Expires |
|---|---|---|---|---|---|
| [GHSA-qwww-vcr4-c8h2](https://github.com/advisories/GHSA-qwww-vcr4-c8h2) | `react-router` | production | RSC/SSR CSRF bypass; MedMemo uses HashRouter in a desktop Wails shell without RSC/SSR/server actions, so the vector is unreachable. | `>=8.3.0` or fixed 7.x patch | 2026-09-05 |
| [GHSA-qwww-vcr4-c8h2](https://github.com/advisories/GHSA-qwww-vcr4-c8h2) | `react-router-dom` | production | Same underlying advisory; client-side HashRouter only, no RSC/SSR action execution path. | `>=8.3.0` or fixed 7.x patch | 2026-09-05 |

## Build Security

- All release binaries are built through GitHub Actions with publicly auditable build logs.
- Release artifacts are accompanied by SHA-256 checksums.
- Users are encouraged to build from source to verify binary integrity.

## Security Best Practices (User-Side)

1. Always download MedMemo from official channels (GitHub Releases or build from source).
2. Update to the latest version regularly to receive security patches.
3. Use system-level full-disk encryption (BitLocker / FileVault / LUKS) for additional data protection.
4. Keep local data backups secure and avoid unencrypted transmission.

---

*Last updated: 2026-08-05*
