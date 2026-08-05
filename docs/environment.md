# Environment Variables

> 🌐 [中文版本](./i18n/zh-Hans-CN/environment.md)

> This document lists all environment variables read by MedMemo in production code. Variables are grouped by scope and override the corresponding configuration file values.

---

## Configuration Overrides

These variables override values in `config.yaml` / `config.json` (see [`docs/DEVELOPMENT.md`](./DEVELOPMENT.md)).

| Variable | Purpose | Default | Example | Scope |
|----------|---------|---------|---------|-------|
| `MEDMEMO_DATA_DIR` | Root directory for encrypted local data (database, logs, memories, embedding models) | `~/.medmemo/data` on macOS/Linux; see [Windows selection](#windows-data-directory-selection) | `/home/user/.medmemo/data` | config / database |
| `MEDMEMO_DEFAULT_MODEL` | Default model ID for new conversations | `kimi-lite` | `kimi-pro` | config |
| `MEDMEMO_PROVIDER_TYPE` | Default provider type | `kimi` | `openai` | config |
| `MEDMEMO_API_ENDPOINT` | Custom API endpoint URL (overrides provider default) | *(empty)* | `https://api.openai.com/v1` | config |
| `MEDMEMO_API_KEY_FILE` | Path to a file containing the API key | *(empty)* | `/run/secrets/medmemo_key` | config |
| `MEDMEMO_MODEL_DIR` | Directory containing the bundled ONNX / embedding model | `resources/models/distilbert-ner` | `/opt/medmemo/models` | config |
| `MEDMEMO_UPDATE_CHECK` | Enable automatic update checks | `true` | `false` | config |
| `MEDMEMO_UPDATE_CHANNEL` | Update channel used by the auto-updater | derived from build (falls back to `beta`) | `stable` | config |
| `MEDMEMO_DESENSITIZATION_LEVEL` | De-identification level before cloud requests | `standard` | `off` / `strict` | config |
| `MEDMEMO_DATA_RETENTION_DAYS` | Number of days to retain conversation / memory data | `30` | `90` | config |
| `MEDMEMO_EMBEDDING_MODEL_DOWNLOAD_URL` | URL used to download the embedding model on first run | *(empty)* | `https://example.com/model.onnx` | config |

## Authentication

OAuth Device Flow requires a per-provider `client_id`. See [`docs/api/auth.md`](./api/auth.md) for registration prerequisites.

| Variable | Purpose | Default | Example | Scope |
|----------|---------|---------|---------|-------|
| `MEDMEMO_KIMI_CLIENT_ID` | Kimi OAuth client ID | *(empty)* | `kimi_xxx` | auth |
| `MEDMEMO_GEMINI_CLIENT_ID` | Gemini OAuth client ID | *(empty)* | `gemini_xxx` | auth |
| `MEDMEMO_MICROSOFT_CLIENT_ID` | Microsoft OAuth client ID | *(empty)* | `ms_xxx` | auth |
| `MEDMEMO_GITHUB_CLIENT_ID` | GitHub OAuth client ID | *(empty)* | `github_xxx` | auth |

## Updater

| Variable | Purpose | Default | Example | Scope |
|----------|---------|---------|---------|-------|
| `MEDMEMO_INSTALL_KIND` | Linux packaging format (overrides marker files and legacy `dpkg`/`rpm` detection) | `unknown` | `deb` / `rpm` | updater |

---

## Windows Data Directory Selection

On Windows, MedMemo uses the following priority order when no explicit `MEDMEMO_DATA_DIR` or `config.yaml` value is set:

1. **Legacy data first** — If `%USERPROFILE%\.medmemo\data\medmemo.db` exists, MedMemo uses `%USERPROFILE%\.medmemo\data`. This keeps existing data visible for users upgrading from v1.1.9 or earlier.
2. **Writable install directory** — If the registry records an install directory (for example, `%LOCALAPPDATA%\Programs\MedMemo`) and `<installDir>\data` is writable, MedMemo uses that directory.
3. **User-profile fallback** — If the install directory is missing or not writable (for example, a read-only `Program Files` install), MedMemo falls back to `%USERPROFILE%\.medmemo\data`.

MedMemo does not automatically move, copy, or merge databases between these locations. To force a specific location, set `MEDMEMO_DATA_DIR` or `data_dir` in `config.yaml`.

---

## Notes

- Boolean values accept `true` / `1` (truthy) or `false` / `0` (falsy).
- `MEDMEMO_DATA_DIR` is read by both the configuration loader and the SQLCipher connector.
- If both a config file and an environment variable are set, the environment variable wins.
- On Windows, `MEDMEMO_DATA_DIR` or `data_dir` in `config.yaml` overrides the automatic selection above.

---

*Last updated: 2026-08-05*
