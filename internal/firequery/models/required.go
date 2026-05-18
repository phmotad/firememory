package models

import (
	"context"
	"sort"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/vector"
)

const (
	SimilarityModelE5Small   = embedder.E5ModelName
	EntityModelGLiNER2Small  = "gliner2-small"
	IntentModelDeBERTaSmall  = "microsoft/deberta-v3-small"
	TriggerModelDeBERTaSmall = "microsoft/deberta-v3-small"
	IntentModelModernBERT    = "answerdotai/ModernBERT-base"
	TriggerModelModernBERT   = "answerdotai/ModernBERT-base"
)

type TextClassificationClient interface {
	Classify(ctx context.Context, modelID string, input TextInput, labels []string) ([]ScoredLabel, error)
}

type EntityExtractionClient interface {
	ExtractEntities(ctx context.Context, modelID string, input TextInput) ([]Entity, error)
}

type QueryPassageEmbedder interface {
	Name() string
	Dimension() int
	EmbedQuery(ctx context.Context, text string) (embedder.Vector, error)
	EmbedPassage(ctx context.Context, text string) (embedder.Vector, error)
}

type ModelBound interface {
	ModelID() string
}

type DeBERTaIntentClassifier struct {
	Client   TextClassificationClient
	Fallback IntentClassifier
	modelID  string
}

func NewDeBERTaIntentClassifier(client TextClassificationClient, fallback IntentClassifier) DeBERTaIntentClassifier {
	return NewConfiguredDeBERTaIntentClassifier(IntentModelDeBERTaSmall, client, fallback)
}

func NewConfiguredDeBERTaIntentClassifier(modelID string, client TextClassificationClient, fallback IntentClassifier) DeBERTaIntentClassifier {
	if fallback == nil {
		fallback = HeuristicIntentClassifier{}
	}
	return DeBERTaIntentClassifier{
		Client:   client,
		Fallback: fallback,
		modelID:  configuredModelID(modelID, IntentModelDeBERTaSmall),
	}
}

func (c DeBERTaIntentClassifier) ModelID() string { return c.modelID }

func (c DeBERTaIntentClassifier) ClassifyIntent(ctx context.Context, input TextInput) (IntentResult, error) {
	if c.Client == nil {
		return c.Fallback.ClassifyIntent(ctx, input)
	}

	labels := []string{
		"remember_information",
		"recall_information",
		"build_context",
		"explain_decision",
		"sync_memory",
	}
	scored, err := c.Client.Classify(ctx, c.modelID, input, labels)
	if err != nil || len(scored) == 0 {
		return c.Fallback.ClassifyIntent(ctx, input)
	}
	top := topLabel(scored)
	return IntentResult{Intent: top.Label, Score: top.Score}, nil
}

type DeBERTaTriggerClassifier struct {
	Client   TextClassificationClient
	Fallback TriggerClassifier
	modelID  string
}

func NewDeBERTaTriggerClassifier(client TextClassificationClient, fallback TriggerClassifier) DeBERTaTriggerClassifier {
	return NewConfiguredDeBERTaTriggerClassifier(TriggerModelDeBERTaSmall, client, fallback)
}

func NewConfiguredDeBERTaTriggerClassifier(modelID string, client TextClassificationClient, fallback TriggerClassifier) DeBERTaTriggerClassifier {
	if fallback == nil {
		fallback = HeuristicTriggerClassifier{}
	}
	return DeBERTaTriggerClassifier{
		Client:   client,
		Fallback: fallback,
		modelID:  configuredModelID(modelID, TriggerModelDeBERTaSmall),
	}
}

func (c DeBERTaTriggerClassifier) ModelID() string { return c.modelID }

func (c DeBERTaTriggerClassifier) ClassifyTrigger(ctx context.Context, input TextInput) (TriggerResult, error) {
	if c.Client == nil {
		return c.Fallback.ClassifyTrigger(ctx, input)
	}

	labels := []string{
		"do_nothing",
		"query_memory",
		"suggest_write",
		"request_confirmation",
	}
	scored, err := c.Client.Classify(ctx, c.modelID, input, labels)
	if err != nil || len(scored) == 0 {
		return c.Fallback.ClassifyTrigger(ctx, input)
	}
	top := topLabel(scored)
	return TriggerResult{Trigger: top.Label, Score: top.Score}, nil
}

type GLiNEREntityExtractor struct {
	Client   EntityExtractionClient
	Fallback EntityExtractor
	modelID  string
}

func NewGLiNEREntityExtractor(client EntityExtractionClient, fallback EntityExtractor) GLiNEREntityExtractor {
	return NewConfiguredGLiNEREntityExtractor(EntityModelGLiNER2Small, client, fallback)
}

func NewConfiguredGLiNEREntityExtractor(modelID string, client EntityExtractionClient, fallback EntityExtractor) GLiNEREntityExtractor {
	if fallback == nil {
		fallback = NewHeuristicEntityExtractor()
	}
	return GLiNEREntityExtractor{
		Client:   client,
		Fallback: fallback,
		modelID:  configuredModelID(modelID, EntityModelGLiNER2Small),
	}
}

func (e GLiNEREntityExtractor) ModelID() string { return e.modelID }

func (e GLiNEREntityExtractor) ExtractEntities(ctx context.Context, input TextInput) ([]Entity, error) {
	if e.Client == nil {
		return e.Fallback.ExtractEntities(ctx, input)
	}
	entities, err := e.Client.ExtractEntities(ctx, e.modelID, input)
	if err != nil || len(entities) == 0 {
		return e.Fallback.ExtractEntities(ctx, input)
	}
	return entities, nil
}

type E5SimilarityEngine struct {
	Embedder QueryPassageEmbedder
	Fallback SimilarityEngine
	modelID  string
}

func NewE5SimilarityEngine(embedder QueryPassageEmbedder, fallback SimilarityEngine) E5SimilarityEngine {
	return NewConfiguredE5SimilarityEngine(SimilarityModelE5Small, embedder, fallback)
}

func NewConfiguredE5SimilarityEngine(modelID string, embedder QueryPassageEmbedder, fallback SimilarityEngine) E5SimilarityEngine {
	if fallback == nil {
		fallback = HeuristicSimilarityEngine{}
	}
	return E5SimilarityEngine{
		Embedder: embedder,
		Fallback: fallback,
		modelID:  configuredModelID(modelID, SimilarityModelE5Small),
	}
}

func (e E5SimilarityEngine) ModelID() string {
	return configuredModelID(e.modelID, SimilarityModelE5Small)
}

func (e E5SimilarityEngine) ScoreCandidates(ctx context.Context, input TextInput, candidates []Candidate) ([]Candidate, error) {
	if e.Embedder == nil {
		return e.Fallback.ScoreCandidates(ctx, input, candidates)
	}

	query, err := e.Embedder.EmbedQuery(ctx, input.Text)
	if err != nil {
		return e.Fallback.ScoreCandidates(ctx, input, candidates)
	}

	scored := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		item := candidate
		passage, err := e.Embedder.EmbedPassage(ctx, candidate.Text)
		if err != nil {
			return e.Fallback.ScoreCandidates(ctx, input, candidates)
		}
		item.Score = vector.CosineSimilarity(query, passage)
		scored = append(scored, item)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].ID < scored[j].ID
		}
		return scored[i].Score > scored[j].Score
	})
	return scored, nil
}

func topLabel(labels []ScoredLabel) ScoredLabel {
	best := labels[0]
	for _, label := range labels[1:] {
		if label.Score > best.Score {
			best = label
		}
	}
	return best
}

func configuredModelID(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
