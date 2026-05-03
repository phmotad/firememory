package extractor

import (
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/phmotad/firememory/internal/dedup"
)

var (
	ErrEmptyContent       = errors.New("content is required")
	ErrGLiNERUnavailable  = errors.New("gliner extractor is not available in the local-first MVP")
	versionPattern        = regexp.MustCompile(`(?i)\b(?:v(?:ersion)?\s*)?(\d+(?:\.\d+){1,3})\b`)
	technicalTokenPattern = regexp.MustCompile(`(?i)\b[a-z][a-z0-9._+-]{2,}\b`)
	properNamePattern     = regexp.MustCompile(`\b[A-Z][a-z]{2,}(?:\s+[A-Z][a-z]{2,})*\b`)
)

var stopWords = map[string]struct{}{
	"a": {}, "apos": {}, "após": {}, "com": {}, "da": {}, "das": {}, "de": {}, "depois": {}, "do": {}, "dos": {},
	"e": {}, "em": {}, "erro": {}, "fiscal": {}, "na": {}, "no": {}, "nos": {}, "nova": {}, "novo": {}, "o": {},
	"os": {}, "ou": {}, "para": {}, "por": {}, "que": {}, "relatou": {}, "sobre": {}, "teve": {}, "uma": {}, "um": {},
}

var technicalTerms = map[string]struct{}{
	"api": {}, "backup": {}, "firebird": {}, "firememory": {}, "gliner": {}, "http": {}, "json": {}, "mcp": {},
	"nfe": {}, "nf-e": {}, "postgres": {}, "sql": {}, "sync": {}, "vector": {}, "versao": {}, "versão": {},
}

var nonEntityNameWords = map[string]struct{}{ //nolint:misspell
	"Cliente": {}, "Clientes": {}, "Servidor": {}, "Sistema": {}, "Versao": {}, "Versão": {}, //nolint:misspell
}

type Extractor interface {
	Extract(input Input) (Result, error)
}

type Input struct {
	MemoryID string
	Content  string
	Scope    string
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.Content) == "" {
		return ErrEmptyContent
	}

	return nil
}

type Result struct {
	Entities []ExtractedEntity
	Facts    []ExtractedFact
	Keywords []string
	Trace    []string
}

type ExtractedEntity struct {
	Name       string
	Type       string
	Confidence float64
}

type ExtractedFact struct {
	Subject    string
	Predicate  string
	Object     string
	Confidence float64
}

type HeuristicExtractor struct{}

func NewHeuristicExtractor() *HeuristicExtractor {
	return &HeuristicExtractor{}
}

func (e *HeuristicExtractor) Extract(input Input) (Result, error) {
	if err := input.Validate(); err != nil {
		return Result{}, err
	}

	normalized := dedup.NormalizeText(input.Content)
	nowKeywords := extractKeywords(normalized)
	versions := extractVersions(input.Content)
	names := extractProperNames(input.Content)
	tech := extractTechnicalTerms(normalized)

	entities := make([]ExtractedEntity, 0, len(versions)+len(names)+len(tech))
	facts := make([]ExtractedFact, 0, len(versions)+len(names))
	trace := []string{
		"normalized extraction input",
		"extracted versions",
		"extracted proper names",
		"extracted technical terms",
		"extracted keywords",
	}

	for _, version := range versions {
		entities = append(entities, ExtractedEntity{
			Name:       version,
			Type:       "version",
			Confidence: 0.95,
		})
		facts = append(facts, ExtractedFact{
			Subject:    input.MemoryIDOrDefault(),
			Predicate:  "mentions_version",
			Object:     version,
			Confidence: 0.9,
		})
	}

	for _, name := range names {
		entities = append(entities, ExtractedEntity{
			Name:       name,
			Type:       "proper_name",
			Confidence: 0.8,
		})
		facts = append(facts, ExtractedFact{
			Subject:    input.MemoryIDOrDefault(),
			Predicate:  "mentions_entity",
			Object:     name,
			Confidence: 0.75,
		})
	}

	for _, term := range tech {
		entities = append(entities, ExtractedEntity{
			Name:       term,
			Type:       "technical_term",
			Confidence: 0.85,
		})
	}

	return Result{
		Entities: dedupeEntities(entities),
		Facts:    dedupeFacts(facts),
		Keywords: nowKeywords,
		Trace:    trace,
	}, nil
}

type GLiNERExtractor struct{}

func NewGLiNERExtractor() *GLiNERExtractor {
	return &GLiNERExtractor{}
}

func (e *GLiNERExtractor) Extract(input Input) (Result, error) {
	if err := input.Validate(); err != nil {
		return Result{}, err
	}

	return Result{}, ErrGLiNERUnavailable
}

func (in Input) MemoryIDOrDefault() string {
	if strings.TrimSpace(in.MemoryID) == "" {
		return "memory"
	}

	return in.MemoryID
}

func extractVersions(content string) []string {
	matches := versionPattern.FindAllStringSubmatch(content, -1)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		version := match[1]
		if _, ok := seen[version]; ok {
			continue
		}
		seen[version] = struct{}{}
		out = append(out, version)
	}
	sort.Strings(out)
	return out
}

func extractProperNames(content string) []string {
	matches := properNamePattern.FindAllString(content, -1)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		for _, candidate := range normalizeProperNameMatch(match) {
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			out = append(out, candidate)
		}
	}
	sort.Strings(out)
	return out
}

func normalizeProperNameMatch(match string) []string {
	cleaned := strings.TrimSpace(match)
	if cleaned == "" {
		return nil
	}

	parts := strings.Fields(cleaned)
	if len(parts) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if _, blocked := nonEntityNameWords[part]; blocked {
			continue
		}

		if _, technical := technicalTerms[strings.ToLower(part)]; technical {
			continue
		}

		filtered = append(filtered, part)
	}

	if len(filtered) == 0 {
		return nil
	}

	if len(filtered) == 1 {
		return filtered
	}

	allSame := true
	for i := 1; i < len(filtered); i++ {
		if filtered[i] != filtered[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return []string{filtered[0]}
	}

	return []string{strings.Join(filtered, " ")}
}

func extractTechnicalTerms(normalized string) []string {
	tokens := strings.Fields(normalized)
	seen := map[string]struct{}{}
	out := make([]string, 0)

	for _, token := range tokens {
		if _, ok := technicalTerms[token]; ok {
			if _, seenAlready := seen[token]; !seenAlready {
				seen[token] = struct{}{}
				out = append(out, token)
			}
			continue
		}

		if technicalTokenPattern.MatchString(token) && hasDigitOrPunctuation(token) {
			if _, seenAlready := seen[token]; !seenAlready {
				seen[token] = struct{}{}
				out = append(out, token)
			}
		}
	}

	sort.Strings(out)
	return out
}

func extractKeywords(normalized string) []string {
	tokens := strings.Fields(normalized)
	counts := map[string]int{}
	order := map[string]int{}

	for i, token := range tokens {
		token = strings.Trim(token, ".,:;!?()[]{}\"'")
		if token == "" {
			continue
		}

		if _, stop := stopWords[token]; stop {
			continue
		}

		if _, seen := order[token]; !seen {
			order[token] = i
		}
		counts[token]++
	}

	keywords := make([]string, 0, len(counts))
	for token := range counts {
		keywords = append(keywords, token)
	}

	sort.Slice(keywords, func(i, j int) bool {
		if counts[keywords[i]] == counts[keywords[j]] {
			return order[keywords[i]] < order[keywords[j]]
		}
		return counts[keywords[i]] > counts[keywords[j]]
	})

	if len(keywords) > 8 {
		keywords = keywords[:8]
	}

	return keywords
}

func hasDigitOrPunctuation(token string) bool {
	for _, r := range token {
		if r >= '0' && r <= '9' {
			return true
		}
		switch r {
		case '.', '-', '_', '+', '/':
			return true
		}
	}

	return false
}

func dedupeEntities(entities []ExtractedEntity) []ExtractedEntity {
	type key struct {
		name string
		typ  string
	}

	seen := map[key]ExtractedEntity{}
	for _, entity := range entities {
		k := key{name: entity.Name, typ: entity.Type}
		existing, ok := seen[k]
		if !ok || entity.Confidence > existing.Confidence {
			seen[k] = entity
		}
	}

	out := make([]ExtractedEntity, 0, len(seen))
	for _, entity := range seen {
		out = append(out, entity)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Type == out[j].Type {
			return out[i].Name < out[j].Name
		}
		return out[i].Type < out[j].Type
	})

	return out
}

func dedupeFacts(facts []ExtractedFact) []ExtractedFact {
	type key struct {
		subject   string
		predicate string
		object    string
	}

	seen := map[key]ExtractedFact{}
	for _, fact := range facts {
		k := key{
			subject:   fact.Subject,
			predicate: fact.Predicate,
			object:    fact.Object,
		}
		existing, ok := seen[k]
		if !ok || fact.Confidence > existing.Confidence {
			seen[k] = fact
		}
	}

	out := make([]ExtractedFact, 0, len(seen))
	for _, fact := range seen {
		out = append(out, fact)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Predicate == out[j].Predicate {
			if out[i].Subject == out[j].Subject {
				return out[i].Object < out[j].Object
			}
			return out[i].Subject < out[j].Subject
		}
		return out[i].Predicate < out[j].Predicate
	})

	return out
}
