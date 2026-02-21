# AI Commands

All AI commands require an API key. Set `ANTHROPIC_API_KEY` for the default provider.

## kit ai summarize

Generate a concise summary of document content.

```bash
kit ai summarize [file] [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as structured JSON |
| `--focus <areas>` | Comma-separated focus areas |
| `--provider <name>` | AI provider override |
| `--model <name>` | Model name override |

### Examples

```bash
kit ai summarize document.txt
kit word read contract.docx | kit ai summarize
kit ai summarize report.txt --focus "risks,dates,parties" --json
```

## kit ai analyze

Structured analysis of data or documents.

```bash
kit ai analyze [file] [flags]
```

## kit ai extract

Extract structured entities from documents.

```bash
kit ai extract --fields "name,date,amount" [file]
```

## kit ai ask

Ask questions about a document.

```bash
kit ai ask "What are the payment terms?" contract.docx
```
