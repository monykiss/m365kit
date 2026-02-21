# Word Commands

## kit word read

Extract text content from a .docx file.

```bash
kit word read <file.docx> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as structured JSON |
| `--markdown` | Output as Markdown |

### Examples

```bash
# Read as plain text
kit word read report.docx

# Read as JSON
kit word read report.docx --json

# Read from stdin
cat report.docx | kit word read -

# Pipe to AI
kit word read report.docx | kit ai summarize
```

## kit word write

Generate a .docx file from template and data. (Coming soon)

## kit word edit

Find and replace text in a .docx file. (Coming soon)

## kit word summarize

AI-powered document summary. (Coming soon — use `kit word read <file> | kit ai summarize`)
