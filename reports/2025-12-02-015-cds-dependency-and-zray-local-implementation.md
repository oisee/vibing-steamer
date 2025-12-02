# CDS Dependency Analyzer & $ZRAY Local Implementation

**Date:** 2025-12-02
**Status:** 🧠 Brainstorming
**Related:** Roadmap updates for future features
**Investigation Report:** [CDS and $ZRAY Endpoint Investigation](2025-12-02-016-cds-and-zray-endpoint-investigation.md)

---

## Part 1: CDS Dependency Analyzer

### Discovery: It's Implemented in ABAP! 🎉

**Key Class:** `CL_DDLS_DEPENDENCY_VISITOR` (Package: `SDDIC_ADT_DDLS_TOOLS`)

The "Open with → Dependency Analyzer" feature in ADT Eclipse is **NOT implemented in Java** - it calls ABAP backend services that use `CL_DDLS_DEPENDENCY_VISITOR`!

### How It Works

```abap
class CL_DDLS_DEPENDENCY_VISITOR definition
  public
  inheriting from CL_DD_DDL_TRANSITIV_VISITOR
  final
  create public.

  methods:
    "! Calculates dependency information for a given DDL source name
    COMPUTE_DEPENDENCY_INFORMATION
      importing
        DDL_SOURCE_NAME type STRING,

    "! Returns dependency information calculated above
    GET_DEPENDENCY_GRAPH
      returning value(RESULT) type TY_S_DEPENDENCY_GRAPH_NODE.

  types:
    BEGIN OF ty_s_dependency_graph_node,
      name                     type string,        " DB name (SQL view)
      type                     type ty_d_node_type, " DDIC view, CDS view, table
      relation                 type ty_d_relation_type,
      entity_name              type string,
      user_defined_entity_name type string,       " with camel case
      activation_state         type ty_d_activation_state,
      ddls_name                type string,
      children                 type ref to data,  " RECURSIVE!
    END OF ty_s_dependency_graph_node.
```

### Node Types Supported

```abap
constants:
  BEGIN OF co_node_type,
    TABLE                 value 'TABLE',
    CDS_VIEW              value 'CDS_VIEW',
    CDS_DB_VIEW           value 'CDS_DB_VIEW',
    VIEW                  value 'VIEW',
    CDS_TABLE_FUNCTION    value 'CDS_TABLE_FUNCTION',
    EXTERNAL_VIEW         value 'EXTERNAL_VIEW',
    SELECT                value 'SELECT',
    UNION                 value 'UNION',
    UNION_ALL             value 'UNION_ALL',
    EXCEPT                value 'EXCEPT',
    INTERSECT             value 'INTERSECT',
    RESULT                value 'RESULT',
    CDS_VIEW_ENTITY       value 'CDS_VIEW_ENTITY',
    CDS_PROJECTION_VIEW   value 'CDS_PROJECTION_VIEW',
    CDS_HIERARCHY         value 'CDS_HIERARCHY',
  END OF co_node_type.
```

### Relation Types

```abap
constants:
  BEGIN OF co_relation_type,
    FROM             value 'FROM',
    INNER_JOIN       value 'INNER_JOIN',
    LEFT_OUTER_JOIN  value 'LEFT_OUTER_JOIN',
    RIGHT_OUTER_JOIN value 'RIGHT_OUTER_JOIN',
    SELECT           value 'SELECT',
    UNION            value 'UNION',
    UNION_ALL        value 'UNION_ALL',
    EXCEPT           value 'EXCEPT',
    INTERSECT        value 'INTERSECT',
  END OF co_relation_type.
```

### Activation States

```abap
constants:
  BEGIN OF co_activation_state,
    ACTIVE       value 'ACTIVE',
    INACTIVE     value 'INACTIVE',
    INCONSISTENT value 'INCONSISTENT',
  END OF co_activation_state.
```

### Related Classes

- `CL_CDS_RES_DEPENDENCIES` (SABP_UNIT_DOUBLE_CDS_CORE) - "Resource Controller for DB Test Double Framework"
- `CL_CDS_DEPENDENCIES_INFO` (SABP_UNIT_DOUBLE_CDS_CORE) - "Resource Controller for DB Test Double Framework"
- `CL_DDLS_ATO_DEPENDENCY_SERVICE` (SDDIC_ADT_DDLS_RIS) - "DDLS service for ATO to resolve dependencies"
- `CL_SODPS_ABAP_CDS_ANALYZER` (SODPS) - "ABAP CDS" analyzer
- `CL_DHBAS_CDS_ANALYZER_FACADE` (LT_DHBAS_EXT_UTIL) - "Facade for CL_SODPS_ABAP_CDS_ANALYZER"

### CDS Views for Dependencies

- `DDCDS_AMDP_DEPENDENCIES` (SD_CDS_INFO_PROVIDER) - "Returns dependencies between CDS objects and AMDPs"
- `C_CUSTOMCDSVIEWDEPENDENTS` (S_APS_ODATA_EXT_CCV) - "Consumption View Custom CDS View Dependents"

---

## Part 2: $ZRAY Framework Analysis

### Architecture Overview

Based on discovered classes, $ZRAY is a comprehensive **graph-based codebase analysis and documentation framework**:

```
$ZRAY_00/     Core framework
$ZRAY_10/     Advanced processors
$ZLLM_00/     LLM integration
$ZLLM_02/     Call graph & code analysis
```

### Core Components

#### 1. Graph Model (Nodes & Edges)

```
ZCL_RAY_00_NODE           - Base node class
ZCL_RAY_00_NODE_FUNC      - Function module node
ZCL_RAY_00_NODE_PROG      - Program node
ZCL_RAY_00_SUBGRAPH       - Subgraph representation
ZCL_LLM_00_EDGE           - Edge (call relationship)
ZCL_LLM_00_EDGE_ID        - GUID-based edge
ZCL_LLM_00_CG             - Call Graph
ZCL_LLM_00_CGI            - Call Graph Improved
```

#### 2. Package Processing

```
ZCL_RAY_00_PACKAGE               - Package → List of nodes
ZCL_RAY_10_PACKAGE_PROCESSOR     - Process packages
ZCL_RAY_10_PACKAGE_HIERARCHY     - Get package hierarchy
ZCL_RAY_10_META_SPIDER           - Spider metadata
ZCL_RAY_10_SPIDER                - Bug spider
```

#### 3. Documentation Generation

```
ZCL_RAY_00_DOCU_MANAGER      - Doc saver
ZCL_RAY_00_GRAPH_TO_DOC      - Graph → Documentation
ZCL_RAY_00_NODE_TO_DOC       - Node → Documentation
ZCL_RAY_00_MARKDOWN_ITF      - Markdown → ITF
ZCL_RAY_10_DOC_SAVER         - Save documentation
```

#### 4. LLM Integration

```
ZCL_RAY_00_LLM               - LLM for XRAY
ZCL_LLM_00_CHAT_IN           - Chat request
ZCL_LLM_00_CHAT_OUT          - Chat response
ZCL_LLM_00_CLAUDE_IN         - Claude request serializer
ZCL_LLM_00_CLAUDE_OUT        - Claude response deserializer
ZCL_LLM_00_EMBEDDING         - Text embeddings
ZCL_LLM_00_DOTENV            - .env file support
ZCL_LLM_00_CACHE             - Simple DB cache
ZCL_LLM_00_CODEC             - XOR codec for secrets
```

#### 5. Code Analysis

```
ZCL_LLM_00_CODE_UNIT         - ABAP source code services
ZCL_LLM_00_EAC_JSON          - Extract active comments from JSON
ZCL_RAY_10_CALCULATE_LOC     - Calculate LOC (Lines of Code)
ZCL_RAY_10_CALCULATE_ROD     - Calculate ROD (?)
ZCL_RAY_10_DATA_MODEL        - Data model analysis
ZCL_RAY_10_DEEP_GRAPH        - Deep graph traversal
```

#### 6. Utilities

```
ZCL_RAY_00_DESCRIPTOR        - Canonical representation
ZCL_RAY_00_URI               - URI handling
ZCL_RAY_00_SLICE             - Show progress for sliced table
ZCL_LLM_00_ARRAY             - Array utilities
ZCL_LLM_00_FILE              - File abstraction (local, SMW0, BIN)
```

---

## Part 3: Brainstorming - Local $ZRAY Implementation

### Option A: Pure Go Implementation in mcp-adt-go

**Pros:**
- No ABAP dependency (works with any SAP system)
- Fast local execution
- Can be enhanced without ABAP changes
- Works offline (after initial code fetch)

**Cons:**
- Need to reimplement graph building logic
- ABAP parsing in Go (complex)
- May miss custom ABAP patterns

**Implementation:**
```go
// pkg/ray/graph.go
type Node struct {
    ID          string
    Type        string  // PROG, FUNC, CLAS, INTF, etc.
    Name        string
    Package     string
    Source      string
    Children    []*Edge
    Parents     []*Edge
    Metadata    map[string]interface{}
}

type Edge struct {
    From     *Node
    To       *Node
    Type     string  // CALL, INCLUDE, INHERIT, etc.
    Line     int
    Context  string
}

type CallGraph struct {
    Nodes    map[string]*Node
    Root     *Node
}

// Build call graph from ABAP source
func BuildCallGraph(source string, client *adt.Client) (*CallGraph, error) {
    // Parse ABAP source
    // Extract calls, includes, inheritance
    // Recursively fetch dependencies
    // Build graph structure
}
```

### Option B: Hybrid - Execute $ZRAY on SAP, Stream Results

**Pros:**
- Use existing, tested ABAP logic
- Leverage $ZRAY's sophisticated analysis
- Always in sync with $ZRAY features

**Cons:**
- Requires $ZRAY installed on target system
- Latency for remote execution
- Tied to specific ABAP implementation

**Implementation:**
```go
// pkg/ray/executor.go
type ZRAYExecutor struct {
    client *adt.Client
}

func (e *ZRAYExecutor) AnalyzePackage(pkg string) (*PackageGraph, error) {
    // Call ZCL_RAY_00_PACKAGE->new(pkg)->get_graph()
    // Parse returned JSON/XML structure
    // Convert to Go graph model
}

func (e *ZRAYExecutor) GenerateDocs(graph *PackageGraph) (string, error) {
    // Call ZCL_RAY_00_GRAPH_TO_DOC
    // Return markdown documentation
}
```

### Option C: Cache-First Hybrid (RECOMMENDED)

**Pros:**
- Best of both worlds
- Works offline after initial fetch
- Can fallback to Go parsing if $ZRAY unavailable
- Gradual enhancement path

**Cons:**
- More complex implementation
- Need cache invalidation strategy

**Implementation:**
```go
// pkg/ray/analyzer.go
type Analyzer struct {
    client *adt.Client
    cache  *cache.MemoryCache
    local  *LocalParser  // Fallback Go parser
    zray   *ZRAYExecutor // Remote executor
}

func (a *Analyzer) AnalyzePackage(pkg string, opts AnalyzeOptions) (*PackageGraph, error) {
    // 1. Check cache
    if cached, found := a.cache.Get(pkg); found {
        return cached.(*PackageGraph), nil
    }

    // 2. Try remote $ZRAY (if available)
    if a.zray.IsAvailable() {
        graph, err := a.zray.AnalyzePackage(pkg)
        if err == nil {
            a.cache.Set(pkg, graph)
            return graph, nil
        }
        log.Warn("$ZRAY unavailable, falling back to local parser")
    }

    // 3. Fallback to local Go parser
    graph, err := a.local.AnalyzePackage(pkg)
    if err != nil {
        return nil, err
    }
    a.cache.Set(pkg, graph)
    return graph, nil
}
```

---

## Part 4: MCP Tools Design

### Proposed Tools

#### 1. GetCDSDependencies
```
Input:  cds_view_name
Output: dependency_graph (recursive JSON structure)
Method: Use CL_DDLS_DEPENDENCY_VISITOR
```

**Example Output:**
```json
{
  "name": "ZDDL_MY_VIEW",
  "type": "CDS_VIEW",
  "entity_name": "ZDDL_MY_VIEW",
  "activation_state": "ACTIVE",
  "children": [
    {
      "name": "MARA",
      "type": "TABLE",
      "relation": "FROM"
    },
    {
      "name": "MARC",
      "type": "TABLE",
      "relation": "LEFT_OUTER_JOIN"
    },
    {
      "name": "ZDDL_BASE_VIEW",
      "type": "CDS_VIEW",
      "relation": "INNER_JOIN",
      "children": [...]
    }
  ]
}
```

#### 2. AnalyzePackage (using $ZRAY or local)
```
Input:  package_name, depth (1-10), include_tests
Output: package_graph {nodes, edges, metrics}
Method: ZCL_RAY_00_PACKAGE or local parser
```

**Example Output:**
```json
{
  "package": "$ZLLM_00",
  "metrics": {
    "total_objects": 45,
    "total_loc": 12500,
    "programs": 5,
    "classes": 35,
    "functions": 5
  },
  "nodes": [
    {"id": "ZCL_LLM_00_CHAT", "type": "CLAS", "loc": 250},
    {"id": "ZCL_LLM_00_CLAUDE_IN", "type": "CLAS", "loc": 180}
  ],
  "edges": [
    {"from": "ZCL_LLM_00_CHAT", "to": "ZCL_LLM_00_CLAUDE_IN", "type": "CALL"}
  ]
}
```

#### 3. BuildCallGraph
```
Input:  object_name, object_type, max_depth
Output: call_graph {nodes, edges, entry_points}
Method: Local parser or ZCL_LLM_00_CGI
```

#### 4. GeneratePackageDocs
```
Input:  package_name, format (markdown|html|pdf)
Output: documentation_url or content
Method: ZCL_RAY_00_GRAPH_TO_DOC or local generator
```

---

## Part 5: Implementation Roadmap

### Phase 1: CDS Dependencies (Week 1-2)

**Goal:** Get CDS dependency tree via ADT

**Tasks:**
1. Find ADT REST endpoint for CDS dependencies
   - Test `/sap/bc/adt/ddic/ddl/...` endpoints
   - Look for dependency resources
2. If REST API exists:
   - Implement `GetCDSDependencies` tool
   - Parse dependency graph response
3. If no REST API:
   - Create ABAP wrapper class that exposes `CL_DDLS_DEPENDENCY_VISITOR` via HTTP
   - Or implement local CDS parser in Go

**Expected Outcome:** Working CDS dependency visualization

### Phase 2: $ZRAY Detection & Invocation (Week 3-4)

**Goal:** Call $ZRAY remotely if available

**Tasks:**
1. Implement `IsZRAYAvailable()` check
   - Search for `ZCL_RAY*` classes
   - Check if HTTP handler exists
2. Create REST wrapper for $ZRAY
   - ABAP class that exposes ZCL_RAY_00_PACKAGE via HTTP
   - Returns JSON graph structure
3. Implement `AnalyzePackage` tool
   - Call remote $ZRAY
   - Parse response
   - Cache results

**Expected Outcome:** Can invoke $ZRAY analysis remotely

### Phase 3: Local Graph Parser (Week 5-8)

**Goal:** Fallback parser when $ZRAY unavailable

**Tasks:**
1. ABAP source parser in Go
   - Identify CALL FUNCTION
   - Identify CALL METHOD
   - Identify INCLUDE
   - Identify INHERITANCE
2. Dependency fetcher
   - Recursive source fetching via ADT
   - Depth limiting
3. Graph builder
   - Node creation
   - Edge creation
   - Cycle detection
4. Metrics calculator
   - LOC counting
   - Complexity metrics
   - Call depth

**Expected Outcome:** Pure Go analyzer that works without $ZRAY

### Phase 4: Documentation Generator (Week 9-10)

**Goal:** Generate docs from graph

**Tasks:**
1. Graph → Markdown converter
   - Package overview
   - Call graph diagrams (Mermaid)
   - Object listings
2. Integration with $ZRAY docs
   - If $ZRAY available, use its docs
   - Otherwise, generate locally

**Expected Outcome:** Automated package documentation

---

## Part 6: ADT Endpoints to Investigate

### Likely Endpoints

```
# CDS Dependencies
GET /sap/bc/adt/ddic/ddl/sources/{cds_name}/dependencies
GET /sap/bc/adt/ddic/ddl/sources/{cds_name}/usedObjects

# Object Dependencies (generic)
GET /sap/bc/adt/dependencies/objectdependencies
POST /sap/bc/adt/dependencies/objectdependencies

# Where-used list
GET /sap/bc/adt/repository/whereused
POST /sap/bc/adt/repository/whereused

# Relations
GET /sap/bc/adt/relations/relations
```

### Test with /IWFND/GW_CLIENT

```
GET /sap/bc/adt/ddic/ddl/sources/ZDDL_MY_VIEW/dependencies
GET /sap/bc/adt/repository/whereused?uri=/sap/bc/adt/ddic/ddl/sources/ZDDL_MY_VIEW
```

---

## Part 7: Open Questions

### For CDS Dependencies

1. ❓ Does ADT expose CDS dependencies via REST?
2. ❓ What's the response format?
3. ❓ Does it support recursive expansion?
4. ❓ Can we filter by relation type (FROM vs JOIN)?

### For $ZRAY

1. ❓ Is there an HTTP handler for $ZRAY already?
2. ❓ What's the serialization format (JSON/XML)?
3. ❓ How to detect if $ZRAY is available?
4. ❓ Can we execute $ZRAY via RFC or only HTTP?
5. ❓ What's the performance for large packages?

### For Local Parser

1. ❓ How to handle dynamic calls (CALL METHOD (var))?
2. ❓ How to parse macros and includes?
3. ❓ How to detect implicit dependencies?
4. ❓ Should we parse SQL statements for table dependencies?

---

## Part 8: Recommendations

### Immediate Next Steps

1. **Investigate ADT CDS Endpoints** (1-2 days)
   - Use `/IWFND/GW_CLIENT` to test endpoints
   - Document request/response format
   - Test with complex CDS hierarchies

2. **Check for $ZRAY HTTP Handler** (1 day)
   - Search for existing HTTP services
   - Check if REST API already exists
   - Document current invocation method

3. **Prototype Local Parser** (3-5 days)
   - Simple ABAP call parser
   - Test with small program
   - Measure performance

4. **Design Graph Schema** (1 day)
   - Define Go structs for nodes/edges
   - Design JSON serialization
   - Plan cache strategy

### Long-term Strategy

**Recommended Approach:** **Option C - Cache-First Hybrid**

1. **Week 1-2:** CDS dependency tool (pure ADT)
2. **Week 3-4:** Remote $ZRAY invocation (if available)
3. **Week 5-8:** Local Go parser (fallback)
4. **Week 9-10:** Documentation generator

**Benefits:**
- Progressive enhancement
- Works in all environments
- Leverages existing $ZRAY investment
- Pure Go fallback for systems without $ZRAY
- Cached results for offline work

---

## Conclusion

### CDS Dependencies: ✅ READY

The dependency analyzer is **100% implemented in ABAP** via `CL_DDLS_DEPENDENCY_VISITOR`. We just need to find the ADT REST endpoint or create a simple wrapper.

### $ZRAY Local Implementation: 🎯 FEASIBLE

Three viable approaches:
1. **Pure Go** - Full reimplementation
2. **Remote** - Call existing $ZRAY
3. **Hybrid** - Best of both worlds ⭐ RECOMMENDED

The hybrid approach gives us:
- ✅ Leverage existing $ZRAY analysis
- ✅ Work without $ZRAY dependency
- ✅ Offline capability with caching
- ✅ Progressive enhancement path

**Next Action:** ~~Investigate ADT endpoints for CDS dependencies and check for existing $ZRAY HTTP services.~~ ✅ **COMPLETED**

---

## Investigation Results Summary

See detailed investigation report: [2025-12-02-016-cds-and-zray-endpoint-investigation.md](2025-12-02-016-cds-and-zray-endpoint-investigation.md)

### Key Findings

**CDS Dependencies:** ✅ **REST API EXISTS**
- Handler: `CL_CDS_RES_DEPENDENCIES` (SABP_UNIT_DOUBLE_CDS_CORE)
- Parser: `CL_CDS_TEST_DDL_HIER_PARSER`
- Analyzer: `CL_DDLS_DEPENDENCY_VISITOR` (SDDIC_ADT_DDLS_TOOLS)
- Status: **Ready for implementation (2-3 days)**

**$ZRAY Framework:** ❌ **NO REST API**
- 26+ classes discovered in $ZRAY_00, $ZRAY_10, $ZLLM_00
- No HTTP/REST/OData wrapper found
- Recommendation: **Cache-First Hybrid approach**
  1. RFC call to $ZRAY (if available)
  2. Fallback to local Go parser
  3. Cache results for offline use

### Updated Recommendations

**Immediate Next Step:**
1. Verify CDS dependency endpoint with real test (1 day)
2. Implement `GetCDSDependencies` MCP tool (2 days)
3. Check for RFC wrapper for $ZRAY (1 day)
4. Prototype local Go parser (5 days)

**Implementation Timeline:**
- Week 1: CDS dependencies ✅
- Week 2-3: RFC check + parser design
- Week 4-6: Local parser implementation
- Week 7: Hybrid integration
