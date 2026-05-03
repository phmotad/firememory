# FireQuery Architecture

## Purpose

FireQuery is the cognitive interface layer between external agents and FireMemory.

It accepts MCP requests, validates them, runs small specialist components, and builds a strict internal request for FireMemory.

FireQuery does not access storage directly.

FireQuery does not bypass FireMemory validation.

## Position in the stack

```txt
Agent / LLM
    |
    v
External MCP Contract
    |
    v
FireQuery
|- MCP Server
|- Contract Validator
|- Contract Builder
|- Device Manager
|- IntentClassifier (ModernBERT or DeBERTa small)
|- TriggerClassifier (ModernBERT or DeBERTa small)
|- EntityExtractor (GLiNER / GLiNER2)
|- FactExtractor
|- RelationClassifier
|- SimilarityEngine (multilingual-e5-small)
|- Reranker
|- FireMemoryClient
`- Runtime
    |
    v
Internal FireQuery -> FireMemory Contract
    |
    v
FireMemory
    |
    v
agent.fbrain
```

## Main responsibilities

- Normalize external requests into a stable contract.
- Reject invalid or unsafe requests before touching FireMemory.
- Classify user intent and operation trigger.
- Extract supporting entities and facts when needed.
- Build an internal request in English.
- Send only validated requests to FireMemory.
- Return a structured response through MCP.

## Main rule

Models suggest.

Go validates.

FireMemory executes.

## Boundaries

FireQuery may:

- read external MCP requests
- run lightweight specialist models
- infer intent and enrich requests
- call FireMemory through a strict client

FireQuery may not:

- write directly to `.fbrain`
- bypass contract validation
- expose storage internals
- use a generative SLM in the MVP

## Request flow

1. Receive MCP tool call from an agent.
2. Validate the external contract.
3. Classify intent and trigger.
4. Extract entities, facts, and hints if useful.
5. Build the internal FireMemory request in English.
6. Validate the internal contract.
7. Call FireMemory.
8. Return a structured MCP response.

## Deployment assumptions

- Local-first by default.
- CPU fallback is mandatory.
- GPU acceleration is optional when available.
- Models are loaded lazily.
- FireMemory remains the source of truth.
