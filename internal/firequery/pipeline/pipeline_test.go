package pipeline

import (
	"context"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/firequery/adapters"
	"github.com/phmotad/firememory/internal/firequery/builder"
	"github.com/phmotad/firememory/internal/firequery/contract"
	fqmcp "github.com/phmotad/firememory/internal/firequery/mcp"
	"github.com/phmotad/firememory/internal/firequery/models"
	"github.com/phmotad/firememory/internal/firequery/validator"
)

func TestDefaultPipelineEndToEnd(t *testing.T) {
	t.Parallel()

	brainPath := t.TempDir() + "/agent.fbrain"
	handle, err := brainfile.Create(brainPath, brainfile.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	p, err := New(Config{
		ExternalValidator:  validator.StrictValidator{},
		InternalValidator:  validator.StrictValidator{},
		FireMemoryClient:   validator.GuardedClient{Validator: validator.StrictValidator{}, Client: adapters.EngineClient{}},
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

	server := fqmcp.NewServer()
	server.Register("firequery.remember", p.HandleMCP)
	server.Register("firequery.recall", p.HandleMCP)

	rememberResponse, err := server.Handle(context.Background(), "firequery.remember", contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_remember",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Operation: "remember",
		Brain:     brainPath,
		Input: map[string]any{
			"content":     "Client Joao uses Firebird 2.5 and reported a fiscal NF-e error after update 3.2",
			"allow_write": true,
		},
	})
	if err != nil {
		t.Fatalf("server.Handle(remember) error = %v", err)
	}
	if !rememberResponse.OK {
		t.Fatalf("remember response = %#v", rememberResponse)
	}

	recallResponse, err := server.Handle(context.Background(), "firequery.recall", contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_recall",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Operation: "recall",
		Brain:     brainPath,
		Input: map[string]any{
			"query": "fiscal NF-e error",
			"top_k": 3,
		},
	})
	if err != nil {
		t.Fatalf("server.Handle(recall) error = %v", err)
	}
	if !recallResponse.OK {
		t.Fatalf("recall response = %#v", recallResponse)
	}

	hits, ok := recallResponse.Data["hits"].([]map[string]any)
	if !ok {
		t.Fatalf("hits type = %T, want []map[string]any", recallResponse.Data["hits"])
	}
	if len(hits) == 0 {
		t.Fatal("expected recall hits")
	}
	if recallResponse.Trace == nil {
		t.Fatal("expected structured trace")
	}
}

func TestDefaultPipelineRejectsInvalidExternalRequest(t *testing.T) {
	t.Parallel()

	client := &recordingClient{}
	p, err := New(Config{
		ExternalValidator:  validator.StrictValidator{},
		InternalValidator:  validator.StrictValidator{},
		FireMemoryClient:   client,
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

	response, err := p.HandleMCP(context.Background(), contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_invalid",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Operation: "recall",
		Brain:     "agent.brain",
		Input:     map[string]any{"query": "x"},
	})
	if err != nil {
		t.Fatalf("HandleMCP() error = %v", err)
	}
	if response.OK || !response.Rejected {
		t.Fatalf("response = %#v, want rejected", response)
	}
	if client.called {
		t.Fatal("firememory client should not be called")
	}
}

type recordingClient struct {
	called bool
}

func (c *recordingClient) Call(context.Context, contract.OperationRequest) (contract.OperationResponse, error) {
	c.called = true
	return contract.OperationResponse{OK: true}, nil
}
