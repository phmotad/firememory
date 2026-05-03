package embedder

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestNormalizeL2(t *testing.T) {
	vector, err := NormalizeL2(Vector{3, 4})
	if err != nil {
		t.Fatalf("expected normalization to succeed, got %v", err)
	}

	if math.Abs(float64(vector[0])-0.6) > 0.0001 {
		t.Fatalf("expected first component near 0.6, got %f", vector[0])
	}

	if math.Abs(float64(vector[1])-0.8) > 0.0001 {
		t.Fatalf("expected second component near 0.8, got %f", vector[1])
	}
}

func TestNormalizeL2RejectsZeroVector(t *testing.T) {
	_, err := NormalizeL2(Vector{0, 0, 0})
	if !errors.Is(err, ErrZeroMagnitude) {
		t.Fatalf("expected ErrZeroMagnitude, got %v", err)
	}
}

func TestDeterministicEmbedderIsStableAndNormalized(t *testing.T) {
	embedder, err := NewDeterministicEmbedder(8)
	if err != nil {
		t.Fatalf("new deterministic embedder: %v", err)
	}

	first, err := embedder.Embed(context.Background(), "Cliente Joao usa Firebird 2.5")
	if err != nil {
		t.Fatalf("first embed: %v", err)
	}

	second, err := embedder.Embed(context.Background(), "Cliente Joao usa Firebird 2.5")
	if err != nil {
		t.Fatalf("second embed: %v", err)
	}

	if len(first) != 8 {
		t.Fatalf("expected dimension 8, got %d", len(first))
	}

	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("expected deterministic output at index %d", i)
		}
	}

	var sum float64
	for _, value := range first {
		sum += float64(value * value)
	}

	if math.Abs(math.Sqrt(sum)-1.0) > 0.0001 {
		t.Fatalf("expected normalized vector, got magnitude %f", math.Sqrt(sum))
	}
}

func TestExternalEmbedderNormalizesAndValidatesDimension(t *testing.T) {
	client := stubClient{
		vector: Vector{3, 4},
	}

	embedder, err := NewExternalEmbedder(client, "test-model", 2)
	if err != nil {
		t.Fatalf("new external embedder: %v", err)
	}

	vector, err := embedder.Embed(context.Background(), "erro fiscal")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}

	if math.Abs(float64(vector[0])-0.6) > 0.0001 {
		t.Fatalf("expected normalized first component, got %f", vector[0])
	}
}

func TestExternalEmbedderRejectsDimensionMismatch(t *testing.T) {
	client := stubClient{
		vector: Vector{1, 2, 3},
	}

	embedder, err := NewExternalEmbedder(client, "test-model", 2)
	if err != nil {
		t.Fatalf("new external embedder: %v", err)
	}

	_, err = embedder.Embed(context.Background(), "erro fiscal")
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Fatalf("expected ErrDimensionMismatch, got %v", err)
	}
}

func TestE5EmbedderPrefixesQueryAndPassage(t *testing.T) {
	client := &recordingClient{
		vector: Vector{1, 2, 3, 4},
	}

	embedder, err := NewE5Embedder(client, 4)
	if err != nil {
		t.Fatalf("new e5 embedder: %v", err)
	}

	if _, err := embedder.EmbedQuery(context.Background(), "problema fiscal"); err != nil {
		t.Fatalf("embed query: %v", err)
	}

	if client.lastModel != E5ModelName {
		t.Fatalf("expected model %q, got %q", E5ModelName, client.lastModel)
	}

	if client.lastText != "query: problema fiscal" {
		t.Fatalf("expected query prefix, got %q", client.lastText)
	}

	if _, err := embedder.EmbedPassage(context.Background(), "Cliente Joao usa Firebird"); err != nil {
		t.Fatalf("embed passage: %v", err)
	}

	if client.lastText != "passage: Cliente Joao usa Firebird" {
		t.Fatalf("expected passage prefix, got %q", client.lastText)
	}
}

func TestValidateDimension(t *testing.T) {
	if err := ValidateDimension(0); !errors.Is(err, ErrInvalidDimension) {
		t.Fatalf("expected ErrInvalidDimension, got %v", err)
	}
}

type stubClient struct {
	vector Vector
	err    error
}

func (c stubClient) Embed(_ context.Context, _, _ string) (Vector, error) {
	if c.err != nil {
		return nil, c.err
	}

	out := make(Vector, len(c.vector))
	copy(out, c.vector)
	return out, nil
}

type recordingClient struct {
	vector    Vector
	lastModel string
	lastText  string
}

func (c *recordingClient) Embed(_ context.Context, model, text string) (Vector, error) {
	c.lastModel = model
	c.lastText = text

	out := make(Vector, len(c.vector))
	copy(out, c.vector)
	return out, nil
}
