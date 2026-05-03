package models

import (
	"context"
	"errors"
	"sort"
	"strings"

	coreextractor "github.com/phmotad/firememory/internal/extractor"
)

var ErrEmptyText = errors.New("models: text is required")

type HeuristicIntentClassifier struct{}

func (HeuristicIntentClassifier) ClassifyIntent(_ context.Context, input TextInput) (IntentResult, error) {
	text, err := normalizedInput(input)
	if err != nil {
		return IntentResult{}, err
	}

	switch {
	case containsAny(text, "explique", "explain", "why", "por que"):
		return IntentResult{Intent: "explain_decision", Score: 0.93}, nil
	case containsAny(text, "context", "contexto", "responder", "answer", "reply"):
		return IntentResult{Intent: "build_context", Score: 0.92}, nil
	case containsAny(text, "sync", "sincron", "consolidate", "consolidar"):
		return IntentResult{Intent: "sync_memory", Score: 0.90}, nil
	case containsAny(text, "remember", "memorize", "anote", "salve", "guarde", "lembre"):
		return IntentResult{Intent: "remember_information", Score: 0.91}, nil
	default:
		return IntentResult{Intent: "recall_information", Score: 0.80}, nil
	}
}

type HeuristicTriggerClassifier struct{}

func (HeuristicTriggerClassifier) ClassifyTrigger(_ context.Context, input TextInput) (TriggerResult, error) {
	text, err := normalizedInput(input)
	if err != nil {
		return TriggerResult{}, err
	}

	switch {
	case containsAny(text, "forget", "delete", "remove", "esquecer", "remover", "apagar"):
		return TriggerResult{Trigger: "request_confirmation", Score: 0.95}, nil
	case containsAny(text, "remember", "memorize", "anote", "salve", "guarde", "lembre"):
		return TriggerResult{Trigger: "suggest_write", Score: 0.92}, nil
	case containsAny(text, "recall", "context", "contexto", "buscar", "procure", "find", "search", "responder", "answer", "explique", "explain"):
		return TriggerResult{Trigger: "query_memory", Score: 0.85}, nil
	default:
		return TriggerResult{Trigger: "do_nothing", Score: 0.60}, nil
	}
}

type HeuristicEntityExtractor struct {
	extractor coreextractor.Extractor
}

func NewHeuristicEntityExtractor() HeuristicEntityExtractor {
	return HeuristicEntityExtractor{extractor: coreextractor.NewHeuristicExtractor()}
}

func (e HeuristicEntityExtractor) ExtractEntities(_ context.Context, input TextInput) ([]Entity, error) {
	text, err := normalizedInput(input)
	if err != nil {
		return nil, err
	}

	result, err := e.extractor.Extract(coreextractor.Input{Content: text})
	if err != nil {
		return nil, err
	}

	entities := make([]Entity, 0, len(result.Entities))
	for _, entity := range result.Entities {
		entities = append(entities, Entity{
			Text:  entity.Name,
			Type:  entity.Type,
			Score: entity.Confidence,
		})
	}
	return entities, nil
}

type HeuristicFactExtractor struct {
	extractor coreextractor.Extractor
}

func NewHeuristicFactExtractor() HeuristicFactExtractor {
	return HeuristicFactExtractor{extractor: coreextractor.NewHeuristicExtractor()}
}

func (e HeuristicFactExtractor) ExtractFacts(_ context.Context, input TextInput) ([]Fact, error) {
	text, err := normalizedInput(input)
	if err != nil {
		return nil, err
	}

	result, err := e.extractor.Extract(coreextractor.Input{Content: text})
	if err != nil {
		return nil, err
	}

	facts := make([]Fact, 0, len(result.Facts))
	for _, fact := range result.Facts {
		facts = append(facts, Fact{
			Text:  fact.Subject + " " + fact.Predicate + " " + fact.Object,
			Score: fact.Confidence,
		})
	}
	return facts, nil
}

type HeuristicRelationClassifier struct{}

func (HeuristicRelationClassifier) ClassifyRelation(_ context.Context, left TextInput, right TextInput) (RelationSuggestion, error) {
	leftText, err := normalizedInput(left)
	if err != nil {
		return RelationSuggestion{}, err
	}
	rightText, err := normalizedInput(right)
	if err != nil {
		return RelationSuggestion{}, err
	}

	if leftText == rightText {
		return RelationSuggestion{Relation: "duplicate", Score: 1.0}, nil
	}

	shared := sharedTokens(leftText, rightText)
	overlap := overlapScore(leftText, rightText)

	if overlap >= 0.88 {
		return RelationSuggestion{Relation: "reinforce", Score: overlap}, nil
	}
	if versionChanged(leftText, rightText) && len(shared) > 0 {
		return RelationSuggestion{Relation: "update", Score: maxFloat(overlap, 0.72)}, nil
	}
	if containsConflict(leftText, rightText) && len(shared) > 0 {
		return RelationSuggestion{Relation: "conflict", Score: maxFloat(overlap, 0.75)}, nil
	}
	return RelationSuggestion{Relation: "complement", Score: maxFloat(overlap, 0.55)}, nil
}

type HeuristicSimilarityEngine struct{}

func (HeuristicSimilarityEngine) ScoreCandidates(_ context.Context, input TextInput, candidates []Candidate) ([]Candidate, error) {
	text, err := normalizedInput(input)
	if err != nil {
		return nil, err
	}

	scored := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		item := candidate
		item.Score = overlapScore(text, strings.ToLower(candidate.Text))
		scored = append(scored, item)
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].ID < scored[j].ID
		}
		return scored[i].Score > scored[j].Score
	})
	return scored, nil
}

type StableReranker struct{}

func (StableReranker) Rerank(_ context.Context, input TextInput, candidates []Candidate) (RankedCandidates, error) {
	if _, err := normalizedInput(input); err != nil {
		return RankedCandidates{}, err
	}

	items := append([]Candidate(nil), candidates...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			if len(items[i].Text) == len(items[j].Text) {
				return items[i].ID < items[j].ID
			}
			return len(items[i].Text) < len(items[j].Text)
		}
		return items[i].Score > items[j].Score
	})
	return RankedCandidates{Items: items}, nil
}

func normalizedInput(input TextInput) (string, error) {
	text := strings.TrimSpace(strings.ToLower(input.Text))
	if text == "" {
		return "", ErrEmptyText
	}
	return text, nil
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func sharedTokens(left, right string) []string {
	leftSet := tokenSet(left)
	rightSet := tokenSet(right)
	shared := make([]string, 0)
	for token := range leftSet {
		if _, ok := rightSet[token]; ok {
			shared = append(shared, token)
		}
	}
	sort.Strings(shared)
	return shared
}

func overlapScore(left, right string) float64 {
	leftSet := tokenSet(left)
	rightSet := tokenSet(right)
	if len(leftSet) == 0 && len(rightSet) == 0 {
		return 0
	}

	intersection := 0
	union := len(leftSet)
	for token := range rightSet {
		if _, ok := leftSet[token]; ok {
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

func tokenSet(text string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, token := range strings.Fields(strings.TrimSpace(text)) {
		token = strings.Trim(token, ".,:;!?()[]{}\"'")
		if token == "" {
			continue
		}
		set[token] = struct{}{}
	}
	return set
}

func versionChanged(left, right string) bool {
	leftVersions := versionTokens(left)
	rightVersions := versionTokens(right)
	if len(leftVersions) == 0 || len(rightVersions) == 0 {
		return false
	}
	return strings.Join(leftVersions, "|") != strings.Join(rightVersions, "|")
}

func versionTokens(text string) []string {
	versions := make([]string, 0)
	for _, token := range strings.Fields(text) {
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

func containsConflict(left, right string) bool {
	positive := []string{"ok", "fixed", "resolved", "works", "corrigido", "resolvido", "funciona"}
	negative := []string{"error", "problem", "failed", "bug", "erro", "problema", "falha"}

	leftPositive := containsAny(left, positive...)
	leftNegative := containsAny(left, negative...)
	rightPositive := containsAny(right, positive...)
	rightNegative := containsAny(right, negative...)

	return (leftPositive && rightNegative) || (leftNegative && rightPositive)
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
