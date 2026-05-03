package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/phmotad/firememory/internal/embedder"
)

func TestContextBuildsExpandedContext(t *testing.T) {
	custom := &recordingEmbedder{
		dimension: 4,
		vectors: map[string]embedder.Vector{
			"Cliente Joao teve erro fiscal na NF-e apos atualizacao 3.2":       {1, 0, 0, 0},
			"Joao relatou novamente problema fiscal na NF-e apos a versao 3.2": {1, 0, 0, 0},
			"responder Joao sobre erro fiscal apos atualizacao":                {1, 0, 0, 0},
		},
	}

	engine := openTestEngine(t, filepath.Join(t.TempDir(), "context.fbrain"), custom, 4)
	defer engine.Close()

	_, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao teve erro fiscal na NF-e apos atualizacao 3.2",
	})
	if err != nil {
		t.Fatalf("remember 1: %v", err)
	}

	_, err = engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Joao relatou novamente problema fiscal na NF-e apos a versao 3.2",
	})
	if err != nil {
		t.Fatalf("remember 2: %v", err)
	}

	if _, err := engine.Sync(SyncInput{BrainPath: engine.Path()}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	result, err := engine.Context(ContextInput{
		BrainPath:    engine.Path(),
		Query:        "responder Joao sobre erro fiscal apos atualizacao",
		TopK:         2,
		BudgetTokens: 200,
		IncludeGraph: true,
		IncludeTrace: true,
	})
	if err != nil {
		t.Fatalf("context: %v", err)
	}

	if len(result.Memories) == 0 {
		t.Fatal("expected memories in context result")
	}

	if len(result.Entities) == 0 {
		t.Fatal("expected entities in context result")
	}

	if len(result.Relations) == 0 {
		t.Fatal("expected relations in context result")
	}

	if result.EstimatedTokens == 0 {
		t.Fatal("expected estimated tokens")
	}

	if !strings.Contains(result.ContextText, "Memories:") {
		t.Fatalf("expected context text to include memories section, got %q", result.ContextText)
	}

	if len(result.Trace) == 0 {
		t.Fatal("expected context trace")
	}
}

func TestContextRespectsBudget(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "context-budget.fbrain"), nil, 384)
	defer engine.Close()

	for _, content := range []string{
		"Cliente A usa Firebird 2.5 com incidente fiscal recorrente na nota eletrônica",
		"Cliente B usa Firebird 3.0 com ambiente legado e notas fiscais antigas",
		"Cliente C usa Firebird 4.0 com operacao fiscal critica e suporte pendente",
	} {
		if _, err := engine.Remember(RememberInput{
			BrainPath: engine.Path(),
			Content:   content,
		}); err != nil {
			t.Fatalf("remember %q: %v", content, err)
		}
	}

	result, err := engine.Context(ContextInput{
		BrainPath:    engine.Path(),
		Query:        "incidente fiscal firebird",
		TopK:         3,
		BudgetTokens: 12,
	})
	if err != nil {
		t.Fatalf("context: %v", err)
	}

	if result.EstimatedTokens > 12 {
		t.Fatalf("expected context budget <= 12, got %d", result.EstimatedTokens)
	}
}

func TestContextWithoutTraceOmitsTrace(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "context-no-trace.fbrain"), nil, 384)
	defer engine.Close()

	if _, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5",
	}); err != nil {
		t.Fatalf("remember: %v", err)
	}

	result, err := engine.Context(ContextInput{
		BrainPath: engine.Path(),
		Query:     "Firebird 2.5",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("context: %v", err)
	}

	if len(result.Trace) != 0 {
		t.Fatalf("expected empty trace, got %v", result.Trace)
	}
}
