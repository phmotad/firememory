//go:build onnx

package onnx

import (
	"fmt"
	"path/filepath"

	tk "github.com/daulet/tokenizers"
)

// tokenizer wraps a HuggingFace fast tokenizer loaded from tokenizer.json.
type tokenizer struct {
	inner *tk.Tokenizer
}

func loadTokenizer(modelDir string) (*tokenizer, error) {
	path := filepath.Join(modelDir, "tokenizer.json")
	t, err := tk.FromFile(path)
	if err != nil {
		return nil, fmt.Errorf("onnx: load tokenizer from %s: %w", path, err)
	}
	return &tokenizer{inner: t}, nil
}

func (t *tokenizer) Close() {
	t.inner.Close()
}

// encodedInput holds the tokenized representation ready for ONNX input tensors.
type encodedInput struct {
	InputIDs      []int64
	AttentionMask []int64
	TokenTypeIDs  []int64
	SeqLen        int
}

// Encode tokenizes text and returns int64 slices matching ONNX int64 tensor types.
func (t *tokenizer) Encode(text string) encodedInput {
	enc := t.inner.Encode(text, true)

	ids := toInt64(enc.IDs)
	mask := toInt64(enc.AttentionMask)
	ttids := toInt64(enc.TypeIDs)

	return encodedInput{
		InputIDs:      ids,
		AttentionMask: mask,
		TokenTypeIDs:  ttids,
		SeqLen:        len(ids),
	}
}

// EncodeWords tokenizes text preserving word-to-subword alignment.
// Returns the encodedInput plus a words_mask (1 at the first subword of each word, 0 elsewhere).
func (t *tokenizer) EncodeWords(text string) (encodedInput, []int64) {
	enc := t.inner.Encode(text, true)

	ids := toInt64(enc.IDs)
	mask := toInt64(enc.AttentionMask)
	ttids := toInt64(enc.TypeIDs)

	wordsMask := computeWordsMask(enc.WordIDs)

	return encodedInput{
		InputIDs:      ids,
		AttentionMask: mask,
		TokenTypeIDs:  ttids,
		SeqLen:        len(ids),
	}, wordsMask
}

// computeWordsMask returns 1 at the first subword of each word, 0 elsewhere.
// WordIDs maps each token position to its originating word index (or -1 for special tokens).
func computeWordsMask(wordIDs []int) []int64 {
	mask := make([]int64, len(wordIDs))
	seen := map[int]bool{}
	for i, wid := range wordIDs {
		if wid < 0 {
			continue
		}
		if !seen[wid] {
			seen[wid] = true
			mask[i] = 1
		}
	}
	return mask
}

func toInt64(src []uint32) []int64 {
	dst := make([]int64, len(src))
	for i, v := range src {
		dst[i] = int64(v)
	}
	return dst
}
