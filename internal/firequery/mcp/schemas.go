package mcp

type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Input       SchemaObject   `json:"input"`
	Output      SchemaObject   `json:"output"`
}

type SchemaObject struct {
	Required []string                 `json:"required,omitempty"`
	Fields   map[string]SchemaField   `json:"fields"`
}

type SchemaField struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

const (
	ToolAsk        = "firequery.ask"
	ToolPlan       = "firequery.plan"
	ToolRemember   = "firequery.remember"
	ToolRecall     = "firequery.recall"
	ToolGetContext = "firequery.get_context"
	ToolExplain    = "firequery.explain"
)

func DefaultSchemas() map[string]ToolSchema {
	return map[string]ToolSchema{
		ToolAsk: {
			Name:        ToolAsk,
			Description: "General cognitive entry point for memory-aware agent requests.",
			Input: SchemaObject{
				Required: []string{"version", "request_id", "actor", "brain", "input"},
				Fields: commonInputFields(map[string]SchemaField{
					"operation": {Type: "string", Description: "Optional operation hint. If omitted, FireQuery infers a safe default."},
				}),
			},
			Output: commonOutputSchema(),
		},
		ToolPlan: {
			Name:        ToolPlan,
			Description: "Builds a context-oriented plan without issuing a write operation.",
			Input: SchemaObject{
				Required: []string{"version", "request_id", "actor", "brain", "input"},
				Fields: commonInputFields(map[string]SchemaField{
					"input.task": {Type: "string", Description: "Task or objective to plan around."},
				}),
			},
			Output: commonOutputSchema(),
		},
		ToolRemember: {
			Name:        ToolRemember,
			Description: "Suggests and validates a remember request for FireMemory.",
			Input: SchemaObject{
				Required: []string{"version", "request_id", "actor", "brain", "input"},
				Fields: commonInputFields(map[string]SchemaField{
					"input.content": {Type: "string", Description: "Content to store in memory."},
				}),
			},
			Output: commonOutputSchema(),
		},
		ToolRecall: {
			Name:        ToolRecall,
			Description: "Recalls relevant memories for a query.",
			Input: SchemaObject{
				Required: []string{"version", "request_id", "actor", "brain", "input"},
				Fields: commonInputFields(map[string]SchemaField{
					"input.query": {Type: "string", Description: "Memory query text."},
				}),
			},
			Output: commonOutputSchema(),
		},
		ToolGetContext: {
			Name:        ToolGetContext,
			Description: "Builds structured context for a task or response.",
			Input: SchemaObject{
				Required: []string{"version", "request_id", "actor", "brain", "input"},
				Fields: commonInputFields(map[string]SchemaField{
					"input.task": {Type: "string", Description: "Task description used to build context."},
				}),
			},
			Output: commonOutputSchema(),
		},
		ToolExplain: {
			Name:        ToolExplain,
			Description: "Explains a retrieval, dedup, or context-building decision.",
			Input: SchemaObject{
				Required: []string{"version", "request_id", "actor", "brain", "input"},
				Fields: commonInputFields(map[string]SchemaField{
					"input.target_operation": {Type: "string", Description: "Operation being explained, such as recall or get_context."},
					"input.memory_id":        {Type: "string", Description: "Optional target memory identifier."},
				}),
			},
			Output: commonOutputSchema(),
		},
	}
}

func commonInputFields(extra map[string]SchemaField) map[string]SchemaField {
	fields := map[string]SchemaField{
		"version":    {Type: "string", Description: "Contract version."},
		"request_id": {Type: "string", Description: "Unique request identifier."},
		"language":   {Type: "string", Description: "User-facing language."},
		"actor":      {Type: "object", Description: "External caller identity."},
		"brain":      {Type: "string", Description: "Target .fbrain file."},
		"input":      {Type: "object", Description: "Tool-specific input payload."},
	}
	for key, value := range extra {
		fields[key] = value
	}
	return fields
}

func commonOutputSchema() SchemaObject {
	return SchemaObject{
		Fields: map[string]SchemaField{
			"ok":         {Type: "boolean", Description: "Whether the request succeeded."},
			"request_id": {Type: "string", Description: "Echoed request identifier."},
			"operation":  {Type: "string", Description: "Resolved FireMemory operation."},
			"data":       {Type: "object", Description: "Structured response payload."},
			"trace":      {Type: "object", Description: "Optional trace information."},
			"rejected":   {Type: "boolean", Description: "Whether the request was rejected before execution."},
			"error":      {Type: "object", Description: "Error payload for rejected or failed requests."},
		},
	}
}
