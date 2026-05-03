package engine

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/phmotad/firememory/internal/dedup"
	"github.com/phmotad/firememory/internal/extractor"
	"github.com/phmotad/firememory/internal/graph"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
	"github.com/phmotad/firememory/internal/vector"
)

const (
	entitiesNamespace  = "entities"
	relationsNamespace = "relations"
	factsNamespace     = "facts"
	defaultSyncTopK    = 3
)

func (e *Base) Sync(input SyncInput) (SyncResult, error) {
	if err := input.Validate(); err != nil {
		return SyncResult{}, err
	}

	if input.BrainPath != e.Path() {
		return SyncResult{}, ErrBrainPathMismatch
	}

	pending, err := e.pendingMemories(input.Limit)
	if err != nil {
		return SyncResult{}, err
	}

	trace := []string{
		"loaded pending_sync memories",
	}
	syncedIDs := make([]string, 0, len(pending))

	for _, mem := range pending {
		if err := e.syncMemory(mem); err != nil {
			return SyncResult{}, err
		}
		syncedIDs = append(syncedIDs, mem.ID)
	}

	trace = append(trace, "executed heuristic extraction", "persisted entities and facts", "classified nearby relations", "updated graph", "marked memories as synced")

	return SyncResult{
		Processed: len(syncedIDs),
		SyncedIDs: syncedIDs,
		Trace:     trace,
	}, nil
}

func (e *Base) pendingMemories(limit int) ([]memory.Memory, error) {
	memories, err := e.listMemories()
	if err != nil {
		return nil, err
	}

	pending := make([]memory.Memory, 0)
	for _, mem := range memories {
		if mem.Status != memory.MemoryStatusPendingSync {
			continue
		}
		pending = append(pending, mem)
	}

	sort.Slice(pending, func(i, j int) bool {
		if pending[i].UpdatedAt.Equal(pending[j].UpdatedAt) {
			return pending[i].ID < pending[j].ID
		}
		return pending[i].UpdatedAt.Before(pending[j].UpdatedAt)
	})

	if limit > 0 && len(pending) > limit {
		pending = pending[:limit]
	}

	return pending, nil
}

func (e *Base) syncMemory(mem memory.Memory) error {
	if err := e.ensureMemoryNode(mem); err != nil {
		return err
	}

	extracted, err := e.extractor.Extract(extractor.Input{
		MemoryID: mem.ID,
		Content:  mem.Content,
		Scope:    mem.Scope,
	})
	if err != nil {
		return err
	}

	for _, entity := range extracted.Entities {
		stored := extractedEntityToDomain(entity, mem.ID)
		if err := persistEntity(e.Store(), stored); err != nil {
			return err
		}

		if err := e.graph.AddNode(graph.Node{
			ID:    stored.ID,
			Label: stored.Name,
			Kind:  memory.MemoryKindConcept,
			Scope: mem.Scope,
			Metadata: map[string]string{
				"entity_type": stored.Type,
			},
		}); err != nil {
			return err
		}

		relation := memory.Relation{
			ID:         relationID(mem.ID, stored.ID, memory.RelationTypeAssociated),
			FromID:     mem.ID,
			ToID:       stored.ID,
			Type:       memory.RelationTypeAssociated,
			Confidence: stored.Confidence,
			CreatedAt:  time.Now().UTC(),
		}
		if err := persistRelation(e.Store(), relation); err != nil {
			return err
		}
		if err := e.graph.AddEdge(graph.Edge{
			ID:     relation.ID,
			FromID: relation.FromID,
			ToID:   relation.ToID,
			Type:   relation.Type,
			Weight: relation.Confidence,
		}); err != nil {
			return err
		}
	}

	for _, fact := range extracted.Facts {
		stored := extractedFactToDomain(fact, mem.ID)
		if err := persistFact(e.Store(), stored); err != nil {
			return err
		}
	}

	if err := e.classifyRelatedMemories(mem); err != nil {
		return err
	}

	mem.Status = memory.MemoryStatusSynced
	mem.UpdatedAt = time.Now().UTC()
	if mem.Metadata == nil {
		mem.Metadata = map[string]string{}
	}
	mem.Metadata["keywords_count"] = strconv.Itoa(len(extracted.Keywords))
	if len(extracted.Keywords) > 0 {
		mem.Metadata["keywords"] = strings.Join(extracted.Keywords, ",")
	}

	if err := e.saveMemory(mem); err != nil {
		return err
	}

	if err := e.Store().Delete(syncQueueNamespace, mem.ID); err != nil {
		return err
	}

	return e.persistTrace(mem.ID, "sync", append([]string{}, extracted.Trace...))
}

func (e *Base) classifyRelatedMemories(mem memory.Memory) error {
	vectorRecord, err := e.Store().Get(vectorsNamespace, mem.ID)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil
		}
		return err
	}

	var stored storedVectorRecord
	if err := json.Unmarshal(vectorRecord, &stored); err != nil {
		return err
	}

	hits, err := e.vectorIndex.Search(vector.SearchInput{
		Vector:   stored.Vector,
		Scope:    mem.Scope,
		TopK:     defaultSyncTopK + 1,
		MinScore: 0.40,
	})
	if err != nil {
		return err
	}

	for _, hit := range hits {
		if hit.ID == mem.ID {
			continue
		}

		other, err := e.loadMemory(hit.ID)
		if err != nil {
			if err == storage.ErrNotFound {
				continue
			}
			return err
		}

		if err := e.ensureMemoryNode(other); err != nil {
			return err
		}

		classified, err := e.relationClassifier.Classify(RelationClassificationInput{
			Left:            mem,
			Right:           other,
			SimilarityScore: hit.Score,
		})
		if err != nil {
			return err
		}

		relation := memory.Relation{
			ID:         relationID(mem.ID, other.ID, classified.Type),
			FromID:     mem.ID,
			ToID:       other.ID,
			Type:       classified.Type,
			Confidence: classified.Confidence,
			CreatedAt:  time.Now().UTC(),
		}

		if err := persistRelation(e.Store(), relation); err != nil {
			return err
		}

		if err := e.graph.AddEdge(graph.Edge{
			ID:     relation.ID,
			FromID: relation.FromID,
			ToID:   relation.ToID,
			Type:   relation.Type,
			Weight: relation.Confidence,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (e *Base) ensureMemoryNode(mem memory.Memory) error {
	return e.graph.AddNode(graph.Node{
		ID:    mem.ID,
		Label: mem.Content,
		Kind:  mem.Kind,
		Scope: mem.Scope,
		Metadata: map[string]string{
			"memory_id": mem.ID,
		},
	})
}

func persistEntity(store storage.Store, entity memory.Entity) error {
	payload, err := json.Marshal(entity)
	if err != nil {
		return err
	}

	return store.Put(entitiesNamespace, entity.ID, payload)
}

func persistFact(store storage.Store, fact memory.Fact) error {
	payload, err := json.Marshal(fact)
	if err != nil {
		return err
	}

	return store.Put(factsNamespace, fact.ID, payload)
}

func persistRelation(store storage.Store, relation memory.Relation) error {
	payload, err := json.Marshal(relation)
	if err != nil {
		return err
	}

	return store.Put(relationsNamespace, relation.ID, payload)
}

func extractedEntityToDomain(entity extractor.ExtractedEntity, sourceMemoryID string) memory.Entity {
	now := time.Now().UTC()
	return memory.Entity{
		ID:             entityID(entity.Type, entity.Name),
		Name:           entity.Name,
		Type:           entity.Type,
		Confidence:     entity.Confidence,
		SourceMemoryID: sourceMemoryID,
		CreatedAt:      now,
	}
}

func extractedFactToDomain(fact extractor.ExtractedFact, sourceMemoryID string) memory.Fact {
	now := time.Now().UTC()
	return memory.Fact{
		ID:             factID(fact.Subject, fact.Predicate, fact.Object),
		Subject:        fact.Subject,
		Predicate:      fact.Predicate,
		Object:         fact.Object,
		Confidence:     fact.Confidence,
		SourceMemoryID: sourceMemoryID,
		CreatedAt:      now,
	}
}

func entityID(typ, name string) string {
	return "entity_" + dedup.HashNormalized(strings.ToLower(typ) + ":" + strings.ToLower(name))[:16]
}

func factID(subject, predicate, object string) string {
	return "fact_" + dedup.HashNormalized(strings.ToLower(subject) + "|" + strings.ToLower(predicate) + "|" + strings.ToLower(object))[:16]
}

func relationID(fromID, toID string, typ memory.RelationType) string {
	sides := []string{fromID, toID}
	sort.Strings(sides)
	return "rel_" + dedup.HashNormalized(sides[0] + "|" + sides[1] + "|" + string(typ))[:16]
}
