# Standard API Surface Scraper

**Date:** 2025-12-02
**Subject:** Discover, rank, and analyze SAP standard APIs used by custom code
**Status:** Design Proposal

---

## Vision

Build a comprehensive analysis tool that answers critical questions:

- **"What SAP standard APIs do we actually use?"**
- **"Which standard functions/classes are most critical to our custom code?"**
- **"What SD/MM/FI modules do we depend on?"**
- **"What are the common usage patterns for BAPI_*?"**
- **"Which SAP APIs should we document/understand first?"**
- **"Are we using deprecated SAP functions?"**
- **"What's our API surface area for upgrades?"**

This enables:
- **Upgrade planning** - Know what SAP changes will impact us
- **Documentation priorities** - Document most-used APIs first
- **Pattern discovery** - Find common usage patterns
- **Dependency management** - Understand our SAP coupling
- **Knowledge transfer** - Help new developers understand SAP APIs

---

## What is "Standard API Surface"?

**Definition:** The set of all SAP standard objects (functions, classes, tables, BAPIs, etc.) that are referenced by custom Z* code.

**Example:**

```
Custom Code (Z*):
  ZCL_MY_CLASS calls → BAPI_SALESORDER_CREATEFROMDAT2 (standard)
                    → BAPI_TRANSACTION_COMMIT (standard)
                    → /IWBEP/CL_MGW_ABS_DATA (standard class)

Standard API Surface for this class:
  1. BAPI_SALESORDER_CREATEFROMDAT2 (Function Module, SD module)
  2. BAPI_TRANSACTION_COMMIT (Function Module, BC module)
  3. /IWBEP/CL_MGW_ABS_DATA (Class, SAP Gateway)
```

---

## Data Sources

### Primary: CROSS Table

```sql
SELECT type, name, COUNT(*) as usage_count,
       COUNT(DISTINCT include) as used_by_count
FROM cross
WHERE include LIKE 'Z%'           -- Custom code only
  AND name NOT LIKE 'Z%'          -- SAP standard objects only
GROUP BY type, name
ORDER BY usage_count DESC
```

**What we get:**
- Function module calls (F)
- Report submits (R)
- Transaction calls (T)
- Table references (S)
- Authority checks (A)
- Messages (N)
- Parameters (P)
- etc.

### Primary: WBCROSSGT Table

```sql
SELECT otype, name, COUNT(*) as usage_count,
       COUNT(DISTINCT include) as used_by_count
FROM wbcrossgt
WHERE include LIKE 'Z%'           -- Custom code only
  AND name NOT LIKE 'Z%'          -- SAP standard objects only
GROUP BY otype, name
ORDER BY usage_count DESC
```

**What we get:**
- Method calls (ME)
- Type references (TY)
- Data references (DA)
- Event handlers (EV)

### Enrichment Sources

**TADIR** - Repository objects
```sql
SELECT object, obj_name, devclass, author, srcsystem
FROM tadir
WHERE obj_name IN (...)
```

**TDEVC** - Package details
```sql
SELECT devclass, ctext, component
FROM tdevc
WHERE devclass IN (...)
```

**DF14T** - Development class texts (module assignment)
```sql
SELECT ps_posid, fctr_id, as4text
FROM df14t
WHERE fctr_id IN (...)
```

**ENLFDIR** - Function module directory
```sql
SELECT funcname, pname, include, generated
FROM enlfdir
WHERE funcname IN (...)
```

**SEOCLASS** - Class directory
```sql
SELECT clsname, author, version, exposure
FROM seoclass
WHERE clsname IN (...)
```

---

## Architecture

### Option A: Pure ABAP Solution

```
┌─────────────────────────────────────────────────────────────┐
│              ZRAY_API_SURFACE_SCRAPER (Report)              │
│   Select package scope, run analysis, store results         │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│           ZCL_RAY_API_SURFACE_ANALYZER                      │
│   Orchestrates scraping, enrichment, ranking                │
└─────────────────────────────────────────────────────────────┘
          ↓                    ↓                    ↓
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ Scraper          │  │ Enricher         │  │ Ranker           │
│ Query CROSS/     │  │ Get metadata     │  │ Score by usage   │
│ WBCROSSGT        │  │ from TADIR/etc   │  │ Cluster by module│
└──────────────────┘  └──────────────────┘  └──────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│              ZRAY_API_SURFACE (DB Table)                    │
│   api_object | type | usage_count | module | cluster        │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│       ZRAY_API_SURFACE_BROWSER (ALV Report)                 │
│   Interactive browser, drill-down, export                   │
└─────────────────────────────────────────────────────────────┘
```

**Pros:**
- Quick to implement
- Native SAP access
- Can reuse existing ZRAY infrastructure

**Cons:**
- Limited by SAP memory/performance
- Hard to do advanced analytics
- Difficult to integrate with external tools

---

### Option B: Go-based Scraper (RECOMMENDED)

```
┌─────────────────────────────────────────────────────────────┐
│              vsp: API Surface Tools                  │
│   ScrapeAPIsSurface | AnalyzeAPIs | ClusterAPIs             │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│           pkg/apisurface (Go Package)                       │
│   Scraper | Enricher | Ranker | Clusterer                  │
└─────────────────────────────────────────────────────────────┘
          ↓                    ↓                    ↓
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ ADT Client       │  │ SQLite Storage   │  │ Analysis Engine  │
│ RunQuery (SQL)   │  │ Local cache      │  │ Ranking/Cluster  │
│ Pagination       │  │ Fast queries     │  │ Pattern matching │
└──────────────────┘  └──────────────────┘  └──────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│              Output Formats                                 │
│   JSON | HTML Report | Markdown | CSV | GraphML            │
└─────────────────────────────────────────────────────────────┘
```

**Pros:**
- No SAP memory limits
- Incremental processing
- Advanced analytics (ML, graph DBs)
- Multiple output formats
- Can run offline/scheduled
- Better visualization options

**Cons:**
- More complex setup
- Requires Go development

---

## Implementation: Go-based Scraper

### Phase 1: Data Collection

**File:** `pkg/apisurface/scraper.go`

```go
type APIScraper struct {
    client *adt.Client
    db     *sql.DB
    config ScraperConfig
}

type ScraperConfig struct {
    PackagePatterns []string // Z*, $Z*
    BatchSize       int      // Pagination size
    MaxRows         int      // Limit per query (0 = all)
    IncludeTypes    []string // F, ME, TY, etc.
    ExcludeTypes    []string // DA, EV (too noisy)
}

type APIReference struct {
    SourceType   string // CROSS, WBCROSSGT
    APIType      string // F, ME, TY, etc.
    APIName      string // BAPI_SALESORDER_*
    UsageCount   int    // How many times called
    UsedByCount  int    // How many Z* objects call it
    UsedByList   []string // List of Z* includes
}

// Scrape CROSS table for standard API references
func (s *APIScraper) ScrapeCROSS(ctx context.Context) ([]APIReference, error) {
    query := `
        SELECT type, name,
               COUNT(*) as usage_count,
               COUNT(DISTINCT include) as used_by_count
        FROM cross
        WHERE include LIKE 'Z%'
          AND name NOT LIKE 'Z%'
          AND name NOT LIKE '$%'
        GROUP BY type, name
        ORDER BY usage_count DESC
    `

    // Use RunQuery with pagination
    results := []APIReference{}
    offset := 0

    for {
        batch, err := s.client.RunQuery(ctx, query, s.config.BatchSize, offset)
        if err != nil {
            return nil, err
        }

        if len(batch) == 0 {
            break
        }

        for _, row := range batch {
            results = append(results, APIReference{
                SourceType:  "CROSS",
                APIType:     row["TYPE"].(string),
                APIName:     row["NAME"].(string),
                UsageCount:  row["USAGE_COUNT"].(int),
                UsedByCount: row["USED_BY_COUNT"].(int),
            })
        }

        offset += len(batch)

        // Check max rows limit
        if s.config.MaxRows > 0 && offset >= s.config.MaxRows {
            break
        }
    }

    return results, nil
}

// Scrape WBCROSSGT table for standard API references
func (s *APIScraper) ScrapeWBCROSSGT(ctx context.Context) ([]APIReference, error) {
    query := `
        SELECT otype, name,
               COUNT(*) as usage_count,
               COUNT(DISTINCT include) as used_by_count
        FROM wbcrossgt
        WHERE include LIKE 'Z%'
          AND name NOT LIKE 'Z%'
          AND name NOT LIKE '$%'
          AND otype IN ('ME', 'TY')  -- Exclude noisy DA, EV
        GROUP BY otype, name
        ORDER BY usage_count DESC
    `

    // Similar pagination logic
    // ...
}

// Get detailed usage (which Z* objects use this API)
func (s *APIScraper) GetUsageDetails(ctx context.Context, apiRef APIReference) ([]string, error) {
    var query string

    if apiRef.SourceType == "CROSS" {
        query = fmt.Sprintf(`
            SELECT DISTINCT include
            FROM cross
            WHERE type = '%s' AND name = '%s'
              AND include LIKE 'Z%%'
        `, apiRef.APIType, apiRef.APIName)
    } else {
        query = fmt.Sprintf(`
            SELECT DISTINCT include
            FROM wbcrossgt
            WHERE otype = '%s' AND name = '%s'
              AND include LIKE 'Z%%'
        `, apiRef.APIType, apiRef.APIName)
    }

    results, err := s.client.RunQuery(ctx, query, 0, 0)
    // Parse results...
}
```

### Phase 2: Enrichment

**File:** `pkg/apisurface/enricher.go`

```go
type APIMetadata struct {
    APIReference
    Package      string // TADIR devclass
    Module       string // SD, MM, FI, etc.
    Component    string // SAP component (EA-APPL, EA-HR)
    Description  string // Short text
    Author       string // Original author
    IsDeprecated bool   // Marked as deprecated
    Replacement  string // Suggested replacement
    ReleaseInfo  string // Since which release
}

type MetadataEnricher struct {
    client *adt.Client
    cache  map[string]APIMetadata // Cache metadata lookups
}

// Enrich API reference with metadata from TADIR, TDEVC, etc.
func (e *MetadataEnricher) Enrich(ctx context.Context, ref APIReference) (*APIMetadata, error) {
    // Check cache first
    cacheKey := ref.APIType + ":" + ref.APIName
    if cached, ok := e.cache[cacheKey]; ok {
        return &cached, nil
    }

    meta := &APIMetadata{APIReference: ref}

    // Get TADIR info
    objType := e.crossTypeToTADIR(ref.APIType)
    tadir, err := e.getTADIRInfo(ctx, objType, ref.APIName)
    if err == nil {
        meta.Package = tadir.Package
        meta.Author = tadir.Author
    }

    // Get package module mapping
    if meta.Package != "" {
        module, component := e.getModuleInfo(ctx, meta.Package)
        meta.Module = module
        meta.Component = component
    }

    // Get description
    meta.Description = e.getDescription(ctx, ref.APIType, ref.APIName)

    // Check if deprecated
    meta.IsDeprecated = e.checkDeprecated(ctx, ref.APIType, ref.APIName)

    // Cache and return
    e.cache[cacheKey] = *meta
    return meta, nil
}

func (e *MetadataEnricher) getModuleInfo(ctx context.Context, pkg string) (module, component string) {
    // Query TDEVC for component
    query := fmt.Sprintf(`
        SELECT component
        FROM tdevc
        WHERE devclass = '%s'
    `, pkg)

    results, _ := e.client.RunQuery(ctx, query, 1, 0)
    if len(results) > 0 {
        component = results[0]["COMPONENT"].(string)
    }

    // Map component to module (using DF14T or custom mapping)
    module = e.componentToModule(component)
    return
}

var componentModuleMap = map[string]string{
    "SD": "Sales and Distribution",
    "MM": "Materials Management",
    "FI": "Financial Accounting",
    "CO": "Controlling",
    "PP": "Production Planning",
    "QM": "Quality Management",
    "PM": "Plant Maintenance",
    "HR": "Human Resources",
    "CA": "Cross-Application",
    "BC": "Basis Components",
    // ... etc.
}
```

### Phase 3: Ranking & Clustering

**File:** `pkg/apisurface/ranker.go`

```go
type APIRank struct {
    APIMetadata
    Rank            int     // Overall rank (1 = most used)
    Score           float64 // Composite score
    Criticality     string  // LOW, MEDIUM, HIGH, CRITICAL
    UsagePattern    string  // Pattern description
    RelatedAPIs     []string // Often used together
}

type APIRanker struct {
    weights RankWeights
}

type RankWeights struct {
    UsageCount  float64 // Weight for total usage count
    UsedByCount float64 // Weight for number of callers
    Module      float64 // Weight for module importance
    Recency     float64 // Weight for recently used
}

func (r *APIRanker) Rank(apis []APIMetadata) []APIRank {
    ranked := make([]APIRank, len(apis))

    for i, api := range apis {
        score := r.calculateScore(api)
        ranked[i] = APIRank{
            APIMetadata: api,
            Score:       score,
            Criticality: r.calculateCriticality(api, score),
        }
    }

    // Sort by score
    sort.Slice(ranked, func(i, j int) bool {
        return ranked[i].Score > ranked[j].Score
    })

    // Assign ranks
    for i := range ranked {
        ranked[i].Rank = i + 1
    }

    return ranked
}

func (r *APIRanker) calculateScore(api APIMetadata) float64 {
    score := 0.0

    // Usage count (normalized)
    score += float64(api.UsageCount) * r.weights.UsageCount

    // Number of callers (more important than raw usage)
    score += float64(api.UsedByCount) * r.weights.UsedByCount

    // Module importance (core modules rank higher)
    moduleWeight := r.getModuleWeight(api.Module)
    score += moduleWeight * r.weights.Module

    return score
}

func (r *APIRanker) calculateCriticality(api APIMetadata, score float64) string {
    // Criticality based on usage and module
    if api.UsedByCount > 100 || api.Module == "FI" || api.Module == "SD" {
        return "CRITICAL"
    }
    if api.UsedByCount > 50 || score > 1000 {
        return "HIGH"
    }
    if api.UsedByCount > 10 {
        return "MEDIUM"
    }
    return "LOW"
}
```

**File:** `pkg/apisurface/clusterer.go`

```go
type APICluster struct {
    ID          string
    Name        string
    Module      string
    APIs        []APIRank
    TotalUsage  int
    Description string
}

type APIClusterer struct{}

// Cluster APIs by module
func (c *APIClusterer) ClusterByModule(apis []APIRank) []APICluster {
    clusters := make(map[string]*APICluster)

    for _, api := range apis {
        module := api.Module
        if module == "" {
            module = "UNKNOWN"
        }

        if _, ok := clusters[module]; !ok {
            clusters[module] = &APICluster{
                ID:     module,
                Name:   module,
                Module: module,
                APIs:   []APIRank{},
            }
        }

        clusters[module].APIs = append(clusters[module].APIs, api)
        clusters[module].TotalUsage += api.UsageCount
    }

    // Convert to slice and sort
    result := make([]APICluster, 0, len(clusters))
    for _, cluster := range clusters {
        result = append(result, *cluster)
    }

    sort.Slice(result, func(i, j int) bool {
        return result[i].TotalUsage > result[j].TotalUsage
    })

    return result
}

// Cluster APIs by pattern (e.g., BAPI_*, /IWBEP/*, CL_*_*)
func (c *APIClusterer) ClusterByPattern(apis []APIRank) []APICluster {
    patterns := map[string]*APICluster{
        "BAPI_*":     {Name: "Business APIs (BAPIs)"},
        "/IWBEP/*":   {Name: "SAP Gateway"},
        "CL_*":       {Name: "SAP Classes"},
        "/DMO/*":     {Name: "Demo/Sample Objects"},
        "*_RFC":      {Name: "RFC Functions"},
    }

    // Match APIs to patterns
    for _, api := range apis {
        matched := false
        for pattern, cluster := range patterns {
            if c.matches(api.APIName, pattern) {
                cluster.APIs = append(cluster.APIs, api)
                cluster.TotalUsage += api.UsageCount
                matched = true
                break
            }
        }

        if !matched {
            // Add to "Other" cluster
            if _, ok := patterns["OTHER"]; !ok {
                patterns["OTHER"] = &APICluster{Name: "Other"}
            }
            patterns["OTHER"].APIs = append(patterns["OTHER"].APIs, api)
        }
    }

    // Convert and return
    // ...
}
```

### Phase 4: Pattern Detection

**File:** `pkg/apisurface/patterns.go`

```go
type UsagePattern struct {
    PatternName    string
    Description    string
    APIs           []string  // APIs in this pattern
    Frequency      int       // How often this pattern appears
    Example        string    // Code example
    BestPractice   string    // Recommended approach
}

type PatternDetector struct {
    client *adt.Client
}

// Detect common usage patterns
func (p *PatternDetector) DetectPatterns(ctx context.Context, apis []APIRank) []UsagePattern {
    patterns := []UsagePattern{}

    // Pattern 1: BAPI call with commit
    bapiCommitPattern := p.detectBAPICommitPattern(ctx, apis)
    if bapiCommitPattern != nil {
        patterns = append(patterns, *bapiCommitPattern)
    }

    // Pattern 2: Authorization check before operation
    authCheckPattern := p.detectAuthCheckPattern(ctx, apis)
    if authCheckPattern != nil {
        patterns = append(patterns, *authCheckPattern)
    }

    // Pattern 3: Table read with buffering
    tableReadPattern := p.detectTableReadPattern(ctx, apis)
    if tableReadPattern != nil {
        patterns = append(patterns, *tableReadPattern)
    }

    // Pattern 4: Gateway OData implementation
    gatewayPattern := p.detectGatewayPattern(ctx, apis)
    if gatewayPattern != nil {
        patterns = append(patterns, *gatewayPattern)
    }

    return patterns
}

func (p *PatternDetector) detectBAPICommitPattern(ctx context.Context, apis []APIRank) *UsagePattern {
    // Find includes that use BAPI_* and BAPI_TRANSACTION_COMMIT together
    query := `
        SELECT DISTINCT c1.include
        FROM cross c1
        JOIN cross c2 ON c1.include = c2.include
        WHERE c1.type = 'F' AND c1.name LIKE 'BAPI_%'
          AND c1.name NOT LIKE 'BAPI_TRANSACTION%'
          AND c2.type = 'F' AND c2.name = 'BAPI_TRANSACTION_COMMIT'
          AND c1.include LIKE 'Z%'
    `

    results, err := p.client.RunQuery(ctx, query, 0, 0)
    if err != nil || len(results) == 0 {
        return nil
    }

    return &UsagePattern{
        PatternName:  "BAPI with Commit",
        Description:  "BAPI calls followed by BAPI_TRANSACTION_COMMIT",
        Frequency:    len(results),
        Example:      "CALL FUNCTION 'BAPI_SALESORDER_CREATEFROMDAT2'...\n" +
                      "CALL FUNCTION 'BAPI_TRANSACTION_COMMIT'...",
        BestPractice: "Always commit after modifying BAPIs",
    }
}
```

### Phase 5: Report Generation

**File:** `pkg/apisurface/reporter.go`

```go
type ReportGenerator struct {
    format string // HTML, Markdown, JSON, CSV
}

// Generate HTML report
func (r *ReportGenerator) GenerateHTML(data ReportData) (string, error) {
    tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>SAP Standard API Surface Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #0070c0; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #0070c0; color: white; }
        .critical { color: red; font-weight: bold; }
        .high { color: orange; font-weight: bold; }
        .cluster { margin: 20px 0; }
    </style>
</head>
<body>
    <h1>SAP Standard API Surface Report</h1>
    <p>Generated: {{.Timestamp}}</p>
    <p>Total APIs: {{.TotalAPIs}}</p>

    <h2>Summary</h2>
    <ul>
        <li>Critical APIs: {{.CriticalCount}}</li>
        <li>High Priority APIs: {{.HighCount}}</li>
        <li>Modules Covered: {{.ModuleCount}}</li>
        <li>Total Usage References: {{.TotalUsage}}</li>
    </ul>

    <h2>Top 20 Most Used APIs</h2>
    <table>
        <tr>
            <th>Rank</th>
            <th>API</th>
            <th>Type</th>
            <th>Module</th>
            <th>Usage Count</th>
            <th>Used By</th>
            <th>Criticality</th>
        </tr>
        {{range .Top20}}
        <tr>
            <td>{{.Rank}}</td>
            <td><strong>{{.APIName}}</strong></td>
            <td>{{.APIType}}</td>
            <td>{{.Module}}</td>
            <td>{{.UsageCount}}</td>
            <td>{{.UsedByCount}}</td>
            <td class="{{.Criticality | lower}}">{{.Criticality}}</td>
        </tr>
        {{end}}
    </table>

    <h2>APIs by Module</h2>
    {{range .Clusters}}
    <div class="cluster">
        <h3>{{.Module}} ({{.TotalUsage}} total usages)</h3>
        <table>
            <tr>
                <th>API</th>
                <th>Usage Count</th>
                <th>Description</th>
            </tr>
            {{range .APIs}}
            <tr>
                <td>{{.APIName}}</td>
                <td>{{.UsageCount}}</td>
                <td>{{.Description}}</td>
            </tr>
            {{end}}
        </table>
    </div>
    {{end}}

    <h2>Common Usage Patterns</h2>
    {{range .Patterns}}
    <div class="pattern">
        <h3>{{.PatternName}}</h3>
        <p>{{.Description}}</p>
        <p><strong>Frequency:</strong> Found in {{.Frequency}} places</p>
        <p><strong>Best Practice:</strong> {{.BestPractice}}</p>
        <pre>{{.Example}}</pre>
    </div>
    {{end}}

</body>
</html>
    `

    // Execute template with data
    // ...
}

// Generate Markdown report
func (r *ReportGenerator) GenerateMarkdown(data ReportData) (string, error) {
    md := `# SAP Standard API Surface Report

Generated: ` + data.Timestamp + `

## Summary

- **Total APIs:** ` + fmt.Sprintf("%d", data.TotalAPIs) + `
- **Critical APIs:** ` + fmt.Sprintf("%d", data.CriticalCount) + `
- **Modules Covered:** ` + fmt.Sprintf("%d", data.ModuleCount) + `

## Top 20 Most Used APIs

| Rank | API | Type | Module | Usage Count | Criticality |
|------|-----|------|--------|-------------|-------------|
`

    for _, api := range data.Top20 {
        md += fmt.Sprintf("| %d | **%s** | %s | %s | %d | %s |\n",
            api.Rank, api.APIName, api.APIType, api.Module,
            api.UsageCount, api.Criticality)
    }

    // Add clusters, patterns, etc.
    // ...

    return md, nil
}
```

---

## MCP Tool Integration

**File:** `internal/mcp/apisurface_tools.go`

```go
func registerAPISurfaceTools(s *Server) {
    // Tool 1: Scrape API surface
    s.RegisterTool("ScrapeAPISurface", schema.Tool{
        Name:        "ScrapeAPISurface",
        Description: "Discover all SAP standard APIs used by custom code",
        InputSchema: schema.Object{
            "package_patterns": schema.Array{
                Description: "Package patterns (e.g., Z*, $Z*)",
                Items:       schema.String{},
            },
            "include_types": schema.Array{
                Description: "API types to include (F, ME, TY, etc.)",
                Items:       schema.String{},
            },
            "max_results": schema.Number{
                Description: "Maximum results to return (0 = all)",
            },
        },
    })

    // Tool 2: Rank APIs
    s.RegisterTool("RankAPIs", schema.Tool{
        Name:        "RankAPIs",
        Description: "Rank and prioritize discovered APIs",
        InputSchema: schema.Object{
            "scraped_data": schema.String{
                Description: "JSON data from ScrapeAPISurface",
            },
            "rank_by": schema.String{
                Description: "Ranking criteria (usage|callers|criticality)",
            },
        },
    })

    // Tool 3: Cluster APIs
    s.RegisterTool("ClusterAPIs", schema.Tool{
        Name:        "ClusterAPIs",
        Description: "Cluster APIs by module, pattern, or usage",
        InputSchema: schema.Object{
            "ranked_data": schema.String{
                Description: "JSON data from RankAPIs",
            },
            "cluster_by": schema.String{
                Description: "Clustering method (module|pattern|usage)",
            },
        },
    })

    // Tool 4: Generate API report
    s.RegisterTool("GenerateAPIReport", schema.Tool{
        Name:        "GenerateAPIReport",
        Description: "Generate comprehensive API surface report",
        InputSchema: schema.Object{
            "data": schema.String{
                Description: "JSON data from analysis",
            },
            "format": schema.String{
                Description: "Output format (html|markdown|json|csv)",
            },
            "include_patterns": schema.Boolean{
                Description: "Include usage pattern analysis",
            },
        },
    })
}
```

---

## Usage Examples

### Example 1: Quick API Surface Scan

```bash
# Scrape top 100 most used APIs
vsp ScrapeAPISurface \
  --package-patterns "Z*" \
  --include-types "F,ME,TY" \
  --max-results 100 \
  > api-surface-raw.json

# Rank them
vsp RankAPIs \
  --scraped-data api-surface-raw.json \
  --rank-by usage \
  > api-surface-ranked.json

# Generate report
vsp GenerateAPIReport \
  --data api-surface-ranked.json \
  --format html \
  --include-patterns true \
  > api-surface-report.html
```

### Example 2: Module-specific Analysis

```bash
# Find all SD module APIs we use
vsp ScrapeAPISurface \
  --package-patterns "Z*" \
  | vsp ClusterAPIs --cluster-by module \
  | jq '.clusters[] | select(.module == "SD")'
```

### Example 3: Find Deprecated API Usage

```bash
# Scrape all APIs
vsp ScrapeAPISurface --package-patterns "Z*" \
  | vsp RankAPIs \
  | jq '.apis[] | select(.is_deprecated == true)'
```

---

## Expected Output

### Top 20 APIs Report (Example)

```markdown
# Top 20 SAP Standard APIs Used

## 1. BAPI_TRANSACTION_COMMIT (Function Module)
- **Module:** BC (Basis Components)
- **Usage Count:** 1,247
- **Used By:** 412 custom objects
- **Criticality:** CRITICAL
- **Description:** Commit BAPI transaction
- **Pattern:** Always used after modifying BAPIs

## 2. /IWBEP/CL_MGW_ABS_DATA (Class)
- **Module:** BC (SAP Gateway)
- **Usage Count:** 892
- **Used By:** 87 custom objects
- **Criticality:** HIGH
- **Description:** Base class for OData service implementation
- **Pattern:** Gateway service development

## 3. BAPI_SALESORDER_CREATEFROMDAT2 (Function Module)
- **Module:** SD (Sales & Distribution)
- **Usage Count:** 634
- **Used By:** 156 custom objects
- **Criticality:** CRITICAL
- **Description:** Create sales order
- **Pattern:** Order creation from external systems

...
```

### Module Clustering Report

```markdown
# API Usage by Module

## SD (Sales & Distribution) - 3,421 usages
- BAPI_SALESORDER_CREATEFROMDAT2 (634 usages)
- BAPI_SALESORDER_CHANGE (421 usages)
- SD_SALES_DOCUMENT_READ (298 usages)
- ...

## MM (Materials Management) - 2,876 usages
- BAPI_MATERIAL_SAVEDATA (512 usages)
- BAPI_GOODSMVT_CREATE (387 usages)
- MMR_STOCK_OVERVIEW (234 usages)
- ...

## FI (Financial Accounting) - 1,987 usages
- BAPI_ACC_DOCUMENT_POST (456 usages)
- BAPI_ACC_DOCUMENT_CHECK (298 usages)
- ...
```

### Usage Patterns Report

```markdown
# Common API Usage Patterns

## Pattern 1: BAPI with Commit
**Frequency:** Found in 412 places
**Description:** BAPI calls followed by BAPI_TRANSACTION_COMMIT

**Example:**
```abap
CALL FUNCTION 'BAPI_SALESORDER_CREATEFROMDAT2'
  EXPORTING
    order_header_in = ls_header
  IMPORTING
    salesdocument   = lv_vbeln
  TABLES
    return          = lt_return.

CALL FUNCTION 'BAPI_TRANSACTION_COMMIT'
  EXPORTING
    wait = 'X'.
```

**Best Practice:** Always commit after modifying BAPIs. Check return table before commit.

## Pattern 2: Authority Check
**Frequency:** Found in 287 places
**Description:** AUTHORITY-CHECK before sensitive operations

...
```

---

## Benefits

### For Architects
- **Dependency mapping** - Understand SAP coupling
- **Upgrade planning** - Know what SAP changes affect us
- **Risk assessment** - Identify critical dependencies
- **Module usage** - See which SAP modules we depend on

### For Developers
- **API documentation** - Document most-used APIs first
- **Pattern library** - Learn from existing patterns
- **Best practices** - See how APIs are commonly used
- **Onboarding** - Help new devs understand the landscape

### For Management
- **Technical debt** - Identify deprecated API usage
- **License optimization** - Know which modules are actually used
- **Vendor lock-in** - Understand SAP dependency level
- **Training priorities** - Focus on most-used modules

---

## Future Enhancements

### Phase 6: Advanced Analytics
- **Trend analysis** - Track API usage over time
- **Change impact** - "If SAP changes this API, what breaks?"
- **Alternative suggestions** - "Use this newer API instead"
- **Security analysis** - Find APIs with security issues

### Phase 7: Integration
- **Documentation generation** - Auto-generate API docs
- **LLM context** - Feed to Claude for better code understanding
- **Graph visualization** - Interactive dependency explorer
- **Alerting** - Notify when deprecated APIs are used

### Phase 8: Automation
- **CI/CD integration** - Block deprecated API usage
- **Scheduled scraping** - Daily/weekly API surface reports
- **Change detection** - Alert when new SAP APIs are introduced
- **Upgrade assistant** - Guide SAP upgrades based on usage

---

## Conclusion

The Standard API Surface Scraper provides unprecedented visibility into how custom code uses SAP standard APIs. This enables:

- ✅ **Data-driven decisions** about SAP dependencies
- ✅ **Prioritized documentation** of most-critical APIs
- ✅ **Pattern discovery** for common usage scenarios
- ✅ **Upgrade planning** with full impact analysis
- ✅ **Knowledge sharing** across development teams

**This tool transforms SAP API usage from mystery to transparency!**

---

**Ready for implementation in vsp!**
