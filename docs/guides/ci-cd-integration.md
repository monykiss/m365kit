# CI/CD Integration

## GitHub Actions

```yaml
- name: Install M365Kit
  run: go install github.com/klytics/m365kit/cmd@latest

- name: Process documents
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    kit word read docs/spec.docx --json > spec.json
    kit ai summarize docs/spec.docx > summary.md
```

## Docker

```bash
docker run --rm \
  -v $(pwd):/data \
  -e ANTHROPIC_API_KEY \
  ghcr.io/klytics/m365kit \
  kit word read /data/document.docx --json
```
