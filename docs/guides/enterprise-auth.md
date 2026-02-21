# Enterprise Authentication

## Configuration File

M365Kit reads configuration from `~/.kit/config.yaml`:

```yaml
provider: anthropic
model: claude-sonnet-4-20250514

api_keys:
  anthropic: sk-ant-...
  openai: sk-...

ollama:
  host: http://localhost:11434

output:
  format: text
  color: true
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `OLLAMA_HOST` | Ollama server URL |
| `KIT_PROVIDER` | Default AI provider |
| `KIT_MODEL` | Default AI model |

Environment variables take precedence over config file values.
