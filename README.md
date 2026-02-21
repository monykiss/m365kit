<p align="center">
  <h1 align="center">M365Kit</h1>
  <p align="center"><strong>The terminal is the new Office.</strong></p>
  <p align="center">
    AI-native CLI for Microsoft 365. Read, write, analyze, transform, and automate<br>
    <code>.docx</code> <code>.xlsx</code> <code>.pptx</code> — locally and in the cloud via Microsoft Graph.
  </p>
</p>

<p align="center">
  <a href="https://github.com/monykiss/m365kit/actions"><img src="https://github.com/monykiss/m365kit/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://www.npmjs.com/package/@m365kit/core"><img src="https://img.shields.io/npm/v/@m365kit/core" alt="npm"></a>
  <a href="https://github.com/monykiss/m365kit/releases"><img src="https://img.shields.io/github/v/release/monykiss/m365kit" alt="Release"></a>
  <a href="https://github.com/monykiss/m365kit/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/monykiss/m365kit"><img src="https://img.shields.io/github/stars/monykiss/m365kit?style=social" alt="Stars"></a>
</p>

---

```bash
# Authenticate with Microsoft 365
kit auth login

# List your OneDrive files
kit onedrive ls /Documents

# Download a contract, analyze it with AI, email the summary
kit onedrive get Documents/contract.docx -o contract.docx
kit word read contract.docx | kit ai summarize > summary.txt
kit send --to counsel@company.com --subject "Contract Review" --body "$(cat summary.txt)"

# Scan local files, find duplicates, organize
kit fs scan ~/Documents -r --hash
kit fs dedupe ~/Documents -r --dry-run
kit fs organize ~/Documents -r --strategy by-type
```

---

## Install

### Homebrew (macOS / Linux)
```bash
brew install monykiss/tap/m365kit
```

### Go
```bash
go install github.com/monykiss/m365kit@latest
```

### Docker *(coming soon)*
```bash
docker pull monykiss/m365kit
```

### TypeScript
```bash
npm install @m365kit/core
```

### Build from source
```bash
git clone https://github.com/monykiss/m365kit.git
cd m365kit
make build
./bin/kit --help
```

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
| **Microsoft 365** | OAuth device code flow | `kit auth login` |
| | OneDrive (ls/get/put/search/share) | `kit onedrive` |
| | SharePoint (sites/libs/audit) | `kit sharepoint` |
| **File System** | Scan documents | `kit fs scan` |
| | Rename (kebab/snake/date) | `kit fs rename` |
| | Deduplicate | `kit fs dedupe` |
| | Find stale files | `kit fs stale` |
| | Organize into folders | `kit fs organize` |
| | JSON manifest | `kit fs manifest` |
| **Automation** | Pipeline workflows | `kit pipeline run` |
| | Batch processing | `kit batch` |
| | Email with AI draft | `kit send` |
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
│   ├── diff/               # kit diff
│   ├── send/               # kit send
│   ├── pipeline/           # kit pipeline run
│   └── batch/              # kit batch
├── internal/
│   ├── auth/               # Microsoft OAuth device code flow
│   ├── graph/              # OneDrive + SharePoint Graph API clients
│   ├── fs/                 # File system scanner, renamer, deduper, organizer
│   ├── formats/            # OOXML parsers (docx, xlsx, pptx)
│   ├── ai/                 # Provider interface + implementations
│   ├── email/              # SMTP email client
│   ├── bridge/             # Go→Node subprocess bridge
│   └── pipeline/           # YAML workflow engine
├── packages/core/          # TypeScript package (@m365kit/core)
├── examples/               # Pipeline YAML examples
└── docs/                   # Documentation
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
