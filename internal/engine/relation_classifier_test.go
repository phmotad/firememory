package engine

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/phmotad/firememory/internal/memory"
)

func TestRelationClassifierDuplicateByHash(t *testing.T) {
	classifier := NewHeuristicMemoryRelationClassifier()

	result, err := classifier.Classify(RelationClassificationInput{
		Left:  testMemory("mem_01", "Cliente Joao usa Firebird 2.5", "same_hash", "support"),
		Right: testMemory("mem_02", "Cliente Joao usa Firebird 2.5", "same_hash", "support"),
	})
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	if result.Type != memory.RelationTypeDuplicate {
		t.Fatalf("expected duplicate, got %q", result.Type)
	}
}

func TestRelationClassifierReinforce(t *testing.T) {
	classifier := NewHeuristicMemoryRelationClassifier()

	result, err := classifier.Classify(RelationClassificationInput{
		Left:            testMemory("mem_01", "Cliente Joao relatou erro fiscal na NF-e", "hash_1", "support"),
		Right:           testMemory("mem_02", "Joao relatou erro fiscal em nota fiscal eletronica", "hash_2", "support"),
		SimilarityScore: 0.91,
	})
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	if result.Type != memory.RelationTypeReinforce {
		t.Fatalf("expected reinforce, got %q", result.Type)
	}
}

func TestRelationClassifierUpdate(t *testing.T) {
	classifier := NewHeuristicMemoryRelationClassifier()

	result, err := classifier.Classify(RelationClassificationInput{
		Left:            testMemory("mem_01", "Cliente Joao usa Firebird 2.5", "hash_1", "support"),
		Right:           testMemory("mem_02", "Cliente Joao usa Firebird 3.2", "hash_2", "support"),
		SimilarityScore: 0.60,
	})
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	if result.Type != memory.RelationTypeUpdate {
		t.Fatalf("expected update, got %q", result.Type)
	}
}

func TestRelationClassifierConflict(t *testing.T) {
	classifier := NewHeuristicMemoryRelationClassifier()

	result, err := classifier.Classify(RelationClassificationInput{
		Left:            testMemory("mem_01", "Cliente Joao relatou erro fiscal na NF-e", "hash_1", "support"),
		Right:           testMemory("mem_02", "Cliente Joao informou NF-e resolvida e funcionando", "hash_2", "support"),
		SimilarityScore: 0.45,
	})
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	if result.Type != memory.RelationTypeConflict {
		t.Fatalf("expected conflict, got %q", result.Type)
	}
}

func TestRelationClassifierComplement(t *testing.T) {
	classifier := NewHeuristicMemoryRelationClassifier()

	result, err := classifier.Classify(RelationClassificationInput{
		Left:            testMemory("mem_01", "Cliente Joao usa Firebird 2.5", "hash_1", "support"),
		Right:           testMemory("mem_02", "Servidor Linux faz backup noturno", "hash_2", "support"),
		SimilarityScore: 0.22,
	})
	if err != nil {
		t.Fatalf("classify: %v", err)
	}

	if result.Type != memory.RelationTypeComplement {
		t.Fatalf("expected complement, got %q", result.Type)
	}
}

func TestRelationClassifierInputValidation(t *testing.T) {
	classifier := NewHeuristicMemoryRelationClassifier()

	_, err := classifier.Classify(RelationClassificationInput{
		Left:            testMemory("mem_01", "Cliente Joao usa Firebird 2.5", "hash_1", "support"),
		Right:           testMemory("mem_02", "Cliente Joao usa Firebird 3.2", "hash_2", "support"),
		SimilarityScore: 1.5,
	})
	if !errors.Is(err, ErrInvalidSimilarityScore) {
		t.Fatalf("expected ErrInvalidSimilarityScore, got %v", err)
	}
}

func testMemory(id, content, hash, scope string) memory.Memory {
	now := time.Now().UTC()
	mem := memory.Memory{
		ID:                id,
		Content:           content,
		NormalizedContent: strings.ToLower(content),
		Hash:              hash,
		Kind:              memory.MemoryKindNote,
		Status:            memory.MemoryStatusPendingSync,
		Scope:             scope,
		Importance:        0.5,
		Confidence:        1.0,
		EmbeddingDim:      384,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	mem.Normalize()
	return mem
}
