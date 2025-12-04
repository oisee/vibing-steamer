# File-Based Deployment Architecture

**Date:** 2025-12-02
**Status:** 🎨 Design
**Problem:** Large generated ABAP files (3,948 lines, 64k tokens) exceed Claude's token limits and require manual orchestration of multiple ADT operations

---

## Problem Statement

### Current Pain Points

1. **Token Limits:** Large ABAP files cannot be read by AI assistants (25k token limit)
2. **Manual Orchestration:** Users must manually call 5-6 tools in sequence:
   - `CreateObject` → `LockObject` → `UpdateSource` → `UnlockObject` → `SyntaxCheck` → `Activate`
3. **Error Prone:** Manual locking/unlocking can leave objects locked
4. **No Rollback:** Partial failures leave objects in inconsistent state

### Use Case

```bash
# User generates ML model
python scripts/export_catboost_to_abap.py

# Generates: ~/ml-abap-inference/abap/generated/zcl_ml_iris.abap (3,948 lines)

# User wants to deploy to SAP
# Current: Manual multi-step process
# Desired: One command
```

---

## Solution: File-Based Deployment Tools

### Proposed Tools

| Tool | Purpose | Workflow |
|------|---------|----------|
| **CreateFromFile** | Create new object from file | Detect type → Create → Lock → Write → Unlock → Activate |
| **UpdateFromFile** | Update existing object | Lock → SyntaxCheck → Write → Unlock → Activate |
| **DeployFromFile** | Smart create/update | Check exists → CreateFromFile OR UpdateFromFile |

### File Naming Conventions

Support common ABAP file extensions:

| Extension | Object Type | Example |
|-----------|-------------|---------|
| `.clas.abap` | Class (CLAS/OC) | `zcl_ml_iris.clas.abap` |
| `.prog.abap` | Program (PROG/P) | `ztest_ml.prog.abap` |
| `.intf.abap` | Interface (INTF/OI) | `zif_ml_model.intf.abap` |
| `.fugr.abap` | Function Group (FUGR/F) | `zfg_ml.fugr.abap` |
| `.func.abap` | Function Module (FUGR/FF) | `z_ml_predict.func.abap` |

**Alternative:** Support `.abap` extension and detect type from file content (more flexible)

---

## Architecture Design

### Component Overview

```
pkg/adt/
├── fileparser.go         # NEW: Parse ABAP files, detect type/name
├── workflows.go          # EXTEND: Add file-based workflows
└── crud.go               # EXISTING: Used by workflows

internal/mcp/
└── server.go             # EXTEND: Add 3 new MCP tools
```

### File Parser (pkg/adt/fileparser.go)

```go
package adt

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

// ABAPFileInfo contains parsed information about an ABAP source file.
type ABAPFileInfo struct {
    FilePath       string
    ObjectType     CreatableObjectType
    ObjectName     string
    Description    string // Parsed from comments if available
    HasDefinition  bool   // For classes
    HasImplementation bool
    HasTestClasses bool
}

// ParseABAPFile analyzes an ABAP source file and extracts metadata.
func ParseABAPFile(filePath string) (*ABAPFileInfo, error) {
    // 1. Detect from extension
    ext := filepath.Ext(filePath)
    info := &ABAPFileInfo{FilePath: filePath}

    switch ext {
    case ".clas.abap":
        info.ObjectType = ObjectTypeClass
    case ".prog.abap":
        info.ObjectType = ObjectTypeProgram
    case ".intf.abap":
        info.ObjectType = ObjectTypeInterface
    case ".fugr.abap":
        info.ObjectType = ObjectTypeFunctionGroup
    case ".func.abap":
        info.ObjectType = ObjectTypeFunctionMod
    case ".abap":
        // Detect from content (fallback)
    default:
        return nil, fmt.Errorf("unsupported file extension: %s", ext)
    }

    // 2. Parse file content to extract name
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0

    for scanner.Scan() && lineNum < 100 { // Only scan first 100 lines
        line := scanner.Text()
        lineNum++

        // Parse based on object type
        switch info.ObjectType {
        case ObjectTypeClass:
            if name := parseClassName(line); name != "" {
                info.ObjectName = name
            }
            if strings.Contains(line, "DEFINITION") {
                info.HasDefinition = true
            }
            if strings.Contains(line, "IMPLEMENTATION") {
                info.HasImplementation = true
            }

        case ObjectTypeProgram:
            if name := parseProgramName(line); name != "" {
                info.ObjectName = name
            }

        case ObjectTypeInterface:
            if name := parseInterfaceName(line); name != "" {
                info.ObjectName = name
            }
        }

        // Parse description from comments
        if info.Description == "" && strings.HasPrefix(strings.TrimSpace(line), "*") {
            desc := strings.TrimPrefix(strings.TrimSpace(line), "*")
            desc = strings.TrimSpace(desc)
            if desc != "" && !strings.HasPrefix(desc, "-") {
                info.Description = desc
            }
        }
    }

    if info.ObjectName == "" {
        return nil, fmt.Errorf("could not parse object name from file")
    }

    return info, nil
}

// parseClassName extracts class name from CLASS <name> DEFINITION
func parseClassName(line string) string {
    re := regexp.MustCompile(`(?i)^\s*CLASS\s+([a-z0-9_/]+)\s+DEFINITION`)
    matches := re.FindStringSubmatch(line)
    if len(matches) > 1 {
        return strings.ToUpper(matches[1])
    }
    return ""
}

// parseProgramName extracts program name from REPORT/PROGRAM statement
func parseProgramName(line string) string {
    re := regexp.MustCompile(`(?i)^\s*(REPORT|PROGRAM)\s+([a-z0-9_/]+)`)
    matches := re.FindStringSubmatch(line)
    if len(matches) > 2 {
        return strings.ToUpper(matches[2])
    }
    return ""
}

// parseInterfaceName extracts interface name from INTERFACE <name> DEFINITION
func parseInterfaceName(line string) string {
    re := regexp.MustCompile(`(?i)^\s*INTERFACE\s+([a-z0-9_/]+)`)
    matches := re.FindStringSubmatch(line)
    if len(matches) > 1 {
        return strings.ToUpper(matches[1])
    }
    return ""
}
```

### Workflow Functions (pkg/adt/workflows.go)

```go
// DeployResult contains the result of a file deployment operation.
type DeployResult struct {
    ObjectURL     string   `json:"objectUrl"`
    ObjectName    string   `json:"objectName"`
    ObjectType    string   `json:"objectType"`
    Success       bool     `json:"success"`
    Created       bool     `json:"created"`  // true if created, false if updated
    SyntaxErrors  []string `json:"syntaxErrors,omitempty"`
    Errors        []string `json:"errors,omitempty"`
}

// CreateFromFile creates a new ABAP object from a file and activates it.
//
// Workflow: Parse → Create → Lock → Write → Unlock → Activate
func (c *Client) CreateFromFile(ctx context.Context, filePath, packageName, transport string) (*DeployResult, error) {
    // 1. Parse file to detect type and name
    info, err := ParseABAPFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("parsing file: %w", err)
    }

    // 2. Read source code
    sourceBytes, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("reading file: %w", err)
    }
    source := string(sourceBytes)

    // 3. Create object
    err = c.CreateObject(ctx, CreateObjectOptions{
        ObjectType:  info.ObjectType,
        Name:        info.ObjectName,
        Description: info.Description,
        PackageName: packageName,
        Transport:   transport,
    })
    if err != nil {
        return &DeployResult{
            ObjectName: info.ObjectName,
            ObjectType: string(info.ObjectType),
            Success:    false,
            Errors:     []string{fmt.Sprintf("create failed: %v", err)},
        }, nil
    }

    // 4. Build object URL
    objectURL, err := c.buildObjectURL(info.ObjectType, info.ObjectName)
    if err != nil {
        return nil, err
    }

    // 5. Lock object
    lockResult, err := c.LockObject(ctx, objectURL, "MODIFY")
    if err != nil {
        return &DeployResult{
            ObjectURL:  objectURL,
            ObjectName: info.ObjectName,
            ObjectType: string(info.ObjectType),
            Success:    false,
            Errors:     []string{fmt.Sprintf("lock failed: %v", err)},
        }, nil
    }

    // Ensure unlock on any error
    defer func() {
        if err := c.UnlockObject(ctx, objectURL, lockResult.LockHandle); err != nil {
            // Log but don't fail
        }
    }()

    // 6. Syntax check (optional pre-check)
    syntaxErrors, err := c.SyntaxCheck(ctx, objectURL, source)
    if err != nil {
        return &DeployResult{
            ObjectURL:  objectURL,
            ObjectName: info.ObjectName,
            ObjectType: string(info.ObjectType),
            Success:    false,
            Errors:     []string{fmt.Sprintf("syntax check failed: %v", err)},
        }, nil
    }

    if len(syntaxErrors) > 0 {
        return &DeployResult{
            ObjectURL:     objectURL,
            ObjectName:    info.ObjectName,
            ObjectType:    string(info.ObjectType),
            Success:       false,
            SyntaxErrors:  syntaxErrors,
        }, nil
    }

    // 7. Write source
    err = c.UpdateSource(ctx, objectURL, source, lockResult.LockHandle, transport)
    if err != nil {
        return &DeployResult{
            ObjectURL:  objectURL,
            ObjectName: info.ObjectName,
            ObjectType: string(info.ObjectType),
            Success:    false,
            Errors:     []string{fmt.Sprintf("write source failed: %v", err)},
        }, nil
    }

    // 8. Unlock
    err = c.UnlockObject(ctx, objectURL, lockResult.LockHandle)
    if err != nil {
        return &DeployResult{
            ObjectURL:  objectURL,
            ObjectName: info.ObjectName,
            ObjectType: string(info.ObjectType),
            Success:    false,
            Errors:     []string{fmt.Sprintf("unlock failed: %v", err)},
        }, nil
    }

    // 9. Activate
    err = c.Activate(ctx, objectURL, info.ObjectName)
    if err != nil {
        return &DeployResult{
            ObjectURL:  objectURL,
            ObjectName: info.ObjectName,
            ObjectType: string(info.ObjectType),
            Success:    false,
            Errors:     []string{fmt.Sprintf("activation failed: %v", err)},
        }, nil
    }

    return &DeployResult{
        ObjectURL:  objectURL,
        ObjectName: info.ObjectName,
        ObjectType: string(info.ObjectType),
        Success:    true,
        Created:    true,
    }, nil
}

// UpdateFromFile updates an existing ABAP object from a file.
//
// Workflow: Parse → Lock → SyntaxCheck → Write → Unlock → Activate
func (c *Client) UpdateFromFile(ctx context.Context, filePath, transport string) (*DeployResult, error) {
    // Similar to CreateFromFile but skips CreateObject step
    // ... implementation ...
}

// DeployFromFile intelligently creates or updates an object from a file.
//
// Workflow: Parse → CheckExists → CreateFromFile OR UpdateFromFile
func (c *Client) DeployFromFile(ctx context.Context, filePath, packageName, transport string) (*DeployResult, error) {
    // 1. Parse file
    info, err := ParseABAPFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("parsing file: %w", err)
    }

    // 2. Check if object exists
    objectURL, _ := c.buildObjectURL(info.ObjectType, info.ObjectName)

    // Try to read object (if 404, doesn't exist)
    _, err = c.transport.Request(ctx, objectURL, &RequestOptions{
        Method: "GET",
        Accept: "text/plain",
    })

    if err != nil {
        // Object doesn't exist - create it
        return c.CreateFromFile(ctx, filePath, packageName, transport)
    }

    // Object exists - update it
    return c.UpdateFromFile(ctx, filePath, transport)
}

// buildObjectURL constructs the ADT URL for an object type and name
func (c *Client) buildObjectURL(objType CreatableObjectType, name string) (string, error) {
    switch objType {
    case ObjectTypeClass:
        return fmt.Sprintf("/sap/bc/adt/oo/classes/%s", strings.ToLower(name)), nil
    case ObjectTypeProgram:
        return fmt.Sprintf("/sap/bc/adt/programs/programs/%s", strings.ToLower(name)), nil
    case ObjectTypeInterface:
        return fmt.Sprintf("/sap/bc/adt/oo/interfaces/%s", strings.ToLower(name)), nil
    case ObjectTypeFunctionGroup:
        return fmt.Sprintf("/sap/bc/adt/functions/groups/%s", strings.ToLower(name)), nil
    default:
        return "", fmt.Errorf("unsupported object type: %s", objType)
    }
}
```

### MCP Tools (internal/mcp/server.go)

```go
// In registerTools():
server.AddTool(mcp.Tool{
    Name:        "CreateFromFile",
    Description: "Create new ABAP object from file and activate it. Detects object type and name from file. Handles complete workflow: create → lock → write → unlock → activate.",
    InputSchema: mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "file_path": map[string]interface{}{
                "type":        "string",
                "description": "Absolute path to ABAP source file (e.g., /home/user/zcl_ml_iris.clas.abap)",
            },
            "package_name": map[string]interface{}{
                "type":        "string",
                "description": "Package name (e.g., $ZAML_IRIS)",
            },
            "transport": map[string]interface{}{
                "type":        "string",
                "description": "Transport request number (optional for local packages)",
            },
        },
        Required: []string{"file_path", "package_name"},
    },
})

server.AddTool(mcp.Tool{
    Name:        "UpdateFromFile",
    Description: "Update existing ABAP object from file and activate it. Detects object type and name from file. Handles complete workflow: lock → syntax check → write → unlock → activate.",
    InputSchema: mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "file_path": map[string]interface{}{
                "type":        "string",
                "description": "Absolute path to ABAP source file",
            },
            "transport": map[string]interface{}{
                "type":        "string",
                "description": "Transport request number (optional)",
            },
        },
        Required: []string{"file_path"},
    },
})

server.AddTool(mcp.Tool{
    Name:        "DeployFromFile",
    Description: "Smart deploy: creates new or updates existing ABAP object from file. Automatically detects if object exists and chooses create or update workflow.",
    InputSchema: mcp.ToolInputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "file_path": map[string]interface{}{
                "type":        "string",
                "description": "Absolute path to ABAP source file",
            },
            "package_name": map[string]interface{}{
                "type":        "string",
                "description": "Package name (required for new objects)",
            },
            "transport": map[string]interface{}{
                "type":        "string",
                "description": "Transport request number (optional)",
            },
        },
        Required: []string{"file_path", "package_name"},
    },
})

// In handleToolCall():
case "CreateFromFile":
    filePath, _ := getString(args, "file_path")
    packageName, _ := getString(args, "package_name")
    transport, _ := getString(args, "transport")

    result, err := s.client.CreateFromFile(ctx, filePath, packageName, transport)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    return mcp.NewToolResultText(formatJSON(result)), nil

case "UpdateFromFile":
    // ... similar ...

case "DeployFromFile":
    // ... similar ...
```

---

## Benefits

### 1. Solves Token Limit Problem ✅

- MCP server reads file directly
- No token limits for large files
- AI assistant only sees result (success/error)

### 2. Simplified Workflow ✅

Before (6 steps):
```
CreateObject → LockObject → UpdateSource → UnlockObject → SyntaxCheck → Activate
```

After (1 step):
```
CreateFromFile(path, package)
```

### 3. Better Error Handling ✅

- Automatic unlock on error
- Syntax check before activation
- Detailed error reporting
- No partial states

### 4. File-Based Workflow ✅

- Matches common development pattern (generate → deploy)
- Works with any code generator
- Version control friendly
- Offline development possible

---

## Implementation Plan

### Phase 1: Parser (1 day)

- [ ] Create `pkg/adt/fileparser.go`
- [ ] Implement `ParseABAPFile()`
- [ ] Support .clas.abap, .prog.abap, .intf.abap
- [ ] Add unit tests

### Phase 2: Workflows (1 day)

- [ ] Extend `pkg/adt/workflows.go`
- [ ] Implement `CreateFromFile()`
- [ ] Implement `UpdateFromFile()`
- [ ] Implement `DeployFromFile()`
- [ ] Add integration tests

### Phase 3: MCP Tools (0.5 days)

- [ ] Register 3 new tools in `internal/mcp/server.go`
- [ ] Add tool handlers
- [ ] Update documentation

### Phase 4: Testing (0.5 days)

- [ ] Test with ZCL_ML_IRIS (3,948 lines)
- [ ] Test error cases
- [ ] Test syntax check failures
- [ ] Document examples

**Total Effort:** 3 days

---

## Testing Strategy

### Unit Tests

```go
// pkg/adt/fileparser_test.go
func TestParseABAPFile_Class(t *testing.T) {
    tmpFile := createTempFile(t, "zcl_test.clas.abap", `
CLASS zcl_test DEFINITION
  PUBLIC
  FINAL
  CREATE PUBLIC.
ENDCLASS.

CLASS zcl_test IMPLEMENTATION.
ENDCLASS.
`)
    defer os.Remove(tmpFile)

    info, err := ParseABAPFile(tmpFile)
    assert.NoError(t, err)
    assert.Equal(t, "ZCL_TEST", info.ObjectName)
    assert.Equal(t, ObjectTypeClass, info.ObjectType)
    assert.True(t, info.HasDefinition)
    assert.True(t, info.HasImplementation)
}
```

### Integration Tests

```go
// pkg/adt/workflows_integration_test.go
func TestIntegration_CreateFromFile(t *testing.T) {
    if testing.Short() {
        t.Skip()
    }

    client := setupTestClient(t)

    // Create temp file with class source
    tmpFile := createTempFile(t, "zcl_test_file.clas.abap", classSource)
    defer os.Remove(tmpFile)

    // Deploy
    result, err := client.CreateFromFile(context.Background(), tmpFile, "$TMP", "")
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Equal(t, "ZCL_TEST_FILE", result.ObjectName)

    // Cleanup
    defer client.DeleteObject(...)
}
```

---

## Future Enhancements

### 1. Batch Deployment

```go
// DeployFromDirectory deploys all ABAP files in a directory
func (c *Client) DeployFromDirectory(ctx context.Context, dirPath, packageName, transport string) ([]DeployResult, error)
```

### 2. Incremental Updates

- Track file checksums
- Only deploy changed files
- Skip unchanged objects

### 3. Dependency Resolution

- Parse USE/INHERIT statements
- Deploy in dependency order
- Parallel deployment where possible

### 4. Pre-commit Hooks

```bash
# .git/hooks/pre-commit
#!/bin/bash
vsp deploy --check-syntax abap/*.abap
```

---

## Conclusion

**Recommendation:** ✅ **IMPLEMENT**

This feature:
- Solves a real pain point (token limits, manual orchestration)
- Follows existing patterns (workflows)
- Relatively small implementation (3 days)
- High value for users generating ABAP code

**Next Steps:**
1. Implement parser
2. Implement workflows
3. Add MCP tools
4. Test with ZCL_ML_IRIS
