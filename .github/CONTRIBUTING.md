# Contributing to M365Kit

Thank you for your interest in contributing to M365Kit! This guide will help you get started.

## Development Setup

1. **Prerequisites:** Go 1.22+, Node.js 20+ (for TypeScript package)
2. **Clone:** `git clone https://github.com/klytics/m365kit.git`
3. **Build:** `make build`
4. **Test:** `make test`

## Making Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Run tests: `make test`
5. Run linter: `make lint`
6. Commit with a descriptive message
7. Open a pull request

## Code Standards

- All public functions must have Go doc comments
- No `panic()` calls — return errors up the stack
- No hardcoded API keys, model names, or file paths
- Error messages must tell the user what went wrong and how to fix it
- Every command must support `--json` flag for machine-readable output

## Testing

- Use golden file tests for parser output
- Test both success and error paths
- Include integration tests for CLI commands

## Reporting Bugs

Use the GitHub issue template. Include:
- What you expected to happen
- What happened instead
- Steps to reproduce
- Your OS and Go version

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
