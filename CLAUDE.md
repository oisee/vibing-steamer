# CLAUDE.md - AI Assistant Guidelines

This file provides context for AI assistants (Claude, etc.) working on this project.

## Project Overview

**mcp-abap-adt-go** is a Go-native MCP (Model Context Protocol) server for SAP ABAP Development Tools (ADT). It provides a single-binary distribution of 36 ADT tools for use with Claude and other MCP-compatible LLMs.

## Quick Reference

### Build & Test

```bash
# Build
go build -o mcp-abap-adt-go ./cmd/mcp-abap-adt-go

# Run unit tests
go test ./...

# Run integration tests (requires SAP system)
SAP_URL=http://host:port SAP_USER=user SAP_PASSWORD=pass SAP_CLIENT=001 \
  go test -tags=integration -v ./pkg/adt/
```

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `SAP_URL` | SAP system URL (e.g., `http://host:50000`) |
| `SAP_USER` | SAP username |
| `SAP_PASSWORD` | SAP password |
| `SAP_CLIENT` | SAP client number (default: 001) |
| `SAP_LANGUAGE` | SAP language (default: EN) |
| `SAP_INSECURE` | Skip TLS verification (default: false) |

## Codebase Structure

```
cmd/mcp-abap-adt-go/main.go  # Entry point
internal/mcp/server.go        # MCP server (36 tool handlers)
pkg/adt/
├── client.go                 # ADT client + read operations
├── crud.go                   # CRUD operations (lock, create, update, delete)
├── devtools.go               # Dev tools (syntax check, activate, unit tests)
├── codeintel.go              # Code intelligence (find def, refs, completion)
├── workflows.go              # High-level workflow operations
├── http.go                   # HTTP transport (CSRF, sessions)
├── config.go                 # Configuration
└── xml.go                    # XML types
```

## Key Files for Common Tasks

| Task | Files |
|------|-------|
| Add new MCP tool | `internal/mcp/server.go` |
| Add ADT read operation | `pkg/adt/client.go` |
| Add CRUD operation | `pkg/adt/crud.go` |
| Add development tool | `pkg/adt/devtools.go` |
| Add code intelligence | `pkg/adt/codeintel.go` |
| Add workflow | `pkg/adt/workflows.go` |
| Add XML types | `pkg/adt/xml.go` |
| Add integration test | `pkg/adt/integration_test.go` |

## Adding a New Tool

1. **Add ADT client method** in appropriate file (`client.go`, `crud.go`, etc.)
2. **Add tool handler** in `internal/mcp/server.go`:
   - Register tool in `registerTools()`
   - Add handler case in `handleToolCall()`
3. **Add integration test** in `pkg/adt/integration_test.go`
4. **Update documentation**:
   - `README.md` tool tables
   - `reports/mcp-abap-adt-go-status.md`

## Code Patterns

### ADT Client Methods

```go
// Read operation pattern
func (c *Client) GetSomething(ctx context.Context, name string) (*Result, error) {
    url := fmt.Sprintf("/sap/bc/adt/path/%s", name)
    resp, err := c.http.Get(ctx, url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    // Parse response
}

// Write operation pattern (requires stateful session)
func (c *Client) UpdateSomething(ctx context.Context, name, content string) error {
    url := fmt.Sprintf("/sap/bc/adt/path/%s", name)
    return c.http.Put(ctx, url, "text/plain", strings.NewReader(content))
}
```

### Tool Handler Pattern

```go
case "NewTool":
    name, _ := getString(args, "name")
    result, err := s.client.NewMethod(ctx, name)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    return mcp.NewToolResultText(formatResult(result)), nil
```

## Testing

- Unit tests mock the HTTP client (see `client_test.go`)
- Integration tests use build tag `integration`
- Integration tests create objects in `$TMP` package and clean up after

## ADT API Reference

The SAP ADT REST API documentation can be found at:
- `/sap/bc/adt/discovery` - API discovery document
- See `reports/adt-abap-internals-documentation.md` for detailed endpoint analysis

## Common Issues

1. **CSRF token errors**: The HTTP transport auto-refreshes tokens; check `http.go`
2. **Lock conflicts**: Objects must be unlocked before other operations
3. **Activation failures**: Check syntax errors first with `SyntaxCheck`
4. **Session issues**: CRUD operations require stateful sessions

## Project Status

See `reports/mcp-abap-adt-go-status.md` for current implementation status.

**Current Phase**: 4 (Code Intelligence)
**Total Tools**: 36
