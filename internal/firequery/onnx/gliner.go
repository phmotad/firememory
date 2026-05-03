//go:build onnx

package onnx

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phmotad/firememory/internal/firequery/models"
	ort "github.com/yalue/onnxruntime_go"
)

// GLiNER ONNX tensor names (gliner_small-v2.1 export).
// Entity type tokens are prepended to the text: [type1] [type2] ... [SEP] [text tokens]
const (
	glinerInputIDs     = "input_ids"
	glinerAttentionMask = "attention_mask"
	glinerWordsMask    = "words_mask"
	glinerTextLengths  = "text_lengths"
	glinerOutput       = "logits"

	// Threshold below which a span prediction is discarded.
	glinerScoreThreshold = 0.5
	// Maximum span length in words.
	glinerMaxSpanLen = 12
)

// glinerExtractor runs GLiNER span-based NER via ONNX Runtime.
type glinerExtractor struct {
	session      *ort.DynamicAdvancedSession
	tokenizer    *tokenizer
	entityLabels []string
}

func newGLiNER(modelDir string, entityLabels []string) (*glinerExtractor, error) {
	tok, err := loadTokenizer(modelDir)
	if err != nil {
		return nil, err
	}

	modelPath := filepath.Join(modelDir, "model.onnx")
	session, err := newDynamicSession(modelPath,
		[]string{glinerInputIDs, glinerAttentionMask, glinerWordsMask, glinerTextLengths},
		[]string{glinerOutput},
	)
	if err != nil {
		tok.Close()
		return nil, fmt.Errorf("onnx: GLiNER session: %w", err)
	}

	return &glinerExtractor{
		session:      session,
		tokenizer:    tok,
		entityLabels: entityLabels,
	}, nil
}

func (g *glinerExtractor) Close() error {
	g.tokenizer.Close()
	return g.session.Destroy()
}

// ExtractEntities implements models.EntityExtractionClient.
func (g *glinerExtractor) ExtractEntities(_ context.Context, _ string, input models.TextInput) ([]models.Entity, error) {
	if input.Text == "" {
		return nil, nil
	}

	// Build joint input: prepend entity type tokens to the text.
	// Format: "[type1] [type2] ... [SEP] [text]"
	// GLiNER v2.x places entity type tokens at the beginning separated by [SEP].
	prompt := strings.Join(g.entityLabels, " ") + " [SEP] " + input.Text

	enc, wordsMaskRaw := g.tokenizer.EncodeWords(prompt)
	if enc.SeqLen == 0 {
		return nil, nil
	}

	// text_lengths: number of words in the text portion (after the entity type prompt).
	// We compute this by counting words in input.Text only.
	textWords := countWords(input.Text)

	seqLen := int64(enc.SeqLen)
	batchShape := ort.NewShape(1, seqLen)
	scalarShape := ort.NewShape(1)

	idsT, err := ort.NewTensor(batchShape, enc.InputIDs)
	if err != nil {
		return nil, fmt.Errorf("onnx: gliner input_ids: %w", err)
	}
	defer idsT.Destroy()

	maskT, err := ort.NewTensor(batchShape, enc.AttentionMask)
	if err != nil {
		return nil, fmt.Errorf("onnx: gliner attention_mask: %w", err)
	}
	defer maskT.Destroy()

	wmT, err := ort.NewTensor(batchShape, wordsMaskRaw)
	if err != nil {
		return nil, fmt.Errorf("onnx: gliner words_mask: %w", err)
	}
	defer wmT.Destroy()

	tlT, err := ort.NewTensor(scalarShape, []int64{int64(textWords)})
	if err != nil {
		return nil, fmt.Errorf("onnx: gliner text_lengths: %w", err)
	}
	defer tlT.Destroy()

	// Output logits: [1, maxWords, maxWords, numEntityTypes]
	numTypes := int64(len(g.entityLabels))
	maxWords := int64(textWords)
	if maxWords == 0 {
		return nil, nil
	}
	outShape := ort.NewShape(1, maxWords, maxWords, numTypes)
	outT, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		return nil, fmt.Errorf("onnx: gliner output tensor: %w", err)
	}
	defer outT.Destroy()

	err = g.session.Run(
		[]ort.ArbitraryTensor{idsT, maskT, wmT, tlT},
		[]ort.ArbitraryTensor{outT},
	)
	if err != nil {
		return nil, fmt.Errorf("onnx: gliner run: %w", err)
	}

	logits := outT.GetData() // [maxWords * maxWords * numTypes] flattened
	return decodeSpans(logits, input.Text, g.entityLabels, int(maxWords)), nil
}

// decodeSpans extracts (start_word, end_word, entity_type) triples above the score threshold.
func decodeSpans(logits []float32, text string, labels []string, maxWords int) []models.Entity {
	words := strings.Fields(text)
	numTypes := len(labels)
	var entities []models.Entity

	for startW := 0; startW < maxWords; startW++ {
		for endW := startW; endW < maxWords && endW-startW < glinerMaxSpanLen; endW++ {
			for k, label := range labels {
				idx := (startW*maxWords+endW)*numTypes + k
				if idx >= len(logits) {
					continue
				}
				score := logits[idx]
				if float64(score) < glinerScoreThreshold {
					continue
				}

				// Reconstruct span text from word boundaries.
				spanWords := words[startW : endW+1]
				spanText := strings.Join(spanWords, " ")

				entities = append(entities, models.Entity{
					Text:  spanText,
					Type:  label,
					Score: float64(score),
				})
			}
		}
	}

	return deduplicateEntities(entities)
}

// deduplicateEntities keeps the highest-scoring span for each (text, type) pair.
func deduplicateEntities(entities []models.Entity) []models.Entity {
	best := map[string]models.Entity{}
	for _, e := range entities {
		key := e.Text + "|" + e.Type
		if prev, ok := best[key]; !ok || e.Score > prev.Score {
			best[key] = e
		}
	}
	out := make([]models.Entity, 0, len(best))
	for _, e := range best {
		out = append(out, e)
	}
	return out
}

func countWords(text string) int {
	return len(strings.Fields(text))
}
