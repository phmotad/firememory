package onnx

// IntentLabelDescriptions maps intent class names to English descriptions used
// for embedding-based zero-shot classification.
var IntentLabelDescriptions = map[string]string{
	"remember_information": "Store new durable information in memory.",
	"recall_information":   "Retrieve known information from memory.",
	"build_context":        "Build context and gather relevant information for answering.",
	"explain_decision":     "Explain a decision or reasoning process.",
	"sync_memory":          "Synchronize and consolidate stored memories.",
}

// TriggerLabelDescriptions maps trigger class names to English descriptions.
var TriggerLabelDescriptions = map[string]string{
	"do_nothing":           "No action required.",
	"query_memory":         "Query stored information from memory.",
	"suggest_write":        "Suggest writing or storing information.",
	"request_confirmation": "Request user confirmation before proceeding.",
}

// EntityLabels are the fixed entity types passed to GLiNER at inference time.
var EntityLabels = []string{
	"person",
	"organization",
	"product",
	"technology",
	"version",
	"issue",
	"document",
	"date",
	"location",
}
