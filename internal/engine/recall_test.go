package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/memory"
)

func TestRecallReturnsHybridRankedHits(t *testing.T) {
	custom := &recordingEmbedder{
		dimension: 4,
		vectors: map[string]embedder.Vector{
			"Cliente Joao teve erro fiscal na NF-e apos atualizacao": {1, 0, 0, 0},
			"Cliente Joao usa Firebird 2.5 em servidor legado":       {0, 1, 0, 0},
			"erro fiscal NF-e": {1, 0, 0, 0},
		},
	}

	engine := openTestEngine(t, filepath.Join(t.TempDir(), "recall.fbrain"), custom, 4)
	defer engine.Close()

	_, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao teve erro fiscal na NF-e apos atualizacao",
	})
	if err != nil {
		t.Fatalf("remember 1: %v", err)
	}

	_, err = engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5 em servidor legado",
	})
	if err != nil {
		t.Fatalf("remember 2: %v", err)
	}

	result, err := engine.Recall(RecallInput{
		BrainPath:    engine.Path(),
		Query:        "erro fiscal NF-e",
		TopK:         2,
		IncludeTrace: true,
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}

	if len(result.Hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(result.Hits))
	}

	if result.Hits[0].Memory.Content != "Cliente Joao teve erro fiscal na NF-e apos atualizacao" {
		t.Fatalf("expected fiscal memory first, got %q", result.Hits[0].Memory.Content)
	}

	reasons := strings.Join(result.Hits[0].Reasons, " | ")
	if !strings.Contains(reasons, "vector similarity") {
		t.Fatalf("expected vector reason, got %q", reasons)
	}

	if len(result.Trace) == 0 {
		t.Fatal("expected recall trace when IncludeTrace is true")
	}
}

func TestRecallRespectsScopeFilter(t *testing.T) {
	custom := &recordingEmbedder{
		dimension: 3,
		vectors: map[string]embedder.Vector{
			"erro fiscal no cliente Joao": {1, 0, 0},
			"erro fiscal no cliente Ana":  {1, 0, 0},
			"erro fiscal":                 {1, 0, 0},
		},
	}

	engine := openTestEngine(t, filepath.Join(t.TempDir(), "scope.fbrain"), custom, 3)
	defer engine.Close()

	_, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "erro fiscal no cliente Joao",
		Scope:     "joao",
	})
	if err != nil {
		t.Fatalf("remember joao: %v", err)
	}

	_, err = engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "erro fiscal no cliente Ana",
		Scope:     "ana",
	})
	if err != nil {
		t.Fatalf("remember ana: %v", err)
	}

	result, err := engine.Recall(RecallInput{
		BrainPath: engine.Path(),
		Query:     "erro fiscal",
		Scope:     "joao",
		TopK:      5,
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}

	if len(result.Hits) != 1 {
		t.Fatalf("expected 1 hit for scope filter, got %d", len(result.Hits))
	}

	if result.Hits[0].Memory.Scope != "joao" {
		t.Fatalf("expected joao scope, got %q", result.Hits[0].Memory.Scope)
	}
}

func TestRecallWithoutTraceOmitsTrace(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "no-trace.fbrain"), nil, 384)
	defer engine.Close()

	_, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5",
	})
	if err != nil {
		t.Fatalf("remember: %v", err)
	}

	result, err := engine.Recall(RecallInput{
		BrainPath: engine.Path(),
		Query:     "Firebird 2.5",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}

	if len(result.Trace) != 0 {
		t.Fatalf("expected empty trace, got %v", result.Trace)
	}

	if len(result.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(result.Hits))
	}
}

func TestRecallAfterReopenUsesPersistedVectors(t *testing.T) {
	custom := &recordingEmbedder{
		dimension: 4,
		vectors: map[string]embedder.Vector{
			"Cliente Joao teve erro fiscal": {1, 0, 0, 0},
			"erro fiscal":                   {1, 0, 0, 0},
		},
	}

	path := filepath.Join(t.TempDir(), "reopen-recall.fbrain")
	engine := openTestEngine(t, path, custom, 4)

	created, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao teve erro fiscal",
	})
	if err != nil {
		t.Fatalf("remember: %v", err)
	}

	if err := engine.Close(); err != nil {
		t.Fatalf("close engine: %v", err)
	}

	reopened, err := Open(Options{
		Path:     path,
		Embedder: custom,
	})
	if err != nil {
		t.Fatalf("reopen engine: %v", err)
	}
	defer reopened.Close()

	result, err := reopened.Recall(RecallInput{
		BrainPath: reopened.Path(),
		Query:     "erro fiscal",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("recall after reopen: %v", err)
	}

	if len(result.Hits) != 1 {
		t.Fatalf("expected 1 hit after reopen, got %d", len(result.Hits))
	}

	if result.Hits[0].Memory.ID != created.Memory.ID {
		t.Fatalf("expected hit %q, got %q", created.Memory.ID, result.Hits[0].Memory.ID)
	}
}

func TestRecallInputNormalizeDefaultsScope(t *testing.T) {
	in := RecallInput{
		BrainPath: "agent.fbrain",
		Query:     "erro fiscal",
	}
	in.Normalize()

	if in.Scope != memory.DefaultScope {
		t.Fatalf("expected default scope %q, got %q", memory.DefaultScope, in.Scope)
	}
}
