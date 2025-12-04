# Test Intelligence System for vsp

## Overview

A comprehensive test intelligence system that enables AI-assisted intelligent test generation, scenario selection, and deep code analysis for SAP ABAP development.

## Three Pillars

### 1. BDD/TDD Test Design (Scenario-Driven)
- Extract behavior specifications from class/interface definitions
- Generate Given-When-Then style test scenarios
- Create test stubs with meaningful names based on method signatures
- Analyze parameters for edge cases (null, empty, boundary values)

### 2. Dependency Graph & Facts Database (DDD-Driven)
- Build dependency graph: class→dependencies, method→method calls
- Test impact analysis: which tests to run for a given change
- Domain insights: aggregate roots, integration points, circular deps
- Facts database: queryable knowledge ("ClassA calls MethodB")

### 3. Deep Probe & Reverse Engineering (Data-Driven)
- Standard code surface scan: analyze FM/class contracts
- Data model surface: table relationships, domain values, data flow
- X-ray capabilities: method-level analysis, call graph tracing
- Test data intelligence: realistic data based on domain knowledge

---

## Recommended Implementation: Phased Approach

### Phase 1: Foundation - Class Analysis & Scenario Generation
**Goal:** Enable AI to understand ABAP class structure and generate test scenarios

**New File:** `pkg/adt/testintel.go`

**New Tools:**
| Tool | Description |
|------|-------------|
| `AnalyzeClass` | Extract methods, interfaces, inheritance, testable behaviors |
| `DesignTests` | Generate BDD scenarios from class analysis |
| `GenerateTestSkeleton` | Create ABAP Unit test code from scenarios |

**Key Data Structures:**
```go
type ClassAnalysis struct {
    ClassName      string
    Interfaces     []InterfaceInfo
    SuperClass     *TypeReference
    PublicMethods  []MethodSignature
    TestableCount  int
}

type TestScenario struct {
    ID          string
    Name        string
    MethodName  string
    Category    string  // happy_path, edge_case, exception, boundary
    Given       []string
    When        string
    Then        []string
    Priority    string  // critical, high, medium, low
}
```

**Scenario Generation Rules:**
| Parameter Type | Generated Scenarios |
|----------------|---------------------|
| `TYPE REF TO` | NULL reference, valid reference |
| `TYPE TABLE OF` | Empty, single row, multiple rows |
| `TYPE I/P/N` | Zero, positive, negative, boundary |
| `TYPE STRING` | Empty, whitespace, typical, max length |
| `OPTIONAL` | With value, without value |

---

### Phase 2: Dependency Analysis & Test Impact
**Goal:** Build knowledge graph for intelligent test selection

**New File:** `pkg/adt/testintel/graph.go`, `builder.go`, `facts.go`

**New Tools:**
| Tool | Description |
|------|-------------|
| `BuildDependencyGraph` | Crawl package and build dependency graph |
| `QueryDependencies` | Query upstream/downstream dependencies |
| `GetTestImpact` | Which tests to run for a change |
| `SuggestMocks` | What to mock for testing a class |

**Key Data Structures:**
```go
type DependencyGraph struct {
    Nodes  map[string]*Node  // ABAP objects
    Edges  []*Edge           // Dependencies (CALLS, INHERITS, USES)
}

type TestImpactResult struct {
    ChangedObject    string
    DirectlyAffected []AffectedObject
    SuggestedTests   []TestSuggestion
}
```

**Graph Building:** Uses existing `FindReferences`, `GetTypeHierarchy`, `GetPackage`

---

### Phase 3: Deep Probe & Data Intelligence
**Goal:** Reverse-engineer standard code and generate realistic test data

**New Files:** `pkg/adt/analysis.go`, `pkg/adt/datamodel.go`

**New Tools:**
| Tool | Description |
|------|-------------|
| `AnalyzeFunctionModule` | Deep FM analysis: params, DB ops, side effects |
| `AnalyzeTable` | Table structure, relationships, domain values |
| `TraceDataFlow` | Find all code reading/writing a table |
| `SuggestTestData` | Generate realistic test data from domains |
| `AnalyzeTestScenario` | Determine test data requirements |

**Key Queries (via existing RunQuery):**
```sql
-- Domain fixed values
SELECT DOMVALUE_L, DDTEXT FROM DD07L/DD07T WHERE DOMNAME = ?

-- Foreign key relationships
SELECT TABNAME, FIELDNAME, CHECKTABLE FROM DD08L WHERE TABNAME = ?

-- Sample check table values (for test data)
SELECT DISTINCT {field} FROM {check_table} UP TO 10 ROWS
```

---

### Phase 4: Quality Integration
**Goal:** Integrate ATC code quality checks

**New Tools:**
| Tool | Description |
|------|-------------|
| `RunATCCheck` | Run ATC code quality checks |
| `GetCoverageMarkers` | Get test coverage data |

**ADT API:** `POST /sap/bc/adt/atc/runs`

---

## MCP Tool Summary (All Phases)

| Phase | Tool | Priority |
|-------|------|----------|
| 1 | AnalyzeClass | High |
| 1 | DesignTests | High |
| 1 | GenerateTestSkeleton | High |
| 2 | BuildDependencyGraph | Medium |
| 2 | QueryDependencies | Medium |
| 2 | GetTestImpact | High |
| 2 | SuggestMocks | Medium |
| 3 | AnalyzeFunctionModule | Medium |
| 3 | AnalyzeTable | Medium |
| 3 | TraceDataFlow | Low |
| 3 | SuggestTestData | High |
| 3 | AnalyzeTestScenario | High |
| 4 | RunATCCheck | Medium |
| 4 | GetCoverageMarkers | Low |

---

## Critical Files to Modify

| File | Change |
|------|--------|
| `pkg/adt/testintel.go` | NEW: Test intelligence types and methods |
| `pkg/adt/testintel/graph.go` | NEW: Dependency graph structures |
| `pkg/adt/testintel/builder.go` | NEW: Graph builder using ADT tools |
| `pkg/adt/testintel/facts.go` | NEW: Facts database |
| `pkg/adt/analysis.go` | NEW: Deep code analysis |
| `pkg/adt/datamodel.go` | NEW: Data model intelligence |
| `internal/mcp/server.go` | MODIFY: Register 10-14 new tools |
| `pkg/adt/devtools.go` | MODIFY: Add ATC, coverage support |

---

## Example AI Workflow

```
User: "Create unit tests for ZCL_ORDER_SERVICE"

Claude:
1. AnalyzeClass("ZCL_ORDER_SERVICE")
   → Methods: create_order, validate_order, calculate_total
   → Interfaces: ZIF_ORDER_SERVICE
   → Dependencies: ZCL_DATABASE, ZCL_VALIDATOR

2. DesignTests("ZCL_ORDER_SERVICE")
   → Scenarios:
     - create_order_happy_path (valid order created)
     - create_order_null_customer (exception expected)
     - create_order_empty_items (edge case)
     - validate_order_invalid_date (boundary)

3. SuggestMocks("ZCL_ORDER_SERVICE")
   → Mock: ZIF_DATABASE_ACCESSOR (DB isolation)
   → Mock: ZCL_RFC_WRAPPER (external calls)

4. GenerateTestSkeleton("ZCL_ORDER_SERVICE", scenarios)
   → ABAP Unit test class with stubs

5. Claude fills in test implementations

6. CreateClassWithTests or UpdateClassInclude to deploy

7. RunUnitTests to verify
```

---

## Storage Strategy

- **In-Memory (Default):** Graph lives in MCP server process
- **JSON Persistence (Optional):** Cache between sessions
- **Future:** SQLite for large codebases

---

## Open Questions

1. **Phase Priority:** Which phase to start with?
2. **Scope:** Custom code only, or include SAP standard analysis?
3. **Storage:** In-memory only, or persist graph to disk?
4. **ATC Integration:** Priority for code quality checks?
