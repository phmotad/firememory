package engine

import (
	"context"
	"sort"
	"strings"

	"github.com/phmotad/firememory/internal/dedup"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
	"github.com/phmotad/firememory/internal/vector"
)

const (
	recallVectorWeight  = 0.7
	recallLexicalWeight = 0.3
)

type scoredHit struct {
	memory       memory.Memory
	vectorScore  float64
	lexicalScore float64
	reasons      []string
}

func (e *Base) Recall(input RecallInput) (RecallResult, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return RecallResult{}, err
	}

	if input.BrainPath != e.Path() {
		return RecallResult{}, ErrBrainPathMismatch
	}

	queryVector, err := e.embedder.Embed(context.Background(), input.Query)
	if err != nil {
		return RecallResult{}, err
	}

	vectorHits, err := e.vectorIndex.Search(vector.SearchInput{
		Vector: queryVector,
		Scope:  input.Scope,
		TopK:   input.TopK,
	})
	if err != nil {
		return RecallResult{}, err
	}

	queryTokens := tokenize(input.Query)
	memories, err := e.listMemories()
	if err != nil {
		return RecallResult{}, err
	}

	combined := map[string]*scoredHit{}
	trace := []string{
		"embedded recall query",
		"executed vector search",
	}

	for _, hit := range vectorHits {
		mem, err := e.loadMemory(hit.ID)
		if err != nil {
			if err == storage.ErrNotFound {
				continue
			}
			return RecallResult{}, err
		}

		combined[mem.ID] = &scoredHit{
			memory:      mem,
			vectorScore: hit.Score,
			reasons:     []string{"vector similarity"},
		}
	}

	trace = append(trace, "loaded candidate memories from vector hits")

	for _, mem := range memories {
		if !matchesRecallScope(mem, input.Scope) {
			continue
		}

		lexicalScore := lexicalSimilarity(queryTokens, mem)
		if lexicalScore == 0 {
			continue
		}

		if existing, ok := combined[mem.ID]; ok {
			existing.lexicalScore = lexicalScore
			existing.reasons = appendReason(existing.reasons, "lexical overlap")
			continue
		}

		combined[mem.ID] = &scoredHit{
			memory:       mem,
			lexicalScore: lexicalScore,
			reasons:      []string{"lexical overlap"},
		}
	}

	trace = append(trace, "executed lexical search", "combined hybrid scores")

	hits := make([]RecallHit, 0, len(combined))
	for _, hit := range combined {
		score := recallVectorWeight*hit.vectorScore + recallLexicalWeight*hit.lexicalScore
		hits = append(hits, RecallHit{
			Memory:  hit.memory,
			Score:   score,
			Reasons: append([]string{}, hit.reasons...),
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Memory.ID < hits[j].Memory.ID
		}
		return hits[i].Score > hits[j].Score
	})

	if input.TopK > 0 && len(hits) > input.TopK {
		hits = hits[:input.TopK]
	}

	if !input.IncludeTrace {
		trace = nil
	}

	return RecallResult{
		Hits:  hits,
		Trace: trace,
	}, nil
}

func (e *Base) listMemories() ([]memory.Memory, error) {
	records, err := e.Store().List(memoriesNamespace, "", 0)
	if err != nil {
		return nil, err
	}

	memories := make([]memory.Memory, 0, len(records))
	for _, record := range records {
		mem, err := e.loadMemory(record.Key)
		if err != nil {
			if err == storage.ErrNotFound {
				continue
			}
			return nil, err
		}

		memories = append(memories, mem)
	}

	return memories, nil
}

func matchesRecallScope(mem memory.Memory, scope string) bool {
	if strings.TrimSpace(scope) == "" {
		return true
	}

	return mem.Scope == scope
}

func lexicalSimilarity(queryTokens []string, mem memory.Memory) float64 {
	if len(queryTokens) == 0 {
		return 0
	}

	content := mem.NormalizedContent
	if strings.TrimSpace(content) == "" {
		content = dedup.NormalizeText(mem.Content)
	}

	memoryTokens := tokenize(content)
	if len(memoryTokens) == 0 {
		return 0
	}

	memorySet := make(map[string]struct{}, len(memoryTokens))
	for _, token := range memoryTokens {
		memorySet[token] = struct{}{}
	}

	querySet := make(map[string]struct{}, len(queryTokens))
	for _, token := range queryTokens {
		querySet[token] = struct{}{}
	}

	intersection := 0
	union := len(memorySet)
	for token := range querySet {
		if _, ok := memorySet[token]; ok {
			intersection++
			continue
		}
		union++
	}

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func tokenize(text string) []string {
	normalized := dedup.NormalizeText(text)
	if normalized == "" {
		return nil
	}

	return strings.Fields(normalized)
}

func appendReason(reasons []string, reason string) []string {
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}

	return append(reasons, reason)
}
