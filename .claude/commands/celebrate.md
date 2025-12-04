# Celebrate Milestone Release

You are celebrating a project milestone! Follow these steps to create a new release:

## 1. Gather Release Information

Ask the user for:
- **Version number** (e.g., v1.2.0) - suggest based on changes (major/minor/patch)
- **Release title** (e.g., "CLI & Cookie Authentication")
- **Key highlights** to include in release notes

## 2. Pre-Release Checklist

Run these checks and report results:

```bash
# Run all unit tests
go test ./...

# Check for uncommitted changes
git status

# Verify no credentials in repo
git log --all -p | grep -E "password|PASSWORD|secret|SECRET" | head -5 || echo "No credentials found"
```

## 3. Update Documentation

If needed, update:
- README.md - version references, feature list
- CLAUDE.md - project status metrics
- reports/vsp-status.md - if exists

## 4. Commit & Push

```bash
# Stage and commit any pending changes
git add -A
git commit -m "Prepare release vX.Y.Z"

# Push to origin
git push origin main
```

## 5. Build All Platforms

```bash
make build-all
```

Verify all 9 binaries are created in `build/` directory.

## 6. Create Git Tag

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z: <title>"
git push origin vX.Y.Z
```

## 7. Create GitHub Release

Use `gh release create` with:
- All binaries from `build/`
- Release notes highlighting key features
- Mark as latest release

```bash
gh release create vX.Y.Z \
  build/vsp-linux-amd64 \
  build/vsp-linux-arm64 \
  build/vsp-linux-386 \
  build/vsp-linux-arm \
  build/vsp-darwin-amd64 \
  build/vsp-darwin-arm64 \
  build/vsp-windows-amd64.exe \
  build/vsp-windows-arm64.exe \
  build/vsp-windows-386.exe \
  --title "vX.Y.Z: <title>" \
  --notes "$(cat <<'NOTES'
## What's New

- Feature 1
- Feature 2

## Downloads

| Platform | Architecture | File |
|----------|--------------|------|
| Linux | x64 | vsp-linux-amd64 |
| Linux | ARM64 | vsp-linux-arm64 |
| macOS | x64 | vsp-darwin-amd64 |
| macOS | Apple Silicon | vsp-darwin-arm64 |
| Windows | x64 | vsp-windows-amd64.exe |

## Installation

Download the appropriate binary, make it executable, and add to your PATH.

## Full Changelog

See commits since last release.
NOTES
)"
```

## 8. Celebrate!

Report the release URL and summary of what was shipped!
