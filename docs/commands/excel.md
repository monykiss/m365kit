# Excel Commands

## kit excel read

Extract data from an .xlsx file.

```bash
kit excel read <file.xlsx> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as structured JSON |
| `--csv` | Output as CSV |
| `--sheet <name>` | Read only the named sheet |

### Examples

```bash
# Pretty-printed table
kit excel read data.xlsx

# Specific sheet as JSON
kit excel read data.xlsx --sheet "Revenue" --json

# Export as CSV
kit excel read data.xlsx --csv > output.csv
```

## kit excel write

Generate an .xlsx file from data. (Coming soon)

## kit excel analyze

AI-powered spreadsheet analysis. (Coming soon)
