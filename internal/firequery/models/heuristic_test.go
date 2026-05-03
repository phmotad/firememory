package models

import (
	"context"
	"testing"
)

func TestHeuristicIntentClassifier(t *testing.T) {
	t.Parallel()

	classifier := HeuristicIntentClassifier{}

	got, err := classifier.ClassifyIntent(context.Background(), TextInput{Text: "responder Joao com contexto fiscal"})
	if err != nil {
		t.Fatalf("ClassifyIntent() error = %v", err)
	}
	if got.Intent != "build_context" {
		t.Fatalf("intent = %q, want build_context", got.Intent)
	}
}

func TestHeuristicTriggerClassifier(t *testing.T) {
	t.Parallel()

	classifier := HeuristicTriggerClassifier{}

	got, err := classifier.ClassifyTrigger(context.Background(), TextInput{Text: "remover essa memoria"})
	if err != nil {
		t.Fatalf("ClassifyTrigger() error = %v", err)
	}
	if got.Trigger != "request_confirmation" {
		t.Fatalf("trigger = %q, want request_confirmation", got.Trigger)
	}
}

func TestHeuristicEntityAndFactExtractors(t *testing.T) {
	t.Parallel()

	entityExtractor := NewHeuristicEntityExtractor()
	factExtractor := NewHeuristicFactExtractor()
	input := TextInput{Text: "Cliente Joao usa Firebird 2.5 no FireMemory"}

	entities, err := entityExtractor.ExtractEntities(context.Background(), input)
	if err != nil {
		t.Fatalf("ExtractEntities() error = %v", err)
	}
	if len(entities) == 0 {
		t.Fatal("expected entities")
	}

	facts, err := factExtractor.ExtractFacts(context.Background(), input)
	if err != nil {
		t.Fatalf("ExtractFacts() error = %v", err)
	}
	if len(facts) == 0 {
		t.Fatal("expected facts")
	}
}

func TestHeuristicRelationClassifier(t *testing.T) {
	t.Parallel()

	classifier := HeuristicRelationClassifier{}

	got, err := classifier.ClassifyRelation(
		context.Background(),
		TextInput{Text: "Erro fiscal na versao 3.1"},
		TextInput{Text: "Erro fiscal na versao 3.2"},
	)
	if err != nil {
		t.Fatalf("ClassifyRelation() error = %v", err)
	}
	if got.Relation != "update" {
		t.Fatalf("relation = %q, want update", got.Relation)
	}
}

func TestHeuristicSimilarityAndReranker(t *testing.T) {
	t.Parallel()

	engine := HeuristicSimilarityEngine{}
	reranker := StableReranker{}
	input := TextInput{Text: "erro fiscal nfe"}
	candidates := []Candidate{
		{ID: "b", Text: "erro fiscal nfe apos atualizacao"},
		{ID: "a", Text: "backup do banco"},
	}

	scored, err := engine.ScoreCandidates(context.Background(), input, candidates)
	if err != nil {
		t.Fatalf("ScoreCandidates() error = %v", err)
	}
	if scored[0].ID != "b" {
		t.Fatalf("top scored = %q, want b", scored[0].ID)
	}

	ranked, err := reranker.Rerank(context.Background(), input, scored)
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if ranked.Items[0].ID != "b" {
		t.Fatalf("top ranked = %q, want b", ranked.Items[0].ID)
	}
}

func TestMockSpecialists(t *testing.T) {
	t.Parallel()

	intent, err := (MockIntentClassifier{
		Result: IntentResult{Intent: "recall_information", Score: 1},
	}).ClassifyIntent(context.Background(), TextInput{Text: "x"})
	if err != nil {
		t.Fatalf("mock intent error = %v", err)
	}
	if intent.Intent != "recall_information" {
		t.Fatalf("intent = %q", intent.Intent)
	}

	ranked, err := (MockReranker{
		Result: RankedCandidates{Items: []Candidate{{ID: "x"}}},
	}).Rerank(context.Background(), TextInput{Text: "x"}, nil)
	if err != nil {
		t.Fatalf("mock reranker error = %v", err)
	}
	if len(ranked.Items) != 1 || ranked.Items[0].ID != "x" {
		t.Fatalf("ranked = %#v", ranked.Items)
	}
}
