package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/memory"
)

func TestRememberCreatesNewMemoryAndPersistsArtifacts(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "agent.fbrain"), nil, 384)
	defer engine.Close()

	result, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5",
		Metadata: map[string]string{
			"source": "ticket",
		},
	})
	if err != nil {
		t.Fatalf("remember: %v", err)
	}

	if result.DedupAction != memory.DedupActionCreateNew {
		t.Fatalf("expected create_new, got %q", result.DedupAction)
	}

	if result.Memory.Status != memory.MemoryStatusPendingSync {
		t.Fatalf("expected pending_sync, got %q", result.Memory.Status)
	}

	if result.Memory.Hash == "" || result.Memory.NormalizedContent == "" {
		t.Fatal("expected hash and normalized content to be set")
	}

	if engine.VectorIndex().Len() != 1 {
		t.Fatalf("expected vector index len 1, got %d", engine.VectorIndex().Len())
	}

	if _, err := engine.Store().Get(memoriesNamespace, result.Memory.ID); err != nil {
		t.Fatalf("expected persisted memory, got %v", err)
	}

	if _, err := engine.Store().Get(vectorsNamespace, result.Memory.ID); err != nil {
		t.Fatalf("expected persisted vector, got %v", err)
	}

	if _, err := engine.Store().Get(hashIndexNamespace, result.Memory.Hash); err != nil {
		t.Fatalf("expected persisted hash index, got %v", err)
	}

	if _, err := engine.Store().Get(syncQueueNamespace, result.Memory.ID); err != nil {
		t.Fatalf("expected sync queue entry, got %v", err)
	}
}

func TestRememberExactDuplicateReinforcesExistingMemory(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "exact.fbrain"), nil, 384)
	defer engine.Close()

	first, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5",
	})
	if err != nil {
		t.Fatalf("first remember: %v", err)
	}

	second, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   " cliente  joao usa firebird 2.5 ",
	})
	if err != nil {
		t.Fatalf("second remember: %v", err)
	}

	if second.DedupAction != memory.DedupActionReinforce {
		t.Fatalf("expected reinforce, got %q", second.DedupAction)
	}

	if second.ReinforcedMemoryID != first.Memory.ID {
		t.Fatalf("expected reinforce id %q, got %q", first.Memory.ID, second.ReinforcedMemoryID)
	}

	if engine.VectorIndex().Len() != 1 {
		t.Fatalf("expected vector index len 1 after dedup, got %d", engine.VectorIndex().Len())
	}

	if second.Memory.Metadata["reinforced_count"] != "1" {
		t.Fatalf("expected reinforced_count 1, got %q", second.Memory.Metadata["reinforced_count"])
	}
}

func TestRememberVectorDuplicateReinforcesExistingMemory(t *testing.T) {
	custom := &recordingEmbedder{
		dimension: 4,
		vectors: map[string]embedder.Vector{
			"Cliente A teve erro fiscal":             {1, 0, 0, 0},
			"Joao relatou novamente problema fiscal": {1, 0, 0, 0},
		},
	}

	engine := openTestEngine(t, filepath.Join(t.TempDir(), "vector.fbrain"), custom, 4)
	defer engine.Close()

	first, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente A teve erro fiscal",
	})
	if err != nil {
		t.Fatalf("first remember: %v", err)
	}

	second, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Joao relatou novamente problema fiscal",
	})
	if err != nil {
		t.Fatalf("second remember: %v", err)
	}

	if second.DedupAction != memory.DedupActionReinforce {
		t.Fatalf("expected reinforce, got %q", second.DedupAction)
	}

	if second.ReinforcedMemoryID != first.Memory.ID {
		t.Fatalf("expected vector reinforce id %q, got %q", first.Memory.ID, second.ReinforcedMemoryID)
	}
}

func TestRememberAfterReopenUsesPersistedHashIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reopen.fbrain")

	engine := openTestEngine(t, path, nil, 384)
	first, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5",
	})
	if err != nil {
		t.Fatalf("first remember: %v", err)
	}

	if err := engine.Close(); err != nil {
		t.Fatalf("close engine: %v", err)
	}

	reopened, err := Open(Options{Path: path})
	if err != nil {
		t.Fatalf("reopen engine: %v", err)
	}
	defer reopened.Close()

	second, err := reopened.Remember(RememberInput{
		BrainPath: reopened.Path(),
		Content:   "Cliente Joao usa Firebird 2.5",
	})
	if err != nil {
		t.Fatalf("second remember: %v", err)
	}

	if second.ReinforcedMemoryID != first.Memory.ID {
		t.Fatalf("expected reopened reinforce id %q, got %q", first.Memory.ID, second.ReinforcedMemoryID)
	}
}

func openTestEngine(t *testing.T, path string, custom embedder.Embedder, embeddingDim int) *Base {
	t.Helper()

	handle, err := brainfile.Create(path, brainfile.CreateOptions{
		Name:           "test",
		EmbeddingModel: brainfile.DefaultEmbedder,
		EmbeddingDim:   embeddingDim,
	})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}

	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{
		Path:     path,
		Embedder: custom,
	})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}

	return engine
}

type recordingEmbedder struct {
	dimension int
	vectors   map[string]embedder.Vector
}

func (e *recordingEmbedder) Name() string {
	return embedder.DeterministicModel
}

func (e *recordingEmbedder) Dimension() int {
	return e.dimension
}

func (e *recordingEmbedder) Embed(_ context.Context, text string) (embedder.Vector, error) {
	if vector, ok := e.vectors[text]; ok {
		out := make(embedder.Vector, len(vector))
		copy(out, vector)
		return out, nil
	}

	out := make(embedder.Vector, e.dimension)
	if e.dimension > 0 {
		out[0] = 1
	}
	return out, nil
}
