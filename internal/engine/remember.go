package engine

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/phmotad/firememory/internal/dedup"
	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/vector"
)

type storedQueueRecord struct {
	MemoryID string    `json:"memory_id"`
	QueuedAt time.Time `json:"queued_at"`
}

type storedTraceRecord struct {
	Operation string    `json:"operation"`
	MemoryID  string    `json:"memory_id"`
	Action    string    `json:"action"`
	Trace     []string  `json:"trace"`
	CreatedAt time.Time `json:"created_at"`
}

func (e *Base) Remember(input RememberInput) (RememberResult, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return RememberResult{}, err
	}

	if input.BrainPath != e.Path() {
		return RememberResult{}, ErrBrainPathMismatch
	}

	embedding, err := e.embedder.Embed(context.Background(), input.Content)
	if err != nil {
		return RememberResult{}, err
	}

	detector, err := dedup.NewDetector(dedup.Config{})
	if err != nil {
		return RememberResult{}, err
	}

	dedupResult, err := detector.Detect(dedup.Input{
		Content:   input.Content,
		Scope:     input.Scope,
		Kind:      input.Kind,
		Vector:    embedding,
		HashIndex: e.hashIndex,
		Index:     e.vectorIndex,
	})
	if err != nil {
		return RememberResult{}, err
	}

	if dedupResult.Action == memory.DedupActionReinforce {
		return e.reinforceExistingMemory(input, dedupResult)
	}

	return e.createNewMemory(input, dedupResult, embedding)
}

func (e *Base) reinforceExistingMemory(input RememberInput, match dedup.Result) (RememberResult, error) {
	mem, err := e.loadMemory(match.MatchedMemoryID)
	if err != nil {
		return RememberResult{}, err
	}

	mem.Normalize()
	mem.UpdatedAt = time.Now().UTC()
	mem.Status = memory.MemoryStatusPendingSync
	mem.SourceRefs = append(mem.SourceRefs, input.SourceRefs...)
	if mem.Metadata == nil {
		mem.Metadata = map[string]string{}
	}

	count, _ := strconv.Atoi(mem.Metadata["reinforced_count"])
	mem.Metadata["reinforced_count"] = strconv.Itoa(count + 1)
	mem.Metadata["last_reinforced_by"] = string(match.MatchType)

	if err := e.saveMemory(mem); err != nil {
		return RememberResult{}, err
	}

	if err := e.persistHashMatch(match.Hash, mem.ID); err != nil {
		return RememberResult{}, err
	}

	if err := e.enqueueSync(mem.ID); err != nil {
		return RememberResult{}, err
	}

	trace := append([]string{}, match.Trace...)
	trace = append(trace, "reinforced existing memory", "queued memory for sync")

	if err := e.persistTrace(mem.ID, string(match.Action), trace); err != nil {
		return RememberResult{}, err
	}

	return RememberResult{
		Memory:             mem,
		DedupAction:        match.Action,
		ReinforcedMemoryID: mem.ID,
		Trace:              trace,
	}, nil
}

func (e *Base) createNewMemory(input RememberInput, match dedup.Result, embedding embedder.Vector) (RememberResult, error) {
	now := time.Now().UTC()
	mem := memory.Memory{
		ID:                newMemoryID(now),
		Content:           input.Content,
		NormalizedContent: match.NormalizedText,
		Hash:              match.Hash,
		Kind:              input.Kind,
		Status:            memory.MemoryStatusPendingSync,
		Scope:             input.Scope,
		Importance:        0.5,
		Confidence:        1.0,
		EmbeddingModel:    e.embedder.Name(),
		EmbeddingDim:      e.embedder.Dimension(),
		SourceRefs:        append([]memory.SourceRef{}, input.SourceRefs...),
		Metadata:          cloneStringMap(input.Metadata),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	mem.Normalize()

	if err := mem.Validate(); err != nil {
		return RememberResult{}, err
	}

	if err := e.saveMemory(mem); err != nil {
		return RememberResult{}, err
	}

	if err := e.persistVector(mem, embedding); err != nil {
		return RememberResult{}, err
	}

	if err := e.vectorIndex.Add(vector.Entry{
		ID:     mem.ID,
		Vector: embedding,
		Scope:  mem.Scope,
		Kind:   mem.Kind,
	}); err != nil {
		return RememberResult{}, err
	}

	if err := e.persistHashMatch(mem.Hash, mem.ID); err != nil {
		return RememberResult{}, err
	}

	if err := e.enqueueSync(mem.ID); err != nil {
		return RememberResult{}, err
	}

	trace := append([]string{}, match.Trace...)
	trace = append(trace, "created new memory", "persisted vector entry", "queued memory for sync")

	if err := e.persistTrace(mem.ID, string(match.Action), trace); err != nil {
		return RememberResult{}, err
	}

	return RememberResult{
		Memory:      mem,
		DedupAction: match.Action,
		Trace:       trace,
	}, nil
}

func (e *Base) loadMemory(id string) (memory.Memory, error) {
	payload, err := e.Store().Get(memoriesNamespace, id)
	if err != nil {
		return memory.Memory{}, err
	}

	var mem memory.Memory
	if err := json.Unmarshal(payload, &mem); err != nil {
		return memory.Memory{}, err
	}

	return mem, nil
}

func (e *Base) saveMemory(mem memory.Memory) error {
	payload, err := json.Marshal(mem)
	if err != nil {
		return err
	}

	return e.Store().Put(memoriesNamespace, mem.ID, payload)
}

func (e *Base) persistVector(mem memory.Memory, values embedder.Vector) error {
	payload, err := json.Marshal(storedVectorRecord{
		Vector: values,
		Scope:  mem.Scope,
		Kind:   mem.Kind,
	})
	if err != nil {
		return err
	}

	return e.Store().Put(vectorsNamespace, mem.ID, payload)
}

func (e *Base) persistHashMatch(hash, memoryID string) error {
	e.hashIndex.Set(hash, memoryID)
	return e.Store().Put(hashIndexNamespace, hash, []byte(memoryID))
}

func (e *Base) enqueueSync(memoryID string) error {
	payload, err := json.Marshal(storedQueueRecord{
		MemoryID: memoryID,
		QueuedAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	return e.Store().Put(syncQueueNamespace, memoryID, payload)
}

func (e *Base) persistTrace(memoryID, action string, trace []string) error {
	record := storedTraceRecord{
		Operation: "remember",
		MemoryID:  memoryID,
		Action:    action,
		Trace:     append([]string{}, trace...),
		CreatedAt: time.Now().UTC(),
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return e.Store().Put(tracesNamespace, newTraceID(record.CreatedAt, memoryID), payload)
}

func newMemoryID(now time.Time) string {
	return "mem_" + now.Format("20060102150405.000000000")
}

func newTraceID(now time.Time, memoryID string) string {
	return "trace_" + now.Format("20060102150405.000000000") + "_" + memoryID
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}

	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
