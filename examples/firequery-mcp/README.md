# FireQuery MCP Examples

## Tool Surface

FireQuery exposes these MCP tools:

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## Example: `firequery.ask`

```json
{
  "version": "0.1",
  "request_id": "req_ask_001",
  "language": "pt-BR",
  "actor": {
    "type": "agent",
    "id": "support-agent"
  },
  "brain": "./agent.fbrain",
  "input": {
    "task": "responder Joao sobre erro fiscal apos atualizacao"
  }
}
```

## Example: `firequery.plan`

```json
{
  "version": "0.1",
  "request_id": "req_plan_001",
  "language": "en",
  "actor": {
    "type": "agent",
    "id": "planner-agent"
  },
  "brain": "./agent.fbrain",
  "input": {
    "task": "prepare a support response for Joao"
  }
}
```

## Example: `firequery.remember`

```json
{
  "version": "0.1",
  "request_id": "req_remember_001",
  "language": "en",
  "actor": {
    "type": "agent",
    "id": "support-agent"
  },
  "brain": "./agent.fbrain",
  "input": {
    "content": "Client Joao uses Firebird 2.5 and reported a fiscal NF-e error after update 3.2"
  }
}
```

## Example: `firequery.recall`

```json
{
  "version": "0.1",
  "request_id": "req_recall_001",
  "language": "en",
  "actor": {
    "type": "agent",
    "id": "support-agent"
  },
  "brain": "./agent.fbrain",
  "input": {
    "query": "fiscal NF-e error",
    "top_k": 5
  }
}
```

## Example: `firequery.get_context`

```json
{
  "version": "0.1",
  "request_id": "req_context_001",
  "language": "en",
  "actor": {
    "type": "agent",
    "id": "support-agent"
  },
  "brain": "./agent.fbrain",
  "input": {
    "task": "answer Joao about the fiscal error after update",
    "budget_tokens": 1500
  }
}
```

## Example: `firequery.explain`

```json
{
  "version": "0.1",
  "request_id": "req_explain_001",
  "language": "en",
  "actor": {
    "type": "agent",
    "id": "support-agent"
  },
  "brain": "./agent.fbrain",
  "input": {
    "target_operation": "recall",
    "memory_id": "mem_123"
  }
}
```
