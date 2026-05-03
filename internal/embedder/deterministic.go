package embedder

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"strings"
)

const DeterministicModel = "deterministic"

type DeterministicEmbedder struct {
	dimension int
}

func NewDeterministicEmbedder(dimension int) (*DeterministicEmbedder, error) {
	if dimension == 0 {
		dimension = DefaultDimension
	}

	if err := ValidateDimension(dimension); err != nil {
		return nil, err
	}

	return &DeterministicEmbedder{dimension: dimension}, nil
}

func (e *DeterministicEmbedder) Name() string {
	return DeterministicModel
}

func (e *DeterministicEmbedder) Dimension() int {
	return e.dimension
}

func (e *DeterministicEmbedder) Embed(_ context.Context, text string) (Vector, error) {
	normalizedText := strings.TrimSpace(text)
	if normalizedText == "" {
		return nil, ErrEmptyInput
	}

	vector := make(Vector, e.dimension)
	seed := []byte(normalizedText)
	for i := 0; i < e.dimension; i++ {
		sum := sha256.Sum256(seed)
		raw := binary.BigEndian.Uint32(sum[:4])
		vector[i] = scaleUint32ToUnit(raw)
		seed = append(sum[:], byte(i))
	}

	return NormalizeL2(vector)
}

func scaleUint32ToUnit(value uint32) float32 {
	normalized := (float64(value) / float64(^uint32(0))) * 2
	return float32(normalized - 1)
}
