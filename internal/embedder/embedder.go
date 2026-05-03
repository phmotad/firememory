package embedder

import (
	"context"
	"errors"
	"math"
)

const DefaultDimension = 384

var (
	ErrEmptyInput        = errors.New("input text is required")
	ErrInvalidDimension  = errors.New("embedding dimension must be greater than zero")
	ErrDimensionMismatch = errors.New("embedding dimension mismatch")
	ErrZeroMagnitude     = errors.New("embedding vector has zero magnitude")
	ErrNilExternalClient = errors.New("external embedding client is required")
	ErrEmptyModelName    = errors.New("embedding model name is required")
)

type Vector = []float32

type Embedder interface {
	Name() string
	Dimension() int
	Embed(ctx context.Context, text string) (Vector, error)
}

type Client interface {
	Embed(ctx context.Context, model, text string) (Vector, error)
}

func ValidateDimension(dimension int) error {
	if dimension <= 0 {
		return ErrInvalidDimension
	}

	return nil
}

func ValidateVectorDimension(vector Vector, dimension int) error {
	if err := ValidateDimension(dimension); err != nil {
		return err
	}

	if len(vector) != dimension {
		return ErrDimensionMismatch
	}

	return nil
}

func NormalizeL2(vector Vector) (Vector, error) {
	var sum float64
	for _, value := range vector {
		sum += float64(value * value)
	}

	magnitude := math.Sqrt(sum)
	if magnitude == 0 {
		return nil, ErrZeroMagnitude
	}

	normalized := make(Vector, len(vector))
	for i, value := range vector {
		normalized[i] = float32(float64(value) / magnitude)
	}

	return normalized, nil
}
