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
	enc := t.inner.EncodeWithOptions(text, true,
		tk.WithReturnTypeIDs(),
		tk.WithReturnAttentionMask(),
	)
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
	enc := t.inner.EncodeWithOptions(text, true,
		tk.WithReturnTypeIDs(),
		tk.WithReturnAttentionMask(),
		tk.WithReturnOffsets(),
	)
	ids := toInt64(enc.IDs)
	mask := toInt64(enc.AttentionMask)
	ttids := toInt64(enc.TypeIDs)
	wordsMask := computeWordsMask(text, enc.Offsets)
	return encodedInput{
		InputIDs:      ids,
		AttentionMask: mask,
		TokenTypeIDs:  ttids,
		SeqLen:        len(ids),
	}, wordsMask
}

// computeWordsMask returns 1 at the first subword token of each word, 0 elsewhere.
// Special tokens have zero-width offsets [0,0] and are skipped.
// A token is word-initial when its start offset is 0 (first word) or is
// immediately preceded by whitespace in the original text.
func computeWordsMask(text string, offsets []tk.Offset) []int64 {
	mask := make([]int64, len(offsets))
	for i, off := range offsets {
		start, end := int(off[0]), int(off[1])
		if start == end {
			continue // special token (zero-width)
		}
		if start == 0 || (start > 0 && isSpace(text[start-1])) {
			mask[i] = 1
		}
	}
	return mask
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func toInt64(src []uint32) []int64 {
	dst := make([]int64, len(src))
	for i, v := range src {
		dst[i] = int64(v)
	}
	return dst
}
