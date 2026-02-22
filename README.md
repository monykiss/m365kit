<div align="center">
  <h1>M365Kit</h1>
  <p><strong>The complete Microsoft 365 CLI.</strong></p>
  <p>
    Process Office documents at scale. Automate SharePoint. Control OneDrive.<br>
    Post to Teams. Read Outlook. Convert formats. All from your terminal.<br>
    No Word required. No GUI. No clicking.
  </p>

  <p>
    <a href="https://github.com/monykiss/m365kit/releases/latest">
      <img src="https://img.shields.io/github/v/release/monykiss/m365kit?color=blue" alt="Latest Release">
    </a>
    <a href="https://github.com/monykiss/m365kit/actions/workflows/ci.yml">
      <img src="https://github.com/monykiss/m365kit/actions/workflows/ci.yml/badge.svg" alt="CI">
    </a>
    <a href="https://pkg.go.dev/github.com/monykiss/m365kit">
      <img src="https://pkg.go.dev/badge/github.com/monykiss/m365kit.svg" alt="Go Reference">
    </a>
    <img src="https://img.shields.io/badge/tests-286%2B-brightgreen" alt="Tests">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="MIT License">
  </p>
</div>

---

## What teams use it for

```bash
# Replace company name in 342 contracts, in 6 minutes
kit batch './contracts/*.docx' --action edit --find "OldName" --replace "NewName"

# Audit SharePoint for external sharing — 3 minutes vs. 2 hours in the web UI
kit acl external --site https://company.sharepoint.com/sites/Legal --domain company.com

# Download every Office attachment from your unread emails
kit outlook inbox --has-attachment --unread
kit outlook download 1 --office-only -o ./downloads

# Post a Word doc summary to Teams automatically
kit word read contract.docx | kit ai summarize | \
  kit teams post --team Legal --channel contracts --message "$(cat /dev/stdin)"

# Convert 50 Word docs to Markdown for your wiki
kit convert report.docx --to md

# Watch a directory and process new files automatically
kit watch start ./incoming -r --ext docx,xlsx --action log
```

**In production:** [Processing 1,200 legal contracts in 47 minutes](docs/case-studies/legal-contracts.md) |
76us per document read | 286+ tests | Pure Go | [Stability policy](docs/stability.md)

---

## Install

### Homebrew (macOS / Linux)
```bash
brew install monykiss/tap/m365kit
```

### Windows

**PowerShell (one-liner, no admin required):**
```powershell
iwr https://raw.githubusercontent.com/monykiss/m365kit/main/packaging/windows/install.ps1 | iex
```

**Windows Installer (.exe):**
Download from [GitHub Releases](https://github.com/monykiss/m365kit/releases/latest)

### Go
```bash
go install github.com/monykiss/m365kit@latest
```

### TypeScript
```bash
npm install @m365kit/core
```

### Docker
```bash
docker pull ghcr.io/monykiss/m365kit:latest
docker run --rm -v "$PWD:/data" ghcr.io/monykiss/m365kit word read /data/report.docx
```

### Build from source
```bash
git clone https://github.com/monykiss/m365kit.git
cd m365kit
make build
./bin/kit --help
```

---

## Why M365Kit?

| Approach | Read .docx | Write .docx | AI | Cloud | Pipeline | Speed |
|----------|-----------|-------------|-----|-------|----------|-------|
| **M365Kit** | `kit word read` | `kit word write` | Built-in | Graph API | YAML | ~76 us/doc |
| python-docx | 10+ lines | 20+ lines | DIY | No | No | ~5 ms |
| LibreOffice CLI | `soffice --convert` | `soffice --convert` | No | No | No | ~1 s |
| Microsoft Graph SDK | 50+ lines | 80+ lines | No | Yes | No | Network |
| Google Apps Script | N/A | N/A | N/A | Google only | Triggers | Slow |

---

## Quick Start

### Local Documents

```bash
# Read a Word document
kit word read report.docx --json | jq .paragraphs

# Read Excel data
kit excel read data.xlsx --sheet "Revenue" --json

# AI summary (set your API key first)
export ANTHROPIC_API_KEY=sk-ant-...
kit word read contract.docx | kit ai summarize

# Compare two documents
kit diff old-version.docx new-version.docx --stats
```

### Microsoft 365 Cloud

```bash
# Authenticate via Azure AD device code flow
export KIT_AZURE_CLIENT_ID="your-app-client-id"
kit auth login

# OneDrive operations
kit onedrive ls /                        # List root
kit onedrive get Documents/report.docx   # Download
kit onedrive put ./report.docx           # Upload
kit onedrive recent                      # Recent files
kit onedrive search "Q1 budget"          # Search
kit onedrive share Documents/report.docx # Create share link

# SharePoint operations
kit sharepoint sites                     # List sites
kit sharepoint libs <site-id>            # List document libraries
kit sharepoint ls <site-id> /Reports     # Browse library files
kit sharepoint get <site-id> report.docx # Download from library
kit sharepoint audit <site-id>           # Activity log

# Teams integration
kit teams list                           # Your teams
kit teams channels --team Engineering    # Channels
kit teams post --team Engineering --channel general --message "Report ready"
kit teams dm --to alice@company.com --message "Contract is ready"
```

### Document Templates

```bash
# Extract variables from a template
kit template vars contract_template.docx

# Apply variables to a template
kit template apply contract_template.docx \
  --set name="John Doe" \
  --set company="Acme Corp" \
  --set date="2025-01-15" \
  -o filled_contract.docx

# Register a template in the library
kit template add invoice --description "Monthly invoice" invoice_template.docx

# Generate reports from data + template
kit report generate --template quarterly.docx --data sales.csv -o report.docx
kit report preview --data sales.csv   # Preview available variables
```

### File Watching

```bash
# Watch a directory for new documents
kit watch start ./incoming -r --ext docx,xlsx --action log

# Check watcher status
kit watch status

# Stop the watcher
kit watch stop
```

### Docker

```bash
# Run kit in Docker
docker compose run kit word read /data/report.docx

# Start a file watcher daemon
docker compose up -d watch
```

### Outlook Email

```bash
# List inbox (with filters)
kit outlook inbox --unread --limit 10
kit outlook inbox --from alice@company.com --has-attachment

# Read a specific email by index
kit outlook read 1

# Download attachments (Office files only)
kit outlook download 3 --office-only -o ./downloads

# List attachments on a message
kit outlook attachments 3

# Mark as read / reply
kit outlook mark-read 1
kit outlook reply 1 --body "Thanks for the update!"
```

### SharePoint Permissions Audit

```bash
# Full ACL audit of a SharePoint site
kit acl audit --site <site-id> --domain company.com

# Find files shared with external users
kit acl external --site <site-id> --domain company.com

# Find files with broken permission inheritance
kit acl broken --site <site-id>

# List all users with site access
kit acl users --site <site-id>

# Export audit report to JSON
kit acl audit --site <site-id> -o audit_report.json
```

### Format Conversion

```bash
# Word to Markdown / HTML / plain text
kit convert report.docx --to md
kit convert report.docx --to html -o report.html
kit convert report.docx --to txt

# Markdown to Word
kit convert notes.md --to docx

# Excel to CSV / JSON / Markdown
kit convert data.xlsx --to csv
kit convert data.xlsx --to json --sheet "Revenue"
kit convert data.xlsx --to md
```

### Enterprise: Org Config + Audit Log + Admin

```bash
# View org policy status (community mode if no org config)
kit org status
kit org show --json

# Generate an org config template for your IT team
kit org init --org-name "Acme Corp" --domain acme.com > /etc/kit/org.yaml

# Validate an org config file before deploying
kit org validate /etc/kit/org.yaml

# View audit log (auto-populated when audit is enabled in org config)
kit audit log --last 20
kit audit log --since 2026-01-01 --user alice@acme.com
kit audit status

# IT admin: usage statistics
kit admin stats --since 2026-01-01
kit admin stats --by user --json
kit admin users

# Telemetry management
kit admin telemetry status
kit admin telemetry clear
```

### File System Intelligence

```bash
# Scan for Office documents
kit fs scan ~/Documents -r

# Rename to consistent convention
kit fs rename ~/Documents -r --pattern kebab --dry-run

# Find and remove duplicates
kit fs dedupe ~/Documents -r --dry-run

# Find stale files (not modified in 90 days)
kit fs stale ~/Documents -r --days 90

# Organize into folders by type
kit fs organize ~/Documents -r --strategy by-type --dry-run

# Generate JSON manifest
kit fs manifest ~/Documents -r > manifest.json
```

---

## Features

| Category | Feature | Command |
|----------|---------|---------|
| **Documents** | Read Word (.docx) | `kit word read` |
| | Write Word (.docx) | `kit word write` |
| | Edit Word (.docx) | `kit word edit` |
| | Read Excel (.xlsx) | `kit excel read` |
| | Write Excel (.xlsx) | `kit excel write` |
| | Analyze Excel (.xlsx) | `kit excel analyze` |
| | Read PowerPoint (.pptx) | `kit pptx read` |
| | Generate PowerPoint | `kit pptx generate` |
| | Compare documents | `kit diff` |
| **AI** | Summarize | `kit ai summarize` |
| | Analyze | `kit ai analyze` |
| | Entity extraction | `kit ai extract` |
| | Q&A | `kit ai ask` |
| | Anthropic / OpenAI / Ollama | `--provider` flag |
| **Conversion** | Word to Markdown | `kit convert report.docx --to md` |
| | Word to HTML | `kit convert report.docx --to html` |
| | Markdown to Word | `kit convert notes.md --to docx` |
| | Excel to CSV/JSON/Markdown | `kit convert data.xlsx --to csv` |
| **Microsoft 365** | OAuth device code flow | `kit auth login` |
| | OneDrive (ls/get/put/search/share) | `kit onedrive` |
| | SharePoint (sites/libs/audit) | `kit sharepoint` |
| | Teams (list/post/share/dm) | `kit teams` |
| | Outlook (inbox/read/download/reply) | `kit outlook` |
| | ACL audit (external/broken/links) | `kit acl audit` |
| **File System** | Scan documents | `kit fs scan` |
| | Rename (kebab/snake/date) | `kit fs rename` |
| | Deduplicate | `kit fs dedupe` |
| | Find stale files | `kit fs stale` |
| | Organize into folders | `kit fs organize` |
| | JSON manifest | `kit fs manifest` |
| **Templates** | Variable extraction | `kit template vars` |
| | Apply variables | `kit template apply` |
| | Template library | `kit template add/list/show/remove` |
| | Report generation | `kit report generate` |
| | Data preview | `kit report preview` |
| **Automation** | Pipeline workflows | `kit pipeline run` |
| | Batch processing | `kit batch` |
| | Email with AI draft | `kit send` |
| | File watcher | `kit watch start/stop/status` |
| **Enterprise** | Org config management | `kit org show/init/validate` |
| | Audit logging (JSONL) | `kit audit log/status/clear` |
| | Usage statistics | `kit admin stats` |
| | User activity | `kit admin users` |
| | Telemetry management | `kit admin telemetry` |
| **Setup** | Config wizard | `kit config init` |
| | Shell completions | `kit completion` |
| | Health check | `kit doctor` |
| | Update checker | `kit update check` |
| **Deploy** | Docker image | `Dockerfile` |
| | Docker Compose | `docker-compose.yml` |
| **Output** | JSON (all commands) | `--json` flag |
| | Markdown | `--markdown` flag |
| | Stdin/stdout piping | All commands |

---

## Authentication Setup

### Microsoft 365 (OneDrive / SharePoint)

1. Register an Azure AD app at [portal.azure.com](https://portal.azure.com)
2. Add delegated permissions: `Files.ReadWrite`, `Sites.ReadWrite.All`, `User.Read`
3. Enable "Allow public client flows" for device code auth

```bash
export KIT_AZURE_CLIENT_ID="your-app-client-id"
kit auth login        # Opens device code flow
kit auth whoami       # Verify identity
kit auth status       # Check token expiry
kit auth refresh      # Refresh token
kit auth logout       # Delete token
```

### AI Providers

```bash
# Anthropic (default)
export ANTHROPIC_API_KEY=sk-ant-api03-...
kit ai summarize document.txt

# OpenAI
export OPENAI_API_KEY=sk-...
kit ai summarize document.txt --provider openai --model gpt-4o

# Ollama (local, no API key)
ollama pull llama3.1
kit ai summarize document.txt --provider ollama --model llama3.1
```

---

## Pipeline Workflows

Define multi-step document automation in YAML:

```yaml
name: contract_review
version: "1.0"
steps:
  - id: read_contract
    action: word.read
    input: ./contracts/agreement.docx

  - id: extract_terms
    action: ai.extract
    input: ${{ steps.read_contract.output }}
    options:
      fields: "parties,effective_date,termination_date,payment_terms"

  - id: risk_analysis
    action: ai.analyze
    input: ${{ steps.read_contract.output }}
    options:
      prompt: "Identify potential risks and unusual clauses"
```

```bash
kit pipeline run contract_review.yaml --verbose
```

See more examples in [`examples/pipelines/`](examples/pipelines/).

---

## Unix Philosophy

Every command is pipe-friendly. Compose them:

```bash
# Chain read -> summarize -> save
kit word read contract.docx | kit ai summarize > summary.txt

# Excel data -> AI analysis
kit excel read metrics.xlsx --json | kit ai analyze --prompt "Find anomalies"

# Download from OneDrive -> analyze -> email results
kit onedrive get Reports/Q1.xlsx -o q1.xlsx
kit excel analyze q1.xlsx --prompt "Key trends" > analysis.txt
kit send --to team@company.com --subject "Q1 Analysis" --attach q1.xlsx --body "$(cat analysis.txt)"

# Batch process with shell
for f in contracts/*.docx; do
  kit word read "$f" | kit ai extract --fields "parties,amount" >> results.jsonl
done
```

---

## Project Structure

```
m365kit/
├── cmd/                    # Cobra CLI commands
│   ├── word/               # kit word read/write/edit
│   ├── excel/              # kit excel read/write/analyze
│   ├── pptx/               # kit pptx read/generate
│   ├── ai/                 # kit ai summarize/analyze/extract/ask
│   ├── auth/               # kit auth login/whoami/status/logout/refresh
│   ├── onedrive/           # kit onedrive ls/get/put/recent/search/share
│   ├── sharepoint/         # kit sharepoint sites/libs/ls/get/put/audit
│   ├── fs/                 # kit fs scan/rename/dedupe/stale/organize/manifest
│   ├── teams/              # kit teams list/post/share/dm
│   ├── config/             # kit config init/show/set/validate
│   ├── completion/         # kit completion bash/zsh/fish/powershell
│   ├── template/           # kit template vars/apply/add/list/show/remove
│   ├── report/             # kit report generate/preview
│   ├── watch/              # kit watch start/stop/status
│   ├── doctor/             # kit doctor
│   ├── update/             # kit update check/install
│   ├── diff/               # kit diff
│   ├── send/               # kit send
│   ├── outlook/            # kit outlook inbox/read/download/reply
│   ├── acl/                # kit acl audit/external/broken/users
│   ├── convert/            # kit convert (docx/xlsx/md/html/csv)
│   ├── org/                # kit org show/init/validate/status
│   ├── audit/              # kit audit log/clear/status
│   ├── admin/              # kit admin stats/users/telemetry
│   ├── pipeline/           # kit pipeline run
│   └── batch/              # kit batch
├── benchmarks/             # Go benchmarks (make benchmark)
├── internal/
│   ├── auth/               # Microsoft OAuth device code flow
│   ├── graph/              # OneDrive + SharePoint + Teams + Outlook + ACL Graph API
│   ├── template/           # Template engine with run-splitting fix
│   ├── report/             # Report generator (CSV/JSON data + templates)
│   ├── watch/              # File system watcher with fsnotify
│   ├── update/             # Update checker
│   ├── fs/                 # File system scanner, renamer, deduper, organizer
│   ├── formats/            # OOXML parsers (docx, xlsx, pptx) + convert
│   ├── ai/                 # Provider interface + implementations
│   ├── email/              # SMTP email client
│   ├── bridge/             # Go→Node subprocess bridge
│   ├── pipeline/           # YAML workflow engine
│   ├── admin/              # IT admin stats aggregation
│   ├── audit/              # JSONL audit logger + redaction
│   └── telemetry/          # Privacy-first local telemetry
├── tests/                  # Smoke / integration tests
├── packages/core/          # TypeScript package (@m365kit/core)
├── examples/               # Pipeline YAML examples
└── docs/                   # Documentation + stability policy + case studies
```

---

## Performance

All benchmarks run on Apple M2, Go 1.23, pure Go (no CGo, no subprocess calls).

| Operation | Time | Allocations |
|-----------|------|-------------|
| Read .docx | ~76 us | 773 allocs |
| Write .docx (6 nodes) | ~93 us | 76 allocs |
| Write .docx (100 paragraphs) | ~142 us | 76 allocs |
| Read + Write round-trip | ~194 us | 859 allocs |
| Plain text extraction | ~0.6 us | 10 allocs |
| Markdown extraction | ~0.6 us | 10 allocs |
| Read .xlsx | ~741 us | 6,410 allocs |
| Write .xlsx | ~974 us | 4,406 allocs |
| Convert .docx to .md | ~76 us | 783 allocs |
| Convert .docx to .html | ~78 us | 779 allocs |
| Convert .xlsx to .csv | ~784 us | 6,514 allocs |
| Convert .md to .docx | ~205 us | 1,048 allocs |

Run benchmarks yourself:
```bash
make benchmark
```

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for guidelines.

```bash
git clone https://github.com/monykiss/m365kit.git
cd m365kit
make build
make test
```

---

## License

[MIT](LICENSE)

---

<p align="center">
  Built by <a href="https://github.com/klytics"><strong>KLYTICS</strong></a>
</p>
