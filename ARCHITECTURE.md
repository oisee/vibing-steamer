# mcp-abap-adt-go Architecture

This document describes the architecture of the Go-native MCP server for SAP ADT.

## Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           MCP Client (Claude)                          │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                              JSON-RPC/stdio
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        internal/mcp/server.go                          │
│                         (36 Tool Handlers)                              │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────────────┐   │
│  │ Read Ops   │ │ CRUD Ops   │ │ Dev Tools  │ │ Code Intelligence  │   │
│  │ (14 tools) │ │ (8 tools)  │ │ (3 tools)  │ │ (7 tools)          │   │
│  └────────────┘ └────────────┘ └────────────┘ └────────────────────┘   │
│                          ┌────────────────┐                             │
│                          │ Workflow Tools │                             │
│                          │ (4 tools)      │                             │
│                          └────────────────┘                             │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                            pkg/adt.Client
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            pkg/adt/                                     │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ client.go - Main Client Facade                                    │  │
│  │   • NewClient(opts...)                                           │  │
│  │   • Read operations: SearchObject, GetProgram, GetClass, etc.    │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────────────────┐  │
│  │ crud.go        │ │ devtools.go    │ │ codeintel.go               │  │
│  │ • LockObject   │ │ • SyntaxCheck  │ │ • FindDefinition           │  │
│  │ • UnlockObject │ │ • Activate     │ │ • FindReferences           │  │
│  │ • UpdateSource │ │ • RunUnitTests │ │ • CodeCompletion           │  │
│  │ • CreateObject │ └────────────────┘ │ • PrettyPrint              │  │
│  │ • DeleteObject │                    │ • GetPrettyPrinterSettings │  │
│  │ • GetClassIncl │ ┌────────────────┐ │ • SetPrettyPrinterSettings │  │
│  │ • CreateTestI. │ │ workflows.go   │ │ • GetTypeHierarchy         │  │
│  │ • UpdateClassI.│ │ • WriteProgram │ └────────────────────────────┘  │
│  └────────────────┘ │ • WriteClass   │                                  │
│                     │ • CreateAndAct.│  ┌────────────────┐             │
│                     │ • CreateClass..│  │ config.go      │             │
│                     └────────────────┘  │ • FromEnv()    │             │
│                                         │ • Options      │             │
│  ┌──────────────────────────────────┐  └────────────────┘             │
│  │ http.go - HTTP Transport         │                                  │
│  │   • CSRF token management        │  ┌────────────────┐             │
│  │   • Session cookies              │  │ xml.go         │             │
│  │   • Stateful sessions            │  │ • ADT XML types│             │
│  │   • Basic authentication         │  │ • Parsing      │             │
│  └──────────────────────────────────┘  └────────────────┘             │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                              HTTPS/HTTP
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        SAP ABAP Development Tools                       │
│                            (ADT REST API)                               │
└─────────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
mcp-abap-adt-go/
├── cmd/mcp-abap-adt-go/
│   └── main.go                  # Entry point, config loading, server start
│
├── internal/mcp/
│   ├── server.go                # MCP server implementation (36 tool handlers)
│   └── server_test.go           # Server unit tests
│
├── pkg/adt/
│   ├── client.go                # ADT client facade + read operations
│   ├── client_test.go           # Client unit tests with mocks
│   ├── config.go                # Configuration (env vars, options)
│   ├── config_test.go           # Config unit tests
│   ├── http.go                  # HTTP transport (CSRF, sessions)
│   ├── http_test.go             # Transport unit tests
│   ├── crud.go                  # CRUD operations (lock, create, update, delete)
│   ├── devtools.go              # Dev tools (syntax check, activate, unit tests)
│   ├── codeintel.go             # Code intelligence (find def, refs, completion)
│   ├── workflows.go             # High-level workflow operations
│   ├── xml.go                   # XML types and parsing
│   ├── xml_test.go              # XML parsing tests
│   └── integration_test.go      # Integration tests (requires SAP system)
│
├── reports/                     # Project documentation
│   ├── mcp-abap-adt-go-status.md    # Implementation status
│   └── *.md                     # Discovery and analysis documents
│
├── testdata/                    # Test fixtures
├── abap-adt-api-lib/           # Reference TypeScript library
└── build/                       # Build artifacts
```

## Component Details

### cmd/mcp-abap-adt-go/main.go

Entry point for the MCP server:
- Loads configuration from environment variables
- Creates ADT client with HTTP transport
- Starts MCP server on stdio

### internal/mcp/server.go

MCP protocol implementation:
- Registers 36 tools with the MCP SDK
- Maps tool calls to ADT client methods
- Handles JSON-RPC communication

### pkg/adt/ - ADT Client Library

#### client.go
Main client facade providing read operations:
- `SearchObject` - Quick search for ABAP objects
- `GetProgram`, `GetClass`, `GetInterface`, `GetInclude`
- `GetFunction`, `GetFunctionGroup`
- `GetTable`, `GetTableContents`, `GetStructure`
- `GetPackage`, `GetTransaction`, `GetTypeInfo`
- `RunQuery` - Freestyle SQL queries

#### http.go
HTTP transport layer:
- Automatic CSRF token fetching and refresh
- Session cookie management
- Stateful session support (required for CRUD)
- Basic authentication
- TLS configuration

#### crud.go
Object modification operations:
- `LockObject` / `UnlockObject` - Edit locks
- `CreateObject` - Create programs, classes, interfaces, includes, function groups, function modules
- `UpdateSource` - Write source code
- `DeleteObject` - Remove objects
- `GetClassInclude` / `CreateTestInclude` / `UpdateClassInclude` - Class include operations

#### devtools.go
Development tools:
- `SyntaxCheck` - Check ABAP source for errors
- `Activate` - Activate objects
- `RunUnitTests` - Execute ABAP Unit tests

#### codeintel.go
Code intelligence features:
- `FindDefinition` - Navigate to symbol definition
- `FindReferences` - Find all usages
- `CodeCompletion` / `CodeCompletionFull` - Code suggestions
- `PrettyPrint` - Format source code
- `GetPrettyPrinterSettings` / `SetPrettyPrinterSettings` - Formatter config
- `GetTypeHierarchy` - Type hierarchy (supertypes/subtypes)

#### workflows.go
High-level operations combining multiple steps:
- `WriteProgram` - Lock → Check → Update → Unlock → Activate
- `WriteClass` - Lock → Check → Update → Unlock → Activate
- `CreateAndActivateProgram` - Create → Update → Activate
- `CreateClassWithTests` - Create class → Create tests → Update → Activate → Run tests

#### config.go
Configuration management:
- `FromEnv()` - Load config from environment
- Functional options pattern for customization

#### xml.go
ADT XML types and parsing utilities for request/response handling.

## Data Flow

### Read Operation Example (GetProgram)

```
1. MCP Client calls "GetProgram" tool with name parameter
2. server.go handler receives call
3. Handler calls client.GetProgram(ctx, name)
4. client.go fetches from /sap/bc/adt/programs/programs/{name}/source/main
5. http.go handles CSRF token, authentication, session
6. Response parsed and returned as tool result
```

### Write Operation Example (WriteProgram)

```
1. MCP Client calls "WriteProgram" with name and source
2. server.go handler receives call
3. Handler calls client.WriteProgram(ctx, name, source)
4. workflows.go orchestrates:
   a. LockObject(name) - Acquire lock
   b. SyntaxCheck(source) - Validate code
   c. UpdateSource(name, source) - Write code
   d. UnlockObject(name) - Release lock
   e. Activate(name) - Activate object
5. Result returned with success/error status
```

## Testing

### Unit Tests
- `*_test.go` files test with mock HTTP client
- No SAP system required
- Run with: `go test ./...`

### Integration Tests
- `integration_test.go` tests against real SAP system
- Build tag: `integration`
- Run with: `go test -tags=integration ./pkg/adt/`
- Requires environment variables: `SAP_URL`, `SAP_USER`, `SAP_PASSWORD`, `SAP_CLIENT`

## Design Decisions

1. **Single Binary**: No runtime dependencies for easy distribution
2. **Functional Options**: Flexible client configuration
3. **Stateful HTTP**: Required for CRUD operations with locks
4. **Workflow Tools**: Reduce round-trips for common operations
5. **Separation of Concerns**: Clean split between MCP, client, and transport layers
