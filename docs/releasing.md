# Release Process

## Steps

1. Bump version in `cmd/version/version.go` and `packages/core/package.json`
2. Commit: `git commit -m "chore: bump version to vX.Y.Z"`
3. Tag: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`
4. GoReleaser builds cross-platform binaries automatically via CI (`.github/workflows/release.yml`)
5. Get sha256 hashes from the release's `checksums.txt` asset
6. Update `homebrew-tap/Formula/m365kit.rb` with new version + sha256s
7. Update `@m365kit/core`: `cd packages/core && npm version X.Y.Z && npm publish --access public`

## CI Secrets Required

- `GITHUB_TOKEN` — automatically provided by GitHub Actions
- `NPM_TOKEN` — must be set as a GitHub Actions secret for npm publish

## Homebrew Formula Update

After GoReleaser creates the release with tarballs:

```bash
# Download checksums
gh release download vX.Y.Z --repo monykiss/m365kit --pattern checksums.txt

# Update formula in homebrew-tap repo
cd /path/to/homebrew-tap
# Edit Formula/m365kit.rb with new version and sha256 values
git commit -am "feat: update m365kit to vX.Y.Z"
git push
```

## Checklist

- [ ] CI green on main before tagging
- [ ] CHANGELOG.md updated
- [ ] homebrew-tap formula updated with sha256 hashes
- [ ] npm package published (`npm show @m365kit/core version`)
- [ ] GitHub Release notes written
