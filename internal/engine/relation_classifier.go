package engine

import (
	"errors"
	"sort"
	"strings"

	"github.com/phmotad/firememory/internal/dedup"
	"github.com/phmotad/firememory/internal/memory"
)

const (
	defaultReinforceThreshold = 0.88
	defaultDuplicateThreshold = 0.98
	defaultConflictThreshold  = 0.30
	defaultUpdateThreshold    = 0.45
)

var (
	ErrInvalidSimilarityScore = errors.New("similarity score must be between 0 and 1")
)

var positiveCues = map[string]struct{}{
	"ok": {}, "corrigido": {}, "corrigida": {}, "resolvido": {}, "resolvida": {}, "funciona": {}, "funcionando": {},
	"sucesso": {}, "normalizado": {}, "estavel": {}, "estável": {},
}

var negativeCues = map[string]struct{}{
	"erro": {}, "falha": {}, "problema": {}, "quebra": {}, "quebrou": {}, "bug": {}, "instavel": {}, "instável": {},
	"nao": {}, "não": {}, "incorreto": {}, "invalido": {}, "inválido": {},
}

type MemoryRelationClassifier interface {
	Classify(input RelationClassificationInput) (RelationClassificationResult, error)
}

type RelationClassificationInput struct {
	Left            memory.Memory
	Right           memory.Memory
	SimilarityScore float64
}

func (in RelationClassificationInput) Validate() error {
	if in.SimilarityScore < 0 || in.SimilarityScore > 1 {
		return ErrInvalidSimilarityScore
	}

	if err := in.Left.Validate(); err != nil {
		return err
	}

	if err := in.Right.Validate(); err != nil {
		return err
	}

	return nil
}

type RelationClassificationResult struct {
	Type       memory.RelationType
	Confidence float64
	Reasons    []string
}

type HeuristicMemoryRelationClassifier struct {
	duplicateThreshold float64
	reinforceThreshold float64
	conflictThreshold  float64
	updateThreshold    float64
}

func NewHeuristicMemoryRelationClassifier() *HeuristicMemoryRelationClassifier {
	return &HeuristicMemoryRelationClassifier{
		duplicateThreshold: defaultDuplicateThreshold,
		reinforceThreshold: defaultReinforceThreshold,
		conflictThreshold:  defaultConflictThreshold,
		updateThreshold:    defaultUpdateThreshold,
	}
}

func (c *HeuristicMemoryRelationClassifier) Classify(input RelationClassificationInput) (RelationClassificationResult, error) {
	if err := input.Validate(); err != nil {
		return RelationClassificationResult{}, err
	}

	leftText := normalizedMemoryText(input.Left)
	rightText := normalizedMemoryText(input.Right)
	leftTokens := tokenSet(leftText)
	rightTokens := tokenSet(rightText)
	overlap := jaccard(leftTokens, rightTokens)
	sharedKeywords := sharedTerms(leftTokens, rightTokens)

	if input.Left.Hash != "" && input.Left.Hash == input.Right.Hash {
		return RelationClassificationResult{
			Type:       memory.RelationTypeDuplicate,
			Confidence: 1.0,
			Reasons:    []string{"same content hash"},
		}, nil
	}

	if leftText == rightText {
		return RelationClassificationResult{
			Type:       memory.RelationTypeDuplicate,
			Confidence: 0.99,
			Reasons:    []string{"same normalized content"},
		}, nil
	}

	if input.SimilarityScore >= c.duplicateThreshold && overlap >= 0.85 {
		return RelationClassificationResult{
			Type:       memory.RelationTypeDuplicate,
			Confidence: average(input.SimilarityScore, overlap),
			Reasons:    []string{"very high similarity", "strong lexical overlap"},
		}, nil
	}

	if isConflict(leftTokens, rightTokens, sharedKeywords) && (overlap >= c.conflictThreshold || len(sharedKeywords) >= 2) {
		return RelationClassificationResult{
			Type:       memory.RelationTypeConflict,
			Confidence: maxFloat(0.75, overlap),
			Reasons:    []string{"shared topic with opposing cues"},
		}, nil
	}

	if isUpdate(input.Left, input.Right, overlap, sharedKeywords) {
		return RelationClassificationResult{
			Type:       memory.RelationTypeUpdate,
			Confidence: maxFloat(0.7, overlap),
			Reasons:    []string{"shared topic with changed detail"},
		}, nil
	}

	if input.SimilarityScore >= c.reinforceThreshold || overlap >= c.reinforceThreshold {
		return RelationClassificationResult{
			Type:       memory.RelationTypeReinforce,
			Confidence: maxFloat(input.SimilarityScore, overlap),
			Reasons:    []string{"high semantic or lexical similarity"},
		}, nil
	}

	return RelationClassificationResult{
		Type:       memory.RelationTypeComplement,
		Confidence: maxFloat(0.55, overlap),
		Reasons:    []string{"related memory adds adjacent detail"},
	}, nil
}

func normalizedMemoryText(mem memory.Memory) string {
	if strings.TrimSpace(mem.NormalizedContent) != "" {
		return mem.NormalizedContent
	}

	return dedup.NormalizeText(mem.Content)
}

func tokenSet(normalized string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, token := range strings.Fields(normalized) {
		if token == "" {
			continue
		}
		set[token] = struct{}{}
	}
	return set
}

func sharedTerms(left, right map[string]struct{}) []string {
	terms := make([]string, 0)
	for token := range left {
		if _, ok := right[token]; ok {
			terms = append(terms, token)
		}
	}
	sort.Strings(terms)
	return terms
}

func jaccard(left, right map[string]struct{}) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}

	intersection := 0
	union := len(left)
	for token := range right {
		if _, ok := left[token]; ok {
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

func isConflict(leftTokens, rightTokens map[string]struct{}, sharedKeywords []string) bool {
	if len(sharedKeywords) == 0 {
		return false
	}

	leftPositive, leftNegative := sentimentFlags(leftTokens)
	rightPositive, rightNegative := sentimentFlags(rightTokens)

	return (leftPositive && rightNegative) || (leftNegative && rightPositive)
}

func isUpdate(left, right memory.Memory, overlap float64, sharedKeywords []string) bool {
	if overlap < defaultUpdateThreshold || len(sharedKeywords) == 0 {
		return false
	}

	if left.Scope != right.Scope {
		return false
	}

	leftVersions := versionTokens(normalizedMemoryText(left))
	rightVersions := versionTokens(normalizedMemoryText(right))
	if len(leftVersions) == 0 || len(rightVersions) == 0 {
		return false
	}

	return strings.Join(leftVersions, "|") != strings.Join(rightVersions, "|")
}

func versionTokens(normalized string) []string {
	tokens := strings.Fields(normalized)
	versions := make([]string, 0)
	for _, token := range tokens {
		hasDigit := false
		hasDot := false
		for _, r := range token {
			if r >= '0' && r <= '9' {
				hasDigit = true
			}
			if r == '.' {
				hasDot = true
			}
		}
		if hasDigit && hasDot {
			versions = append(versions, token)
		}
	}
	sort.Strings(versions)
	return versions
}

func sentimentFlags(tokens map[string]struct{}) (positive bool, negative bool) {
	for token := range tokens {
		if _, ok := positiveCues[token]; ok {
			positive = true
		}
		if _, ok := negativeCues[token]; ok {
			negative = true
		}
	}
	return positive, negative
}

func average(left, right float64) float64 {
	return (left + right) / 2
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
