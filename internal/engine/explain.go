package engine

import (
	"encoding/json"
	"sort"
	"strings"
)

func (e *Base) Explain(input ExplainInput) (ExplainResult, error) {
	if err := input.Validate(); err != nil {
		return ExplainResult{}, err
	}

	if input.BrainPath != e.Path() {
		return ExplainResult{}, ErrBrainPathMismatch
	}

	trace := append([]string{}, input.Trace...)
	if len(trace) == 0 {
		stored, err := e.loadTraceRecords(input.MemoryID, input.Operation)
		if err != nil {
			return ExplainResult{}, err
		}

		for _, record := range stored {
			trace = append(trace, record.Trace...)
		}
	}

	summary := summarizeExplanation(input.Operation, input.MemoryID, trace)

	return ExplainResult{
		Operation: input.Operation,
		Summary:   summary,
		Trace:     trace,
	}, nil
}

func (e *Base) loadTraceRecords(memoryID, operation string) ([]storedTraceRecord, error) {
	records, err := e.Store().List(tracesNamespace, "", 0)
	if err != nil {
		return nil, err
	}

	matches := make([]storedTraceRecord, 0)
	for _, record := range records {
		var stored storedTraceRecord
		if err := json.Unmarshal(record.Value, &stored); err != nil {
			return nil, err
		}

		if memoryID != "" && stored.MemoryID != memoryID {
			continue
		}

		if operation != "" && stored.Operation != operation && stored.Action != operation {
			continue
		}

		matches = append(matches, stored)
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].CreatedAt.Equal(matches[j].CreatedAt) {
			return matches[i].MemoryID < matches[j].MemoryID
		}
		return matches[i].CreatedAt.Before(matches[j].CreatedAt)
	})

	return matches, nil
}

func summarizeExplanation(operation, memoryID string, trace []string) string {
	var prefix string
	switch operation {
	case "dedup":
		prefix = "Dedup explanation"
	case "recall":
		prefix = "Recall explanation"
	case "context":
		prefix = "Context explanation"
	default:
		prefix = "Operation explanation"
	}

	parts := []string{prefix}
	if memoryID != "" {
		parts = append(parts, "for "+memoryID)
	}

	if len(trace) == 0 {
		return strings.Join(parts, " ") + ": no trace available."
	}

	unique := uniqueTrace(trace)
	return strings.Join(parts, " ") + ": " + strings.Join(unique, "; ") + "."
}

func uniqueTrace(trace []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(trace))
	for _, item := range trace {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

