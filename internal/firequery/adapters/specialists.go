package adapters

import (
	"context"
	"strings"

	coreengine "github.com/phmotad/firememory/internal/engine"
	coreextractor "github.com/phmotad/firememory/internal/extractor"
	"github.com/phmotad/firememory/internal/firequery/models"
	"github.com/phmotad/firememory/internal/memory"
)

type ExtractorAdapter struct {
	Extractor coreextractor.Extractor
}

func (a ExtractorAdapter) ExtractEntities(_ context.Context, input models.TextInput) ([]models.Entity, error) {
	result, err := a.Extractor.Extract(coreextractor.Input{Content: input.Text})
	if err != nil {
		return nil, err
	}

	entities := make([]models.Entity, 0, len(result.Entities))
	for _, entity := range result.Entities {
		entities = append(entities, models.Entity{
			Text:  entity.Name,
			Type:  entity.Type,
			Score: entity.Confidence,
		})
	}
	return entities, nil
}

func (a ExtractorAdapter) ExtractFacts(_ context.Context, input models.TextInput) ([]models.Fact, error) {
	result, err := a.Extractor.Extract(coreextractor.Input{Content: input.Text})
	if err != nil {
		return nil, err
	}

	facts := make([]models.Fact, 0, len(result.Facts))
	for _, fact := range result.Facts {
		facts = append(facts, models.Fact{
			Text:  strings.TrimSpace(fact.Subject + " " + fact.Predicate + " " + fact.Object),
			Score: fact.Confidence,
		})
	}
	return facts, nil
}

type RelationClassifierAdapter struct {
	Classifier coreengine.MemoryRelationClassifier
}

func (a RelationClassifierAdapter) ClassifyRelation(_ context.Context, left models.TextInput, right models.TextInput) (models.RelationSuggestion, error) {
	result, err := a.Classifier.Classify(coreengine.RelationClassificationInput{
		Left: memory.Memory{
			Content:    left.Text,
			Kind:       memory.MemoryKindNote,
			Status:     memory.MemoryStatusActive,
			Scope:      memory.DefaultScope,
			Importance: 0.5,
			Confidence: 0.8,
		},
		Right: memory.Memory{
			Content:    right.Text,
			Kind:       memory.MemoryKindNote,
			Status:     memory.MemoryStatusActive,
			Scope:      memory.DefaultScope,
			Importance: 0.5,
			Confidence: 0.8,
		},
		SimilarityScore: lexicalSimilarity(left.Text, right.Text),
	})
	if err != nil {
		return models.RelationSuggestion{}, err
	}

	return models.RelationSuggestion{
		Relation: string(result.Type),
		Score:    result.Confidence,
	}, nil
}

func lexicalSimilarity(left, right string) float64 {
	leftSet := map[string]struct{}{}
	for _, token := range strings.Fields(strings.ToLower(left)) {
		leftSet[token] = struct{}{}
	}

	if len(leftSet) == 0 {
		return 0
	}

	intersection := 0
	union := len(leftSet)
	for _, token := range strings.Fields(strings.ToLower(right)) {
		if _, ok := leftSet[token]; ok {
			intersection++
			continue
		}
		union++
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
