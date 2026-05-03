package models

import "context"

type MockIntentClassifier struct {
	Result IntentResult
	Err    error
}

func (m MockIntentClassifier) ClassifyIntent(context.Context, TextInput) (IntentResult, error) {
	return m.Result, m.Err
}

type MockTriggerClassifier struct {
	Result TriggerResult
	Err    error
}

func (m MockTriggerClassifier) ClassifyTrigger(context.Context, TextInput) (TriggerResult, error) {
	return m.Result, m.Err
}

type MockEntityExtractor struct {
	Result []Entity
	Err    error
}

func (m MockEntityExtractor) ExtractEntities(context.Context, TextInput) ([]Entity, error) {
	return append([]Entity(nil), m.Result...), m.Err
}

type MockFactExtractor struct {
	Result []Fact
	Err    error
}

func (m MockFactExtractor) ExtractFacts(context.Context, TextInput) ([]Fact, error) {
	return append([]Fact(nil), m.Result...), m.Err
}

type MockRelationClassifier struct {
	Result RelationSuggestion
	Err    error
}

func (m MockRelationClassifier) ClassifyRelation(context.Context, TextInput, TextInput) (RelationSuggestion, error) {
	return m.Result, m.Err
}

type MockSimilarityEngine struct {
	Result []Candidate
	Err    error
}

func (m MockSimilarityEngine) ScoreCandidates(context.Context, TextInput, []Candidate) ([]Candidate, error) {
	return append([]Candidate(nil), m.Result...), m.Err
}

type MockReranker struct {
	Result RankedCandidates
	Err    error
}

func (m MockReranker) Rerank(context.Context, TextInput, []Candidate) (RankedCandidates, error) {
	items := append([]Candidate(nil), m.Result.Items...)
	return RankedCandidates{Items: items}, m.Err
}
