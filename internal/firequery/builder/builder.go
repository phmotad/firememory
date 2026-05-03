package builder

import (
	"strings"

	"github.com/phmotad/firememory/internal/engine"
	"github.com/phmotad/firememory/internal/firequery/contract"
	"github.com/phmotad/firememory/internal/firequery/models"
)

const DefaultActorID = "firequery-mcp"

type Inputs struct {
	Intent           models.IntentResult
	Trigger          models.TriggerResult
	Entities         []models.Entity
	Facts            []models.Fact
	Relation         models.RelationSuggestion
	RankedCandidates models.RankedCandidates
}

type Builder interface {
	Build(request contract.ExternalRequest, inputs Inputs) contract.OperationRequest
}

type GoContractBuilder struct {
	actorID string
}

func NewGoContractBuilder(actorID string) *GoContractBuilder {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = DefaultActorID
	}
	return &GoContractBuilder{actorID: actorID}
}

func (b *GoContractBuilder) Build(request contract.ExternalRequest, inputs Inputs) contract.OperationRequest {
	internalInput := cloneMap(request.Input)
	internalInput["trigger"] = inputs.Trigger.Trigger
	if inputs.Relation.Relation != "" {
		internalInput["relation_hint"] = inputs.Relation.Relation
	}
	if len(inputs.Entities) > 0 {
		internalInput["entities"] = inputs.Entities
	}
	if len(inputs.Facts) > 0 {
		internalInput["facts"] = inputs.Facts
	}
	if len(inputs.RankedCandidates.Items) > 0 {
		internalInput["ranked_candidates"] = inputs.RankedCandidates.Items
	}

	return contract.OperationRequest{
		Version:   request.Version,
		RequestID: request.RequestID,
		Language:  "en",
		Actor: contract.Actor{
			Type: "firequery",
			ID:   b.actorID,
		},
		Operation:   request.Operation,
		Intent:      inputs.Intent.Intent,
		Brain:       request.Brain,
		Scope:       stringValue(request.Input, "scope", "default"),
		Input:       normalizeInputForOperation(request.Operation, internalInput),
		Permissions: buildPermissions(request),
		Thresholds:  buildThresholds(request),
		Options: &contract.Options{
			IncludeGraph: boolValue(request.Input, "include_graph", request.Operation == "get_context"),
			IncludeTrace: boolValue(request.Input, "include_trace", true),
		},
	}
}

func buildPermissions(request contract.ExternalRequest) *contract.Permissions {
	allowWrite := boolValue(request.Input, "allow_write", false)
	requiresConfirmation := boolValue(request.Input, "requires_confirmation", request.Operation == "forget")
	return &contract.Permissions{
		AllowWrite:           allowWrite,
		RequiresConfirmation: requiresConfirmation,
	}
}

func buildThresholds(request contract.ExternalRequest) *contract.Thresholds {
	switch request.Operation {
	case "recall", "get_context":
		return &contract.Thresholds{
			TopK:                intValue(request.Input, "top_k", engine.DefaultTopK),
			SimilarityThreshold: floatValue(request.Input, "similarity_threshold", 0.7),
			BudgetTokens:        intValue(request.Input, "budget_tokens", engine.DefaultBudgetTokens),
		}
	default:
		return nil
	}
}

func normalizeInputForOperation(operation string, input map[string]any) map[string]any {
	switch operation {
	case "get_context":
		if task, ok := input["task"].(string); ok && strings.TrimSpace(task) != "" {
			input["query"] = task
		}
	case "explain":
		if target, ok := input["target_operation"].(string); ok && strings.TrimSpace(target) != "" {
			input["operation"] = target
		}
	}
	return input
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func stringValue(input map[string]any, key, fallback string) string {
	if value, ok := input[key].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func boolValue(input map[string]any, key string, fallback bool) bool {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.TrimSpace(strings.ToLower(typed)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return fallback
}

func intValue(input map[string]any, key string, fallback int) int {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	}
	return fallback
}

func floatValue(input map[string]any, key string, fallback float64) float64 {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	}
	return fallback
}
