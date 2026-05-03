package engine

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/graph"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
	"github.com/phmotad/firememory/internal/vector"
)

func TestOpenUsesManifestDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{Name: "agent"})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{Path: path})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}
	defer engine.Close()

	if engine.Manifest().Name != "agent" {
		t.Fatalf("expected manifest name %q, got %q", "agent", engine.Manifest().Name)
	}

	if engine.Embedder().Name() != embedder.DeterministicModel {
		t.Fatalf("expected deterministic embedder, got %q", engine.Embedder().Name())
	}

	if engine.VectorIndex().Dimension() != brainfile.DefaultEmbedDim {
		t.Fatalf("expected vector dimension %d, got %d", brainfile.DefaultEmbedDim, engine.VectorIndex().Dimension())
	}

	if engine.Graph() == nil {
		t.Fatal("expected graph to be initialized")
	}
}

func TestOpenRebuildsVectorIndexAndGraph(t *testing.T) {
	path := filepath.Join(t.TempDir(), "support.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{
		Name:         "support",
		EmbeddingDim: 3,
	})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}

	payload, err := json.Marshal(storedVectorRecord{
		Vector: embedder.Vector{1, 0, 0},
		Scope:  memory.DefaultScope,
		Kind:   memory.MemoryKindNote,
	})
	if err != nil {
		t.Fatalf("marshal vector record: %v", err)
	}

	if err := handle.Store().Put(vectorsNamespace, "mem_01", payload); err != nil {
		t.Fatalf("persist vector: %v", err)
	}

	g, err := graph.New(handle.Store())
	if err != nil {
		t.Fatalf("new graph: %v", err)
	}

	if err := g.AddNode(graph.Node{ID: "mem_01", Kind: memory.MemoryKindNote, Scope: memory.DefaultScope}); err != nil {
		t.Fatalf("add node mem_01: %v", err)
	}

	if err := g.AddNode(graph.Node{ID: "mem_02", Kind: memory.MemoryKindFact, Scope: memory.DefaultScope}); err != nil {
		t.Fatalf("add node mem_02: %v", err)
	}

	if err := g.AddEdge(graph.Edge{
		ID:     "edge_01",
		FromID: "mem_01",
		ToID:   "mem_02",
		Type:   memory.RelationTypeAssociated,
	}); err != nil {
		t.Fatalf("add edge: %v", err)
	}

	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{Path: path})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}
	defer engine.Close()

	hits, err := engine.VectorIndex().Search(vector.SearchInput{
		Vector: embedder.Vector{1, 0, 0},
		TopK:   1,
	})
	if err != nil {
		t.Fatalf("search rebuilt vector index: %v", err)
	}

	if len(hits) != 1 || hits[0].ID != "mem_01" {
		t.Fatalf("expected rebuilt vector hit mem_01, got %+v", hits)
	}

	neighbors, err := engine.Graph().Neighbors("mem_01")
	if err != nil {
		t.Fatalf("graph neighbors: %v", err)
	}

	if len(neighbors) != 1 || neighbors[0].ID != "mem_02" {
		t.Fatalf("expected graph neighbor mem_02, got %+v", neighbors)
	}
}

func TestOpenValidatesInjectedDependencies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "billing.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{
		Name:         "billing",
		EmbeddingDim: 4,
	})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	wrongEmbedder, err := embedder.NewDeterministicEmbedder(3)
	if err != nil {
		t.Fatalf("new embedder: %v", err)
	}

	_, err = Open(Options{
		Path:     path,
		Embedder: wrongEmbedder,
	})
	if !errors.Is(err, ErrEmbedderDimensionMismatch) {
		t.Fatalf("expected ErrEmbedderDimensionMismatch, got %v", err)
	}

	index, err := vector.NewLinearVectorIndex(3)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	_, err = Open(Options{
		Path:        path,
		VectorIndex: index,
	})
	if !errors.Is(err, ErrVectorIndexDimensionMismatch) {
		t.Fatalf("expected ErrVectorIndexDimensionMismatch, got %v", err)
	}
}

func TestCloseClosesUnderlyingStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "close-test.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{Name: "close-test"})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{Path: path})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}

	if err := engine.Close(); err != nil {
		t.Fatalf("close engine: %v", err)
	}

	if _, err := engine.Store().Snapshot(); !errors.Is(err, storage.ErrStoreClosed) {
		t.Fatalf("expected closed store error, got %v", err)
	}
}
