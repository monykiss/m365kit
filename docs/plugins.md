# M365Kit Plugin System

M365Kit v1.2 supports plugins — custom commands installed in `~/.kit/plugins/`.

## Installing Plugins

```bash
# Local development install
kit plugin install --local ./my-plugin/

# List installed plugins
kit plugin list

# Remove a plugin
kit plugin remove my-plugin
```

## Writing a Shell Plugin

Generate a scaffold:

```bash
kit plugin new --name my-review --type shell
cd my-review/
```

This creates:

```
my-review/
├── kit-my-review          # the executable (chmod +x)
├── plugin.yaml            # plugin metadata
└── README.md              # usage documentation
```

Edit `kit-my-review` with your logic, then install:

```bash
kit plugin install --local ./my-review/
kit my-review <args>
```

### Environment Variables

Plugins receive M365Kit context via environment variables:

| Variable | Description |
|----------|-------------|
| `KIT_VERSION` | Current M365Kit version |
| `KIT_CONFIG_PATH` | Path to user's config.yaml |
| `KIT_TOKEN_PATH` | Path to OAuth token.json |
| `KIT_JSON` | `"true"` if `--json` was requested |
| `KIT_VERBOSE` | `"true"` if `--verbose` was set |

### Plugin Manifest (plugin.yaml)

```yaml
name: my-review
version: 0.1.0
description: "AI-powered document review"
author: "Your Name"
min_version: "1.2.0"
commands:
  - my-review
```

The `commands` field lists top-level command names that this plugin registers.
When installed, `kit my-review` will invoke the plugin directly.

## Writing a Go Plugin

```bash
kit plugin new --name my-tool --type go
cd my-tool/
```

Edit `main.go`, then build and install:

```go
package main

import (
    "fmt"
    "os"
)

func main() {
    args := os.Args[1:]
    if len(args) == 0 {
        fmt.Fprintln(os.Stderr, "Usage: kit my-tool <args>")
        os.Exit(1)
    }

    jsonOutput := os.Getenv("KIT_JSON") == "true"
    _ = jsonOutput

    fmt.Printf("Processing: %s\n", args[0])
}
```

```bash
go build -o kit-my-tool .
kit plugin install --local .
kit my-tool document.docx
```

## Example: NDA Review Plugin

```bash
#!/bin/bash
# kit-nda-review — Analyze an NDA for unusual clauses
set -euo pipefail

doc="${1:-}"
[ -z "$doc" ] && { echo "Usage: kit nda-review <nda.docx>" >&2; exit 1; }

kit ai analyze "$doc" \
  --prompt "Identify: 1) Non-compete scope and duration, 2) IP assignment clauses, 3) Any unusual confidentiality carve-outs, 4) Governing law" \
  ${KIT_JSON:+--json}
```

## Plugin Discovery Order

1. `~/.kit/plugins/kit-<name>` (direct executable)
2. `~/.kit/plugins/<name>/kit-<name>` (subdirectory install)
3. `kit-<name>` in `$PATH` (system-installed)

## Plugin Management

```bash
kit plugin list                    # List installed plugins
kit plugin show my-review          # Show details
kit plugin remove my-review        # Uninstall
kit plugin new --name x --type go  # Generate scaffold
```
