package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/oisee/vibing-steampunk/pkg/cache"
)

func TestMemoryCache_BasicOperations(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	// Test PutNode and GetNode
	node := &cache.Node{
		ID:          "ME.ZCL_TEST\\ME:METHOD",
		ObjectType:  "CLAS",
		ObjectName:  "ZCL_TEST",
		Package:     "$TMP",
		SourceHash:  "abc123",
		Valid:       true,
	}

	err := c.PutNode(ctx, node)
	if err != nil {
		t.Fatalf("PutNode failed: %v", err)
	}

	retrieved, err := c.GetNode(ctx, node.ID)
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if retrieved.ID != node.ID {
		t.Errorf("Expected ID %s, got %s", node.ID, retrieved.ID)
	}

	if retrieved.ObjectType != node.ObjectType {
		t.Errorf("Expected ObjectType %s, got %s", node.ObjectType, retrieved.ObjectType)
	}
}

func TestMemoryCache_NotFound(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	_, err := c.GetNode(ctx, "nonexistent")
	if err != cache.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryCache_Invalidation(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	node := &cache.Node{
		ID:         "TEST_NODE",
		ObjectType: "CLAS",
		ObjectName: "ZCL_TEST",
		Valid:      true,
	}

	c.PutNode(ctx, node)

	// Invalidate
	err := c.InvalidateNode(ctx, node.ID, "test invalidation")
	if err != nil {
		t.Fatalf("InvalidateNode failed: %v", err)
	}

	// Should return ErrInvalidated
	_, err = c.GetNode(ctx, node.ID)
	if err != cache.ErrInvalidated {
		t.Errorf("Expected ErrInvalidated, got %v", err)
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	ctx := context.Background()

	config := cache.DefaultConfig()
	config.InvalidationPolicy.UseTTL = true
	config.InvalidationPolicy.TTL = 1 * time.Millisecond

	c := cache.NewMemoryCache(config)

	node := &cache.Node{
		ID:         "TEST_NODE",
		ObjectType: "CLAS",
		ObjectName: "ZCL_TEST",
		Valid:      true,
		CachedAt:   time.Now(),
	}

	c.PutNode(ctx, node)

	// Wait for TTL to expire
	time.Sleep(10 * time.Millisecond)

	// Should return ErrExpired
	_, err := c.GetNode(ctx, node.ID)
	if err != cache.ErrExpired {
		t.Errorf("Expected ErrExpired, got %v", err)
	}
}

func TestMemoryCache_Edges(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	edge := &cache.Edge{
		FromID:   "NODE_A",
		ToID:     "NODE_B",
		EdgeType: "CALLS",
		Source:   "WBCROSSGT",
		Valid:    true,
	}

	err := c.PutEdge(ctx, edge)
	if err != nil {
		t.Fatalf("PutEdge failed: %v", err)
	}

	// Get edges from NODE_A
	edges, err := c.GetEdgesFrom(ctx, "NODE_A")
	if err != nil {
		t.Fatalf("GetEdgesFrom failed: %v", err)
	}

	if len(edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0].ToID != "NODE_B" {
		t.Errorf("Expected ToID NODE_B, got %s", edges[0].ToID)
	}

	// Get edges to NODE_B
	edges, err = c.GetEdgesTo(ctx, "NODE_B")
	if err != nil {
		t.Fatalf("GetEdgesTo failed: %v", err)
	}

	if len(edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(edges))
	}

	if edges[0].FromID != "NODE_A" {
		t.Errorf("Expected FromID NODE_A, got %s", edges[0].FromID)
	}
}

func TestMemoryCache_GetNodesByPackage(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	nodes := []*cache.Node{
		{ID: "NODE1", ObjectType: "CLAS", ObjectName: "ZCL_A", Package: "$ZRAY", Valid: true},
		{ID: "NODE2", ObjectType: "CLAS", ObjectName: "ZCL_B", Package: "$ZRAY", Valid: true},
		{ID: "NODE3", ObjectType: "PROG", ObjectName: "ZPROG", Package: "$TMP", Valid: true},
	}

	for _, node := range nodes {
		c.PutNode(ctx, node)
	}

	// Get nodes in $ZRAY package
	packageNodes, err := c.GetNodesByPackage(ctx, "$ZRAY")
	if err != nil {
		t.Fatalf("GetNodesByPackage failed: %v", err)
	}

	if len(packageNodes) != 2 {
		t.Errorf("Expected 2 nodes in $ZRAY, got %d", len(packageNodes))
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	// Add some data
	c.PutNode(ctx, &cache.Node{ID: "N1", ObjectType: "CLAS", ObjectName: "ZCL_A", Valid: true})
	c.PutNode(ctx, &cache.Node{ID: "N2", ObjectType: "CLAS", ObjectName: "ZCL_B", Valid: true})
	c.PutEdge(ctx, &cache.Edge{FromID: "N1", ToID: "N2", EdgeType: "CALLS", Source: "WBCROSSGT", Valid: true})

	stats, err := c.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.NodeCount != 2 {
		t.Errorf("Expected 2 nodes, got %d", stats.NodeCount)
	}

	if stats.ValidNodeCount != 2 {
		t.Errorf("Expected 2 valid nodes, got %d", stats.ValidNodeCount)
	}

	if stats.EdgeCount != 1 {
		t.Errorf("Expected 1 edge, got %d", stats.EdgeCount)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	// Add data
	c.PutNode(ctx, &cache.Node{ID: "N1", ObjectType: "CLAS", ObjectName: "ZCL_A", Valid: true})
	c.PutEdge(ctx, &cache.Edge{FromID: "N1", ToID: "N2", EdgeType: "CALLS", Source: "WBCROSSGT", Valid: true})

	// Clear
	err := c.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify empty
	stats, _ := c.Stats(ctx)
	if stats.NodeCount != 0 || stats.EdgeCount != 0 {
		t.Errorf("Cache not empty after Clear: %d nodes, %d edges", stats.NodeCount, stats.EdgeCount)
	}
}

func BenchmarkMemoryCache_PutNode(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	node := &cache.Node{
		ObjectType: "CLAS",
		ObjectName: "ZCL_TEST",
		Package:    "$TMP",
		Valid:      true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node.ID = string(rune(i))
		c.PutNode(ctx, node)
	}
}

func BenchmarkMemoryCache_GetNode(b *testing.B) {
	ctx := context.Background()
	c := cache.NewMemoryCache(cache.DefaultConfig())

	// Prepopulate
	for i := 0; i < 1000; i++ {
		c.PutNode(ctx, &cache.Node{
			ID:         string(rune(i)),
			ObjectType: "CLAS",
			ObjectName: "ZCL_TEST",
			Valid:      true,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.GetNode(ctx, string(rune(i%1000)))
	}
}
