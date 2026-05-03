package models

import "context"

type TextInput struct {
	Language string
	Text     string
}

type ScoredLabel struct {
	Label string
	Score float64
}

type IntentResult struct {
	Intent string
	Score  float64
}

type TriggerResult struct {
	Trigger string
	Score   float64
}

type Entity struct {
	Text  string
	Type  string
	Score float64
}

type Fact struct {
	Text  string
	Score float64
}

type RelationSuggestion struct {
	Relation string
	Score    float64
}

type Candidate struct {
	ID    string
	Text  string
	Score float64
}

type RankedCandidates struct {
	Items []Candidate
}

type IntentClassifier interface {
	ClassifyIntent(ctx context.Context, input TextInput) (IntentResult, error)
}

type TriggerClassifier interface {
	ClassifyTrigger(ctx context.Context, input TextInput) (TriggerResult, error)
}

type EntityExtractor interface {
	ExtractEntities(ctx context.Context, input TextInput) ([]Entity, error)
}

type FactExtractor interface {
	ExtractFacts(ctx context.Context, input TextInput) ([]Fact, error)
}

type RelationClassifier interface {
	ClassifyRelation(ctx context.Context, left TextInput, right TextInput) (RelationSuggestion, error)
}

type SimilarityEngine interface {
	ScoreCandidates(ctx context.Context, input TextInput, candidates []Candidate) ([]Candidate, error)
}

type Reranker interface {
	Rerank(ctx context.Context, input TextInput, candidates []Candidate) (RankedCandidates, error)
}
