package firequery

import (
	"context"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/firequery/adapters"
	"github.com/phmotad/firememory/internal/firequery/builder"
	"github.com/phmotad/firememory/internal/firequery/contract"
	"github.com/phmotad/firememory/internal/firequery/doctor"
	fqmcp "github.com/phmotad/firememory/internal/firequery/mcp"
	"github.com/phmotad/firememory/internal/firequery/models"
	"github.com/phmotad/firememory/internal/firequery/pipeline"
	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
	"github.com/phmotad/firememory/internal/firequery/validator"
)

func TestFireQueryAcceptance(t *testing.T) {
	t.Parallel()

	brainPath := t.TempDir() + "/agent.fbrain"
	handle, err := brainfile.Create(brainPath, brainfile.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	p, err := pipeline.New(pipeline.Config{
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
		t.Fatalf("pipeline.New() error = %v", err)
	}

	runtimeManager := fqruntime.StaticManager{
		Status: fqruntime.Health{
			Ready:   true,
			Backend: fqruntime.BackendCPU,
			Notes:   []string{"cpu fallback enabled"},
			Devices: []fqruntime.Device{{Kind: fqruntime.DeviceCPU, Name: "cpu", Available: true}},
			Models: []fqruntime.ModelState{
				{ID: models.IntentModelDeBERTaSmall, Backend: fqruntime.BackendCPU, Healthy: true},
				{ID: models.TriggerModelDeBERTaSmall, Backend: fqruntime.BackendCPU, Healthy: true},
				{ID: models.EntityModelGLiNER2Small, Backend: fqruntime.BackendCPU, Healthy: true},
				{ID: models.SimilarityModelE5Small, Backend: fqruntime.BackendCPU, Healthy: true},
			},
		},
	}
	reporter := doctor.RuntimeReporter{Runtime: runtimeManager}
	server := fqmcp.NewServer()
	server.RegisterDefaultTools(p.HandleMCP)

	service, err := New(Config{
		Pipeline: p,
		Runtime:  runtimeManager,
		Doctor:   reporter,
		MCP:      server,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	devices, err := service.Runtime().Devices(context.Background())
	if err != nil {
		t.Fatalf("Runtime().Devices() error = %v", err)
	}
	if len(devices) == 0 || devices[0].Supports() != fqruntime.BackendCPU {
		t.Fatalf("devices = %#v, want cpu fallback", devices)
	}

	report, err := service.Doctor().Run(context.Background())
	if err != nil {
		t.Fatalf("Doctor().Run() error = %v", err)
	}
	if !report.Ready {
		t.Fatalf("report = %#v, want ready", report)
	}

	rememberResponse, err := service.MCP().Handle(context.Background(), fqmcp.ToolRemember, contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_remember",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Brain:     brainPath,
		Input: map[string]any{
			"content":     "Client Joao reported fiscal NF-e error after update 3.2",
			"allow_write": true,
		},
	})
	if err != nil {
		t.Fatalf("MCP remember error = %v", err)
	}
	if !rememberResponse.OK {
		t.Fatalf("remember response = %#v", rememberResponse)
	}

	recallResponse, err := service.MCP().Handle(context.Background(), fqmcp.ToolRecall, contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_recall",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Brain:     brainPath,
		Input: map[string]any{
			"query": "fiscal NF-e error",
		},
	})
	if err != nil {
		t.Fatalf("MCP recall error = %v", err)
	}
	if !recallResponse.OK {
		t.Fatalf("recall response = %#v", recallResponse)
	}

	askResponse, err := service.MCP().Handle(context.Background(), fqmcp.ToolAsk, contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_ask",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Brain:     brainPath,
		Input: map[string]any{
			"task": "answer Joao about the fiscal error",
		},
	})
	if err != nil {
		t.Fatalf("MCP ask error = %v", err)
	}
	if !askResponse.OK || askResponse.Operation != "get_context" {
		t.Fatalf("ask response = %#v", askResponse)
	}

	invalidResponse, err := service.MCP().Handle(context.Background(), fqmcp.ToolRecall, contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_invalid",
		Language:  "pt-BR",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Brain:     "agent.brain",
		Input: map[string]any{
			"query": "fiscal NF-e error",
		},
	})
	if err != nil {
		t.Fatalf("MCP invalid error = %v", err)
	}
	if invalidResponse.OK || !invalidResponse.Rejected {
		t.Fatalf("invalid response = %#v, want rejected", invalidResponse)
	}

	writeDeniedResponse, err := service.MCP().Handle(context.Background(), fqmcp.ToolRemember, contract.ExternalRequest{
		Version:   "0.1",
		RequestID: "req_write_denied",
		Language:  "en",
		Actor:     contract.Actor{Type: "agent", ID: "support-agent"},
		Brain:     brainPath,
		Input: map[string]any{
			"content":     "Store this but deny writes",
			"allow_write": false,
		},
	})
	if err != nil {
		t.Fatalf("MCP write denied error = %v", err)
	}
	if writeDeniedResponse.OK || !writeDeniedResponse.Rejected {
		t.Fatalf("write denied response = %#v, want rejected", writeDeniedResponse)
	}
}
