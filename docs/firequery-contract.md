# FireQuery Contract

## Purpose

FireQuery sits between external agents and FireMemory.

It accepts an external request, validates it, builds an internal request, validates that request again, and only then calls FireMemory.

## Contract layers

FireQuery has two contract layers:

1. Agent -> FireQuery
2. FireQuery -> FireMemory

The external contract may use the user language.

The internal contract must always be in English.

## External contract

### Required fields

- `version`
- `request_id`
- `actor`
- `operation`
- `brain`
- `input`

### Example

```json
{
  "version": "0.1",
  "request_id": "req_ext_001",
  "language": "pt-BR",
  "actor": {
    "type": "agent",
    "id": "support-agent"
  },
  "operation": "get_context",
  "brain": "agent.fbrain",
  "input": {
    "task": "responder Joao sobre erro fiscal"
  }
}
```

## Internal contract

### Main rule

All FireQuery -> FireMemory requests must be in English and must follow a rigid schema.

### Required fields

- `version`
- `request_id`
- `language = "en"`
- `actor`
- `operation`
- `intent`
- `brain`
- `scope`
- `permissions`

Thresholds are required when the operation needs ranking or filtering.

### Example

```json
{
  "version": "0.1",
  "request_id": "req_int_001",
  "language": "en",
  "actor": {
    "type": "firequery",
    "id": "firequery-mcp"
  },
  "operation": "get_context",
  "intent": "build_context",
  "brain": "agent.fbrain",
  "scope": "default",
  "input": {
    "task": "answer Joao about fiscal error after update"
  },
  "permissions": {
    "allow_write": false,
    "requires_confirmation": false
  },
  "thresholds": {
    "top_k": 8,
    "similarity_threshold": 0.7,
    "budget_tokens": 2000
  },
  "options": {
    "include_graph": true,
    "include_trace": true
  }
}
```

## Validation rules

Reject if:

- `version` is missing
- `request_id` is missing
- `language` is not `en` in the internal contract
- `actor` is missing
- `operation` is missing
- `intent` is missing
- `brain` is missing
- `brain` does not end with `.fbrain`
- `scope` is missing
- `permissions` is missing
- a write operation has `allow_write = false`
- `forget` does not require confirmation
- thresholds are outside valid bounds
- `top_k` is invalid
- `budget_tokens` is invalid
- `operation` is unknown
- `intent` does not match `operation`

## Rejection response

```json
{
  "ok": false,
  "rejected": true,
  "error": {
    "code": "CONTRACT_VALIDATION_FAILED",
    "message": "Missing required field: scope"
  }
}
```

## Safety rule

FireQuery must never call FireMemory if the request is invalid.
