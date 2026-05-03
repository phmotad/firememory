package engine

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExplainUsesProvidedRecallTrace(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "explain-recall.fbrain"), nil, 384)
	defer engine.Close()

	result, err := engine.Explain(ExplainInput{
		BrainPath: engine.Path(),
		Operation: "recall",
		MemoryID:  "mem_01",
		Trace:     []string{"embedded recall query", "executed vector search", "combined hybrid scores"},
	})
	if err != nil {
		t.Fatalf("explain: %v", err)
	}

	if !strings.Contains(result.Summary, "Recall explanation") {
		t.Fatalf("expected recall summary, got %q", result.Summary)
	}

	if len(result.Trace) != 3 {
		t.Fatalf("expected provided trace to be returned, got %v", result.Trace)
	}
}

func TestExplainLoadsPersistedDedupTrace(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "explain-dedup.fbrain"), nil, 384)
	defer engine.Close()

	remembered, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5",
	})
	if err != nil {
		t.Fatalf("remember: %v", err)
	}

	result, err := engine.Explain(ExplainInput{
		BrainPath: engine.Path(),
		Operation: "create_new",
		MemoryID:  remembered.Memory.ID,
	})
	if err != nil {
		t.Fatalf("explain: %v", err)
	}

	if !strings.Contains(result.Summary, remembered.Memory.ID) {
		t.Fatalf("expected summary to mention memory id, got %q", result.Summary)
	}

	if len(result.Trace) == 0 {
		t.Fatal("expected persisted trace to be loaded")
	}

	joined := strings.Join(result.Trace, " | ")
	if !strings.Contains(joined, "created new memory") {
		t.Fatalf("expected remember trace details, got %q", joined)
	}
}

func TestExplainForContextWithoutTraceReportsNoTrace(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "explain-context.fbrain"), nil, 384)
	defer engine.Close()

	result, err := engine.Explain(ExplainInput{
		BrainPath: engine.Path(),
		Operation: "context",
	})
	if err != nil {
		t.Fatalf("explain: %v", err)
	}

	if !strings.Contains(result.Summary, "no trace available") {
		t.Fatalf("expected no-trace summary, got %q", result.Summary)
	}
}
