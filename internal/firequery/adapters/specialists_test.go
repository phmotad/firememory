package adapters

import (
	"context"
	"testing"

	coreengine "github.com/phmotad/firememory/internal/engine"
	coreextractor "github.com/phmotad/firememory/internal/extractor"
	"github.com/phmotad/firememory/internal/firequery/models"
)

func TestExtractorAdapter(t *testing.T) {
	t.Parallel()

	adapter := ExtractorAdapter{Extractor: coreextractor.NewHeuristicExtractor()}

	entities, err := adapter.ExtractEntities(context.Background(), models.TextInput{Text: "Cliente Joao usa Firebird 2.5"})
	if err != nil {
		t.Fatalf("ExtractEntities() error = %v", err)
	}
	if len(entities) == 0 {
		t.Fatal("expected entities")
	}

	facts, err := adapter.ExtractFacts(context.Background(), models.TextInput{Text: "Cliente Joao usa Firebird 2.5"})
	if err != nil {
		t.Fatalf("ExtractFacts() error = %v", err)
	}
	if len(facts) == 0 {
		t.Fatal("expected facts")
	}
}

func TestRelationClassifierAdapter(t *testing.T) {
	t.Parallel()

	adapter := RelationClassifierAdapter{Classifier: coreengine.NewHeuristicMemoryRelationClassifier()}

	result, err := adapter.ClassifyRelation(
		context.Background(),
		models.TextInput{Text: "Erro fiscal na versao 3.1"},
		models.TextInput{Text: "Erro fiscal na versao 3.2"},
	)
	if err != nil {
		t.Fatalf("ClassifyRelation() error = %v", err)
	}
	if result.Relation == "" {
		t.Fatal("expected relation")
	}
}
