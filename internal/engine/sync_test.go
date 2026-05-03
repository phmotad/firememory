package engine

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/embedder"
	"github.com/phmotad/firememory/internal/extractor"
	"github.com/phmotad/firememory/internal/memory"
)

func TestSyncProcessesPendingMemoriesAndMarksSynced(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "sync.fbrain"), nil, 384)
	defer engine.Close()

	first, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente Joao usa Firebird 2.5 e teve erro fiscal na NF-e apos atualizacao 3.2",
	})
	if err != nil {
		t.Fatalf("remember first: %v", err)
	}

	second, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Joao relatou novamente problema fiscal na NF-e apos a versao 3.2",
	})
	if err != nil {
		t.Fatalf("remember second: %v", err)
	}

	result, err := engine.Sync(SyncInput{
		BrainPath: engine.Path(),
	})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if result.Processed != 2 {
		t.Fatalf("expected 2 processed memories, got %d", result.Processed)
	}

	mem1, err := engine.loadMemory(first.Memory.ID)
	if err != nil {
		t.Fatalf("load mem1: %v", err)
	}
	if mem1.Status != memory.MemoryStatusSynced {
		t.Fatalf("expected mem1 synced, got %q", mem1.Status)
	}

	mem2, err := engine.loadMemory(second.Memory.ID)
	if err != nil {
		t.Fatalf("load mem2: %v", err)
	}
	if mem2.Status != memory.MemoryStatusSynced {
		t.Fatalf("expected mem2 synced, got %q", mem2.Status)
	}

	entityRecords, err := engine.Store().List(entitiesNamespace, "", 0)
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}
	if len(entityRecords) == 0 {
		t.Fatal("expected extracted entities to be persisted")
	}

	factRecords, err := engine.Store().List(factsNamespace, "", 0)
	if err != nil {
		t.Fatalf("list facts: %v", err)
	}
	if len(factRecords) == 0 {
		t.Fatal("expected extracted facts to be persisted")
	}

	relationRecords, err := engine.Store().List(relationsNamespace, "", 0)
	if err != nil {
		t.Fatalf("list relations: %v", err)
	}
	if len(relationRecords) == 0 {
		t.Fatal("expected relations to be persisted")
	}

	if engine.Graph().NodeCount() < 2 {
		t.Fatalf("expected graph nodes to be created, got %d", engine.Graph().NodeCount())
	}
}

func TestSyncRespectsLimit(t *testing.T) {
	engine := openTestEngine(t, filepath.Join(t.TempDir(), "sync-limit.fbrain"), nil, 384)
	defer engine.Close()

	for _, content := range []string{
		"Cliente A usa Firebird 2.5",
		"Cliente B usa Firebird 3.0",
		"Cliente C usa Firebird 4.0",
	} {
		if _, err := engine.Remember(RememberInput{
			BrainPath: engine.Path(),
			Content:   content,
		}); err != nil {
			t.Fatalf("remember %q: %v", content, err)
		}
	}

	result, err := engine.Sync(SyncInput{
		BrainPath: engine.Path(),
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if result.Processed != 2 {
		t.Fatalf("expected 2 processed memories, got %d", result.Processed)
	}

	memories, err := engine.listMemories()
	if err != nil {
		t.Fatalf("list memories: %v", err)
	}

	pending := 0
	for _, mem := range memories {
		if mem.Status == memory.MemoryStatusPendingSync {
			pending++
		}
	}

	if pending != 1 {
		t.Fatalf("expected 1 pending memory after limited sync, got %d", pending)
	}
}

func TestSyncUsesInjectedExtractor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync-custom.fbrain")
	handle, err := brainfile.Create(path, brainfile.CreateOptions{
		Name:           "custom",
		EmbeddingModel: brainfile.DefaultEmbedder,
		EmbeddingDim:   4,
	})
	if err != nil {
		t.Fatalf("create brainfile: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("close brainfile: %v", err)
	}

	engine, err := Open(Options{
		Path: path,
		Embedder: &recordingEmbedder{
			dimension: 4,
			vectors: map[string]embedder.Vector{
				"Cliente A usa Firebird 2.5": {1, 0, 0, 0},
			},
		},
		Extractor: stubExtractor{
			result: extractor.Result{
				Entities: []extractor.ExtractedEntity{
					{Name: "Cliente A", Type: "proper_name", Confidence: 0.9},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("open engine: %v", err)
	}
	defer engine.Close()

	remembered, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente A usa Firebird 2.5",
	})
	if err != nil {
		t.Fatalf("remember: %v", err)
	}

	if _, err := engine.Sync(SyncInput{BrainPath: engine.Path()}); err != nil {
		t.Fatalf("sync: %v", err)
	}

	records, err := engine.Store().List(entitiesNamespace, "", 0)
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(records))
	}

	var entity memory.Entity
	if err := json.Unmarshal(records[0].Value, &entity); err != nil {
		t.Fatalf("unmarshal entity: %v", err)
	}

	if entity.SourceMemoryID != remembered.Memory.ID {
		t.Fatalf("expected source memory id %q, got %q", remembered.Memory.ID, entity.SourceMemoryID)
	}
}

func TestSyncCreatesGraphNodesForRelatedPendingMemories(t *testing.T) {
	custom := &recordingEmbedder{
		dimension: 4,
		vectors: map[string]embedder.Vector{
			"Cliente A teve erro fiscal na NF-e":          {1, 0, 0, 0},
			"Cliente A reportou novo erro fiscal na NF-e": {0.7, 0.7, 0, 0},
		},
	}

	engine := openTestEngine(t, filepath.Join(t.TempDir(), "sync-related.fbrain"), custom, 4)
	defer engine.Close()

	first, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente A teve erro fiscal na NF-e",
	})
	if err != nil {
		t.Fatalf("remember first: %v", err)
	}

	second, err := engine.Remember(RememberInput{
		BrainPath: engine.Path(),
		Content:   "Cliente A reportou novo erro fiscal na NF-e",
	})
	if err != nil {
		t.Fatalf("remember second: %v", err)
	}

	result, err := engine.Sync(SyncInput{BrainPath: engine.Path()})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if result.Processed != 2 {
		t.Fatalf("expected 2 processed memories, got %d", result.Processed)
	}

	if _, err := engine.Graph().GetNode(first.Memory.ID); err != nil {
		t.Fatalf("expected first memory node in graph: %v", err)
	}

	if _, err := engine.Graph().GetNode(second.Memory.ID); err != nil {
		t.Fatalf("expected second memory node in graph: %v", err)
	}

	if engine.Graph().EdgeCount() == 0 {
		t.Fatal("expected graph edge to be created for related memories")
	}
}

type stubExtractor struct {
	result extractor.Result
	err    error
}

func (s stubExtractor) Extract(input extractor.Input) (extractor.Result, error) {
	if s.err != nil {
		return extractor.Result{}, s.err
	}
	return s.result, nil
}
