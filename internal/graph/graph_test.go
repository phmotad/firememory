package graph

import (
	"path/filepath"
	"testing"

	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
)

func TestPersistentGraphAddNodeAndNeighbors(t *testing.T) {
	g, err := New(storage.NewFakeStore())
	if err != nil {
		t.Fatalf("new graph: %v", err)
	}

	nodes := []Node{
		{ID: "mem_01", Kind: memory.MemoryKindNote, Scope: memory.DefaultScope, Label: "A"},
		{ID: "mem_02", Kind: memory.MemoryKindFact, Scope: memory.DefaultScope, Label: "B"},
		{ID: "mem_03", Kind: memory.MemoryKindConcept, Scope: "support", Label: "C"},
	}
	for _, node := range nodes {
		if err := g.AddNode(node); err != nil {
			t.Fatalf("add node %s: %v", node.ID, err)
		}
	}

	if err := g.AddEdge(Edge{
		ID:     "edge_01",
		FromID: "mem_01",
		ToID:   "mem_02",
		Type:   memory.RelationTypeAssociated,
	}); err != nil {
		t.Fatalf("add edge_01: %v", err)
	}

	if err := g.AddEdge(Edge{
		ID:     "edge_02",
		FromID: "mem_01",
		ToID:   "mem_03",
		Type:   memory.RelationTypeComplement,
	}); err != nil {
		t.Fatalf("add edge_02: %v", err)
	}

	neighbors, err := g.Neighbors("mem_01")
	if err != nil {
		t.Fatalf("neighbors: %v", err)
	}

	if len(neighbors) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(neighbors))
	}

	if neighbors[0].ID != "mem_02" || neighbors[1].ID != "mem_03" {
		t.Fatalf("expected neighbors [mem_02 mem_03], got [%s %s]", neighbors[0].ID, neighbors[1].ID)
	}
}

func TestPersistentGraphTraverseDepth(t *testing.T) {
	g, err := New(storage.NewFakeStore())
	if err != nil {
		t.Fatalf("new graph: %v", err)
	}

	for _, id := range []string{"mem_01", "mem_02", "mem_03", "mem_04"} {
		if err := g.AddNode(Node{
			ID:    id,
			Kind:  memory.MemoryKindNote,
			Scope: memory.DefaultScope,
		}); err != nil {
			t.Fatalf("add node %s: %v", id, err)
		}
	}

	edges := []Edge{
		{ID: "edge_01", FromID: "mem_01", ToID: "mem_02", Type: memory.RelationTypeAssociated},
		{ID: "edge_02", FromID: "mem_02", ToID: "mem_03", Type: memory.RelationTypeAssociated},
		{ID: "edge_03", FromID: "mem_03", ToID: "mem_04", Type: memory.RelationTypeAssociated},
	}
	for _, edge := range edges {
		if err := g.AddEdge(edge); err != nil {
			t.Fatalf("add edge %s: %v", edge.ID, err)
		}
	}

	related, err := g.Related("mem_01", 2)
	if err != nil {
		t.Fatalf("related: %v", err)
	}

	if len(related) != 2 {
		t.Fatalf("expected 2 related nodes at depth 2, got %d", len(related))
	}

	if related[0].ID != "mem_02" || related[1].ID != "mem_03" {
		t.Fatalf("expected traversal [mem_02 mem_03], got [%s %s]", related[0].ID, related[1].ID)
	}
}

func TestPersistentGraphPersistsNodesAndEdges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	store, err := storage.OpenBboltStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	g, err := New(store)
	if err != nil {
		t.Fatalf("new graph: %v", err)
	}

	if err := g.AddNode(Node{ID: "mem_01", Kind: memory.MemoryKindNote, Scope: memory.DefaultScope}); err != nil {
		t.Fatalf("add node mem_01: %v", err)
	}
	if err := g.AddNode(Node{ID: "mem_02", Kind: memory.MemoryKindFact, Scope: memory.DefaultScope}); err != nil {
		t.Fatalf("add node mem_02: %v", err)
	}
	if err := g.AddEdge(Edge{
		ID:     "edge_01",
		FromID: "mem_01",
		ToID:   "mem_02",
		Type:   memory.RelationTypeReinforce,
		Weight: 0.8,
	}); err != nil {
		t.Fatalf("add edge: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := storage.OpenBboltStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer reopened.Close()

	loaded, err := New(reopened)
	if err != nil {
		t.Fatalf("reload graph: %v", err)
	}

	if loaded.NodeCount() != 2 {
		t.Fatalf("expected 2 nodes after reload, got %d", loaded.NodeCount())
	}

	if loaded.EdgeCount() != 1 {
		t.Fatalf("expected 1 edge after reload, got %d", loaded.EdgeCount())
	}

	neighbors, err := loaded.Neighbors("mem_01")
	if err != nil {
		t.Fatalf("neighbors after reload: %v", err)
	}

	if len(neighbors) != 1 || neighbors[0].ID != "mem_02" {
		t.Fatalf("expected reloaded neighbor mem_02, got %+v", neighbors)
	}
}

func TestPersistentGraphValidation(t *testing.T) {
	g, err := New(storage.NewFakeStore())
	if err != nil {
		t.Fatalf("new graph: %v", err)
	}

	if err := g.AddNode(Node{}); err != ErrNodeIDRequired {
		t.Fatalf("expected ErrNodeIDRequired, got %v", err)
	}

	if err := g.AddNode(Node{ID: "mem_01", Kind: memory.MemoryKindNote, Scope: memory.DefaultScope}); err != nil {
		t.Fatalf("add node mem_01: %v", err)
	}

	err = g.AddEdge(Edge{
		ID:     "edge_01",
		FromID: "mem_01",
		ToID:   "missing",
		Type:   memory.RelationTypeAssociated,
	})
	if err != ErrNodeNotFound {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}

	_, err = g.TraverseDepth("mem_01", -1)
	if err != ErrDepthNegative {
		t.Fatalf("expected ErrDepthNegative, got %v", err)
	}
}
