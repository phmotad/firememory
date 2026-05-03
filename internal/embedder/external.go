package embedder

import (
	"context"
	"strings"
)

type ExternalEmbedder struct {
	client    Client
	modelName string
	dimension int
}

func NewExternalEmbedder(client Client, modelName string, dimension int) (*ExternalEmbedder, error) {
	if client == nil {
		return nil, ErrNilExternalClient
	}

	if strings.TrimSpace(modelName) == "" {
		return nil, ErrEmptyModelName
	}

	if err := ValidateDimension(dimension); err != nil {
		return nil, err
	}

	return &ExternalEmbedder{
		client:    client,
		modelName: modelName,
		dimension: dimension,
	}, nil
}

func (e *ExternalEmbedder) Name() string {
	return e.modelName
}

func (e *ExternalEmbedder) Dimension() int {
	return e.dimension
}

func (e *ExternalEmbedder) Embed(ctx context.Context, text string) (Vector, error) {
	normalizedText := strings.TrimSpace(text)
	if normalizedText == "" {
		return nil, ErrEmptyInput
	}

	vector, err := e.client.Embed(ctx, e.modelName, normalizedText)
	if err != nil {
		return nil, err
	}

	if err := ValidateVectorDimension(vector, e.dimension); err != nil {
		return nil, err
	}

	return NormalizeL2(vector)
}
