# FireQuery Models

## Goal

FireQuery uses small specialist components.

The MVP does not use a generative SLM.

## Mandatory specialists

These components are mandatory in the FireQuery architecture:

- `SimilarityEngine`: `intfloat/multilingual-e5-small`
- `EntityExtractor`: `GLiNER` or `GLiNER2`
- `IntentClassifier`: `ModernBERT` or a small `DeBERTa`
- `TriggerClassifier`: `ModernBERT` or a small `DeBERTa`
- `ContractBuilder`: Go
- `ContractValidator`: Go
- `DeviceManager`: Go

## Current required model bindings

- `SimilarityEngine`: `intfloat/multilingual-e5-small`
- `EntityExtractor`: `GLiNER2` adapter with local fallback
- `IntentClassifier`: `microsoft/deberta-v3-small` adapter with local fallback
- `TriggerClassifier`: `microsoft/deberta-v3-small` adapter with local fallback

## Specialists

### IntentClassifier

Maps a user request to an allowed FireMemory intent such as:

- `remember_information`
- `recall_information`
- `build_context`
- `explain_decision`
- `sync_memory`

### TriggerClassifier

Decides whether the user message should:

- do nothing
- query memory
- suggest a write
- request confirmation

### EntityExtractor

Extracts candidate entities from the user request or recalled context.

### FactExtractor

Extracts candidate facts and structured memory claims.

### RelationClassifier

Suggests relation types between new information and known memory:

- `duplicate`
- `reinforce`
- `complement`
- `update`
- `conflict`

### SimilarityEngine

Scores candidate memories or candidate facts before reranking.

### Reranker

Produces a stable final ordering for retrieved or generated candidates.

## Later-stage specialists

These remain important but are lower-priority than the mandatory set above:

- `RelationClassifier`
- `Reranker`
- a more advanced `FactExtractor`

## Design rules

- Each specialist must have a narrow interface.
- Each specialist must be replaceable by mocks in tests.
- Each specialist must support deterministic test behavior.
- Specialists do not execute writes directly.
- Specialists return suggestions, scores, and traces.

## Model policy

- Prefer lightweight local models.
- CPU fallback is required.
- GPU support is optional.
- Lazy loading is preferred.
- Batch size and memory budget must be configurable.

## Testing policy

- Unit tests use mocks or deterministic implementations.
- Runtime selection must be testable without hardware dependency.
- FireMemory integration tests must not require external model downloads.
