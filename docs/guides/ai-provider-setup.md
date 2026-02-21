# AI Provider Setup

## Anthropic (Default)

```bash
export ANTHROPIC_API_KEY=sk-ant-api03-...
kit ai summarize document.txt
```

Or add to `~/.kit/config.yaml`:

```yaml
provider: anthropic
api_keys:
  anthropic: sk-ant-api03-...
```

## OpenAI

```bash
export OPENAI_API_KEY=sk-...
kit ai summarize document.txt --provider openai --model gpt-4o
```

## Ollama (Local)

```bash
ollama serve
ollama pull llama3.1
kit ai summarize document.txt --provider ollama --model llama3.1
```

Configure host in `~/.kit/config.yaml`:

```yaml
provider: ollama
ollama:
  host: http://localhost:11434
```
