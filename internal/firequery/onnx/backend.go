//go:build onnx

package onnx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/firequery/models"
)

const (
	// ModelDirE5 is the subdirectory name for the E5 similarity model.
	ModelDirE5      = "multilingual-e5-small"
	// ModelDirDeBERTa is the subdirectory name for the DeBERTa classification model.
	ModelDirDeBERTa = "deberta-v3-small"
	// ModelDirGLiNER is the subdirectory name for the GLiNER entity extractor.
	ModelDirGLiNER  = "gliner-small-v2.1"
)

type onnxBackend struct {
	e5enc      *encoder           // E5 similarity encoder
	classEnc   *encoder           // DeBERTa classification encoder
	classifier *embeddingClassifier
	gliner     *glinerExtractor
	similarityModelID string
	dimension         int
}

// New initialises the ONNX backend from the given models directory.
// Each model is expected in its own subdirectory containing model.onnx and tokenizer.json.
// Returns ErrModelNotFound if any required model directory is missing.
func New(modelsDir string) (Backend, error) {
	if modelsDir == "" {
		modelsDir = DefaultModelsDir()
	}

	e5Dir := filepath.Join(modelsDir, ModelDirE5)
	debertaDir := filepath.Join(modelsDir, ModelDirDeBERTa)
	glinerDir := filepath.Join(modelsDir, ModelDirGLiNER)

	for _, dir := range []string{e5Dir, debertaDir, glinerDir} {
		if _, err := os.Stat(filepath.Join(dir, "model.onnx")); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrModelNotFound, dir)
		}
	}

	e5enc, err := newEncoder(e5Dir, models.SimilarityModelE5Small)
	if err != nil {
		return nil, fmt.Errorf("onnx: load E5 encoder: %w", err)
	}

	classEnc, err := newEncoder(debertaDir, models.IntentModelDeBERTaSmall)
	if err != nil {
		_ = e5enc.Close()
		return nil, fmt.Errorf("onnx: load DeBERTa encoder: %w", err)
	}

	gliner, err := newGLiNER(glinerDir, EntityLabels)
	if err != nil {
		_ = e5enc.Close()
		_ = classEnc.Close()
		return nil, fmt.Errorf("onnx: load GLiNER: %w", err)
	}

	return &onnxBackend{
		e5enc:             e5enc,
		classEnc:          classEnc,
		classifier:        newEmbeddingClassifier(classEnc),
		gliner:            gliner,
		similarityModelID: models.SimilarityModelE5Small,
		dimension:         e5enc.dimension,
	}, nil
}

func (b *onnxBackend) Close() error {
	var errs []error
	if err := b.e5enc.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := b.classEnc.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := b.gliner.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("onnx: close: %v", errs)
	}
	return nil
}

// Embed implements embedder.Client using the E5 encoder.
func (b *onnxBackend) Embed(ctx context.Context, _, text string) (embedder.Vector, error) {
	return b.e5enc.Embed(ctx, "", text)
}

// EmbedQuery prefixes "query: " for E5 retrieval.
func (b *onnxBackend) EmbedQuery(_ context.Context, text string) (embedder.Vector, error) {
	return b.e5enc.embed("query: " + text)
}

// EmbedPassage prefixes "passage: " for E5 retrieval.
func (b *onnxBackend) EmbedPassage(_ context.Context, text string) (embedder.Vector, error) {
	return b.e5enc.embed("passage: " + text)
}

// Classify implements models.TextClassificationClient using DeBERTa embeddings.
func (b *onnxBackend) Classify(ctx context.Context, modelID string, input models.TextInput, labels []string) ([]models.ScoredLabel, error) {
	return b.classifier.Classify(ctx, modelID, input, labels)
}

// ExtractEntities implements models.EntityExtractionClient using GLiNER.
func (b *onnxBackend) ExtractEntities(ctx context.Context, modelID string, input models.TextInput) ([]models.Entity, error) {
	return b.gliner.ExtractEntities(ctx, modelID, input)
}

// Name returns the similarity model ID. Satisfies models.QueryPassageEmbedder.
func (b *onnxBackend) Name() string  { return b.similarityModelID }
func (b *onnxBackend) Dimension() int { return b.dimension }
