# PowerPoint Commands

## kit pptx read

Extract slide content from a .pptx file.

```bash
kit pptx read <file.pptx> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as structured JSON |

### Examples

```bash
kit pptx read presentation.pptx
kit pptx read presentation.pptx --json
```

## kit pptx generate

Generate a presentation from template and data. (Coming soon)
