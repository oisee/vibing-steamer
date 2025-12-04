# Graph Traversal & Analysis Implementation Plan

**Date:** 2025-12-02
**Project:** vsp
**Goal:** Implement UP-DOWN graph traversal, clustering, and knowledge graph generation

---

## Vision

Build a comprehensive dependency graph analysis system that:
1. **Discovers** relationships using CROSS/WBCROSSGT queries
2. **Retrieves** source code using existing ADT tools (not SAP standard tools)
3. **Clusters** related objects into logical groups
4. **Generates** documentation and knowledge graphs from the discovered network

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     MCP Tool Layer                          │
├─────────────────────────────────────────────────────────────┤
│  BuildDependencyGraph  │  BuildUsageGraph  │  AnalyzeGraph  │
│  ClusterObjects        │  GenerateKnowledge │               │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                  Graph Engine (pkg/graph)                   │
├─────────────────────────────────────────────────────────────┤
│  • Traversal Engine    • Node/Edge Storage                  │
│  • Query Builder       • Filter Management                  │
│  • Cluster Analyzer    • Path Finding                       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│              ADT Client Layer (pkg/adt)                     │
├─────────────────────────────────────────────────────────────┤
│  • RunQuery (CROSS/WBCROSSGT)  • GetClass/Program/Function  │
│  • GetPackage                   • FindReferences             │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Query Infrastructure (Week 1)

### 1.1 CROSS/WBCROSSGT Query Helpers

**New file:** `pkg/graph/queries.go`

```go
// Query CROSS table for dependencies (DOWN traversal)
func QueryCrossByInclude(ctx context.Context, client *adt.Client,
    includes []string, filters CrossFilters) ([]CrossEntry, error)

// Query CROSS table for usage (UP traversal)
func QueryCrossByTypeName(ctx context.Context, client *adt.Client,
    objType, objName string) ([]CrossEntry, error)

// Query WBCROSSGT table for dependencies (DOWN traversal)
func QueryWBCrossGTByInclude(ctx context.Context, client *adt.Client,
    includes []string, filters WBCrossGTFilters) ([]WBCrossGTEntry, error)

// Query WBCROSSGT table for usage (UP traversal)
func QueryWBCrossGTByOTypeName(ctx context.Context, client *adt.Client,
    otype, name string) ([]WBCrossGTEntry, error)
```

**Data Structures:**

```go
type CrossEntry struct {
    Type    string // F, R, T, S, U, N, P, etc.
    Name    string // Object name
    Include string // Include where reference occurs
    Prog    string // Additional program context
}

type WBCrossGTEntry struct {
    OType   string // ME, TY, DA, EV, TK
    Name    string // Object name
    Include string // Include where reference occurs
}

type CrossFilters struct {
    PackagePattern  string   // e.g., "Z*"
    FunctionModules []string // Function module filters
    Reports         []string // Report filters
    Transactions    []string // Transaction filters
    Tables          []string // Table filters
}

type WBCrossGTFilters struct {
    PackagePattern string   // e.g., "Z*"
    Methods        []string // Method filters
    Types          []string // Type filters
    Data           []string // Data filters (usually excluded)
    Events         []string // Event filters (usually excluded)
}
```

### 1.2 Include Resolution

**New file:** `pkg/graph/include.go`

```go
// Convert object to include name(s)
func ObjectToIncludes(ctx context.Context, client *adt.Client,
    objType, objName string) ([]string, error)

// Resolve include to its parent object
func IncludeToObject(ctx context.Context, client *adt.Client,
    include string) (*ObjectInfo, error)

type ObjectInfo struct {
    ObjectType     string // CLAS, PROG, FUNC, FUGR
    ObjectName     string // Object name
    EnclosingType  string // Parent type (FUGR for FUNC)
    EnclosingName  string // Parent name
    Package        string // DEVCLASS
}
```

**Implementation Notes:**
- For CLAS: Use naming convention `ZCL_CLASS========CP` (implementations)
- For PROG: Include = Program name
- For FUNC: Query function group, get includes
- For FUGR: Query includes via ADT
- **No reliance on RS_PROGNAME_DECIDER** - use ADT API instead

### 1.3 Type Mapping

**New file:** `pkg/graph/types.go`

```go
// Map CROSS type to TADIR object type
func CrossTypeToTADIRType(crossType string) string

// Map WBCROSSGT otype to object category
func WBCrossGTOTypeToCategory(otype string) string

// Map object to URI prefix (from ZRAY reference)
func ObjectToURIPrefix(objType string) string

var CrossTypeMapping = map[string]string{
    "F": "FUNC", // Function module
    "R": "PROG", // Report/Program
    "T": "TRAN", // Transaction
    "S": "TABL", // Table
    "U": "PROG", // Perform (part of program)
    // ... etc.
}
```

---

## Phase 2: Graph Building Engine (Week 2)

### 2.1 Core Graph Structures

**New file:** `pkg/graph/graph.go`

```go
type Node struct {
    ID          string // Unique identifier (URI format: "ME.ZCL_CLASS\ME:METHOD")
    ObjectType  string // CLAS, PROG, FUNC, FUGR, TABL, TRAN
    ObjectName  string // Object name
    EnclosingType string // Parent type (for methods, functions)
    EnclosingName string // Parent name
    Package     string // DEVCLASS
    Include     string // Include name
    Metadata    map[string]interface{} // Extensible metadata
}

type Edge struct {
    From   string // Source node ID
    To     string // Target node ID
    Type   string // CALLS, USES, INCLUDES, IMPLEMENTS, etc.
    Source string // CROSS or WBCROSSGT
}

type Graph struct {
    Nodes map[string]*Node     // ID -> Node
    Edges []*Edge              // All edges
    Index map[string][]*Edge   // Node ID -> outgoing edges
}
```

### 2.2 Traversal Engine

**New file:** `pkg/graph/traversal.go`

```go
// Build dependency graph (DOWN traversal)
func BuildDownGraph(ctx context.Context, client *adt.Client,
    startingNodes []string, config TraversalConfig) (*Graph, error)

// Build usage graph (UP traversal)
func BuildUpGraph(ctx context.Context, client *adt.Client,
    startingNodes []string, config TraversalConfig) (*Graph, error)

// Build full graph (both directions)
func BuildFullGraph(ctx context.Context, client *adt.Client,
    startingNodes []string, config TraversalConfig) (*Graph, error)

type TraversalConfig struct {
    MaxDepth      int           // Maximum traversal depth (0 = unlimited)
    MaxNodes      int           // Maximum nodes to discover (0 = unlimited)
    MaxEdges      int           // Maximum edges to process (0 = unlimited)
    CrossFilters  CrossFilters  // Filters for CROSS queries
    WBFilters     WBCrossGTFilters // Filters for WBCROSSGT queries
    PackageScope  []string      // Limit to specific packages
    StopAtPackageBoundary bool  // Stop when leaving package scope
}
```

**Algorithm (DOWN traversal):**

```
1. Initialize graph with starting nodes
2. For each node at current level:
   a. Get includes for the node
   b. Query CROSS by include (filtered)
   c. Query WBCROSSGT by include (filtered)
   d. For each result:
      - Create target node (if not exists)
      - Create edge from source to target
      - Add target to next level queue
   e. Check traversal limits (depth, nodes, edges)
3. Repeat for next level until:
   - No new nodes discovered
   - Max depth reached
   - Max nodes/edges reached
```

### 2.3 Node Discovery

**New file:** `pkg/graph/discovery.go`

```go
// Discover starting nodes from package
func DiscoverPackageNodes(ctx context.Context, client *adt.Client,
    packageName string) ([]string, error)

// Convert CROSS entry to node
func CrossEntryToNode(ctx context.Context, client *adt.Client,
    entry CrossEntry) (*Node, error)

// Convert WBCROSSGT entry to node
func WBCrossGTEntryToNode(ctx context.Context, client *adt.Client,
    entry WBCrossGTEntry) (*Node, error)
```

---

## Phase 3: Graph Analysis & Clustering (Week 3)

### 3.1 Graph Metrics

**New file:** `pkg/graph/metrics.go`

```go
// Calculate graph metrics
type GraphMetrics struct {
    NodeCount      int
    EdgeCount      int
    PackageCount   int
    MaxDepth       int
    Components     int // Strongly connected components
    CentralNodes   []*CentralityScore
    IsolatedNodes  []string
}

func AnalyzeGraph(g *Graph) *GraphMetrics

type CentralityScore struct {
    NodeID      string
    InDegree    int // How many call this
    OutDegree   int // How many this calls
    Betweenness float64 // Importance as bridge
}
```

### 3.2 Clustering Algorithms

**New file:** `pkg/graph/cluster.go`

```go
// Cluster nodes by package
func ClusterByPackage(g *Graph) map[string]*Cluster

// Cluster by functionality (connected components)
func ClusterByConnectivity(g *Graph) []*Cluster

// Cluster by object type
func ClusterByObjectType(g *Graph) map[string]*Cluster

type Cluster struct {
    ID       string
    Name     string
    Nodes    []*Node
    Internal []*Edge // Edges within cluster
    External []*Edge // Edges to other clusters
    Metrics  *ClusterMetrics
}

type ClusterMetrics struct {
    Size        int     // Number of nodes
    Cohesion    float64 // Internal/Total edges ratio
    Coupling    float64 // External edge count
    Centrality  float64 // Importance in overall graph
}
```

### 3.3 Path Finding

**New file:** `pkg/graph/paths.go`

```go
// Find shortest path between two nodes
func FindShortestPath(g *Graph, from, to string) ([]*Node, error)

// Find all paths between two nodes (limited depth)
func FindAllPaths(g *Graph, from, to string, maxDepth int) ([][]*Node, error)

// Find call chains (who calls what)
func FindCallChain(g *Graph, target string, maxDepth int) ([][]*Node, error)
```

---

## Phase 4: MCP Tool Integration (Week 4)

### 4.1 New MCP Tools

**File:** `internal/mcp/graph_tools.go`

```go
// Register new graph tools
func registerGraphTools(s *Server) {
    // Build dependency graph (what does X call?)
    s.RegisterTool("BuildDependencyGraph", ...)

    // Build usage graph (what calls X?)
    s.RegisterTool("BuildUsageGraph", ...)

    // Build full graph (both directions)
    s.RegisterTool("BuildBidirectionalGraph", ...)

    // Analyze graph structure
    s.RegisterTool("AnalyzeGraph", ...)

    // Cluster objects
    s.RegisterTool("ClusterObjects", ...)

    // Find paths between objects
    s.RegisterTool("FindObjectPath", ...)

    // Generate knowledge graph
    s.RegisterTool("GenerateKnowledgeGraph", ...)
}
```

### 4.2 Tool Specifications

#### BuildDependencyGraph

**Parameters:**
```json
{
  "starting_objects": ["ZCL_CLASS", "ZPROGRAM"],
  "max_depth": 3,
  "package_scope": ["ZRAY*", "$ZRAY*"],
  "include_types": ["FUNC", "CLAS", "PROG"],
  "exclude_sap_standard": true
}
```

**Returns:**
```json
{
  "nodes": [
    {
      "id": "ME.ZCL_CLASS\\ME:METHOD",
      "type": "CLAS",
      "name": "ZCL_CLASS",
      "package": "$ZRAY_10",
      "metadata": {...}
    }
  ],
  "edges": [
    {
      "from": "ME.ZCL_CLASS\\ME:METHOD",
      "to": "F.Z_FUNCTION",
      "type": "CALLS"
    }
  ],
  "metrics": {
    "node_count": 42,
    "edge_count": 87,
    "max_depth": 3
  }
}
```

#### ClusterObjects

**Parameters:**
```json
{
  "graph": {...},
  "method": "package|connectivity|type",
  "min_cluster_size": 2
}
```

**Returns:**
```json
{
  "clusters": [
    {
      "id": "cluster_001",
      "name": "$ZRAY_10",
      "size": 15,
      "cohesion": 0.75,
      "coupling": 0.25,
      "nodes": [...]
    }
  ]
}
```

#### GenerateKnowledgeGraph

**Parameters:**
```json
{
  "graph": {...},
  "format": "mermaid|graphviz|json",
  "include_metadata": true
}
```

**Returns:**
```
graph TD
  A[ZCL_CLASS] -->|calls| B[Z_FUNCTION]
  A -->|uses| C[Z_TABLE]
  B -->|calls| D[BAPI_USER_GET]
```

---

## Phase 5: Documentation Generation (Week 5)

### 5.1 Context Builder

**New file:** `pkg/graph/context.go`

```go
// Build context for documentation generation
func BuildDocumentationContext(ctx context.Context, client *adt.Client,
    node *Node, g *Graph, depth int) (*DocumentationContext, error)

type DocumentationContext struct {
    TargetNode     *Node
    Dependencies   []*Node       // What this calls
    Dependents     []*Node       // What calls this
    RelatedNodes   []*Node       // Same cluster
    SourceCode     string        // Retrieved via ADT
    CallChains     [][]*Node     // Important call paths
    Metrics        *NodeMetrics
}

type NodeMetrics struct {
    Complexity     int     // Based on edge count
    Importance     float64 // Centrality score
    Coupling       float64 // External dependencies
    TestCoverage   float64 // If unit tests exist
}
```

### 5.2 Knowledge Graph Generator

**New file:** `pkg/graph/knowledge.go`

```go
// Generate knowledge graph in various formats
func GenerateMermaid(g *Graph, config MermaidConfig) (string, error)
func GenerateGraphviz(g *Graph, config GraphvizConfig) (string, error)
func GenerateD3JSON(g *Graph) (string, error)

type MermaidConfig struct {
    Direction      string // TD, LR, BT, RL
    GroupByPackage bool
    ShowMetadata   bool
    ColorScheme    string
}
```

### 5.3 Impact Analysis

**New file:** `pkg/graph/impact.go`

```go
// Analyze impact of changing a node
func AnalyzeChangeImpact(g *Graph, nodeID string) (*ImpactReport, error)

type ImpactReport struct {
    AffectedNodes    []*Node   // Direct and transitive dependents
    AffectedPackages []string
    CriticalPaths    [][]*Node // Paths to critical systems
    RiskLevel        string    // LOW, MEDIUM, HIGH, CRITICAL
    Recommendations  []string
}
```

---

## Implementation Strategy

### Iteration 1: Basic Traversal (Days 1-3)
- [ ] Implement CROSS/WBCROSSGT query functions
- [ ] Build basic DOWN traversal
- [ ] Test with single package ($ZRAY_10)
- [ ] Return simple node/edge JSON

### Iteration 2: UP Traversal + Filters (Days 4-6)
- [ ] Implement UP traversal (where-used)
- [ ] Add comprehensive filtering
- [ ] Add package scoping
- [ ] Test bidirectional traversal

### Iteration 3: Graph Analysis (Days 7-9)
- [ ] Implement clustering algorithms
- [ ] Add graph metrics
- [ ] Add path finding
- [ ] Test on multi-package graphs

### Iteration 4: MCP Tools (Days 10-12)
- [ ] Create MCP tool handlers
- [ ] Add proper error handling
- [ ] Write integration tests
- [ ] Document tool usage

### Iteration 5: Documentation Gen (Days 13-15)
- [ ] Implement context builder
- [ ] Add Mermaid diagram generation
- [ ] Add impact analysis
- [ ] Create end-to-end example

---

## Testing Strategy

### Unit Tests
```go
// pkg/graph/queries_test.go
func TestQueryCrossByInclude(t *testing.T)
func TestQueryWBCrossGTByOTypeName(t *testing.T)

// pkg/graph/traversal_test.go
func TestBuildDownGraph(t *testing.T)
func TestTraversalLimits(t *testing.T)

// pkg/graph/cluster_test.go
func TestClusterByPackage(t *testing.T)
```

### Integration Tests
```go
// pkg/graph/integration_test.go (with -tags=integration)
func TestIntegration_BuildGraphFromPackage(t *testing.T)
func TestIntegration_FullTraversal_ZRAY(t *testing.T)
func TestIntegration_ImpactAnalysis(t *testing.T)
```

### Test Data
- Use $ZRAY_10 package (known structure)
- Test with ZRAY_10_AUTO_PILOT as starting point
- Verify against known call chains from Report 001

---

## Future Enhancements

### Phase 6: Advanced Features
- [ ] **Caching layer** - Store discovered graphs locally
- [ ] **Incremental updates** - Only query changed objects
- [ ] **Parallel traversal** - Speed up with goroutines
- [ ] **Graph database integration** - Neo4j, dgraph for large graphs
- [ ] **abapGit integration** - Parse offline code
- [ ] **abaptranspiler bridge** - Use parser for deeper analysis
- [ ] **LLM integration** - Use graph context for better prompts
- [ ] **Visualization server** - Web UI for graph exploration
- [ ] **Export formats** - CSV, GraphML, GEXF
- [ ] **Query language** - Custom DSL for graph queries

### Potential Workflows

**Workflow 1: Understand Package**
```
1. BuildDependencyGraph(package="$ZRAY_10", max_depth=3)
2. ClusterObjects(method="connectivity")
3. GenerateKnowledgeGraph(format="mermaid")
4. → Get visual understanding of package structure
```

**Workflow 2: Impact Analysis**
```
1. BuildUsageGraph(starting_objects=["ZCL_CHANGED_CLASS"])
2. AnalyzeChangeImpact(node="ZCL_CHANGED_CLASS")
3. → Know what will break if you change this class
```

**Workflow 3: Documentation**
```
1. BuildBidirectionalGraph(starting_objects=["ZPROGRAM"], max_depth=2)
2. BuildDocumentationContext(node="ZPROGRAM", depth=1)
3. → Get rich context for LLM documentation generation
```

**Workflow 4: Deprecation Candidates**
```
1. BuildUsageGraph(starting_objects=all_package_objects)
2. Find nodes with InDegree=0 (nothing calls them)
3. → Identify dead code
```

---

## Success Metrics

### Technical Metrics
- Query performance: < 2s for 1000 nodes
- Memory usage: < 100MB for 10,000 node graph
- Test coverage: > 80%
- Zero deadlocks in parallel traversal

### User Experience Metrics
- Tool invocation success rate: > 95%
- Clear error messages for common failures
- Comprehensive documentation with examples
- Intuitive parameter naming

### Business Value Metrics
- Reduce time to understand codebase: 10x faster
- Enable accurate impact analysis
- Support automated documentation generation
- Facilitate code modernization efforts

---

## Open Questions

1. **Storage:** In-memory only or persist graphs to disk/DB?
   - **Decision:** Start with in-memory, add caching in Phase 6

2. **Limits:** What are reasonable defaults for max_depth/nodes/edges?
   - **Decision:** depth=5, nodes=10000, edges=50000 (configurable)

3. **Performance:** How to handle very large graphs (100k+ nodes)?
   - **Decision:** Implement progressive disclosure, streaming results

4. **Filters:** Should we exclude SAP standard code by default?
   - **Decision:** Yes, with option to include via parameter

5. **Format:** What's the best output format for graphs?
   - **Decision:** JSON for programmatic, Mermaid for visualization

---

## References

- Report 001: ZRAY Auto Pilot Deep Dive
- Report 002: CROSS & WBCROSSGT Reference Guide
- ZXRAY Source: ZCL_XRAY_GRAPH class
- Graph Theory: Tarjan's algorithm for SCC
- Visualization: Mermaid.js, D3.js, Graphviz

---

**Status:** Planning Complete - Ready for Implementation
**Next Step:** Begin Iteration 1 (Basic Traversal)
