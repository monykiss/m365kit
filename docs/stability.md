# M365Kit Stability Policy

## v1.0 Stability Guarantee

Starting with v1.0.0, M365Kit follows [Semantic Versioning](https://semver.org):

- **PATCH** (v1.0.x) — Bug fixes only. No new flags, no changed output format.
- **MINOR** (v1.x.0) — New commands or flags. Existing behavior unchanged.
- **MAJOR** (vX.0.0) — Breaking changes. Announced 60 days in advance.

## What is stable

- All command names (`kit word read`, `kit onedrive ls`, etc.)
- All required flags (their names and semantics)
- JSON output structure (`--json` flag output keys and types)
- Exit codes: 0 = success, 1 = user error, 2 = system error
- Config file location (`~/.kit/config.yaml`)
- Token file location (`~/.kit/token.json`)

## What may change in minor versions

- New optional flags added to existing commands
- New commands added (no existing commands removed)
- New JSON output fields added (existing fields not removed or renamed)
- Error message wording (exit codes remain stable)
- Performance improvements

## What changes in major versions (with 60-day notice)

- Command renamed or removed
- Required flag changed
- JSON output field removed or renamed
- Config file structure changed
- Authentication flow changed

## Deprecation process

1. Deprecated feature marked with `[DEPRECATED]` in `--help` output
2. Deprecation warning printed to stderr (not stdout) on use
3. Listed in CHANGELOG.md with target removal version
4. Removed in next major version (no sooner than 60 days after announcement)

## Reporting stability issues

If v1.x breaks something that worked in v1.0.x, open an issue tagged `bug/regression`.
Regressions are treated as P0 bugs and fixed in the next patch release.
