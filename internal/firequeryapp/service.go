package firequeryapp

import (
	"context"
	"strings"

	"github.com/phmotad/firememory/internal/engine"
	"github.com/phmotad/firememory/internal/firequery"
	"github.com/phmotad/firememory/internal/firequery/adapters"
	"github.com/phmotad/firememory/internal/firequery/builder"
	"github.com/phmotad/firememory/internal/firequery/contract"
	"github.com/phmotad/firememory/internal/firequery/doctor"
	fqmcp "github.com/phmotad/firememory/internal/firequery/mcp"
	"github.com/phmotad/firememory/internal/firequery/models"
	fqonnx "github.com/phmotad/firememory/internal/firequery/onnx"
	"github.com/phmotad/firememory/internal/firequery/pipeline"
	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
	"github.com/phmotad/firememory/internal/firequery/validator"
)

type EnvLookup func(string) string

const (
	envIntentModel     = "FIREQUERY_INTENT_MODEL"
	envTriggerModel    = "FIREQUERY_TRIGGER_MODEL"
	envEntityModel     = "FIREQUERY_ENTITY_MODEL"
	envSimilarityModel = "FIREQUERY_SIMILARITY_MODEL"
	envRequireReal     = "FIREQUERY_REQUIRE_REAL_MODELS"
	envModelsDir       = "FIREMEMORY_MODELS_DIR"
)

type ModelConfig struct {
	IntentModelID     string
	TriggerModelID    string
	EntityModelID     string
	SimilarityModelID string
}

func BuildService(lookupEnv EnvLookup) (*firequery.Service, error) {
	manager := BuildRuntimeManager(lookupEnv)

	p, err := buildPipelineWithClient(lookupEnv,
		validator.GuardedClient{Validator: validator.StrictValidator{}, Client: adapters.EngineClient{}})
	if err != nil {
		return nil, err
	}

	mcpServer := fqmcp.NewServer()
	mcpServer.RegisterDefaultTools(func(ctx context.Context, request contract.ExternalRequest) (contract.ExternalResponse, error) {
		return p.HandleMCP(ctx, request)
	})

	return firequery.New(firequery.Config{
		Pipeline: p,
		Runtime:  manager,
		Doctor:   doctor.RuntimeReporter{Runtime: manager},
		MCP:      mcpServer,
	})
}

// BuildDaemonHandler builds a full pipeline backed by a pre-opened engine and
// returns a handler func suitable for use by the daemon's HTTP server.
// The returned handler runs the complete ML pipeline (intent/entity/similarity)
// plus the engine operation — all inside the daemon process.
func BuildDaemonHandler(lookupEnv EnvLookup, eng engine.Engine) (func(context.Context, contract.ExternalRequest) (contract.ExternalResponse, error), error) {
	p, err := buildPipelineWithClient(lookupEnv,
		validator.GuardedClient{
			Validator: validator.StrictValidator{},
			Client:    adapters.NewPersistentEngineClient(eng),
		})
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, req contract.ExternalRequest) (contract.ExternalResponse, error) {
		return p.HandleMCP(ctx, req)
	}, nil
}

// buildPipelineWithClient constructs the ML pipeline with the given storage client.
// This is the shared setup for both ephemeral (EngineClient) and daemon
// (PersistentEngineClient) modes.
func buildPipelineWithClient(lookupEnv EnvLookup, client adapters.FireMemoryClient) (*pipeline.DefaultPipeline, error) {
	modelsConfig := ResolveModelConfig(lookupEnv)
	modelsDir := envOrDefault(lookupEnv, envModelsDir, fqonnx.DefaultModelsDir())
	requireReal := envTruthy(lookupEnv, envRequireReal)

	intentClassifier := models.NewConfiguredDeBERTaIntentClassifier(modelsConfig.IntentModelID, nil, nil)
	triggerClassifier := models.NewConfiguredDeBERTaTriggerClassifier(modelsConfig.TriggerModelID, nil, nil)
	entityExtractor := models.NewConfiguredGLiNEREntityExtractor(modelsConfig.EntityModelID, nil, nil)
	similarityEngine := models.NewConfiguredE5SimilarityEngine(modelsConfig.SimilarityModelID, nil, nil)

	onnxBackend, err := fqonnx.New(modelsDir)
	if err != nil {
		if requireReal {
			return nil, err
		}
	} else {
		textClient := onnxTextClient{backend: onnxBackend}
		entityClient := onnxEntityClient{backend: onnxBackend}

		intentClassifier = models.NewConfiguredDeBERTaIntentClassifier(modelsConfig.IntentModelID, textClient, nil)
		triggerClassifier = models.NewConfiguredDeBERTaTriggerClassifier(modelsConfig.TriggerModelID, textClient, nil)
		entityExtractor = models.NewConfiguredGLiNEREntityExtractor(modelsConfig.EntityModelID, entityClient, nil)
		similarityEngine = models.NewConfiguredE5SimilarityEngine(modelsConfig.SimilarityModelID, onnxBackend, nil)
	}

	return pipeline.New(pipeline.Config{
		ExternalValidator:  validator.StrictValidator{},
		InternalValidator:  validator.StrictValidator{},
		FireMemoryClient:   client,
		ContractBuilder:    builder.NewGoContractBuilder(builder.DefaultActorID),
		IntentClassifier:   intentClassifier,
		TriggerClassifier:  triggerClassifier,
		EntityExtractor:    entityExtractor,
		FactExtractor:      models.NewHeuristicFactExtractor(),
		RelationClassifier: models.HeuristicRelationClassifier{},
		SimilarityEngine:   similarityEngine,
		Reranker:           models.StableReranker{},
	})
}

func BuildRuntimeManager(lookupEnv EnvLookup) fqruntime.Manager {
	modelsConfig := ResolveModelConfig(lookupEnv)
	modelsDir := envOrDefault(lookupEnv, envModelsDir, fqonnx.DefaultModelsDir())
	devices := DetectDevices(lookupEnv)

	runtimeReady := true
	runtimeNotes := []string{"onnx backend"}
	modelNote := "onnx backend active"

	_, err := fqonnx.New(modelsDir)
	if err != nil {
		runtimeNotes = []string{"models not available: " + err.Error()}
		modelNote = "models not downloaded (run: fquery models pull)"
		if envTruthy(lookupEnv, envRequireReal) {
			runtimeReady = false
		}
	}

	registered := []fqruntime.ModelState{
		{ID: modelsConfig.IntentModelID, Version: "0.1", Backend: fqruntime.BackendCPU, Loaded: runtimeReady, Healthy: runtimeReady, Notes: []string{modelNote}},
		{ID: modelsConfig.TriggerModelID, Version: "0.1", Backend: fqruntime.BackendCPU, Loaded: runtimeReady, Healthy: runtimeReady, Notes: []string{modelNote}},
		{ID: modelsConfig.EntityModelID, Version: "0.1", Backend: fqruntime.BackendCPU, Loaded: runtimeReady, Healthy: runtimeReady, Notes: []string{modelNote}},
		{ID: modelsConfig.SimilarityModelID, Version: "0.1", Backend: fqruntime.BackendCPU, Loaded: runtimeReady, Healthy: runtimeReady, Notes: []string{modelNote}},
		{ID: "fact-extractor", Version: "0.1", Backend: fqruntime.BackendCPU, Loaded: true, Healthy: true, Notes: []string{"registered"}},
		{ID: "relation-classifier", Version: "0.1", Backend: fqruntime.BackendCPU, Loaded: true, Healthy: true, Notes: []string{"registered"}},
		{ID: "reranker", Version: "0.1", Backend: fqruntime.BackendCPU, Loaded: true, Healthy: true, Notes: []string{"registered"}},
	}

	return fqruntime.StaticManager{
		DetectedDevices: devices,
		Status: fqruntime.Health{
			Ready:   runtimeReady,
			Backend: fqruntime.BackendCPU,
			Notes:   runtimeNotes,
			Devices: devices,
			Models:  registered,
		},
		Registered: registered,
	}
}

func ResolveModelConfig(lookupEnv EnvLookup) ModelConfig {
	return ModelConfig{
		IntentModelID:     envOrDefault(lookupEnv, envIntentModel, models.IntentModelDeBERTaSmall),
		TriggerModelID:    envOrDefault(lookupEnv, envTriggerModel, models.TriggerModelDeBERTaSmall),
		EntityModelID:     envOrDefault(lookupEnv, envEntityModel, models.EntityModelGLiNER2Small),
		SimilarityModelID: envOrDefault(lookupEnv, envSimilarityModel, models.SimilarityModelE5Small),
	}
}

func DetectDevices(lookupEnv EnvLookup) []fqruntime.Device {
	detector := fqruntime.EnvironmentDetector{
		LookupEnv: func(key string) string {
			if lookupEnv == nil {
				return ""
			}
			return lookupEnv(key)
		},
	}
	devices, err := detector.Detect(context.Background())
	if err != nil {
		return []fqruntime.Device{{Kind: fqruntime.DeviceCPU, Name: "cpu", Available: true}}
	}
	return devices
}

// onnxTextClient adapts fqonnx.Backend to models.TextClassificationClient.
type onnxTextClient struct {
	backend fqonnx.Backend
}

func (c onnxTextClient) Classify(ctx context.Context, modelID string, input models.TextInput, labels []string) ([]models.ScoredLabel, error) {
	return c.backend.Classify(ctx, modelID, input, labels)
}

// onnxEntityClient adapts fqonnx.Backend to models.EntityExtractionClient.
type onnxEntityClient struct {
	backend fqonnx.Backend
}

func (c onnxEntityClient) ExtractEntities(ctx context.Context, modelID string, input models.TextInput) ([]models.Entity, error) {
	return c.backend.ExtractEntities(ctx, modelID, input)
}

func envOrDefault(lookupEnv EnvLookup, key, fallback string) string {
	if lookupEnv == nil {
		return fallback
	}
	value := strings.TrimSpace(lookupEnv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envTruthy(lookupEnv EnvLookup, key string) bool { //nolint:unparam
	value := strings.ToLower(strings.TrimSpace(envOrDefault(lookupEnv, key, "")))
	switch value {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
