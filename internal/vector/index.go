package vector

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/memory"
)

const DefaultTopK = 8

var (
	ErrEntryIDRequired      = errors.New("entry id is required")
	ErrVectorRequired       = errors.New("vector is required")
	ErrInvalidTopK          = errors.New("top_k must be greater than zero")
	ErrInvalidMinScore      = errors.New("min_score must be between -1 and 1")
	ErrInvalidVectorKind    = errors.New("invalid vector kind")
	ErrDimensionMismatch    = errors.New("vector dimension mismatch")
	ErrIndexDimension       = errors.New("index dimension must be greater than zero")
	ErrSearchVectorRequired = errors.New("query vector is required")
)

type Entry struct {
	ID     string
	Vector embedder.Vector
	Scope  string
	Kind   memory.MemoryKind
}

func (e *Entry) Normalize() {
	if strings.TrimSpace(e.Scope) == "" {
		e.Scope = memory.DefaultScope
	}

	if e.Kind == "" {
		e.Kind = memory.MemoryKindNote
	}
}

func (e Entry) Validate(dimension int) error {
	if strings.TrimSpace(e.ID) == "" {
		return ErrEntryIDRequired
	}

	if len(e.Vector) == 0 {
		return ErrVectorRequired
	}

	if err := embedder.ValidateVectorDimension(e.Vector, dimension); err != nil {
		if errors.Is(err, embedder.ErrDimensionMismatch) {
			return ErrDimensionMismatch
		}
		return err
	}

	if !e.Kind.Valid() {
		return ErrInvalidVectorKind
	}

	return nil
}

type SearchInput struct {
	Vector   embedder.Vector
	Scope    string
	Kinds    []memory.MemoryKind
	TopK     int
	MinScore float64
}

func (in *SearchInput) Normalize() {
	if in.TopK == 0 {
		in.TopK = DefaultTopK
	}
}

func (in SearchInput) Validate(dimension int) error {
	if len(in.Vector) == 0 {
		return ErrSearchVectorRequired
	}

	if err := embedder.ValidateVectorDimension(in.Vector, dimension); err != nil {
		if errors.Is(err, embedder.ErrDimensionMismatch) {
			return ErrDimensionMismatch
		}
		return err
	}

	if in.TopK < 0 {
		return ErrInvalidTopK
	}

	if in.MinScore < -1 || in.MinScore > 1 {
		return ErrInvalidMinScore
	}

	for _, kind := range in.Kinds {
		if !kind.Valid() {
			return ErrInvalidVectorKind
		}
	}

	return nil
}

type SearchResult struct {
	ID     string
	Score  float64
	Scope  string
	Kind   memory.MemoryKind
	Vector embedder.Vector
}

type VectorIndex interface {
	Dimension() int
	Add(entry Entry) error
	Search(input SearchInput) ([]SearchResult, error)
	Remove(id string) error
	Len() int
}

type LinearVectorIndex struct {
	mu        sync.RWMutex
	dimension int
	entries   map[string]Entry
}

func NewLinearVectorIndex(dimension int) (*LinearVectorIndex, error) {
	if err := embedder.ValidateDimension(dimension); err != nil {
		return nil, ErrIndexDimension
	}

	return &LinearVectorIndex{
		dimension: dimension,
		entries:   map[string]Entry{},
	}, nil
}

func (idx *LinearVectorIndex) Dimension() int {
	return idx.dimension
}

func (idx *LinearVectorIndex) Add(entry Entry) error {
	entry.Normalize()
	if err := entry.Validate(idx.dimension); err != nil {
		return err
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	cloned := Entry{
		ID:     entry.ID,
		Vector: cloneVector(entry.Vector),
		Scope:  entry.Scope,
		Kind:   entry.Kind,
	}
	idx.entries[entry.ID] = cloned
	return nil
}

func (idx *LinearVectorIndex) Search(input SearchInput) ([]SearchResult, error) {
	input.Normalize()
	if err := input.Validate(idx.dimension); err != nil {
		return nil, err
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	results := make([]SearchResult, 0, len(idx.entries))
	for _, entry := range idx.entries {
		if !matchesScope(entry, input.Scope) {
			continue
		}

		if !matchesKind(entry, input.Kinds) {
			continue
		}

		score := CosineSimilarity(input.Vector, entry.Vector)
		if score < input.MinScore {
			continue
		}

		results = append(results, SearchResult{
			ID:     entry.ID,
			Score:  score,
			Scope:  entry.Scope,
			Kind:   entry.Kind,
			Vector: cloneVector(entry.Vector),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].ID < results[j].ID
		}
		return results[i].Score > results[j].Score
	})

	if input.TopK > 0 && len(results) > input.TopK {
		results = results[:input.TopK]
	}

	return results, nil
}

func (idx *LinearVectorIndex) Remove(id string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.entries, id)
	return nil
}

func (idx *LinearVectorIndex) Len() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.entries)
}

func CosineSimilarity(left, right embedder.Vector) float64 {
	if len(left) == 0 || len(right) == 0 || len(left) != len(right) {
		return 0
	}

	var dot float64
	var leftNorm float64
	var rightNorm float64

	for i := range left {
		lv := float64(left[i])
		rv := float64(right[i])
		dot += lv * rv
		leftNorm += lv * lv
		rightNorm += rv * rv
	}

	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}

	return dot / (sqrt(leftNorm) * sqrt(rightNorm))
}

func matchesScope(entry Entry, scope string) bool {
	if strings.TrimSpace(scope) == "" {
		return true
	}

	return entry.Scope == scope
}

func matchesKind(entry Entry, kinds []memory.MemoryKind) bool {
	if len(kinds) == 0 {
		return true
	}

	for _, kind := range kinds {
		if entry.Kind == kind {
			return true
		}
	}

	return false
}

func cloneVector(vector embedder.Vector) embedder.Vector {
	if vector == nil {
		return nil
	}

	out := make(embedder.Vector, len(vector))
	copy(out, vector)
	return out
}

func sqrt(value float64) float64 {
	// Local helper keeps the package dependency surface small.
	// The index only needs square roots for cosine normalization.
	z := value
	if z == 0 {
		return 0
	}

	x := value
	for i := 0; i < 10; i++ {
		x = 0.5 * (x + z/x)
	}
	return x
}
