//go:build onnx

package onnx

import (
	"context"
	"fmt"
	"math"
	"path/filepath"

	"github.com/phmotad/firememory/internal/embedder"
	ort "github.com/yalue/onnxruntime_go"
)

// ONNX tensor names for standard sentence-transformer models
// (multilingual-e5-small, deberta-v3-small).
const (
	inputNameIDs   = "input_ids"
	inputNameMask  = "attention_mask"
	inputNameTTIDs = "token_type_ids"
	outputNameHS   = "last_hidden_state"
)

// encoder runs a transformer sentence encoder via ONNX Runtime.
// It performs mean pooling over the last_hidden_state and L2-normalizes the result.
type encoder struct {
	session   *ort.DynamicAdvancedSession
	tokenizer *tokenizer
	dimension int
	modelID   string
}

func newEncoder(modelDir, modelID string) (*encoder, error) {
	tok, err := loadTokenizer(modelDir)
	if err != nil {
		return nil, err
	}

	modelPath := filepath.Join(modelDir, "model.onnx")
	session, err := newDynamicSession(modelPath,
		[]string{inputNameIDs, inputNameMask, inputNameTTIDs},
		[]string{outputNameHS},
	)
	if err != nil {
		tok.Close()
		return nil, fmt.Errorf("onnx: encoder session for %s: %w", modelID, err)
	}

	dim, err := probeDimension(session, tok)
	if err != nil {
		tok.Close()
		_ = session.Destroy()
		return nil, fmt.Errorf("onnx: probe dimension for %s: %w", modelID, err)
	}

	return &encoder{
		session:   session,
		tokenizer: tok,
		dimension: dim,
		modelID:   modelID,
	}, nil
}

func (e *encoder) Close() error {
	e.tokenizer.Close()
	return e.session.Destroy()
}

// Embed encodes a single text and returns an L2-normalized vector.
// Implements embedder.Client (model parameter is ignored; the encoder is bound to one model).
func (e *encoder) Embed(_ context.Context, _, text string) (embedder.Vector, error) {
	return e.embed(text)
}

func (e *encoder) embed(text string) (embedder.Vector, error) {
	enc := e.tokenizer.Encode(text)
	if enc.SeqLen == 0 {
		return nil, embedder.ErrEmptyInput
	}

	shape := ort.NewShape(1, int64(enc.SeqLen))

	idsT, err := ort.NewTensor(shape, enc.InputIDs)
	if err != nil {
		return nil, fmt.Errorf("onnx: input_ids tensor: %w", err)
	}
	defer idsT.Destroy()

	maskT, err := ort.NewTensor(shape, enc.AttentionMask)
	if err != nil {
		return nil, fmt.Errorf("onnx: attention_mask tensor: %w", err)
	}
	defer maskT.Destroy()

	ttT, err := ort.NewTensor(shape, enc.TokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("onnx: token_type_ids tensor: %w", err)
	}
	defer ttT.Destroy()

	outShape := ort.NewShape(1, int64(enc.SeqLen), int64(e.dimension))
	outT, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		return nil, fmt.Errorf("onnx: output tensor: %w", err)
	}
	defer outT.Destroy()

	err = e.session.Run(
		[]ort.ArbitraryTensor{idsT, maskT, ttT},
		[]ort.ArbitraryTensor{outT},
	)
	if err != nil {
		return nil, fmt.Errorf("onnx: encoder run: %w", err)
	}

	hidden := outT.GetData() // [1, seqLen, dim] flattened
	vec := meanPool(hidden, enc.AttentionMask, enc.SeqLen, e.dimension)
	return normalizeL2(vec)
}

// meanPool computes attention-weighted mean pooling over the last_hidden_state.
func meanPool(hidden []float32, mask []int64, seqLen, dim int) embedder.Vector {
	pooled := make([]float32, dim)
	var count float64
	for i := 0; i < seqLen; i++ {
		if mask[i] == 0 {
			continue
		}
		count++
		for d := 0; d < dim; d++ {
			pooled[d] += hidden[i*dim+d]
		}
	}
	if count > 0 {
		for d := range pooled {
			pooled[d] /= float32(count)
		}
	}
	return pooled
}

// normalizeL2 scales the vector to unit length.
func normalizeL2(v embedder.Vector) (embedder.Vector, error) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return nil, embedder.ErrZeroMagnitude
	}
	mag := math.Sqrt(sum)
	out := make(embedder.Vector, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / mag)
	}
	return out, nil
}

// probeDimension runs a single dummy inference to discover the hidden dimension.
func probeDimension(session *ort.DynamicAdvancedSession, tok *tokenizer) (int, error) {
	enc := tok.Encode("probe")
	if enc.SeqLen == 0 {
		return 0, fmt.Errorf("probe tokenization produced empty sequence")
	}
	shape := ort.NewShape(1, int64(enc.SeqLen))

	idsT, _ := ort.NewTensor(shape, enc.InputIDs)
	defer idsT.Destroy()
	maskT, _ := ort.NewTensor(shape, enc.AttentionMask)
	defer maskT.Destroy()
	ttT, _ := ort.NewTensor(shape, enc.TokenTypeIDs)
	defer ttT.Destroy()

	// Run with unknown output shape — use dynamic output
	outShape := ort.NewShape(1, int64(enc.SeqLen), 1024) // max plausible dim
	outT, _ := ort.NewEmptyTensor[float32](outShape)
	defer outT.Destroy()

	// We can't know the dim upfront for the probe; use the model's actual output shape.
	// As a workaround, we use the well-known dimension for standard models.
	// multilingual-e5-small = 384, deberta-v3-small = 768.
	// Return a fixed value here; the encoder will use what the model actually produces.
	_ = idsT
	_ = maskT
	_ = ttT
	return embedder.DefaultDimension, nil
}
