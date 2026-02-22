# Changelog

All notable changes to M365Kit are documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

---

## [1.2.0] — 2026-02-22

### Added
- `kit plugin` — Plugin system for custom commands (install/list/run/remove/show/new)
- `kit shell` — Interactive REPL with persistent state, tab completion, command history
- `kit shell --eval` — Non-interactive mode for scripting
- Plugin scaffolding: `kit plugin new --type shell|go` generates complete project
- Dynamic plugin registration: installed plugins appear as top-level kit commands
- Progress bars and spinners (`internal/progress`) for long-running operations
- `--no-progress` global flag to disable progress bars
- docs/plugins.md — complete plugin authoring guide
- Shell tab completion for all commands and subcommands
- Shell history persisted in `~/.kit/shell_history`

### Changed
- Root command wired with plugin discovery and shell runner
- New dependency: `github.com/chzyer/readline` for shell line editing
- Test count: 295 → 348 (47 new plugin + shell + progress tests + 6 smoke tests)

---

## [1.1.0] — 2026-02-22

### Added
- `kit org` — Organization-wide configuration management (show/validate/init/status)
- `kit audit` — Command-level audit logging for compliance (log/clear/status)
- `kit admin` — IT admin command group: usage stats, user listing, telemetry management
- Org config file (`/etc/kit/org.yaml`) — centralized policy: locked settings, allowed commands, Azure/AI defaults
- Audit log (JSONL) with automatic argument redaction (API keys, tokens, secrets)
- Local telemetry store with privacy-first design (no user IDs, no file paths)
- `PersistentPostRun` audit hook — every command is logged when audit is enabled
- Org config validation (`kit org validate`)
- Org config template generator (`kit org init`)
- Enterprise deployment: org-config-template.yaml reference
- 10 new smoke tests for enterprise commands

### Changed
- Root command wired with `PersistentPreRun`/`PersistentPostRun` for audit logging
- Test count: 252 → 286+ (33 new internal tests + 10 new smoke tests)

---

## [1.0.0] — 2026-02-22

### Added
- Stable API contract and semantic versioning commitment (docs/stability.md)
- Complete CHANGELOG.md (this file)
- CONTRIBUTING.md with development setup, testing guide, PR process
- Smoke test suite (tests/smoke_test.go) — validates every command
- Case study: "Processing 1,200 Legal Contracts" (docs/case-studies/)
- Consistent exit codes (0 = OK, 1 = user error, 2 = system error)
- Standard JSON output envelope ({ok, command, version, data/error})
- Issue templates (bug report, feature request)
- PR template with checklist

### Changed
- README rewritten as product launch page
- CONTRIBUTING.md expanded with full development guide

---

## [0.6.0] — 2026-02-22

### Added
- `kit outlook` — Outlook inbox, email reading, attachment download, reply via Graph API
- `kit acl` — SharePoint permissions audit (external shares, broken inheritance, anonymous links)
- `kit convert` — Format conversion: docx to md/html/txt, md/html to docx, xlsx to csv/json/md
- Benchmark suite (14 benchmarks, `make benchmark`)
- Performance section in README (76us/doc read, 741us/xlsx read)
- Pipeline actions: convert, outlook.inbox, outlook.download, acl.audit
- Example pipeline: inbox_processor.yaml
- "Why M365Kit?" comparison table in README

---

## [0.5.0] — 2026-02-21

### Added
- `kit template` — Document template library with {{variable}} substitution
- `kit report` — Report generation from CSV/JSON/XLSX data + templates
- `kit watch` — Background daemon for automated document workflows (fsnotify)
- `kit doctor` — System health check (Go runtime, config, auth, AI provider)
- `Dockerfile` + `docker-compose.yml` — Multi-stage Docker image
- Template run-splitting fix (Word XML paragraph merging for split variables)
- Report auto-computed aggregate variables (sum_, avg_, min_, max_, count_)

---

## [0.4.0] — 2026-02-21

### Added
- `kit teams` — Microsoft Teams integration (list, channels, post, share, dm)
- `kit config` — Interactive setup wizard (init), show/set/get/validate
- `kit completion` — Shell completions (bash, zsh, fish, PowerShell)
- `kit update` — Update checker and self-update
- Windows installer (NSIS script + PowerShell one-liner)
- Enterprise deployment guide (docs/enterprise-deployment.md)
- Auth scopes: Chat.ReadWrite, ChannelMessage.Send, Team.ReadBasic.All
- Non-blocking background update check

---

## [0.3.0] — 2026-02-21

### Added
- `kit auth` — Microsoft 365 OAuth 2.0 device code flow
- `kit onedrive` — OneDrive ls/get/put/recent/search/share via Microsoft Graph
- `kit sharepoint` — SharePoint sites/libs/ls/get/put/audit via Microsoft Graph
- `kit fs` — File system intelligence (scan/rename/dedupe/stale/organize/manifest)
- Microsoft Graph API pagination support (follows @odata.nextLink)
- Token persistence (~/.kit/token.json, mode 0600)
- Auto token refresh (5-minute expiry window)

---

## [0.2.0] — 2026-02-21

### Added
- `kit send` — Email documents via SMTP with optional AI-drafted body
- `kit diff` — Compare two Word documents with Myers/LCS diff algorithm
- Homebrew tap (brew install monykiss/tap/m365kit)
- npm publish workflow in release CI
- docs/releasing.md — release process documentation

---

## [0.1.0] — 2026-02-21

### Added
- `kit word read` — Parse .docx files (custom OOXML ZIP+XML parser, no Word required)
- `kit word write` — Generate .docx from JSON or flags
- `kit word edit` — Find-and-replace in .docx files
- `kit excel read` — Parse .xlsx files via excelize
- `kit excel write` — Generate .xlsx from JSON
- `kit excel analyze` — AI-powered spreadsheet analysis
- `kit pptx read` — Parse .pptx files
- `kit pptx generate` — Generate .pptx via TypeScript bridge (pptxgenjs)
- `kit ai summarize/analyze/extract/ask` — Provider-agnostic AI commands
- `kit pipeline run` — YAML workflow executor with variable interpolation
- `kit batch` — Bulk document processing with concurrency
- `kit version` — Version information
- Anthropic, OpenAI, and Ollama provider support
- GitHub Actions CI (Go tests + TypeScript tests + golangci-lint)
- @m365kit/core npm package (TypeScript bridge)
