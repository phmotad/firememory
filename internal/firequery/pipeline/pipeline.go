package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/phmotad/firememory/internal/firequery/adapters"
	"github.com/phmotad/firememory/internal/firequery/builder"
	"github.com/phmotad/firememory/internal/firequery/contract"
	"github.com/phmotad/firememory/internal/firequery/models"
	"github.com/phmotad/firememory/internal/firequery/validator"
	"github.com/phmotad/firememory/internal/util"
)

var (
	ErrExternalValidatorRequired  = errors.New("firequery/pipeline: external validator is required")
	ErrInternalValidatorRequired  = errors.New("firequery/pipeline: internal validator is required")
	ErrFireMemoryClientRequired   = errors.New("firequery/pipeline: firememory client is required")
	ErrContractBuilderRequired    = errors.New("firequery/pipeline: contract builder is required")
	ErrIntentClassifierRequired   = errors.New("firequery/pipeline: intent classifier is required")
	ErrTriggerClassifierRequired  = errors.New("firequery/pipeline: trigger classifier is required")
	ErrEntityExtractorRequired    = errors.New("firequery/pipeline: entity extractor is required")
	ErrFactExtractorRequired      = errors.New("firequery/pipeline: fact extractor is required")
	ErrRelationClassifierRequired = errors.New("firequery/pipeline: relation classifier is required")
	ErrSimilarityEngineRequired   = errors.New("firequery/pipeline: similarity engine is required")
	ErrRerankerRequired           = errors.New("firequery/pipeline: reranker is required")
)

type Input struct {
	Request contract.ExternalRequest
}

type Output struct {
	InternalRequest *contract.OperationRequest
	Response        *contract.OperationResponse
}

type Pipeline interface {
	Run(ctx context.Context, input Input) (Output, error)
}

type NoopPipeline struct{}

func (NoopPipeline) Run(context.Context, Input) (Output, error) {
	return Output{}, nil
}

type Config struct {
	ExternalValidator  validator.ExternalValidator
	InternalValidator  validator.InternalValidator
	FireMemoryClient   adapters.FireMemoryClient
	ContractBuilder    builder.Builder
	IntentClassifier   models.IntentClassifier
	TriggerClassifier  models.TriggerClassifier
	EntityExtractor    models.EntityExtractor
	FactExtractor      models.FactExtractor
	RelationClassifier models.RelationClassifier
	SimilarityEngine   models.SimilarityEngine
	Reranker           models.Reranker
	ActorID            string
}

type DefaultPipeline struct {
	externalValidator  validator.ExternalValidator
	internalValidator  validator.InternalValidator
	fireMemoryClient   adapters.FireMemoryClient
	contractBuilder    builder.Builder
	intentClassifier   models.IntentClassifier
	triggerClassifier  models.TriggerClassifier
	entityExtractor    models.EntityExtractor
	factExtractor      models.FactExtractor
	relationClassifier models.RelationClassifier
	similarityEngine   models.SimilarityEngine
	reranker           models.Reranker
}

func New(config Config) (*DefaultPipeline, error) {
	switch {
	case config.ExternalValidator == nil:
		return nil, ErrExternalValidatorRequired
	case config.InternalValidator == nil:
		return nil, ErrInternalValidatorRequired
	case config.FireMemoryClient == nil:
		return nil, ErrFireMemoryClientRequired
	case config.ContractBuilder == nil:
		return nil, ErrContractBuilderRequired
	case config.IntentClassifier == nil:
		return nil, ErrIntentClassifierRequired
	case config.TriggerClassifier == nil:
		return nil, ErrTriggerClassifierRequired
	case config.EntityExtractor == nil:
		return nil, ErrEntityExtractorRequired
	case config.FactExtractor == nil:
		return nil, ErrFactExtractorRequired
	case config.RelationClassifier == nil:
		return nil, ErrRelationClassifierRequired
	case config.SimilarityEngine == nil:
		return nil, ErrSimilarityEngineRequired
	case config.Reranker == nil:
		return nil, ErrRerankerRequired
	}

	return &DefaultPipeline{
		externalValidator:  config.ExternalValidator,
		internalValidator:  config.InternalValidator,
		fireMemoryClient:   config.FireMemoryClient,
		contractBuilder:    config.ContractBuilder,
		intentClassifier:   config.IntentClassifier,
		triggerClassifier:  config.TriggerClassifier,
		entityExtractor:    config.EntityExtractor,
		factExtractor:      config.FactExtractor,
		relationClassifier: config.RelationClassifier,
		similarityEngine:   config.SimilarityEngine,
		reranker:           config.Reranker,
	}, nil
}

func (p *DefaultPipeline) Run(ctx context.Context, input Input) (Output, error) {
	externalResult := p.externalValidator.ValidateExternal(input.Request)
	if !externalResult.OK {
		response := externalRejectResponse(input.Request, externalResult)
		return Output{Response: &response}, nil
	}

	text := extractPrimaryText(input.Request)
	textInput := models.TextInput{
		Language: input.Request.Language,
		Text:     text,
	}

	intentResult, err := p.classifyIntent(ctx, input.Request, textInput)
	if err != nil {
		return Output{}, err
	}

	triggerResult, err := p.classifyTrigger(ctx, textInput, input.Request.Operation)
	if err != nil {
		return Output{}, err
	}

	entities, err := p.extractEntities(ctx, textInput)
	if err != nil {
		return Output{}, err
	}

	facts, err := p.extractFacts(ctx, textInput)
	if err != nil {
		return Output{}, err
	}

	candidates := buildCandidates(entities, facts)
	scoredCandidates, err := p.scoreCandidates(ctx, textInput, candidates)
	if err != nil {
		return Output{}, err
	}

	rankedCandidates, err := p.reranker.Rerank(ctx, textInput, scoredCandidates)
	if err != nil {
		return Output{}, err
	}

	relation, err := p.classifyRelation(ctx, textInput, rankedCandidates)
	if err != nil {
		return Output{}, err
	}

	internalRequest := p.contractBuilder.Build(input.Request, builder.Inputs{
		Intent:           intentResult,
		Trigger:          triggerResult,
		Entities:         entities,
		Facts:            facts,
		Relation:         relation,
		RankedCandidates: rankedCandidates,
	})
	internalResult := p.internalValidator.ValidateInternal(internalRequest)
	if !internalResult.OK {
		response := internalRejectResponse(internalRequest, internalResult)
		return Output{
			InternalRequest: &internalRequest,
			Response:        &response,
		}, nil
	}

	response, err := p.fireMemoryClient.Call(ctx, internalRequest)
	if err != nil {
		return Output{}, err
	}

	enrichResponse(&response, internalRequest, intentResult, triggerResult, entities, facts, rankedCandidates, relation)

	return Output{
		InternalRequest: &internalRequest,
		Response:        &response,
	}, nil
}

func (p *DefaultPipeline) HandleMCP(ctx context.Context, request contract.ExternalRequest) (contract.OperationResponse, error) {
	output, err := p.Run(ctx, Input{Request: request})
	if err != nil {
		return contract.OperationResponse{}, err
	}
	if output.Response == nil {
		return contract.OperationResponse{
			OK:        false,
			RequestID: request.RequestID,
			Operation: request.Operation,
			Rejected:  true,
			Error: &contract.Error{
				Code:    "PIPELINE_NO_RESPONSE",
				Message: "Pipeline finished without response",
			},
		}, nil
	}
	return *output.Response, nil
}

func (p *DefaultPipeline) classifyIntent(ctx context.Context, request contract.ExternalRequest, input models.TextInput) (models.IntentResult, error) {
	if strings.TrimSpace(input.Text) == "" {
		return models.IntentResult{
			Intent: mappedIntent(request.Operation),
			Score:  1.0,
		}, nil
	}

	intentResult, err := p.intentClassifier.ClassifyIntent(ctx, input)
	if err != nil {
		return models.IntentResult{}, err
	}
	intentResult.Intent = mappedIntent(request.Operation)
	if intentResult.Score == 0 {
		intentResult.Score = 1.0
	}
	return intentResult, nil
}

func (p *DefaultPipeline) classifyTrigger(ctx context.Context, input models.TextInput, operation string) (models.TriggerResult, error) {
	if strings.TrimSpace(input.Text) == "" {
		return fallbackTrigger(operation), nil
	}

	triggerResult, err := p.triggerClassifier.ClassifyTrigger(ctx, input)
	if err != nil {
		return models.TriggerResult{}, err
	}
	if strings.TrimSpace(triggerResult.Trigger) == "" {
		return fallbackTrigger(operation), nil
	}
	return triggerResult, nil
}

func (p *DefaultPipeline) extractEntities(ctx context.Context, input models.TextInput) ([]models.Entity, error) {
	if strings.TrimSpace(input.Text) == "" {
		return nil, nil
	}
	return p.entityExtractor.ExtractEntities(ctx, input)
}

func (p *DefaultPipeline) extractFacts(ctx context.Context, input models.TextInput) ([]models.Fact, error) {
	if strings.TrimSpace(input.Text) == "" {
		return nil, nil
	}
	return p.factExtractor.ExtractFacts(ctx, input)
}

func (p *DefaultPipeline) scoreCandidates(ctx context.Context, input models.TextInput, candidates []models.Candidate) ([]models.Candidate, error) {
	if strings.TrimSpace(input.Text) == "" || len(candidates) == 0 {
		return candidates, nil
	}
	return p.similarityEngine.ScoreCandidates(ctx, input, candidates)
}

func (p *DefaultPipeline) classifyRelation(ctx context.Context, input models.TextInput, ranked models.RankedCandidates) (models.RelationSuggestion, error) {
	if strings.TrimSpace(input.Text) == "" || len(ranked.Items) == 0 {
		return models.RelationSuggestion{}, nil
	}
	return p.relationClassifier.ClassifyRelation(ctx, input, models.TextInput{
		Language: "en",
		Text:     ranked.Items[0].Text,
	})
}

func mappedIntent(operation string) string {
	switch operation {
	case "remember":
		return "remember_information"
	case "recall":
		return "recall_information"
	case "relate":
		return "relate_memory"
	case "forget":
		return "forget_memory"
	case "consolidate":
		return "consolidate_memory"
	case "get_context":
		return "build_context"
	case "explain":
		return "explain_decision"
	case "sync":
		return "sync_memory"
	default:
		return "recall_information"
	}
}

func fallbackTrigger(operation string) models.TriggerResult {
	switch operation {
	case "remember", "relate", "consolidate", "sync":
		return models.TriggerResult{Trigger: "suggest_write", Score: 1.0}
	case "forget":
		return models.TriggerResult{Trigger: "request_confirmation", Score: 1.0}
	default:
		return models.TriggerResult{Trigger: "query_memory", Score: 1.0}
	}
}

func buildCandidates(entities []models.Entity, facts []models.Fact) []models.Candidate {
	candidates := make([]models.Candidate, 0, len(entities)+len(facts))
	for _, entity := range entities {
		candidates = append(candidates, models.Candidate{
			ID:    "entity:" + entity.Type + ":" + entity.Text,
			Text:  entity.Text,
			Score: entity.Score,
		})
	}
	for index, fact := range facts {
		candidates = append(candidates, models.Candidate{
			ID:    fmt.Sprintf("fact:%d", index),
			Text:  fact.Text,
			Score: fact.Score,
		})
	}
	return candidates
}

func enrichResponse(
	response *contract.OperationResponse,
	internalRequest contract.OperationRequest,
	intent models.IntentResult,
	trigger models.TriggerResult,
	entities []models.Entity,
	facts []models.Fact,
	ranked models.RankedCandidates,
	relation models.RelationSuggestion,
) {
	if response.Data == nil {
		response.Data = map[string]any{}
	}
	response.RequestID = internalRequest.RequestID
	response.Operation = internalRequest.Operation
	response.Data["intent"] = intent.Intent
	response.Data["intent_score"] = intent.Score
	response.Data["trigger"] = trigger.Trigger
	response.Data["trigger_score"] = trigger.Score
	response.Data["entity_count"] = len(entities)
	response.Data["fact_count"] = len(facts)
	response.Data["ranked_candidate_count"] = len(ranked.Items)
	if relation.Relation != "" {
		response.Data["relation_hint"] = relation.Relation
		response.Data["relation_score"] = relation.Score
	}
	if response.Trace == nil {
		response.Trace = map[string]any{}
	}
	response.Trace["pipeline"] = util.StructuredTrace("firequery.pipeline", []string{
		"validated external contract",
		"classified intent",
		"detected trigger",
		"extracted entities",
		"extracted facts",
		"scored candidates",
		"reranked candidates",
		"built internal contract",
		"validated internal contract",
		"called firememory",
	})
}

func externalRejectResponse(request contract.ExternalRequest, result validator.Result) contract.OperationResponse {
	return contract.OperationResponse{
		OK:        false,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Rejected:  true,
		Error:     result.Error,
	}
}

func internalRejectResponse(request contract.OperationRequest, result validator.Result) contract.OperationResponse {
	return contract.OperationResponse{
		OK:        false,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Rejected:  true,
		Error:     result.Error,
	}
}

func extractPrimaryText(request contract.ExternalRequest) string {
	for _, key := range []string{"content", "query", "task", "text"} {
		if value, ok := request.Input[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
