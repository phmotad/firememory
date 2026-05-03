package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/phmotad/firememory/internal/engine"
	"github.com/phmotad/firememory/internal/firequery/contract"
	"github.com/phmotad/firememory/internal/util"
)

type EngineClient struct{}

func (EngineClient) Call(_ context.Context, request contract.OperationRequest) (contract.OperationResponse, error) {
	eng, err := engine.Open(engine.Options{Path: request.Brain})
	if err != nil {
		return contract.OperationResponse{}, err
	}
	defer eng.Close()

	switch request.Operation {
	case "remember":
		return callRemember(eng, request)
	case "recall":
		return callRecall(eng, request)
	case "get_context":
		return callContext(eng, request)
	case "explain":
		return callExplain(eng, request)
	case "sync":
		return callSync(eng, request)
	default:
		return contract.OperationResponse{}, fmt.Errorf("firequery/adapters: unsupported firememory operation %q", request.Operation)
	}
}

func callRemember(eng engine.Engine, request contract.OperationRequest) (contract.OperationResponse, error) {
	result, err := eng.Remember(engine.RememberInput{
		BrainPath: request.Brain,
		Content:   stringValue(request.Input, "content", ""),
		Scope:     request.Scope,
	})
	if err != nil {
		return contract.OperationResponse{}, err
	}

	return contract.OperationResponse{
		OK:        true,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Data: map[string]any{
			"memory_id":             result.Memory.ID,
			"dedup_action":          result.DedupAction,
			"reinforced_memory_id":  result.ReinforcedMemoryID,
			"status":                result.Memory.Status,
			"normalized_content":    result.Memory.NormalizedContent,
			"memory_scope":          result.Memory.Scope,
		},
		Trace: map[string]any{
			"firememory": util.StructuredTrace("firememory.engine", result.Trace),
		},
	}, nil
}

func callRecall(eng engine.Engine, request contract.OperationRequest) (contract.OperationResponse, error) {
	topK := engine.DefaultTopK
	includeTrace := true
	if request.Thresholds != nil && request.Thresholds.TopK > 0 {
		topK = request.Thresholds.TopK
	}
	if request.Options != nil {
		includeTrace = request.Options.IncludeTrace
	}

	result, err := eng.Recall(engine.RecallInput{
		BrainPath:    request.Brain,
		Query:        stringValue(request.Input, "query", ""),
		Scope:        request.Scope,
		TopK:         topK,
		IncludeTrace: includeTrace,
	})
	if err != nil {
		return contract.OperationResponse{}, err
	}

	hits := make([]map[string]any, 0, len(result.Hits))
	for _, hit := range result.Hits {
		hits = append(hits, map[string]any{
			"memory_id": hit.Memory.ID,
			"content":   hit.Memory.Content,
			"score":     hit.Score,
			"reasons":   hit.Reasons,
		})
	}

	return contract.OperationResponse{
		OK:        true,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Data: map[string]any{
			"hits": hits,
		},
		Trace: map[string]any{
			"firememory": util.StructuredTrace("firememory.engine", result.Trace),
		},
	}, nil
}

func callContext(eng engine.Engine, request contract.OperationRequest) (contract.OperationResponse, error) {
	topK := engine.DefaultTopK
	budgetTokens := engine.DefaultBudgetTokens
	includeGraph := true
	includeTrace := true
	if request.Thresholds != nil {
		if request.Thresholds.TopK > 0 {
			topK = request.Thresholds.TopK
		}
		if request.Thresholds.BudgetTokens > 0 {
			budgetTokens = request.Thresholds.BudgetTokens
		}
	}
	if request.Options != nil {
		includeGraph = request.Options.IncludeGraph
		includeTrace = request.Options.IncludeTrace
	}

	result, err := eng.Context(engine.ContextInput{
		BrainPath:    request.Brain,
		Query:        stringValue(request.Input, "query", ""),
		Scope:        request.Scope,
		TopK:         topK,
		BudgetTokens: budgetTokens,
		IncludeGraph: includeGraph,
		IncludeTrace: includeTrace,
	})
	if err != nil {
		return contract.OperationResponse{}, err
	}

	return contract.OperationResponse{
		OK:        true,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Data: map[string]any{
			"context":          result.ContextText,
			"estimated_tokens": result.EstimatedTokens,
			"memory_count":     len(result.Memories),
			"entity_count":     len(result.Entities),
			"fact_count":       len(result.Facts),
			"relation_count":   len(result.Relations),
		},
		Trace: map[string]any{
			"firememory": util.StructuredTrace("firememory.engine", result.Trace),
		},
	}, nil
}

func callExplain(eng engine.Engine, request contract.OperationRequest) (contract.OperationResponse, error) {
	result, err := eng.Explain(engine.ExplainInput{
		BrainPath: request.Brain,
		Operation: stringValue(request.Input, "operation", ""),
		MemoryID:  stringValue(request.Input, "memory_id", ""),
	})
	if err != nil {
		return contract.OperationResponse{}, err
	}

	return contract.OperationResponse{
		OK:        true,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Data: map[string]any{
			"summary": result.Summary,
		},
		Trace: map[string]any{
			"firememory": util.StructuredTrace("firememory.engine", result.Trace),
		},
	}, nil
}

func callSync(eng engine.Engine, request contract.OperationRequest) (contract.OperationResponse, error) {
	result, err := eng.Sync(engine.SyncInput{
		BrainPath: request.Brain,
		Limit:     intValue(request.Input, "limit", 0),
	})
	if err != nil {
		return contract.OperationResponse{}, err
	}

	return contract.OperationResponse{
		OK:        true,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Data: map[string]any{
			"processed":  result.Processed,
			"synced_ids": result.SyncedIDs,
		},
		Trace: map[string]any{
			"firememory": util.StructuredTrace("firememory.engine", result.Trace),
		},
	}, nil
}

func stringValue(input map[string]any, key, fallback string) string {
	if value, ok := input[key].(string); ok && strings.TrimSpace(value) != "" {
		return value
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
	default:
		return fallback
	}
}
