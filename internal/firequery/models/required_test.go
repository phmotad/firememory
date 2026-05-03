package models

import (
	"context"
	"testing"

	"github.com/phmotad/firememory/internal/embedder"
)

func TestRequiredModelAdaptersFallback(t *testing.T) {
	t.Parallel()

	intent := NewDeBERTaIntentClassifier(nil, nil)
	intentResult, err := intent.ClassifyIntent(context.Background(), TextInput{Language: "pt-BR", Text: "responder Joao sobre erro fiscal"})
	if err != nil {
		t.Fatalf("ClassifyIntent() error = %v", err)
	}
	if intent.ModelID() != IntentModelDeBERTaSmall {
		t.Fatalf("intent model = %q", intent.ModelID())
	}
	if intentResult.Intent == "" {
		t.Fatal("expected non-empty intent")
	}

	trigger := NewDeBERTaTriggerClassifier(nil, nil)
	triggerResult, err := trigger.ClassifyTrigger(context.Background(), TextInput{Language: "pt-BR", Text: "salve isso na memoria"})
	if err != nil {
		t.Fatalf("ClassifyTrigger() error = %v", err)
	}
	if trigger.ModelID() != TriggerModelDeBERTaSmall {
		t.Fatalf("trigger model = %q", trigger.ModelID())
	}
	if triggerResult.Trigger == "" {
		t.Fatal("expected non-empty trigger")
	}

	entityExtractor := NewGLiNEREntityExtractor(nil, nil)
	entities, err := entityExtractor.ExtractEntities(context.Background(), TextInput{Language: "en", Text: "Joao uses Firebird 2.5"})
	if err != nil {
		t.Fatalf("ExtractEntities() error = %v", err)
	}
	if entityExtractor.ModelID() != EntityModelGLiNER2Small {
		t.Fatalf("entity model = %q", entityExtractor.ModelID())
	}
	if len(entities) == 0 {
		t.Fatal("expected fallback entities")
	}
}

func TestConfiguredRequiredModelAdapters(t *testing.T) {
	t.Parallel()

	intent := NewConfiguredDeBERTaIntentClassifier(IntentModelModernBERT, nil, nil)
	if intent.ModelID() != IntentModelModernBERT {
		t.Fatalf("intent model = %q", intent.ModelID())
	}

	trigger := NewConfiguredDeBERTaTriggerClassifier(TriggerModelModernBERT, nil, nil)
	if trigger.ModelID() != TriggerModelModernBERT {
		t.Fatalf("trigger model = %q", trigger.ModelID())
	}

	entityExtractor := NewConfiguredGLiNEREntityExtractor("urchade/gliner_medium-v2.1", nil, nil)
	if entityExtractor.ModelID() != "urchade/gliner_medium-v2.1" {
		t.Fatalf("entity model = %q", entityExtractor.ModelID())
	}

	engine := NewConfiguredE5SimilarityEngine("intfloat/multilingual-e5-small", nil, nil)
	if engine.ModelID() != "intfloat/multilingual-e5-small" {
		t.Fatalf("similarity model = %q", engine.ModelID())
	}
}

func TestE5SimilarityEngineScoresCandidates(t *testing.T) {
	t.Parallel()

	engine := NewE5SimilarityEngine(fakeE5Embedder{
		query: map[string]embedder.Vector{
			"fiscal error": {1, 0},
		},
		passage: map[string]embedder.Vector{
			"fiscal candidate": {1, 0},
			"other candidate":  {0, 1},
		},
	}, nil)

	scored, err := engine.ScoreCandidates(context.Background(), TextInput{Language: "en", Text: "fiscal error"}, []Candidate{
		{ID: "b", Text: "other candidate"},
		{ID: "a", Text: "fiscal candidate"},
	})
	if err != nil {
		t.Fatalf("ScoreCandidates() error = %v", err)
	}
	if engine.ModelID() != SimilarityModelE5Small {
		t.Fatalf("similarity model = %q", engine.ModelID())
	}
	if len(scored) != 2 || scored[0].ID != "a" {
		t.Fatalf("scored = %#v", scored)
	}
}

type fakeE5Embedder struct {
	query   map[string]embedder.Vector
	passage map[string]embedder.Vector
}

func (f fakeE5Embedder) Name() string   { return SimilarityModelE5Small }
func (f fakeE5Embedder) Dimension() int { return 2 }
func (f fakeE5Embedder) EmbedQuery(_ context.Context, text string) (embedder.Vector, error) {
	return append(embedder.Vector(nil), f.query[text]...), nil
}
func (f fakeE5Embedder) EmbedPassage(_ context.Context, text string) (embedder.Vector, error) {
	return append(embedder.Vector(nil), f.passage[text]...), nil
}
