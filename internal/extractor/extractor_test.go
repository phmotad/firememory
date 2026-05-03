package extractor

import (
	"errors"
	"strings"
	"testing"
)

func TestHeuristicExtractorExtractsVersionsNamesTermsAndKeywords(t *testing.T) {
	ex := NewHeuristicExtractor()

	result, err := ex.Extract(Input{
		MemoryID: "mem_01",
		Content:  "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2",
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	if len(result.Trace) == 0 {
		t.Fatal("expected trace entries")
	}

	assertEntity(t, result.Entities, "2.5", "version")
	assertEntity(t, result.Entities, "3.2", "version")
	assertEntity(t, result.Entities, "Joao", "proper_name")
	assertEntity(t, result.Entities, "firebird", "technical_term")
	assertEntity(t, result.Entities, "nf-e", "technical_term")

	if len(result.Facts) == 0 {
		t.Fatal("expected extracted facts")
	}

	joinedKeywords := strings.Join(result.Keywords, "|")
	if !strings.Contains(joinedKeywords, "firebird") {
		t.Fatalf("expected keywords to include firebird, got %v", result.Keywords)
	}
}

func TestHeuristicExtractorDedupeResults(t *testing.T) {
	ex := NewHeuristicExtractor()

	result, err := ex.Extract(Input{
		MemoryID: "mem_02",
		Content:  "Joao Joao atualizou para versao 3.2 e versao 3.2 no Firebird",
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	versionCount := 0
	nameCount := 0
	for _, entity := range result.Entities {
		if entity.Type == "version" && entity.Name == "3.2" {
			versionCount++
		}
		if entity.Type == "proper_name" && entity.Name == "Joao" {
			nameCount++
		}
	}

	if versionCount != 1 {
		t.Fatalf("expected deduped version entity, got %d", versionCount)
	}

	if nameCount != 1 {
		t.Fatalf("expected deduped name entity, got %d", nameCount)
	}
}

func TestGLiNERExtractorReturnsUnavailableInMVP(t *testing.T) {
	ex := NewGLiNERExtractor()

	_, err := ex.Extract(Input{
		Content: "Cliente Joao usa Firebird 2.5",
	})
	if !errors.Is(err, ErrGLiNERUnavailable) {
		t.Fatalf("expected ErrGLiNERUnavailable, got %v", err)
	}
}

func TestInputValidation(t *testing.T) {
	ex := NewHeuristicExtractor()

	_, err := ex.Extract(Input{})
	if !errors.Is(err, ErrEmptyContent) {
		t.Fatalf("expected ErrEmptyContent, got %v", err)
	}
}

func assertEntity(t *testing.T, entities []ExtractedEntity, name, typ string) {
	t.Helper()

	for _, entity := range entities {
		if entity.Name == name && entity.Type == typ {
			return
		}
	}

	t.Fatalf("expected entity %q of type %q, got %+v", name, typ, entities)
}
