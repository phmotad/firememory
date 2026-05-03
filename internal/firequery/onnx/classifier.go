//go:build onnx

package onnx

import (
	"context"
	"fmt"
	"sort"

	"github.com/phmotad/firememory/internal/firequery/models"
	"github.com/phmotad/firememory/internal/vector"
)

// embeddingClassifier implements models.TextClassificationClient using
// embedding-based cosine similarity (same approach as the Python backend).
// The text and each label description are encoded, then ranked by cosine similarity.
type embeddingClassifier struct {
	enc              *encoder
	labelDescriptions map[string]map[string]string // modelID → label → description
}

func newEmbeddingClassifier(enc *encoder) *embeddingClassifier {
	return &embeddingClassifier{
		enc: enc,
		labelDescriptions: map[string]map[string]string{
			models.IntentModelDeBERTaSmall:   IntentLabelDescriptions,
			models.TriggerModelDeBERTaSmall:  TriggerLabelDescriptions,
		},
	}
}

// Classify implements models.TextClassificationClient.
func (c *embeddingClassifier) Classify(ctx context.Context, modelID string, input models.TextInput, labels []string) ([]models.ScoredLabel, error) {
	if input.Text == "" {
		return nil, fmt.Errorf("onnx: classify: empty input text")
	}

	textVec, err := c.enc.embed(input.Text)
	if err != nil {
		return nil, fmt.Errorf("onnx: classify text embed: %w", err)
	}

	descriptions, ok := c.labelDescriptions[modelID]
	if !ok {
		// Fall back to using the label name directly as description.
		descriptions = map[string]string{}
	}

	results := make([]models.ScoredLabel, 0, len(labels))
	for _, label := range labels {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		desc, ok := descriptions[label]
		if !ok {
			desc = label
		}

		labelVec, err := c.enc.embed(desc)
		if err != nil {
			return nil, fmt.Errorf("onnx: classify label embed %q: %w", label, err)
		}

		score := vector.CosineSimilarity(textVec, labelVec)
		results = append(results, models.ScoredLabel{Label: label, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results, nil
}
