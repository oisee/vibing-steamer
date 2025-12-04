# Library Architecture & Caching Strategy for Graph/API Tools

**Date:** 2025-12-02
**Report ID:** 009
**Subject:** Multi-layer architecture for programmatic usage and intelligent caching
**Related Documents:** Reports 005-007 (Graph & API Surface designs)

---

## Executive Summary

This document addresses three critical architectural questions:

1. **How to expose tools?** MCP-only vs Library vs Standalone vs Hybrid
2. **Where to cache graphs?** Storage options and data models
3. **How to invalidate cache?** Smart change detection strategies

**Recommendation:** Build as a **Go library** with multiple consumption layers (MCP, CLI, programmatic) and **SQLite-based caching** with hash-based invalidation.

---

## Part 1: Architecture Options Analysis

### Option 1: MCP Tools Only (Current State)

```
┌─────────────────────────────────────┐
│         MCP Server                  │
│  (internal/mcp/server.go)           │
│                                     │
│  Tool handlers directly implement   │
│  all logic inline                   │
└─────────────────────────────────────┘
```

**Pros:**
- Simple, no additional layers
- Works with Claude immediately
- Single codebase

**Cons:**
- ❌ Not reusable outside MCP
- ❌ Hard to test (need full MCP setup)
- ❌ Can't script/automate
- ❌ No library for other Go programs

---

### Option 2: Standalone Executable + MCP

```
┌──────────────────┐     ┌──────────────────┐
│   MCP Server     │────>│   Subprocess     │
│   (wrapper)      │     │   CLI tool       │
└──────────────────┘     └──────────────────┘
```

**Pros:**
- Can use CLI for scripting
- Separation of concerns
- MCP just wraps CLI

**Cons:**
- ❌ Two codebases to maintain
- ❌ Subprocess overhead
- ❌ Complex error handling
- ❌ Still not a library

---

### Option 3: Go Library + Multiple Consumers (RECOMMENDED)

```
┌─────────────────────────────────────────────────────────┐
│              Core Library (pkg/apisurface)              │
│                      (pkg/graph)                        │
│                                                         │
│  • Pure Go packages                                     │
│  • No dependencies on MCP or CLI                        │
│  • Fully testable                                       │
│  • Can be imported by any Go program                    │
└─────────────────────────────────────────────────────────┘
            ↑              ↑              ↑
            │              │              │
┌───────────┴──┐  ┌────────┴────┐  ┌─────┴─────────┐
│  MCP Tools   │  │  CLI Tool   │  │  Other Go     │
│  (wrapper)   │  │  (wrapper)  │  │  Programs     │
└──────────────┘  └─────────────┘  └───────────────┘
```

**Pros:**
- ✅ Maximum reusability
- ✅ Easy to test (unit tests)
- ✅ Can be imported by other projects
- ✅ MCP and CLI are just thin wrappers
- ✅ Can build .so/.dll later

**Cons:**
- Slightly more initial work (but worth it)

---

### Option 4: Future Enhancement - C-Compatible Library

```
┌─────────────────────────────────────────────────────────┐
│              Core Go Library                            │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│         CGO Wrapper (libapisurface.so/.dll)             │
│         Exports C-compatible API                        │
└─────────────────────────────────────────────────────────┘
            ↑              ↑              ↑
            │              │              │
┌───────────┴──┐  ┌────────┴────┐  ┌─────┴─────────┐
│  Python      │  │  Node.js    │  │  Ruby         │
│  (ctypes)    │  │  (ffi)      │  │  (fiddle)     │
└──────────────┘  └─────────────┘  └───────────────┘
```

**Future consideration** - park this for Phase 10+

---

## Part 2: Recommended Architecture

### Directory Structure

```
pkg/
├── apisurface/           # API Surface Scraper library
│   ├── scraper.go        # Core scraping logic
│   ├── enricher.go       # Metadata enrichment
│   ├── ranker.go         # Ranking algorithms
│   ├── clusterer.go      # Clustering logic
│   ├── patterns.go       # Pattern detection
│   ├── reporter.go       # Report generation
│   ├── cache.go          # Cache layer
│   └── types.go          # Shared types
│
├── graph/                # Graph Analysis library
│   ├── query.go          # CROSS/WBCROSSGT queries
│   ├── traversal.go      # UP/DOWN traversal
│   ├── graph.go          # Core graph structure
│   ├── analyzer.go       # Graph analysis
│   ├── cache.go          # Graph caching
│   └── types.go          # Shared types
│
└── cache/                # Shared caching infrastructure
    ├── sqlite.go         # SQLite backend
    ├── memory.go         # In-memory cache
    ├── types.go          # Cache types
    └── invalidation.go   # Invalidation strategies

cmd/
├── vsp/           # MCP Server (existing)
│   └── main.go
│
├── adt-cli/              # New: Standalone CLI tool
│   └── main.go
│
└── api-scraper/          # New: Specialized API scraper CLI
    └── main.go

internal/
├── mcp/
│   ├── server.go         # MCP server core
│   ├── apisurface_tools.go  # API surface MCP wrappers
│   └── graph_tools.go    # Graph MCP wrappers
│
└── cli/
    ├── api_commands.go   # CLI commands for API scraper
    └── graph_commands.go # CLI commands for graph tools
```

### Library-First Design Principles

1. **Pure Go packages** in `pkg/` - no dependencies on CLI or MCP
2. **Thin wrappers** in `cmd/` and `internal/` - just call library
3. **Interface-based** - easy to mock and test
4. **Configuration structs** - not CLI flags in library code
5. **Context-aware** - proper context.Context throughout

### Example: Using as Library

```go
package main

import (
    "context"
    "github.com/oisee/vibing-steampunk/pkg/apisurface"
    "github.com/oisee/vibing-steampunk/pkg/adt"
)

func main() {
    // Create ADT client
    client, _ := adt.NewClient(adt.Config{
        URL:      "http://host:50000",
        Username: "user",
        Password: "pass",
    })

    // Create API scraper
    scraper := apisurface.NewScraper(client, apisurface.Config{
        Packages:     []string{"Z*", "$ZRAY*"},
        IncludeTypes: []string{"F", "ME", "TY"},
        UseCache:     true,
        CacheTTL:     24 * time.Hour,
    })

    // Scrape APIs
    ctx := context.Background()
    apis, err := scraper.ScrapeAPIs(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Rank APIs
    ranker := apisurface.NewRanker(apisurface.DefaultWeights)
    ranked := ranker.Rank(apis)

    // Cluster by module
    clusterer := apisurface.NewClusterer()
    clusters := clusterer.ClusterByModule(ranked)

    // Generate report
    reporter := apisurface.NewReporter("html")
    html, _ := reporter.Generate(clusters)

    // Save to file
    os.WriteFile("api-report.html", []byte(html), 0644)
}
```

### Example: CLI Usage

```bash
# Scrape APIs
adt-cli api scrape \
  --packages "Z*" \
  --include-types "F,ME,TY" \
  --cache \
  --output apis.json

# Rank and cluster
adt-cli api rank apis.json \
  | adt-cli api cluster --by module \
  > clustered.json

# Generate report
adt-cli api report clustered.json \
  --format html \
  --output report.html
```

### Example: MCP Tool Usage

```json
{
  "tool": "ScrapeAPISurface",
  "arguments": {
    "packages": ["Z*", "$ZRAY*"],
    "include_types": ["F", "ME", "TY"],
    "use_cache": true,
    "max_results": 100
  }
}
```

---

## Part 3: Caching Architecture

### Storage Options Comparison

| Option | Pros | Cons | Recommendation |
|--------|------|------|----------------|
| **In-Memory** | Fast, simple | Lost on restart, memory limits | Development only |
| **SQLite** | Fast, portable, SQL queries | Single process | ✅ **Primary choice** |
| **PostgreSQL** | Production-grade, concurrent | Complex setup | Future (Phase 8+) |
| **Neo4j** | Graph queries, visualization | Heavy, expensive | Future (Phase 10+) |
| **File-based JSON** | Simple, human-readable | Slow, no queries | Backup/export only |

**Decision: SQLite for local caching, PostgreSQL as optional future upgrade**

### SQLite Schema

```sql
-- Cached nodes (graph vertices)
CREATE TABLE cached_nodes (
    id TEXT PRIMARY KEY,                    -- Node ID (e.g., "ME.ZCL_CLASS\ME:METHOD")
    object_type TEXT NOT NULL,              -- CLAS, PROG, FUNC, etc.
    object_name TEXT NOT NULL,              -- Object name
    package TEXT,                           -- DEVCLASS

    -- Change detection
    source_hash TEXT,                       -- SHA256 of source code
    last_modified_adt INTEGER,              -- ADT timestamp (Unix)
    cached_at INTEGER NOT NULL,             -- When cached (Unix)

    -- Validity
    valid INTEGER DEFAULT 1,                -- 0 = invalidated, 1 = valid
    invalidated_at INTEGER,                 -- When invalidated
    invalidation_reason TEXT,               -- Why invalidated

    -- Metadata (JSON)
    metadata TEXT,                          -- Flexible JSON storage

    -- Indexes
    INDEX idx_object_type_name ON cached_nodes(object_type, object_name),
    INDEX idx_package ON cached_nodes(package),
    INDEX idx_valid ON cached_nodes(valid),
    INDEX idx_cached_at ON cached_nodes(cached_at)
);

-- Cached edges (graph edges)
CREATE TABLE cached_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id TEXT NOT NULL,                  -- Source node ID
    to_id TEXT NOT NULL,                    -- Target node ID
    edge_type TEXT NOT NULL,                -- CALLS, USES, IMPLEMENTS, etc.
    source TEXT NOT NULL,                   -- CROSS, WBCROSSGT

    -- Change tracking
    discovered_at INTEGER NOT NULL,         -- Unix timestamp
    valid INTEGER DEFAULT 1,                -- 0 = invalidated, 1 = valid

    -- Indexes
    INDEX idx_from_id ON cached_edges(from_id),
    INDEX idx_to_id ON cached_edges(to_id),
    INDEX idx_valid_edges ON cached_edges(valid),

    -- Unique constraint
    UNIQUE(from_id, to_id, edge_type)
);

-- Cache metadata (for invalidation policies)
CREATE TABLE cache_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Store config like: last_full_scan, cache_version, ttl_seconds, etc.
INSERT INTO cache_metadata VALUES
    ('cache_version', '1.0.0', strftime('%s', 'now')),
    ('default_ttl_seconds', '86400', strftime('%s', 'now')),
    ('last_full_scan', '', strftime('%s', 'now'));

-- API surface cache (separate from graph)
CREATE TABLE cached_apis (
    api_name TEXT NOT NULL,                 -- BAPI_SALESORDER_*
    api_type TEXT NOT NULL,                 -- F, ME, TY
    source TEXT NOT NULL,                   -- CROSS, WBCROSSGT

    -- Usage stats
    usage_count INTEGER DEFAULT 0,
    used_by_count INTEGER DEFAULT 0,

    -- Enrichment data
    package TEXT,
    module TEXT,                            -- SD, MM, FI
    description TEXT,
    is_deprecated INTEGER DEFAULT 0,

    -- Change tracking
    cached_at INTEGER NOT NULL,
    valid INTEGER DEFAULT 1,

    PRIMARY KEY (api_name, api_type),
    INDEX idx_api_module ON cached_apis(module),
    INDEX idx_api_usage ON cached_apis(usage_count DESC)
);
```

### Cache Implementation

**File:** `pkg/cache/sqlite.go`

```go
type SQLiteCache struct {
    db *sql.DB
    config CacheConfig
}

type CacheConfig struct {
    Path string        // Path to SQLite file
    TTL  time.Duration // Default TTL
}

type CachedNode struct {
    ID               string
    ObjectType       string
    ObjectName       string
    Package          string
    SourceHash       string
    LastModifiedADT  time.Time
    CachedAt         time.Time
    Valid            bool
    InvalidatedAt    *time.Time
    InvalidationReason string
    Metadata         map[string]interface{}
}

type CachedEdge struct {
    FromID       string
    ToID         string
    EdgeType     string
    Source       string
    DiscoveredAt time.Time
    Valid        bool
}

func NewSQLiteCache(config CacheConfig) (*SQLiteCache, error) {
    db, err := sql.Open("sqlite3", config.Path)
    if err != nil {
        return nil, err
    }

    // Initialize schema
    if err := initSchema(db); err != nil {
        return nil, err
    }

    return &SQLiteCache{db: db, config: config}, nil
}

// Store node
func (c *SQLiteCache) PutNode(ctx context.Context, node *CachedNode) error {
    query := `
        INSERT INTO cached_nodes
        (id, object_type, object_name, package, source_hash,
         last_modified_adt, cached_at, valid, metadata)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            source_hash = excluded.source_hash,
            last_modified_adt = excluded.last_modified_adt,
            cached_at = excluded.cached_at,
            valid = excluded.valid,
            metadata = excluded.metadata
    `

    metadataJSON, _ := json.Marshal(node.Metadata)

    _, err := c.db.ExecContext(ctx, query,
        node.ID,
        node.ObjectType,
        node.ObjectName,
        node.Package,
        node.SourceHash,
        node.LastModifiedADT.Unix(),
        node.CachedAt.Unix(),
        node.Valid,
        string(metadataJSON),
    )

    return err
}

// Get node
func (c *SQLiteCache) GetNode(ctx context.Context, id string) (*CachedNode, error) {
    query := `
        SELECT id, object_type, object_name, package, source_hash,
               last_modified_adt, cached_at, valid, invalidated_at,
               invalidation_reason, metadata
        FROM cached_nodes
        WHERE id = ? AND valid = 1
    `

    var node CachedNode
    var metadataJSON string
    var lastModifiedUnix, cachedAtUnix int64
    var invalidatedAtUnix sql.NullInt64

    err := c.db.QueryRowContext(ctx, query, id).Scan(
        &node.ID,
        &node.ObjectType,
        &node.ObjectName,
        &node.Package,
        &node.SourceHash,
        &lastModifiedUnix,
        &cachedAtUnix,
        &node.Valid,
        &invalidatedAtUnix,
        &node.InvalidationReason,
        &metadataJSON,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrNotFound
        }
        return nil, err
    }

    node.LastModifiedADT = time.Unix(lastModifiedUnix, 0)
    node.CachedAt = time.Unix(cachedAtUnix, 0)

    if invalidatedAtUnix.Valid {
        t := time.Unix(invalidatedAtUnix.Int64, 0)
        node.InvalidatedAt = &t
    }

    json.Unmarshal([]byte(metadataJSON), &node.Metadata)

    return &node, nil
}

// Check if node is still valid
func (c *SQLiteCache) IsNodeValid(ctx context.Context, id string) (bool, error) {
    // Check if exists and not expired
    node, err := c.GetNode(ctx, id)
    if err != nil {
        if err == ErrNotFound {
            return false, nil
        }
        return false, err
    }

    // Check TTL
    if time.Since(node.CachedAt) > c.config.TTL {
        return false, nil
    }

    return node.Valid, nil
}

// Invalidate node
func (c *SQLiteCache) InvalidateNode(ctx context.Context, id string, reason string) error {
    query := `
        UPDATE cached_nodes
        SET valid = 0,
            invalidated_at = ?,
            invalidation_reason = ?
        WHERE id = ?
    `

    _, err := c.db.ExecContext(ctx, query,
        time.Now().Unix(),
        reason,
        id,
    )

    return err
}

// Batch operations
func (c *SQLiteCache) PutNodes(ctx context.Context, nodes []*CachedNode) error {
    tx, err := c.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO cached_nodes
        (id, object_type, object_name, package, source_hash,
         last_modified_adt, cached_at, valid, metadata)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            source_hash = excluded.source_hash,
            last_modified_adt = excluded.last_modified_adt,
            cached_at = excluded.cached_at,
            valid = excluded.valid,
            metadata = excluded.metadata
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, node := range nodes {
        metadataJSON, _ := json.Marshal(node.Metadata)
        _, err = stmt.ExecContext(ctx,
            node.ID, node.ObjectType, node.ObjectName, node.Package,
            node.SourceHash, node.LastModifiedADT.Unix(), node.CachedAt.Unix(),
            node.Valid, string(metadataJSON),
        )
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

---

## Part 4: Change Detection & Invalidation

### Change Detection Strategies

#### 1. Hash-Based (RECOMMENDED)

```go
type HashInvalidator struct {
    client *adt.Client
    cache  Cache
}

func (h *HashInvalidator) CheckAndInvalidate(ctx context.Context, nodeID string) error {
    // Get cached node
    cached, err := h.cache.GetNode(ctx, nodeID)
    if err != nil {
        return err
    }

    // Get current source code from ADT
    source, err := h.getSourceCode(ctx, cached.ObjectType, cached.ObjectName)
    if err != nil {
        return err
    }

    // Calculate current hash
    currentHash := sha256.Sum256([]byte(source))
    currentHashStr := hex.EncodeToString(currentHash[:])

    // Compare with cached hash
    if currentHashStr != cached.SourceHash {
        // Code changed - invalidate
        return h.cache.InvalidateNode(ctx, nodeID,
            fmt.Sprintf("Source code changed (hash mismatch)"))
    }

    return nil
}

func (h *HashInvalidator) getSourceCode(ctx context.Context, objType, objName string) (string, error) {
    switch objType {
    case "CLAS":
        class, err := h.client.GetClass(ctx, objName)
        if err != nil {
            return "", err
        }
        return class.Source, nil
    case "PROG":
        prog, err := h.client.GetProgram(ctx, objName)
        if err != nil {
            return "", err
        }
        return prog.Source, nil
    case "FUNC":
        // Get function source
        // ...
    }
    return "", fmt.Errorf("unsupported object type: %s", objType)
}
```

#### 2. Timestamp-Based

```go
type TimestampInvalidator struct {
    client *adt.Client
    cache  Cache
}

func (t *TimestampInvalidator) CheckAndInvalidate(ctx context.Context, nodeID string) error {
    cached, err := t.cache.GetNode(ctx, nodeID)
    if err != nil {
        return err
    }

    // Get last modified timestamp from ADT
    adtTimestamp, err := t.getLastModified(ctx, cached.ObjectType, cached.ObjectName)
    if err != nil {
        return err
    }

    // Compare with cached timestamp
    if adtTimestamp.After(cached.LastModifiedADT) {
        return t.cache.InvalidateNode(ctx, nodeID,
            "ADT timestamp is newer than cached timestamp")
    }

    return nil
}
```

#### 3. TTL-Based (Simple)

```go
type TTLInvalidator struct {
    cache Cache
    ttl   time.Duration
}

func (t *TTLInvalidator) CheckAndInvalidate(ctx context.Context, nodeID string) error {
    cached, err := t.cache.GetNode(ctx, nodeID)
    if err != nil {
        return err
    }

    // Check if expired
    if time.Since(cached.CachedAt) > t.ttl {
        return t.cache.InvalidateNode(ctx, nodeID,
            fmt.Sprintf("TTL expired (%v)", t.ttl))
    }

    return nil
}
```

#### 4. Composite Invalidator (BEST)

```go
type CompositeInvalidator struct {
    invalidators []Invalidator
}

func (c *CompositeInvalidator) CheckAndInvalidate(ctx context.Context, nodeID string) error {
    for _, inv := range c.invalidators {
        if err := inv.CheckAndInvalidate(ctx, nodeID); err != nil {
            return err
        }
    }
    return nil
}

// Usage:
invalidator := &CompositeInvalidator{
    invalidators: []Invalidator{
        NewTTLInvalidator(cache, 7*24*time.Hour),   // Expire after 7 days
        NewHashInvalidator(client, cache),          // Check hash if < 7 days old
    },
}
```

### Invalidation Policies

```go
type InvalidationPolicy struct {
    // When to check
    CheckOnRead  bool          // Check validity on every cache read
    CheckPeriod  time.Duration // Background check interval

    // What to check
    UseHashCheck      bool
    UseTimestampCheck bool
    UseTTL            bool
    TTL               time.Duration

    // How to handle
    InvalidateEdges   bool // Also invalidate related edges
    Cascade           bool // Invalidate dependent nodes
}

// Example policies
var (
    // Aggressive: Always fresh, check on read
    AggressivePolicy = InvalidationPolicy{
        CheckOnRead:       true,
        UseHashCheck:      true,
        UseTimestampCheck: true,
        InvalidateEdges:   true,
        Cascade:           true,
    }

    // Balanced: TTL + hash check
    BalancedPolicy = InvalidationPolicy{
        CheckOnRead:       false,
        CheckPeriod:       1 * time.Hour,
        UseHashCheck:      true,
        UseTTL:            true,
        TTL:               24 * time.Hour,
        InvalidateEdges:   true,
    }

    // Lazy: TTL only
    LazyPolicy = InvalidationPolicy{
        CheckOnRead:       false,
        CheckPeriod:       24 * time.Hour,
        UseTTL:            true,
        TTL:               7 * 24 * time.Hour,
        InvalidateEdges:   false,
    }
)
```

---

## Part 5: Consumer API Design

### Flexible Starting Point Configuration

```go
type ScrapeConfig struct {
    // Starting Points (at least one required)
    Packages      []string      `json:"packages"`       // ["$ZRAY*", "ZLLM*"]
    PackageMask   string        `json:"package_mask"`   // "Z*" (simplified)
    Objects       []ObjectRef   `json:"objects"`        // Specific objects

    // Scope Control
    MaxDepth      int           `json:"max_depth"`      // Graph traversal depth
    MaxNodes      int           `json:"max_nodes"`      // Stop after N nodes
    MaxAPIs       int           `json:"max_apis"`       // Stop after N APIs

    // Type Filters
    IncludeTypes  []string      `json:"include_types"`  // ["F", "ME", "TY"]
    ExcludeTypes  []string      `json:"exclude_types"`  // ["DA", "EV"]
    OnlyStandard  bool          `json:"only_standard"`  // Only SAP standard APIs
    OnlyCustom    bool          `json:"only_custom"`    // Only Z* APIs

    // Cache Control
    UseCache      bool          `json:"use_cache"`      // Use cache if available
    CacheTTL      time.Duration `json:"cache_ttl"`      // Cache validity period
    ForceRefresh  bool          `json:"force_refresh"`  // Ignore cache, force rescan

    // Output Control
    OutputFormat  string        `json:"output_format"`  // "json", "html", "markdown"
    OutputPath    string        `json:"output_path"`    // Where to save
    Pretty        bool          `json:"pretty"`         // Pretty-print JSON
}

type ObjectRef struct {
    Type string `json:"type"` // "CLAS", "PROG", "FUNC", "FUGR", "INTF"
    Name string `json:"name"` // "ZCL_MY_CLASS"
}

// Constructor helpers
func FromPackages(packages ...string) ScrapeConfig {
    return ScrapeConfig{Packages: packages}
}

func FromPackageMask(mask string) ScrapeConfig {
    return ScrapeConfig{PackageMask: mask}
}

func FromObjects(objects ...ObjectRef) ScrapeConfig {
    return ScrapeConfig{Objects: objects}
}

// Fluent builder
type ConfigBuilder struct {
    config ScrapeConfig
}

func NewConfig() *ConfigBuilder {
    return &ConfigBuilder{
        config: ScrapeConfig{
            MaxDepth:     5,
            UseCache:     true,
            CacheTTL:     24 * time.Hour,
            OutputFormat: "json",
        },
    }
}

func (b *ConfigBuilder) Packages(packages ...string) *ConfigBuilder {
    b.config.Packages = packages
    return b
}

func (b *ConfigBuilder) Object(typ, name string) *ConfigBuilder {
    b.config.Objects = append(b.config.Objects, ObjectRef{Type: typ, Name: name})
    return b
}

func (b *ConfigBuilder) MaxDepth(depth int) *ConfigBuilder {
    b.config.MaxDepth = depth
    return b
}

func (b *ConfigBuilder) UseCache(use bool) *ConfigBuilder {
    b.config.UseCache = use
    return b
}

func (b *ConfigBuilder) Build() ScrapeConfig {
    return b.config
}
```

### Usage Examples

#### From MCP Tool

```json
{
  "tool": "ScrapeAPISurface",
  "arguments": {
    "packages": ["$ZRAY*"],
    "max_apis": 100,
    "include_types": ["F", "ME"],
    "only_standard": true,
    "use_cache": true,
    "output_format": "json"
  }
}
```

#### From CLI

```bash
# From packages
adt-cli api-surface scrape \
  --packages "$ZRAY*,ZLLM*" \
  --max-apis 100 \
  --include-types "F,ME,TY" \
  --only-standard \
  --cache \
  --output apis.json

# From specific object
adt-cli api-surface scrape \
  --object "CLAS:ZCL_RAY_10_SPIDER" \
  --max-depth 3 \
  --cache \
  --output spider-deps.json

# From package mask
adt-cli api-surface scrape \
  --package-mask "Z*" \
  --max-nodes 1000 \
  --force-refresh \
  --output all-z-apis.json
```

#### From Go Code

```go
// Simple: Just packages
config := apisurface.FromPackages("$ZRAY*", "ZLLM*")
scraper := apisurface.NewScraper(client, config)
apis, err := scraper.ScrapeAPIs(ctx)

// Fluent: More control
config := apisurface.NewConfig().
    Packages("$ZRAY*").
    Object("CLAS", "ZCL_RAY_10_SPIDER").
    MaxDepth(3).
    IncludeTypes("F", "ME", "TY").
    OnlyStandard(true).
    UseCache(true).
    Build()

scraper := apisurface.NewScraper(client, config)
apis, err := scraper.ScrapeAPIs(ctx)

// Custom objects
config := apisurface.FromObjects(
    apisurface.ObjectRef{Type: "CLAS", Name: "ZCL_MY_CLASS"},
    apisurface.ObjectRef{Type: "PROG", Name: "ZREPORT"},
    apisurface.ObjectRef{Type: "FUNC", Name: "Z_MY_FUNCTION"},
)
```

---

## Part 6: Implementation Roadmap

### Phase 1: Core Library (Week 1)
- [ ] Create `pkg/cache` with SQLite backend
- [ ] Implement cache CRUD operations
- [ ] Add hash-based invalidation
- [ ] Unit tests for cache layer

### Phase 2: API Surface Library (Week 2)
- [ ] Refactor existing scraper code to `pkg/apisurface`
- [ ] Add cache integration
- [ ] Implement flexible config (packages/mask/objects)
- [ ] Unit tests

### Phase 3: Graph Library (Week 3)
- [ ] Create `pkg/graph` package
- [ ] Add cache integration
- [ ] Implement change detection
- [ ] Unit tests

### Phase 4: MCP Integration (Week 4)
- [ ] Update MCP tools to use libraries
- [ ] Add cache control parameters
- [ ] Integration tests
- [ ] Documentation

### Phase 5: CLI Tool (Week 5)
- [ ] Create `cmd/adt-cli` with cobra
- [ ] Implement all commands (api, graph, cache)
- [ ] Add shell completion
- [ ] User documentation

### Phase 6: Advanced Features (Week 6+)
- [ ] Pattern detection
- [ ] Report generation (HTML/MD)
- [ ] Batch operations
- [ ] Performance optimization

---

## Part 7: Decision Summary

### ✅ Decisions Made

1. **Architecture:** Go Library + Multiple Consumers
   - Core logic in `pkg/apisurface` and `pkg/graph`
   - MCP tools as thin wrappers
   - CLI tool as separate binary
   - Can be imported by other Go programs

2. **Caching:** SQLite-based with hash invalidation
   - Portable, fast, SQL queries
   - Hash-based change detection (primary)
   - TTL-based expiration (fallback)
   - Composite invalidation strategy

3. **API Design:** Flexible starting points
   - Packages (array)
   - Package mask (wildcard)
   - Specific objects (array of refs)
   - Fluent builder for complex configs

### 🚧 Parked for Future

1. **C-Compatible Library** (.so/.dll)
   - CGO wrapper for Python/Node/Ruby
   - Phase 10+ (if needed)

2. **Graph Database** (Neo4j)
   - Advanced graph queries
   - Visualization
   - Phase 10+ (if scale demands)

3. **PostgreSQL Backend**
   - Multi-user caching
   - Production deployment
   - Phase 8+ (if needed)

---

## Conclusion

**Recommended approach:**

1. ✅ Build as Go library first (`pkg/apisurface`, `pkg/graph`, `pkg/cache`)
2. ✅ Add SQLite-based caching with hash invalidation
3. ✅ Wrap with MCP tools and CLI
4. ✅ Support flexible starting points (packages/mask/objects)
5. 🚧 Park shared library (.so/.dll) for future

This gives us:
- **Maximum reusability** (library can be imported)
- **Scriptability** (CLI for automation)
- **Claude integration** (MCP tools)
- **Smart caching** (hash-based invalidation)
- **Flexibility** (multiple starting point options)

**Start with Phase 1 (Core Library + Caching) and iterate!**
