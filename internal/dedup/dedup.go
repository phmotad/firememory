package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/vector"
)

const DefaultVectorThreshold = 0.92

var (
	ErrContentRequired        = errors.New("content is required")
	ErrInvalidVectorThreshold = errors.New("vector threshold must be between 0 and 1")
)

type MatchType string

const (
	MatchTypeNone   MatchType = "none"
	MatchTypeExact  MatchType = "exact"
	MatchTypeVector MatchType = "vector"
)

type HashIndex interface {
	Get(hash string) (memoryID string, ok bool)
	Set(hash, memoryID string)
}

type InMemoryHashIndex struct {
	items map[string]string
}

func NewInMemoryHashIndex() *InMemoryHashIndex {
	return &InMemoryHashIndex{
		items: map[string]string{},
	}
}

func (idx *InMemoryHashIndex) Get(hash string) (string, bool) {
	memoryID, ok := idx.items[hash]
	return memoryID, ok
}

func (idx *InMemoryHashIndex) Set(hash, memoryID string) {
	if idx.items == nil {
		idx.items = map[string]string{}
	}

	idx.items[hash] = memoryID
}

type Config struct {
	VectorThreshold float64
}

func (c *Config) Normalize() {
	if c.VectorThreshold == 0 {
		c.VectorThreshold = DefaultVectorThreshold
	}
}

func (c Config) Validate() error {
	if c.VectorThreshold < 0 || c.VectorThreshold > 1 {
		return ErrInvalidVectorThreshold
	}

	return nil
}

type Detector struct {
	config Config
}

func NewDetector(config Config) (*Detector, error) {
	config.Normalize()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Detector{config: config}, nil
}

type Input struct {
	Content   string
	Scope     string
	Kind      memory.MemoryKind
	Vector    embedder.Vector
	HashIndex HashIndex
	Index     vector.VectorIndex
}

func (in *Input) Normalize() {
	if in.Kind == "" {
		in.Kind = memory.MemoryKindNote
	}

	if strings.TrimSpace(in.Scope) == "" {
		in.Scope = memory.DefaultScope
	}
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.Content) == "" {
		return ErrContentRequired
	}

	if in.Kind != "" && !in.Kind.Valid() {
		return memory.ErrInvalidMemoryKind
	}

	return nil
}

type Result struct {
	Action          memory.DedupAction
	MatchType       MatchType
	MatchedMemoryID string
	NormalizedText  string
	Hash            string
	SimilarityScore float64
	Trace           []string
}

func (d *Detector) Detect(input Input) (Result, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return Result{}, err
	}

	normalized := NormalizeText(input.Content)
	hash := HashNormalized(normalized)

	result := Result{
		Action:         memory.DedupActionCreateNew,
		MatchType:      MatchTypeNone,
		NormalizedText: normalized,
		Hash:           hash,
		Trace: []string{
			"normalized content",
			"generated content hash",
		},
	}

	if input.HashIndex != nil {
		if memoryID, ok := input.HashIndex.Get(hash); ok {
			result.Action = memory.DedupActionReinforce
			result.MatchType = MatchTypeExact
			result.MatchedMemoryID = memoryID
			result.Trace = append(result.Trace, "exact hash match found")
			return result, nil
		}
	}

	result.Trace = append(result.Trace, "no exact hash match found")

	if input.Index == nil || len(input.Vector) == 0 {
		result.Trace = append(result.Trace, "vector dedup skipped")
		return result, nil
	}

	hits, err := input.Index.Search(vector.SearchInput{
		Vector:   input.Vector,
		Scope:    input.Scope,
		Kinds:    []memory.MemoryKind{input.Kind},
		TopK:     1,
		MinScore: d.config.VectorThreshold,
	})
	if err != nil {
		return Result{}, err
	}

	if len(hits) == 0 {
		result.Trace = append(result.Trace, "no vector match above threshold")
		return result, nil
	}

	result.Action = memory.DedupActionReinforce
	result.MatchType = MatchTypeVector
	result.MatchedMemoryID = hits[0].ID
	result.SimilarityScore = hits[0].Score
	result.Trace = append(result.Trace, "vector match found above threshold")
	return result, nil
}

func NormalizeText(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	parts := strings.Fields(strings.ToLower(trimmed))
	return strings.Join(parts, " ")
}

func HashNormalized(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
