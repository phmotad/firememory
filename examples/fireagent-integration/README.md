# FireQuery Agent Integration Example

This example shows the supported external integration path:

Agent -> FireQuery MCP -> FireMemory

Agents should not call FireMemory directly.

## Available MCP Tools

- `firequery.ask`
- `firequery.plan`
- `firequery.remember`
- `firequery.recall`
- `firequery.get_context`
- `firequery.explain`

## Sample Calls

Remember:

```json
{
  "tool": "firequery.remember",
  "arguments": {
    "version": "0.1",
    "request_id": "req_remember_001",
    "language": "pt-BR",
    "actor": {
      "type": "agent",
      "id": "support-agent"
    },
    "brain": "./agent.fbrain",
    "input": {
      "content": "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2",
      "allow_write": true
    }
  }
}
```

Get context:

```json
{
  "tool": "firequery.get_context",
  "arguments": {
    "version": "0.1",
    "request_id": "req_context_001",
    "language": "pt-BR",
    "actor": {
      "type": "agent",
      "id": "support-agent"
    },
    "brain": "./agent.fbrain",
    "input": {
      "task": "responder Joao sobre erro fiscal apos atualizacao",
      "top_k": 5,
      "budget_tokens": 800,
      "include_graph": true,
      "include_trace": true
    }
  }
}
```

Explain:

```json
{
  "tool": "firequery.explain",
  "arguments": {
    "version": "0.1",
    "request_id": "req_explain_001",
    "language": "pt-BR",
    "actor": {
      "type": "agent",
      "id": "support-agent"
    },
    "brain": "./agent.fbrain",
    "input": {
      "target_operation": "get_context"
    }
  }
}
```

## Notes

- The agent talks to FireQuery, not to FireMemory directly.
- FireQuery validates the external request and builds the internal contract.
- FireMemory remains behind the FireQuery boundary.
