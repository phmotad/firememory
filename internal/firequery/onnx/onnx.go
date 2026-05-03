// Package onnx provides ONNX Runtime-based inference for FireQuery specialists.
//
// Build with -tags onnx to enable real model inference.
// Without that tag, New() returns ErrNotAvailable and the heuristic fallbacks remain active.
package onnx

import (
	"context"
	"errors"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/firequery/models"
)

var (
	ErrNotAvailable = errors.New("onnx: backend not available (rebuild with -tags onnx)")
	ErrModelNotFound = errors.New("onnx: model files not found (run: fquery models pull)")
)

// Backend provides ONNX-based inference for all specialist components.
// It replaces the Python subprocess backend with a pure-Go ONNX Runtime binding.
type Backend interface {
	// Embed implements embedder.Client for sentence encoding.
	Embed(ctx context.Context, modelID, text string) (embedder.Vector, error)

	// Classify implements models.TextClassificationClient.
	// Uses embedding-based cosine similarity against label descriptions.
	Classify(ctx context.Context, modelID string, input models.TextInput, labels []string) ([]models.ScoredLabel, error)

	// ExtractEntities implements models.EntityExtractionClient via GLiNER.
	ExtractEntities(ctx context.Context, modelID string, input models.TextInput) ([]models.Entity, error)

	// EmbedQuery returns an L2-normalized embedding with "query: " prefix (for E5).
	EmbedQuery(ctx context.Context, text string) (embedder.Vector, error)

	// EmbedPassage returns an L2-normalized embedding with "passage: " prefix (for E5).
	EmbedPassage(ctx context.Context, text string) (embedder.Vector, error)

	// Name returns the similarity model identifier. Satisfies models.QueryPassageEmbedder.
	Name() string

	// Dimension returns the embedding vector dimension.
	Dimension() int

	// Close releases all ONNX sessions and tokenizer resources.
	Close() error
}
