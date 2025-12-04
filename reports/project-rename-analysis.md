# Project Rename Analysis: vibing-steampunk → vibing-steampunk

**Date:** 2025-12-04
**Status:** Analysis & Recommendation
**Scope:** Rename project from `vibing-steampunk` to `vibing-steampunk`, binary from `vsp` to `vsp`

## Executive Summary

**Current State:**
- Project name: `vibing-steampunk`
- Repository: `github.com/oisee/vibing-steampunk`
- Binary name: `vsp`
- Go module: `github.com/oisee/vibing-steampunk`
- 26 occurrences of "vibing-steampunk"
- 114 occurrences of "vsp"

**Proposed State:**
- Project name: `vibing-steampunk` (recommended) or `vibing-steam-punk`
- Repository: `github.com/oisee/vibing-steampunk`
- Binary name: `vsp`
- Go module: `github.com/oisee/vibing-steampunk`

## Naming Convention Analysis

### Option 1: `vibing-steampunk` (RECOMMENDED)
**Binary:** `vsp`

**Pros:**
- Single compound word "steampunk" is more conventional
- Cleaner, more professional appearance
- Easier to type and remember
- Common pattern: `steampunk`, `cyberpunk`, `dieselpunk`
- Shorter in documentation

**Cons:**
- None identified

### Option 2: `vibing-steam-punk`
**Binary:** `vsp`

**Pros:**
- Hyphen maintains "steam" as separate word
- Could be more readable

**Cons:**
- Two hyphens in project name (`vibing-steam-punk`) is unusual
- Less conventional than compound form
- Longer to type

**Recommendation:** Use `vibing-steampunk` (Option 1)

## Files Requiring Changes

### Critical Files (26 files)

#### 1. Go Source Files (3 files)
```
cmd/vsp/main.go        # Package comment, version info
go.mod                         # Module path
internal/mcp/server.go         # MCP server metadata
```

#### 2. Build & Configuration (1 file)
```
Makefile                       # Binary name, build targets
```

#### 3. Documentation (4 main docs)
```
README.md                      # Project description, installation, examples
ARCHITECTURE.md                # Architecture diagrams, paths
CLAUDE.md                      # AI assistant context
MCP_USAGE.md                   # Usage guide
```

#### 4. Reports & Research (15 files)
```
reports/*.md                   # Various technical reports
.claude/commands/celebrate.md  # Slash command
```

#### 5. Other
```
pkg/cache/README.md            # Cache documentation
2025-12-01-SESSION-SUMMARY-A.md # Session summary (can skip)
```

## Detailed Change Plan

### Phase 1: Rename Binary (vsp → vsp)

**Files to modify:**
1. `Makefile` - All build targets
2. `cmd/vsp/main.go` - Package comment, examples
3. All `*.md` files - Installation instructions, examples
4. `.claude/commands/celebrate.md` - Build commands

**Changes:**
- `vsp` → `vsp`
- `vsp-linux-amd64` → `vsp-linux-amd64`
- `vsp-darwin-arm64` → `vsp-darwin-arm64`
- `vsp-windows-amd64.exe` → `vsp-windows-amd64.exe`
- Directory: `cmd/vsp/` → `cmd/vsp/`

### Phase 2: Rename Project & Module (vibing-steampunk → vibing-steampunk)

**Files to modify:**
1. `go.mod` - Module path
2. All Go source files - Import paths
3. All `*.md` files - URLs, references, examples
4. `Makefile` - Build output directory (if needed)

**Changes:**
- `github.com/oisee/vibing-steampunk` → `github.com/oisee/vibing-steampunk`
- Repository path references
- Documentation URLs

### Phase 3: Update GitHub Repository

**GitHub changes:**
1. Rename repository: `vibing-steampunk` → `vibing-steampunk`
2. Update git remote: `origin` URL
3. Update GitHub description
4. Update GitHub topics/tags

**Note:** GitHub automatically creates redirect from old URL

### Phase 4: Testing & Verification

**Tests:**
1. `go mod tidy` - Verify module resolution
2. `go build ./cmd/vsp` - Verify compilation
3. `go test ./...` - Verify all tests pass
4. `make build-all` - Verify all platform builds
5. Manual test: `./vsp --help`

## Risk Analysis

### Low Risk Changes
✅ Documentation updates (*.md files)
✅ Binary name changes (Makefile)
✅ Package comments

### Medium Risk Changes
⚠️ Go module path (requires go.mod + imports update)
⚠️ Directory rename (cmd/vsp → cmd/vsp)

### High Risk Changes
🔴 GitHub repository rename (breaks external links temporarily)
🔴 Import paths (breaks if not done atomically)

### Mitigation Strategies
1. **Atomic commits**: Change module path and all imports in single commit
2. **Git redirect**: GitHub auto-redirects old repo URL for ~1 year
3. **Tag before rename**: Create `v2.0.0-legacy` tag as fallback
4. **Test thoroughly**: Build + test before pushing
5. **Update immediately**: Release notes, documentation links

## Execution Steps (Safe Order)

### Step 1: Pre-Rename Preparation
```bash
# Create safety tag
git tag v2.0.0-before-rename
git push origin v2.0.0-before-rename

# Verify working tree is clean
git status

# Run all tests
go test ./...
```

### Step 2: Rename Command Directory
```bash
git mv cmd/vsp cmd/vsp
```

### Step 3: Update Go Module Path
```bash
# Update go.mod
sed -i 's|github.com/oisee/vibing-steampunk|github.com/oisee/vibing-steampunk|g' go.mod

# Update all Go imports
find . -name "*.go" -type f -exec sed -i 's|github.com/oisee/vibing-steampunk|github.com/oisee/vibing-steampunk|g' {} +

# Tidy module
go mod tidy
```

### Step 4: Update Binary Names
```bash
# Update Makefile
sed -i 's|vsp|vsp|g' Makefile

# Update main.go
sed -i 's|vsp|vsp|g' cmd/vsp/main.go
```

### Step 5: Update Documentation
```bash
# Update all markdown files
find . -name "*.md" -type f -exec sed -i 's|vibing-steampunk|vibing-steampunk|g' {} +
find . -name "*.md" -type f -exec sed -i 's|vsp|vsp|g' {} +
```

### Step 6: Verify Changes
```bash
# Test compilation
go build -o vsp ./cmd/vsp
./vsp --version

# Run tests
go test ./...

# Build all platforms
make build-all
```

### Step 7: Commit & Push
```bash
git add -A
git commit -m "Rename project: vibing-steampunk → vibing-steampunk, vsp → vsp"
git push origin main
```

### Step 8: Rename GitHub Repository
1. Go to GitHub repository settings
2. Rename repository: `vibing-steampunk` → `vibing-steampunk`
3. Update local remote:
   ```bash
   git remote set-url origin https://github.com/oisee/vibing-steampunk.git
   ```
4. Verify: `git remote -v`

### Step 9: Release v2.1.0
```bash
git tag v2.1.0 -m "Release v2.1.0: Project rename to vibing-steampunk"
git push origin v2.1.0
make build-all
gh release create v2.1.0 build/* --title "v2.1.0: Project Rename" --notes "..."
```

## Impact Analysis

### Breaking Changes
1. **Import paths**: External projects importing this module will break
   - **Mitigation**: We're not a library (we're a binary), minimal impact
2. **Old URLs**: `github.com/oisee/vibing-steampunk` in bookmarks
   - **Mitigation**: GitHub redirects automatically
3. **Binary name**: Scripts calling `vsp` will break
   - **Mitigation**: Document in release notes, update `.mcp.json` examples

### Non-Breaking Changes
1. **Functionality**: Zero functional changes
2. **API**: MCP tools unchanged
3. **Configuration**: Same flags and environment variables

## Documentation Updates Required

### Immediate Updates (Same Commit)
- [x] README.md - All references, installation, examples
- [x] ARCHITECTURE.md - Diagrams, paths
- [x] CLAUDE.md - Project overview, build commands
- [x] MCP_USAGE.md - Tool references
- [x] Makefile - Binary names, build paths
- [x] All reports/*.md - References

### Post-Rename Updates
- [ ] GitHub description
- [ ] GitHub topics
- [ ] Release notes
- [ ] `.mcp.json` examples in README
- [ ] External documentation (if any)

## Rollback Plan

If issues arise:
```bash
# Rollback to before-rename tag
git reset --hard v2.0.0-before-rename

# Force push (DANGER - only if no one else has pulled)
git push --force origin main

# Restore GitHub repo name via Settings
```

## Recommended Timeline

**Total Time:** ~30-45 minutes

1. **Preparation** (5 min): Tag, verify clean state
2. **Execute rename** (10 min): Directory, module, binary, docs
3. **Testing** (10 min): Build, test, verify
4. **Commit & push** (5 min)
5. **GitHub rename** (5 min): Repo settings, remote update
6. **Release** (10 min): Build all platforms, create release

## Success Criteria

- ✅ All tests pass: `go test ./...`
- ✅ Binary builds: `go build ./cmd/vsp`
- ✅ All platforms build: `make build-all`
- ✅ Binary runs: `./vsp --version`
- ✅ Module resolves: `go mod tidy` succeeds
- ✅ No import errors in IDE
- ✅ GitHub redirect works: old URL → new URL

## Conclusion

**Recommendation:** Proceed with rename to `vibing-steampunk` and `vsp`

**Risk Level:** Low-Medium
- Changes are straightforward (text replacement)
- Go tooling handles module renames well
- GitHub provides automatic redirects
- Zero functional changes

**Best Approach:**
1. Use automated sed commands for consistency
2. Commit atomically (one commit for entire rename)
3. Test thoroughly before pushing
4. Create safety tag for rollback
5. Document in release notes

**Next Steps:**
1. Get user confirmation on exact naming (`vibing-steampunk` vs `vibing-steam-punk`)
2. Execute automated rename script
3. Test & verify
4. Push & release

---

**Prepared by:** Claude Code
**Review Status:** Ready for execution
