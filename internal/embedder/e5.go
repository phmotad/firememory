package embedder

import (
	"context"
	"strings"
)

const (
	E5ModelName   = "intfloat/multilingual-e5-small"
	E5QueryMode   = "query"
	E5PassageMode = "passage"
)

type E5Embedder struct {
	base *ExternalEmbedder
	mode string
}

func NewE5Embedder(client Client, dimension int) (*E5Embedder, error) {
	if dimension == 0 {
		dimension = DefaultDimension
	}

	base, err := NewExternalEmbedder(client, E5ModelName, dimension)
	if err != nil {
		return nil, err
	}

	return &E5Embedder{
		base: base,
		mode: E5PassageMode,
	}, nil
}

func (e *E5Embedder) Name() string {
	return e.base.Name()
}

func (e *E5Embedder) Dimension() int {
	return e.base.Dimension()
}

func (e *E5Embedder) Embed(ctx context.Context, text string) (Vector, error) {
	return e.EmbedPassage(ctx, text)
}

func (e *E5Embedder) EmbedQuery(ctx context.Context, text string) (Vector, error) {
	return e.embedWithPrefix(ctx, E5QueryMode, text)
}

func (e *E5Embedder) EmbedPassage(ctx context.Context, text string) (Vector, error) {
	return e.embedWithPrefix(ctx, E5PassageMode, text)
}

func (e *E5Embedder) embedWithPrefix(ctx context.Context, mode, text string) (Vector, error) {
	normalizedText := strings.TrimSpace(text)
	if normalizedText == "" {
		return nil, ErrEmptyInput
	}

	return e.base.Embed(ctx, mode+": "+normalizedText)
}
