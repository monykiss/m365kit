<p align="center">
  <h1 align="center">M365Kit</h1>
  <p align="center"><strong>The terminal is the new Office.</strong></p>
  <p align="center">
    AI-native CLI for Microsoft 365 documents. Read, write, analyze, transform, and automate<br>
    <code>.docx</code> <code>.xlsx</code> <code>.pptx</code> from your terminal.
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
# Read a Word doc and get structured JSON
kit word read contract.docx --json | jq .wordCount

# AI-powered document summary in one pipe
kit word read contract.docx | kit ai summarize

# Extract entities from any document
kit ai extract --fields "parties,dates,amounts" contract.docx
```

<!-- TODO: Replace with actual demo GIF -->
<!-- ![M365Kit Demo](docs/assets/demo.gif) -->

---

## Install

**Homebrew** (macOS / Linux):
```bash
brew install monykiss/tap/kit
```

**Go Install**:
```bash
go install github.com/monykiss/m365kit@latest
```

**NPM** (TypeScript package only):
```bash
npm install @m365kit/core
```

**Docker**:
```bash
docker run --rm -v $(pwd):/data ghcr.io/monykiss/m365kit kit word read /data/doc.docx
```

**Build from source**:
```bash
git clone https://github.com/monykiss/m365kit.git
cd m365kit
make build
./bin/kit --help
```

---

## Quick Start

```bash
# 1. Read a Word document
kit word read report.docx

# 2. Read as structured JSON
kit word read report.docx --json | jq .paragraphs

# 3. Read Excel data
kit excel read data.xlsx --sheet "Revenue" --json

# 4. AI summary (set your API key first)
export ANTHROPIC_API_KEY=sk-ant-...
kit word read contract.docx | kit ai summarize

# 5. Run an automation pipeline
kit pipeline run examples/pipelines/contract_review.yaml --verbose
```

---

## Features

| Feature | Status | Command |
|---------|--------|---------|
| Read Word (.docx) | ✅ | `kit word read` |
| Read Excel (.xlsx) | ✅ | `kit excel read` |
| Read PowerPoint (.pptx) | ✅ | `kit pptx read` |
| AI Summarize | ✅ | `kit ai summarize` |
| AI Analyze | ✅ | `kit ai analyze` |
| AI Entity Extraction | ✅ | `kit ai extract` |
| AI Q&A | ✅ | `kit ai ask` |
| Pipeline Workflows | ✅ | `kit pipeline run` |
| JSON output (all commands) | ✅ | `--json` flag |
| Markdown output | ✅ | `--markdown` flag |
| Stdin/stdout piping | ✅ | All commands |
| Anthropic (Claude) | ✅ | Default provider |
| OpenAI (GPT-4o) | ✅ | `--provider openai` |
| Ollama (local) | ✅ | `--provider ollama` |
| Write Word (.docx) | ✅ | `kit word write` |
| Edit Word (.docx) | ✅ | `kit word edit` |
| Write Excel (.xlsx) | ✅ | `kit excel write` |
| Batch processing | ✅ | `kit batch` |
| Pipeline dry-run | ✅ | `--dry-run` flag |
| Generate PowerPoint | 🚧 | Coming soon |

---

## AI Provider Setup

### Anthropic (default)

```bash
export ANTHROPIC_API_KEY=sk-ant-api03-...
kit ai summarize document.txt
```

### OpenAI

```bash
export OPENAI_API_KEY=sk-...
kit ai summarize document.txt --provider openai --model gpt-4o
```

### Ollama (local, no API key needed)

```bash
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

Run it:
```bash
kit pipeline run contract_review.yaml --verbose
```

See more examples in [`examples/pipelines/`](examples/pipelines/).

---

## Unix Philosophy

Every command is pipe-friendly. Compose them:

```bash
# Chain read → summarize → save
kit word read contract.docx | kit ai summarize > summary.txt

# Excel data → AI analysis
kit excel read metrics.xlsx --json | kit ai analyze --prompt "Find anomalies"

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
│   └── pipeline/           # kit pipeline run
├── internal/
│   ├── formats/            # OOXML parsers (docx, xlsx, pptx)
│   ├── ai/                 # Provider interface + implementations
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
  Built by <a href="https://github.com/klytics"><strong>KLYTICS</strong></a> 🇵🇷
</p>
