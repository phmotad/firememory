package dedup

import (
	"errors"
	"strings"
	"testing"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/vector"
)

func TestNormalizeText(t *testing.T) {
	got := NormalizeText("  Cliente   JOAO \n usa Firebird 2.5  ")
	want := "cliente joao usa firebird 2.5"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestHashNormalizedIsStable(t *testing.T) {
	first := HashNormalized("cliente joao usa firebird")
	second := HashNormalized("cliente joao usa firebird")
	if first != second {
		t.Fatal("expected deterministic hash")
	}

	if len(first) != 64 {
		t.Fatalf("expected 64-char sha256 hex, got %d", len(first))
	}
}

func TestDetectorReturnsExactReinforce(t *testing.T) {
	idx := NewInMemoryHashIndex()
	hash := HashNormalized("cliente joao usa firebird 2.5")
	idx.Set(hash, "mem_01")

	detector, err := NewDetector(Config{})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(Input{
		Content:   " Cliente Joao usa Firebird 2.5 ",
		HashIndex: idx,
	})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Action != memory.DedupActionReinforce {
		t.Fatalf("expected reinforce, got %q", result.Action)
	}

	if result.MatchType != MatchTypeExact {
		t.Fatalf("expected exact match, got %q", result.MatchType)
	}

	if result.MatchedMemoryID != "mem_01" {
		t.Fatalf("expected mem_01, got %q", result.MatchedMemoryID)
	}
}

func TestDetectorReturnsVectorReinforce(t *testing.T) {
	index, err := vector.NewLinearVectorIndex(3)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	if err := index.Add(vector.Entry{
		ID:     "mem_02",
		Vector: embedder.Vector{1, 0, 0},
		Scope:  memory.DefaultScope,
		Kind:   memory.MemoryKindNote,
	}); err != nil {
		t.Fatalf("add entry: %v", err)
	}

	detector, err := NewDetector(Config{VectorThreshold: 0.8})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(Input{
		Content: "Problema fiscal na NF-e",
		Scope:   memory.DefaultScope,
		Kind:    memory.MemoryKindNote,
		Vector:  embedder.Vector{1, 0, 0},
		Index:   index,
	})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Action != memory.DedupActionReinforce {
		t.Fatalf("expected reinforce, got %q", result.Action)
	}

	if result.MatchType != MatchTypeVector {
		t.Fatalf("expected vector match, got %q", result.MatchType)
	}

	if result.MatchedMemoryID != "mem_02" {
		t.Fatalf("expected mem_02, got %q", result.MatchedMemoryID)
	}
}

func TestDetectorReturnsCreateNewBelowThreshold(t *testing.T) {
	index, err := vector.NewLinearVectorIndex(2)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}

	if err := index.Add(vector.Entry{
		ID:     "mem_03",
		Vector: embedder.Vector{1, 0},
		Scope:  memory.DefaultScope,
		Kind:   memory.MemoryKindNote,
	}); err != nil {
		t.Fatalf("add entry: %v", err)
	}

	detector, err := NewDetector(Config{VectorThreshold: 0.95})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(Input{
		Content: "Novo relato fiscal",
		Scope:   memory.DefaultScope,
		Kind:    memory.MemoryKindNote,
		Vector:  embedder.Vector{0, 1},
		Index:   index,
	})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Action != memory.DedupActionCreateNew {
		t.Fatalf("expected create_new, got %q", result.Action)
	}

	if result.MatchType != MatchTypeNone {
		t.Fatalf("expected no match, got %q", result.MatchType)
	}
}

func TestDetectorSkipsVectorWithoutIndex(t *testing.T) {
	detector, err := NewDetector(Config{})
	if err != nil {
		t.Fatalf("new detector: %v", err)
	}

	result, err := detector.Detect(Input{
		Content: "Cliente Joao usa Firebird",
	})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if result.Action != memory.DedupActionCreateNew {
		t.Fatalf("expected create_new, got %q", result.Action)
	}

	trace := strings.Join(result.Trace, " | ")
	if !strings.Contains(trace, "vector dedup skipped") {
		t.Fatalf("expected trace to mention vector skip, got %q", trace)
	}
}

func TestConfigValidation(t *testing.T) {
	_, err := NewDetector(Config{VectorThreshold: 1.5})
	if !errors.Is(err, ErrInvalidVectorThreshold) {
		t.Fatalf("expected ErrInvalidVectorThreshold, got %v", err)
	}
}
