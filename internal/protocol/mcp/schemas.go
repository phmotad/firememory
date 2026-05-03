package mcp

func rememberSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{
			"brain_path",
			"content",
		},
		"properties": map[string]any{
			"brain_path": map[string]any{"type": "string"},
			"content":    map[string]any{"type": "string"},
			"scope":      map[string]any{"type": "string"},
			"kind":       map[string]any{"type": "string"},
			"metadata":   map[string]any{"type": "object"},
		},
	}
}

func recallSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{
			"brain_path",
			"query",
		},
		"properties": map[string]any{
			"brain_path":    map[string]any{"type": "string"},
			"query":         map[string]any{"type": "string"},
			"scope":         map[string]any{"type": "string"},
			"top_k":         map[string]any{"type": "integer"},
			"include_trace": map[string]any{"type": "boolean"},
		},
	}
}

func contextSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{
			"brain_path",
			"query",
		},
		"properties": map[string]any{
			"brain_path":    map[string]any{"type": "string"},
			"query":         map[string]any{"type": "string"},
			"scope":         map[string]any{"type": "string"},
			"top_k":         map[string]any{"type": "integer"},
			"budget_tokens": map[string]any{"type": "integer"},
			"include_graph": map[string]any{"type": "boolean"},
			"include_trace": map[string]any{"type": "boolean"},
		},
	}
}

func syncSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{
			"brain_path",
		},
		"properties": map[string]any{
			"brain_path": map[string]any{"type": "string"},
			"limit":      map[string]any{"type": "integer"},
		},
	}
}

func explainSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{
			"brain_path",
			"operation",
		},
		"properties": map[string]any{
			"brain_path": map[string]any{"type": "string"},
			"operation":  map[string]any{"type": "string"},
			"memory_id":  map[string]any{"type": "string"},
			"trace": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		},
	}
}
