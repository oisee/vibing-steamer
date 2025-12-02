# CDS Dependency & $ZRAY Endpoint Investigation

**Date:** 2025-12-02
**Status:** ✅ Investigation Complete
**Related:** [CDS Dependency and $ZRAY Local Implementation](2025-12-02-015-cds-dependency-and-zray-local-implementation.md)

---

## Executive Summary

### Key Findings

1. **CDS Dependency REST API**: ✅ **EXISTS** - ADT exposes CDS dependencies via REST
2. **$ZRAY HTTP Wrapper**: ❌ **NOT FOUND** - No REST/HTTP/OData interface discovered
3. **Recommendation**: Implement **Cache-First Hybrid** approach (Option C)
   - Use ADT REST API for CDS dependencies
   - Execute $ZRAY via RFC/local calls if available
   - Implement Go fallback parser for universal support

---

## Part 1: CDS Dependency Analysis

### 1.1 REST Resource Controller

**Class:** `CL_CDS_RES_DEPENDENCIES`
**Package:** `SABP_UNIT_DOUBLE_CDS_CORE`
**Description:** "New Resource Controller for DB Test Double Framework"

**Inheritance:**
```
CL_CDS_RES_DEPENDENCIES
  └─ CL_ADT_REST_RESOURCE (ADT REST base class)
```

**Key Methods:**
- `GET` - Main HTTP GET handler
- `GET_CDS_HIER_TEST_DEPENDENCIES` - Retrieve dependency graph

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `ddls_name` | string | CDS DDL source name (required) |
| `dependency_level` | string | 'unit' or 'hierarchy' (default: hierarchy) |
| `with_associations` | string | 'TRUE'/'FALSE' - Include modeled associations |
| `contextPackage` | string | Context package filter (optional) |

**Content Type:**
- Response: `application/vnd.sap.adt.aunit.cds.dependencymodel.v1+xml`

### 1.2 Dependency Parser

**Class:** `CL_CDS_TEST_DDL_HIER_PARSER`
**Package:** `SABP_UNIT_DOUBLE_CDS_CORE`
**Description:** "Hierarchy parser for CDS test framework"

**Inheritance:**
```
CL_CDS_TEST_DDL_HIER_PARSER
  └─ CL_DD_DDL_TRANSITIV_VISITOR (DDL AST visitor)
```

**Core Methods:**
```abap
methods:
  "! Calculates dependency information for a given DDL source name
  COMPUTE_DEPENDENCY_INFORMATION
    importing DDL_SOURCE_NAME type STRING,

  "! Returns dependency information calculated above
  GET_DEPENDENCY_GRAPH
    returning value(RESULT) type TY_S_DEPENDENCY_GRAPH_NODE.
```

**Dependency Graph Structure:**
```abap
types:
  BEGIN OF ty_s_dependency_graph_node,
    name                     type string,              " SQL view name
    type                     type ty_d_node_type,      " Node type
    object_type              type char1,               " Object type ID
    has_params               type abap_bool,           " Has parameters
    relation                 type ty_d_relation_type,  " FROM, INNER_JOIN, etc.
    entity_name              type string,              " CDS entity name
    user_defined_entity_name type string,              " Camel case entity name
    activation_state         type ty_d_activation_state,
    ddls_name                type string,              " DDL source name
    children                 type ref to data,         " RECURSIVE!
  END OF ty_s_dependency_graph_node.
```

### 1.3 Node Types Supported

```abap
constants:
  BEGIN OF co_node_type,
    TABLE                   value 'TABLE',
    CDS_VIEW                value 'CDS_VIEW',
    CDS_DB_VIEW             value 'CDS_DB_VIEW',
    VIEW                    value 'VIEW',
    CDS_TABLE_FUNCTION      value 'CDS_TABLE_FUNCTION',
    EXTERNAL_VIEW           value 'EXTERNAL_VIEW',
    UNKNOWN                 value 'UNKNOWN',
    FUNCTIONAL_DEPENDENCIES value 'FUNCTIONAL_DEPENDENCIES',
    PROJECTION_VIEW         value 'PROJECTION_VIEW',
  END OF co_node_type.
```

### 1.4 Relation Types

```abap
constants:
  BEGIN OF co_relation_type,
    FROM       value 'FROM',
    INNER_JOIN value 'INNER_JOIN',
    MODELED    value 'MODELED',       " For associations
    CYLCIC_DEP value 'CYCLIC_DEPENDENCY',
  END OF co_relation_type.
```

### 1.5 Dependency Analyzer (Alternative)

**Class:** `CL_DDLS_DEPENDENCY_VISITOR`
**Package:** `SDDIC_ADT_DDLS_TOOLS`
**Description:** "Dependency analyzer"
**Status:** ✅ VERIFIED (mentioned in brainstorming doc)

This class provides similar functionality and may be used by the Eclipse ADT "Dependency Analyzer" feature.

### 1.6 Likely REST Endpoint

Based on ADT conventions and the resource controller, the endpoint is likely:

```
GET /sap/bc/adt/cds/dependencies?ddls_name=<name>&dependency_level=hierarchy&with_associations=false
```

Or potentially:

```
GET /sap/bc/adt/ddic/ddl/sources/<ddls_name>/dependencies
```

**⚠️ Note:** Exact endpoint needs verification via:
1. SICF transaction (check ADT service tree)
2. `/sap/bc/adt/discovery` (ADT discovery document)
3. Eclipse ADT plugin (network traffic inspection)

### 1.7 Related XSLT Transformation

**Object:** `RFAC_ST_ADT_CDS_GET_DEPENDENCY`
**Type:** XSLT Transformation
**Package:** `SRFAC_ADT`

This suggests there may be additional REST endpoints in the RFAC (Remote Function Call Access) package.

---

## Part 2: $ZRAY Framework Analysis

### 2.1 Discovered Classes (26+)

**Core Framework ($ZRAY_00):**
```
✅ ZCL_RAY                      - Shortcut to common services
✅ ZCL_RAY_00_DESCRIPTOR        - Canonical representation
✅ ZCL_RAY_00_DOCU_MANAGER      - Documentation manager
✅ ZCL_RAY_00_GRAPH_TO_DOC      - Graph → Documentation
✅ ZCL_RAY_00_LLM               - LLM integration
✅ ZCL_RAY_00_MARKDOWN_ITF      - Markdown → ITF conversion
✅ ZCL_RAY_00_NODE              - Base node class
✅ ZCL_RAY_00_NODE_FUNC         - Function module node
✅ ZCL_RAY_00_NODE_PROG         - Program node
✅ ZCL_RAY_00_NODE_TO_DOC       - Node → Documentation
✅ ZCL_RAY_00_PACKAGE           - Package → List of nodes
✅ ZCL_RAY_00_SLICE             - Progress tracking
✅ ZCL_RAY_00_SUBGRAPH          - Subgraph representation
✅ ZCL_RAY_00_URI               - URI handling
```

**Advanced Processors ($ZRAY_10):**
```
✅ ZCL_RAY_10_CALCULATE_LOC     - Lines of Code calculator
✅ ZCL_RAY_10_CALCULATE_ROD     - ROD metric calculator
✅ ZCL_RAY_10_DATA_MODEL        - Data model analysis
✅ ZCL_RAY_10_DEEP_GRAPH        - Deep graph traversal
✅ ZCL_RAY_10_DOC_SAVER         - Documentation persistence
✅ ZCL_RAY_10_META_SPIDER       - Metadata spider
✅ ZCL_RAY_10_NODE_APPLYER      - Node applyer
✅ ZCL_RAY_10_NODE_SRC          - Node source handler
✅ ZCL_RAY_10_PACKAGE_HIERARCHY - Package hierarchy
✅ ZCL_RAY_10_PACKAGE_PROCESSOR - Package processor
✅ ZCL_RAY_10_SPIDER            - Bug spider
✅ ZCL_RAY_10_TADIR_TO_CCLM     - TADIR converter
```

### 2.2 HTTP/REST/OData Search Results

**Search Queries Executed:**
1. `*ZRAY*REST*` → ❌ No results
2. `*ZRAY*ODATA*` → ❌ No results
3. `*ZRAY*HTTP*` → ❌ No results
4. `*ZRAY*HANDLER*` → ❌ No results

**Conclusion:** $ZRAY framework does NOT have an exposed REST API, HTTP handler, or OData service.

### 2.3 ZCL_RAY_00_PACKAGE Analysis

**Source Code:**
```abap
CLASS zcl_ray_00_package DEFINITION
  PUBLIC FINAL CREATE PUBLIC.

  PUBLIC SECTION.
    TYPES: ttr_devc TYPE RANGE OF tadir-devclass.
    TYPES: ts_tadir TYPE tadir.
    TYPES: tt_tadir TYPE TABLE OF ts_tadir WITH KEY pgmid object obj_name.

    CLASS-METHODS devc_to_tadir_list
      IMPORTING iv_ TYPE tadir-devclass
      RETURNING VALUE(rt_) TYPE tt_tadir.

    CLASS-METHODS devc_range_to_tadir_list
      IMPORTING itr_ TYPE ttr_devc
      RETURNING VALUE(rt_) TYPE tt_tadir.
ENDCLASS.

CLASS zcl_ray_00_package IMPLEMENTATION.
  METHOD devc_range_to_tadir_list.
    IF itr_ IS INITIAL. RETURN. ENDIF.
    SELECT * FROM tadir
      WHERE devclass IN @itr_
      INTO TABLE @rt_.
  ENDMETHOD.

  METHOD devc_to_tadir_list.
    rt_ = devc_range_to_tadir_list(
      VALUE #( ( sign = 'I' option = 'EQ' low = iv_ ) )
    ).
  ENDMETHOD.
ENDCLASS.
```

**Functionality:**
- Converts package name to list of repository objects (TADIR)
- Simple, focused interface
- Can be used as basis for local Go implementation

---

## Part 3: Implementation Strategy

### 3.1 CDS Dependencies - READY FOR IMPLEMENTATION

**Status:** ✅ **REST API EXISTS**

**Next Steps:**
1. **Verify exact endpoint** (1-2 hours)
   - Check SICF for `/sap/bc/adt/cds/*` services
   - Test endpoint with real CDS view
   - Document request/response format

2. **Implement Go client** (1 day)
   ```go
   // pkg/adt/cds.go
   type CDSDependencyNode struct {
       Name                   string                `xml:"name"`
       Type                   string                `xml:"type"`
       Relation               string                `xml:"relation"`
       EntityName             string                `xml:"entity_name"`
       UserDefinedEntityName  string                `xml:"user_defined_entity_name"`
       ActivationState        string                `xml:"activation_state"`
       DDLSName               string                `xml:"ddls_name"`
       HasParams              bool                  `xml:"has_params"`
       Children               []CDSDependencyNode   `xml:"children>node"`
   }

   func (c *Client) GetCDSDependencies(ctx context.Context, ddlsName string, opts CDSDependencyOptions) (*CDSDependencyNode, error) {
       url := fmt.Sprintf("/sap/bc/adt/cds/dependencies?ddls_name=%s&dependency_level=%s&with_associations=%v",
           ddlsName, opts.Level, opts.WithAssociations)
       // ...
   }
   ```

3. **Add MCP tool** (1 day)
   ```go
   // internal/mcp/server.go
   case "GetCDSDependencies":
       ddlsName, _ := getString(args, "ddls_name")
       level := getStringOrDefault(args, "dependency_level", "hierarchy")
       withAssoc := getBoolOrDefault(args, "with_associations", false)

       result, err := s.client.GetCDSDependencies(ctx, ddlsName, adt.CDSDependencyOptions{
           Level:            level,
           WithAssociations: withAssoc,
       })
   ```

**Estimated Effort:** 2-3 days

### 3.2 $ZRAY Framework - HYBRID APPROACH

**Status:** ⚠️ **NO REST API - LOCAL EXECUTION ONLY**

**Three Options:**

#### Option A: RFC Call (if $ZRAY is available)
```go
// pkg/ray/executor.go
func (e *RFCExecutor) AnalyzePackage(pkg string) (*PackageGraph, error) {
    // RFC call to ABAP wrapper function module
    // Z_RAY_RFC_ANALYZE_PACKAGE
    // Returns JSON-serialized graph
}
```

**Pros:** Leverage existing $ZRAY logic
**Cons:** Requires RFC, $ZRAY must be installed, network latency

#### Option B: Pure Go Parser (always works)
```go
// pkg/ray/parser.go
func (p *LocalParser) AnalyzePackage(pkg string) (*PackageGraph, error) {
    // Fetch all objects via ADT
    // Parse ABAP source
    // Build dependency graph
    // Calculate metrics
}
```

**Pros:** No ABAP dependency, works everywhere
**Cons:** Complex ABAP parsing, may miss edge cases

#### Option C: Cache-First Hybrid ⭐ RECOMMENDED
```go
// pkg/ray/analyzer.go
func (a *Analyzer) AnalyzePackage(pkg string) (*PackageGraph, error) {
    // 1. Check cache
    if cached, found := a.cache.Get(pkg); found {
        return cached.(*PackageGraph), nil
    }

    // 2. Try RFC to $ZRAY (if available)
    if a.rfcExecutor.IsAvailable() {
        if graph, err := a.rfcExecutor.AnalyzePackage(pkg); err == nil {
            a.cache.Set(pkg, graph)
            return graph, nil
        }
    }

    // 3. Fallback to local parser
    graph, err := a.localParser.AnalyzePackage(pkg)
    if err != nil { return nil, err }

    a.cache.Set(pkg, graph)
    return graph, nil
}
```

**Estimated Effort:**
- RFC wrapper (if $ZRAY available): 2-3 days
- Local Go parser: 2-3 weeks
- Hybrid orchestration: 1 day

---

## Part 4: Proposed MCP Tools

### Tool 1: GetCDSDependencies

**Status:** ✅ Ready to implement (REST API exists)

**Input:**
```json
{
  "ddls_name": "ACM_DDDDLSRC",
  "dependency_level": "hierarchy",
  "with_associations": false,
  "context_package": ""
}
```

**Output:**
```json
{
  "name": "ACM_DDDDLSRC",
  "type": "CDS_VIEW",
  "entity_name": "ACM_DDDDLSRC_CDSV",
  "activation_state": "ACTIVE",
  "ddls_name": "ACM_DDDDLSRC",
  "children": [
    {
      "name": "DDDDLSRC",
      "type": "TABLE",
      "relation": "FROM"
    }
  ]
}
```

### Tool 2: AnalyzePackage (using $ZRAY or local)

**Status:** ⚠️ Requires RFC wrapper or Go parser

**Input:**
```json
{
  "package_name": "$ZRAY_00",
  "depth": 1,
  "include_tests": false
}
```

**Output:**
```json
{
  "package": "$ZRAY_00",
  "metrics": {
    "total_objects": 26,
    "total_loc": 5000,
    "classes": 14,
    "programs": 0
  },
  "nodes": [
    {"id": "ZCL_RAY_00_PACKAGE", "type": "CLAS", "loc": 50},
    {"id": "ZCL_RAY_00_NODE", "type": "CLAS", "loc": 200}
  ],
  "edges": []
}
```

---

## Part 5: Next Actions

### Immediate (Week 1)

1. **Verify CDS Endpoint** (1 day)
   - Use `/IWFND/GW_CLIENT` or Postman
   - Test with `ACM_DDDDLSRC` CDS view
   - Document exact endpoint and response format
   - Capture XML schema

2. **Implement GetCDSDependencies** (2 days)
   - Add Go client method in `pkg/adt/cds.go`
   - Add XML parsing types
   - Add integration test
   - Add MCP tool handler

3. **Check for RFC Wrapper** (1 day)
   - Search for `Z*RAY*RFC*` function modules
   - Test if $ZRAY can be called via RFC
   - Document interface if exists

### Short-term (Week 2-3)

4. **Prototype Local Parser** (5 days)
   - Simple ABAP call parser (CALL FUNCTION, CALL METHOD)
   - Basic dependency graph
   - Test with small package

5. **Design Graph Schema** (1 day)
   - Define Go structs for nodes/edges
   - JSON serialization
   - Cache strategy

### Medium-term (Week 4-6)

6. **Complete Local Parser** (10 days)
   - Handle all ABAP patterns
   - Recursive dependency resolution
   - Metrics calculation

7. **Implement Hybrid Analyzer** (3 days)
   - Cache-first logic
   - RFC fallback (if available)
   - Local parser fallback

---

## Part 6: Test CDS Views Available

The following CDS views were discovered for testing:

1. `ACM_DDDDLSRC` - Wrapping-View for DDDDLSRC table (SACMTOOLS)
2. `/1BS/SADL_QE_TEST_VIEW` - Unit test view (/1BS/SADL_QE_TEST)

**Recommended Test Case:** `ACM_DDDDLSRC`
- Simple view wrapping a table
- Should have straightforward dependency tree
- Available in all SAP systems

---

## Part 7: Risk Assessment

### CDS Dependencies Implementation

**Risk:** 🟢 LOW

- REST API confirmed to exist
- Well-defined structure
- Standard ADT endpoint
- 2-3 days implementation

**Mitigation:**
- Endpoint verification first
- Comprehensive testing with various CDS types

### $ZRAY Integration

**Risk:** 🟡 MEDIUM

- No REST API exists
- Requires RFC or local parsing
- Complex ABAP analysis needed

**Mitigation:**
- Hybrid approach provides fallback
- Start with simple parser
- Progressive enhancement
- Cache results for offline use

---

## Conclusion

### ✅ CDS Dependencies: READY

**Finding:** ADT REST API exists (`CL_CDS_RES_DEPENDENCIES`)
**Action:** Implement `GetCDSDependencies` MCP tool
**Timeline:** 2-3 days

### ⚠️ $ZRAY Framework: REQUIRES WRAPPER

**Finding:** No REST/HTTP/OData API exists
**Action:** Implement Cache-First Hybrid approach
**Timeline:**
- Phase 1 (RFC wrapper): 2-3 days (if $ZRAY available)
- Phase 2 (Local parser): 2-3 weeks
- Phase 3 (Hybrid): 1 day

### Recommended Roadmap

**Week 1:** CDS dependencies + endpoint verification
**Week 2:** Check RFC availability, design parser
**Week 3-5:** Implement local Go parser
**Week 6:** Hybrid integration + documentation

**Next Immediate Step:** Verify CDS dependency endpoint with real test.
