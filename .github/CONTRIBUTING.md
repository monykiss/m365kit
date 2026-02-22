# Contributing to M365Kit

Thank you for your interest in contributing! M365Kit is an open-source
project built by [KLYTICS LLC](https://github.com/klytics) and the community.

## Quick Start

    git clone https://github.com/monykiss/m365kit.git
    cd m365kit
    go mod download
    make build
    make test

## Development Requirements

- Go 1.22+
- Node.js 22+ (for `kit pptx generate` TypeScript bridge only)
- golangci-lint (for `make lint`)

Optional for Microsoft 365 features:
- Azure AD app registration with `KIT_AZURE_CLIENT_ID`

## Project Structure

    cmd/           CLI commands (one package per command)
    internal/      Internal packages (not part of public API)
      ai/          Provider interface + implementations
      auth/        Microsoft OAuth device code flow
      bridge/      Go-Node subprocess bridge
      config/      Config management
      email/       SMTP client
      formats/     OOXML parsers (docx, xlsx, pptx) + convert
      fs/          File system scanner, renamer, deduper
      graph/       Microsoft Graph API clients
      output/      Output formatting and JSON envelope
      pipeline/    YAML workflow engine + actions registry
      report/      Report generator
      template/    Document template engine
      update/      Update checker
      watch/       File watcher (fsnotify)
    benchmarks/    Go benchmark suite
    packages/core/ TypeScript bridge (@m365kit/core)
    docs/          Documentation
    examples/      Example pipeline YAML files
    testdata/      Test fixtures
    tests/         Smoke / integration tests

## Adding a New Command

1. Create `cmd/mycommand/mycommand.go` with `NewCommand() *cobra.Command`
2. Register in `cmd/root.go`
3. Write tests in the corresponding `internal/` package
4. Add `--json` flag support using `internal/output` helpers
5. Add `--dry-run` for any command that writes files or makes network calls
6. Update the Features table in README.md
7. Add a smoke test in `tests/smoke_test.go`

## Testing

    make test         # run all unit tests
    make smoke        # build + run smoke tests
    make benchmark    # run benchmarks

Every new command must have:
- At least 3 unit tests
- A smoke test that validates --help exits 0

## Code Style

- `golangci-lint run ./...` must be clean
- `go vet ./...` must be clean
- Error messages are lowercase, human-readable, with actionable fix instructions
- Token/API key values are NEVER logged or included in error messages

## Pull Request Process

1. Fork the repo and create your branch: `git checkout -b feat/my-feature`
2. Write tests for your changes
3. Confirm: `make build && make test && go vet ./...`
4. Open a PR with the checklist filled out (see `.github/pull_request_template.md`)
5. A maintainer will review within 48 hours

## Reporting Security Issues

Do NOT open a public GitHub issue for security vulnerabilities.
Email `security@klytics.com` with the subject "M365Kit Security".
We will respond within 24 hours and coordinate a fix + disclosure.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
