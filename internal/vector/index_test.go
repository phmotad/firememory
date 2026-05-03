package vector

import (
	"errors"
	"math"
	"testing"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/memory"
)

func TestCosineSimilarity(t *testing.T) {
	same := CosineSimilarity(embedder.Vector{1, 0}, embedder.Vector{1, 0})
	if math.Abs(same-1.0) > 0.0001 {
		t.Fatalf("expected cosine 1.0, got %f", same)
	}

	orthogonal := CosineSimilarity(embedder.Vector{1, 0}, embedder.Vector{0, 1})
	if math.Abs(orthogonal-0.0) > 0.0001 {
		t.Fatalf("expected cosine 0.0, got %f", orthogonal)
	}
}

func TestLinearVectorIndexAddSearchRemove(t *testing.T) {
	idx, err := NewLinearVectorIndex(3)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	if err := idx.Add(Entry{
		ID:     "mem_01",
		Vector: embedder.Vector{1, 0, 0},
		Scope:  memory.DefaultScope,
		Kind:   memory.MemoryKindNote,
	}); err != nil {
		t.Fatalf("add mem_01: %v", err)
	}

	if err := idx.Add(Entry{
		ID:     "mem_02",
		Vector: embedder.Vector{0, 1, 0},
		Scope:  memory.DefaultScope,
		Kind:   memory.MemoryKindFact,
	}); err != nil {
		t.Fatalf("add mem_02: %v", err)
	}

	results, err := idx.Search(SearchInput{
		Vector: embedder.Vector{1, 0, 0},
		TopK:   2,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].ID != "mem_01" {
		t.Fatalf("expected mem_01 as best match, got %q", results[0].ID)
	}

	if err := idx.Remove("mem_01"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	results, err = idx.Search(SearchInput{
		Vector: embedder.Vector{1, 0, 0},
		TopK:   2,
	})
	if err != nil {
		t.Fatalf("search after remove: %v", err)
	}

	if len(results) != 1 || results[0].ID != "mem_02" {
		t.Fatalf("expected only mem_02 after remove, got %+v", results)
	}
}

func TestLinearVectorIndexFiltersByScopeAndKind(t *testing.T) {
	idx, err := NewLinearVectorIndex(2)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	entries := []Entry{
		{ID: "mem_01", Vector: embedder.Vector{1, 0}, Scope: "support", Kind: memory.MemoryKindNote},
		{ID: "mem_02", Vector: embedder.Vector{1, 0}, Scope: "billing", Kind: memory.MemoryKindNote},
		{ID: "mem_03", Vector: embedder.Vector{1, 0}, Scope: "support", Kind: memory.MemoryKindFact},
	}

	for _, entry := range entries {
		if err := idx.Add(entry); err != nil {
			t.Fatalf("add %s: %v", entry.ID, err)
		}
	}

	results, err := idx.Search(SearchInput{
		Vector: embedder.Vector{1, 0},
		Scope:  "support",
		Kinds:  []memory.MemoryKind{memory.MemoryKindFact},
		TopK:   5,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].ID != "mem_03" {
		t.Fatalf("expected mem_03, got %q", results[0].ID)
	}
}

func TestLinearVectorIndexTopKAndStableOrdering(t *testing.T) {
	idx, err := NewLinearVectorIndex(2)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	for _, id := range []string{"mem_b", "mem_a", "mem_c"} {
		if err := idx.Add(Entry{
			ID:     id,
			Vector: embedder.Vector{1, 0},
			Scope:  memory.DefaultScope,
			Kind:   memory.MemoryKindNote,
		}); err != nil {
			t.Fatalf("add %s: %v", id, err)
		}
	}

	results, err := idx.Search(SearchInput{
		Vector: embedder.Vector{1, 0},
		TopK:   2,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected top 2 results, got %d", len(results))
	}

	if results[0].ID != "mem_a" || results[1].ID != "mem_b" {
		t.Fatalf("expected stable ordering by id on tie, got [%s %s]", results[0].ID, results[1].ID)
	}
}

func TestLinearVectorIndexValidatesDimensions(t *testing.T) {
	idx, err := NewLinearVectorIndex(2)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	err = idx.Add(Entry{
		ID:     "mem_01",
		Vector: embedder.Vector{1, 0, 0},
		Scope:  memory.DefaultScope,
		Kind:   memory.MemoryKindNote,
	})
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Fatalf("expected ErrDimensionMismatch on add, got %v", err)
	}

	_, err = idx.Search(SearchInput{
		Vector: embedder.Vector{1, 0, 0},
		TopK:   1,
	})
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Fatalf("expected ErrDimensionMismatch on search, got %v", err)
	}
}

func TestSearchInputDefaultsAndMinScore(t *testing.T) {
	idx, err := NewLinearVectorIndex(2)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	if err := idx.Add(Entry{
		ID:     "mem_01",
		Vector: embedder.Vector{1, 0},
		Kind:   memory.MemoryKindNote,
	}); err != nil {
		t.Fatalf("add mem_01: %v", err)
	}

	input := SearchInput{
		Vector:   embedder.Vector{1, 0},
		MinScore: 0.9,
	}

	results, err := idx.Search(input)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result above threshold, got %d", len(results))
	}

	bad := SearchInput{
		Vector:   embedder.Vector{1, 0},
		TopK:     -1,
		MinScore: 2,
	}
	bad.Normalize()
	if err := bad.Validate(idx.Dimension()); !errors.Is(err, ErrInvalidTopK) {
		t.Fatalf("expected ErrInvalidTopK first, got %v", err)
	}
}
