package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/phmotad/firememory/internal/firequery/adapters"
	"github.com/phmotad/firememory/internal/firequery/builder"
	"github.com/phmotad/firememory/internal/firequery/contract"
	"github.com/phmotad/firememory/internal/firequery/models"
	"github.com/phmotad/firememory/internal/firequery/validator"
)

func TestPipelineFailureInjectionOnFireMemoryClientError(t *testing.T) {
	t.Parallel()

	p, err := New(Config{
		ExternalValidator:  validator.StrictValidator{},
		InternalValidator:  validator.StrictValidator{},
		FireMemoryClient:   failingFireMemoryClient{err: errors.New("synthetic firememory failure")},
		ContractBuilder:    builder.NewGoContractBuilder(builder.DefaultActorID),
		IntentClassifier:   models.NewDeBERTaIntentClassifier(nil, nil),
		TriggerClassifier:  models.NewDeBERTaTriggerClassifier(nil, nil),
		EntityExtractor:    models.NewGLiNEREntityExtractor(nil, nil),
		FactExtractor:      models.NewHeuristicFactExtractor(),
		RelationClassifier: models.HeuristicRelationClassifier{},
		SimilarityEngine:   models.NewE5SimilarityEngine(nil, nil),
		Reranker:           models.StableReranker{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = p.Run(context.Background(), Input{
		Request: contract.ExternalRequest{
			Version:   "0.1",
			RequestID: "req_failure",
			Language:  "en",
			Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
			Operation: "recall",
			Brain:     "agent.fbrain",
			Input:     map[string]any{"query": "fiscal issue"},
		},
	})
	if err == nil || err.Error() != "synthetic firememory failure" {
		t.Fatalf("expected synthetic firememory failure, got %v", err)
	}
}

type failingFireMemoryClient struct {
	err error
}

func (c failingFireMemoryClient) Call(context.Context, contract.OperationRequest) (contract.OperationResponse, error) {
	return contract.OperationResponse{}, c.err
}

var _ adapters.FireMemoryClient = failingFireMemoryClient{}
