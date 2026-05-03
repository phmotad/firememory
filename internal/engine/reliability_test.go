package engine

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/embedder"
)

func TestRememberWithLargeBrainfile(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "large.fbrain"), nil, 384)
	defer engine.Close()

	for i := range 150 {
		content := fmt.Sprintf("customer %03d reported fiscal issue after update %d.%d %s", i, i%7, i%5, strings.Repeat("payload ", 64))
		if _, err := engine.Remember(RememberInput{
			BrainPath: engine.Path(),
			Content:   content,
		}); err != nil {
			t.Fatalf("remember %d: %v", i, err)
		}
	}

	result, err := engine.Recall(RecallInput{
		BrainPath: engine.Path(),
		Query:     "fiscal issue payload",
		TopK:      10,
	})
	if err != nil {
		t.Fatalf("recall large brainfile: %v", err)
	}
	if len(result.Hits) == 0 {
		t.Fatal("expected hits from large brainfile")
	}

	snapshot, err := engine.Store().Snapshot()
	if err != nil {
		t.Fatalf("snapshot large brainfile: %v", err)
	}
	if len(snapshot.Namespaces["memories"]) < 100 {
		t.Fatalf("memory count = %d, want at least 100", len(snapshot.Namespaces["memories"]))
	}
}

func TestRememberFailureInjectionOnEmbedderError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embedder-failure.fbrain")
	handle, err := brainfile.Create(path, brainfile.CreateOptions{
		Name:           "test",
		EmbeddingModel: brainfile.DefaultEmbedder,
		EmbeddingDim:   4,
	})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{
		Path: path,
		Embedder: failureEmbedder{
			dimension: 4,
			err:       errors.New("synthetic embedder failure"),
		},
	})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}
	defer engine.Close()

	_, err = engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "this write should fail",
	})
	if err == nil || !strings.Contains(err.Error(), "synthetic embedder failure") {
		t.Fatalf("expected embedder failure, got %v", err)
	}

	records, err := engine.Store().List(memoriesNamespace, "", 0)
	if err != nil {
		t.Fatalf("list memories after failed remember: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected no persisted memories after failed remember, got %d", len(records))
	}
}

func BenchmarkRemember(b *testing.B) {
	path := filepath.Join(b.TempDir(), "benchmark-remember.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{
		Name:           "bench",
		EmbeddingModel: brainfile.DefaultEmbedder,
		EmbeddingDim:   384,
	})
	if err != nil {
		b.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		b.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{Path: path})
	if err != nil {
		b.Fatalf("open engine: %v", err)
	}
	defer engine.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Remember(RememberInput{
			BrainPath: engine.Path(),
			Content:   fmt.Sprintf("benchmark remember content %d", i),
		})
		if err != nil {
			b.Fatalf("remember %d: %v", i, err)
		}
	}
}

func BenchmarkRecall(b *testing.B) {
	path := filepath.Join(b.TempDir(), "benchmark-recall.fbrain")

	handle, err := brainfile.Create(path, brainfile.CreateOptions{
		Name:           "bench",
		EmbeddingModel: brainfile.DefaultEmbedder,
		EmbeddingDim:   384,
	})
	if err != nil {
		b.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		b.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{Path: path})
	if err != nil {
		b.Fatalf("open engine: %v", err)
	}
	defer engine.Close()

	for i := range 100 {
		_, err := engine.Remember(RememberInput{
			BrainPath: engine.Path(),
			Content:   fmt.Sprintf("benchmark recall content %d fiscal issue", i),
		})
		if err != nil {
			b.Fatalf("seed remember %d: %v", i, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Recall(RecallInput{
			BrainPath: engine.Path(),
			Query:     "fiscal issue",
			TopK:      5,
		})
		if err != nil {
			b.Fatalf("recall %d: %v", i, err)
		}
	}
}

type failureEmbedder struct {
	dimension int
	err       error
}

func (e failureEmbedder) Name() string {
	return embedder.DeterministicModel
}

func (e failureEmbedder) Dimension() int {
	return e.dimension
}

func (e failureEmbedder) Embed(context.Context, string) (embedder.Vector, error) {
	return nil, e.err
}
