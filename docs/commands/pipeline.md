# Pipeline Commands

## kit pipeline run

Execute a multi-step workflow from a YAML file.

```bash
kit pipeline run <workflow.yaml> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output step results as JSON |
| `--verbose` | Show detailed execution progress |

### Pipeline YAML Format

```yaml
name: my_workflow
version: "1.0"
steps:
  - id: step_name
    action: excel.read
    input: ./data.xlsx
    options:
      sheet: Revenue
  - id: analysis
    action: ai.analyze
    input: ${{ steps.step_name.output }}
```

### Variable Interpolation

Use `${{ steps.<id>.output }}` to reference the output of a previous step.

Built-in variables:
- `${{ date.today }}` — current date (YYYY-MM-DD)
- `${{ date.now }}` — current timestamp (RFC 3339)
