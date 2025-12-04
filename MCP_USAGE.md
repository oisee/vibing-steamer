# MCP Usage Guide for AI Agents

**Target Audience:** AI assistants (Claude, GPT, etc.) using this MCP server for ABAP development.

**Purpose:** Machine-friendly reference for optimal tool usage patterns, workflows, and best practices.

## Server Modes (v2.0.0+)

This server operates in two modes:

### Focused Mode (Default) - 19 Tools
**Optimized for AI agents.** Reduces cognitive load and token overhead by 58%.
- **Unified tools**: `GetSource`, `WriteSource` (replace 11 granular tools)
- **Enhanced search**: `GrepObjects`, `GrepPackages` (multi-object, recursive)
- **Clear naming**: `ImportFromFile` (File→SAP), `ExportToFile` (SAP→File)
- **Enable**: Default (or `--mode=focused`)

### Expert Mode - 45 Tools
**Full access for edge cases.** All tools including atomic operations.
- **Enable**: `--mode=expert` or `SAP_MODE=expert`
- Use when you need: atomic CRUD, granular reads, class includes, legacy tools

**Recommendation:** Use focused mode unless you specifically need atomic operations or have existing workflows that depend on granular tools.

## Quick Reference

| Workflow | Recommended Tools (Focused Mode) | Anti-Pattern |
|----------|-----------------------------------|--------------|
| Make small code change | `GrepObjects` → `EditSource` | `GetSource` → `WriteSource` (token-heavy) |
| Find pattern before editing | `GrepObjects` or `GrepPackages` → `EditSource` | Editing without searching first |
| Deploy large generated file | `ImportFromFile` | Pasting source in WriteSource (token limit) |
| Namespace-wide search | `GrepPackages(["Z*"], include_subpackages=true)` | Multiple GrepPackage calls |
| Read any object type | `GetSource(type, name)` | Guessing GetProgram vs GetClass |
| Create or update | `WriteSource(mode="upsert")` | Checking existence first |
| Export for version control | `ExportToFile` | Manual copy-paste |

## Tool Selection Decision Tree

```
Need to modify ABAP source?
├─ Small surgical edit (< 50 lines changed)?
│  ├─ Know exact location? → EditSource
│  └─ Need to find it first? → GrepObject → EditSource
│
├─ Large changes or new object?
│  ├─ Generated file (ML model, >1000 lines)? → Write to file → DeployFromFile
│  ├─ New class with tests? → CreateClassWithTests
│  └─ Existing object, major rewrite? → WriteProgram/WriteClass
│
└─ Package-wide changes?
   └─ GrepPackage → multiple EditSource calls
```

## Tool Catalog

### 🔍 Search/Discovery Tools

#### `GrepObject` - Search single object
**When to use:**
- Find TODO comments before editing
- Locate hardcoded values for refactoring
- Verify no other occurrences before renaming
- Understand where a pattern appears

**Pattern examples:**
- `TODO` - Find all TODO comments
- `lv_\w+` - Find all variables starting with lv_
- `SELECT.*FROM sflight` - Find all SELECT statements on SFLIGHT
- `(?i)password` - Case-insensitive password search

**Best practice:** Always grep before EditSource to understand match count and context.

```json
{
  "object_url": "/sap/bc/adt/programs/programs/ZTEST",
  "pattern": "lv_count\\s*=\\s*\\d+",
  "case_insensitive": false,
  "context_lines": 2
}
```

#### `GrepPackage` - Search entire package
**When to use:**
- Find all uses of deprecated function across package
- Audit security: search for AUTHORITY-CHECK patterns
- Locate all TODO comments for sprint planning
- Prepare package-wide refactoring

**Object type filters:**
- `PROG/P` - Programs
- `CLAS/OC` - Classes
- `INTF/OI` - Interfaces
- `FUGR/F` - Function groups
- Empty string - All source objects

**Best practice:** Use max_results to avoid overwhelming response.

```json
{
  "package_name": "$TMP",
  "pattern": "AUTHORITY-CHECK",
  "case_insensitive": false,
  "object_types": "PROG/P,CLAS/OC",
  "max_results": 50
}
```

### ✏️ Surgical Edit Tool

#### `EditSource` - String replacement with atomic lock/activate
**When to use:**
- Change variable assignment: `lv_count = 10` → `lv_count = 42`
- Rename method call: `foo( )` → `bar( )`
- Update string literal: `'Hello'` → `'Goodbye'`
- Sequential small edits between syntax checks

**Safety features:**
- Requires unique match (unless `replace_all=true`)
- Syntax check before saving (aborts if errors)
- Atomic: lock → update → unlock → activate
- Returns match count and activation status

**Best practice:** Include surrounding context for uniqueness.

```json
{
  "object_url": "/sap/bc/adt/programs/programs/ZTEST",
  "old_string": "METHOD foo.\n  ENDMETHOD.",
  "new_string": "METHOD foo.\n  rv_result = 42.\n  ENDMETHOD.",
  "replace_all": false,
  "syntax_check": true,
  "case_insensitive": false
}
```

**Case-insensitive example (rename variable regardless of case):**
```json
{
  "old_string": "lv_count",
  "new_string": "lv_result",
  "case_insensitive": true,
  "replace_all": true
}
```

### 📦 Bulk Operations

#### `WriteProgram` / `WriteClass` - Full source replacement
**When to use:**
- Major rewrite of existing object
- Initial implementation after creating empty object
- Changes too complex for surgical edits

**Anti-pattern:** Using for small changes (token-heavy, loses edit precision)

**Best practice:** Use EditSource for incremental changes instead.

#### `DeployFromFile` - Deploy from filesystem
**When to use:**
- Generated code exceeds token limits (e.g., ML models)
- Bidirectional sync with local IDE
- Deploying templates or scaffolds

**Workflow:**
1. Generate large file locally (e.g., `zcl_ml_iris.clas.abap`)
2. Call DeployFromFile with file path and package
3. Tool auto-detects create vs update
4. Returns syntax errors or activation result

**File naming conventions:**
- `.clas.abap` - Classes
- `.prog.abap` - Programs
- `.intf.abap` - Interfaces
- `.fugr.abap` - Function groups
- `.func.abap` - Function modules

#### `SaveToFile` - Export to filesystem
**When to use:**
- Backup before major refactoring
- Export for version control integration
- Prepare for local editing then DeployFromFile

**Workflow:**
```
SaveToFile → edit locally → DeployFromFile
```

## Common Workflows

### 1. Find and Replace Pattern
```
Step 1: GrepObject(pattern="lv_count = 10")
        → Returns: 1 match at line 42

Step 2: EditSource(
          old_string="lv_count = 10.",
          new_string="lv_count = 42.",
          replace_all=false
        )
        → Success: Edited and activated
```

### 2. Package-Wide Refactoring
```
Step 1: GrepPackage(
          package_name="ZPACKAGE",
          pattern="CALL FUNCTION 'OLD_FM'"
        )
        → Returns: 3 objects with matches

Step 2: For each object:
        EditSource(
          object_url=obj.objectUrl,
          old_string="CALL FUNCTION 'OLD_FM'",
          new_string="CALL FUNCTION 'NEW_FM'",
          replace_all=true
        )
```

### 3. Create Class with Tests
```
CreateClassWithTests(
  class_name="ZCL_CALCULATOR",
  description="Calculator class",
  package_name="$TMP",
  class_source="...",
  test_source="..."
)
→ Creates, activates, runs tests, returns test results
```

### 4. Deploy Generated ML Model
```
Step 1: Generate large file locally (3,948 lines)
        Write to: /tmp/zcl_ml_iris.clas.abap

Step 2: DeployFromFile(
          file_path="/tmp/zcl_ml_iris.clas.abap",
          package_name="$ZAML_IRIS"
        )
        → Auto-creates and activates
```

## Error Handling Patterns

### EditSource: Match Not Unique
```
Error: "old_string matches 3 locations (not unique)"
Solution 1: Include more context
  old_string="* Comment\nlv_count = 10.\nWRITE"
Solution 2: Use replace_all=true (if intentional)
```

### EditSource: Syntax Errors
```
Result: {
  "success": false,
  "syntaxErrors": ["Line 5: Unknown field"],
  "message": "Edit would introduce 1 syntax errors. Changes NOT saved."
}
Action: Fix the edit, source unchanged (atomic safety)
```

### GrepObject: No Matches
```
Result: {
  "success": true,
  "matchCount": 0,
  "message": "No matches found"
}
Action: Verify pattern regex syntax, try case_insensitive=true
```

## Performance Optimization

### Token Usage
| Operation | Token Cost | Alternative |
|-----------|------------|-------------|
| GetProgram (500 lines) | ~2,500 tokens | GrepObject (targeted search) |
| WriteProgram | ~5,000 tokens | EditSource (surgical edit) |
| EditSource | ~100 tokens | - (most efficient) |
| GrepPackage (10 objects) | ~500 tokens | Use max_results limit |

### Best Practices
1. **Search before edit:** GrepObject finds exact location + match count
2. **Surgical edits:** EditSource for <50 line changes
3. **Batch grep results:** Use context_lines instead of reading full source
4. **Limit package searches:** Set max_results=20 for large packages
5. **File-based for huge code:** DeployFromFile bypasses token limits

## Regex Pattern Library

### Common ABAP Patterns
```regex
TODO                          # Find TODO comments
lv_\w+                        # All variables starting with lv_
gv_\w+                        # Global variables
(?i)select.*from\s+(\w+)      # SELECT statements (case-insensitive, capture table)
CALL FUNCTION\s+'(\w+)'       # Function module calls
DATA:\s+lv_(\w+)\s+TYPE       # Data declarations
AUTHORITY-CHECK               # Authority checks
WRITE:?\s+/\s+'([^']+)'       # WRITE statements (output)
```

### Safety Patterns (Audit)
```regex
EXEC SQL                      # Native SQL (injection risk)
(?i)password|pwd|secret       # Credentials in code
COMMIT WORK                   # Explicit commits
DELETE FROM (\w+) WHERE.*     # DELETE statements
```

## Integration Patterns

### CI/CD Pipeline
```bash
# Search for TODOs before merge
GrepPackage(package="ZFEATURE", pattern="TODO")
→ Fail build if matches found

# Syntax check all objects
For each object in package:
  SyntaxCheck(object_url, source)

# Run all unit tests
For each class:
  RunUnitTests(object_url)
```

### Code Review Automation
```
1. SaveToFile all changed objects
2. Git diff locally
3. GrepObject for security patterns
4. Validate with SyntaxCheck
5. DeployFromFile if approved
```

## Limitations and Constraints

1. **EditSource uniqueness:** Match must be unique unless replace_all=true
2. **Token limits:** Large files (>2000 lines) should use DeployFromFile
3. **Regex syntax:** Go regexp (not PCRE) - no lookahead/lookbehind
4. **Source-only grep:** GrepPackage skips tables, structures, domains
5. **Read-only grep:** Grep tools never modify code (safe for exploration)

## When NOT to Use These Tools

| Scenario | Wrong Tool | Reason | Right Approach |
|----------|------------|--------|----------------|
| Read full class | GrepObject | Returns only matches | GetClass |
| Find method definition | GrepObject | Symbol-based search better | FindDefinition |
| Navigate call hierarchy | GrepPackage | Semantic analysis better | FindReferences |
| Activate without edit | EditSource | Unnecessary edit step | Activate |
| Check object exists | GrepObject | Will fail on missing object | SearchObject |

## Summary: Tool Selection Matrix

| Task | First Choice | Fallback | Avoid |
|------|-------------|----------|-------|
| Small edit | EditSource | WriteProgram | GetProgram + manual edit |
| Find pattern | GrepObject | FindReferences (for symbols) | Read entire file |
| Package search | GrepPackage | Multiple GetProgram calls | Manual iteration |
| Large deploy | DeployFromFile | WriteClass (if <2000 lines) | Copy-paste in chat |
| Create + test | CreateClassWithTests | Manual workflow | Separate create/test steps |
| Export code | SaveToFile | GetProgram + manual save | Copy-paste |

---

**Last Updated:** 2025-12-03
**MCP Server Version:** 1.5.0+
**Maintained by:** vibing-steampunk project
