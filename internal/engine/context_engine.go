package engine

import (
	"encoding/json"
	"sort"
	"strings"

	fmcontext "github.com/phmotad/firememory/internal/context"
	"github.com/phmotad/firememory/internal/memory"
	"github.com/phmotad/firememory/internal/storage"
)

func (e *Base) Context(input ContextInput) (ContextResult, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return ContextResult{}, err
	}

	if input.BrainPath != e.Path() {
		return ContextResult{}, ErrBrainPathMismatch
	}

	recallResult, err := e.Recall(RecallInput{
		BrainPath:    input.BrainPath,
		Query:        input.Query,
		Scope:        input.Scope,
		TopK:         input.TopK,
		IncludeTrace: input.IncludeTrace,
	})
	if err != nil {
		return ContextResult{}, err
	}

	memories := orderedUniqueMemories(recallResult.Hits)
	trace := []string{
		"recalled relevant memories",
	}

	if input.IncludeGraph {
		expanded, err := e.expandGraphMemories(memories)
		if err != nil {
			return ContextResult{}, err
		}
		memories = mergeMemories(memories, expanded)
		trace = append(trace, "expanded graph neighborhood")
	}

	entities, err := e.loadEntitiesForMemories(memories)
	if err != nil {
		return ContextResult{}, err
	}

	facts, err := e.loadFactsForMemories(memories)
	if err != nil {
		return ContextResult{}, err
	}

	relations, err := e.loadRelationsForMemories(memories)
	if err != nil {
		return ContextResult{}, err
	}

	trace = append(trace, "loaded related entities, facts, and relations")

	selectedMemories, selectedEntities, selectedFacts, selectedRelations := fitContextBudget(memories, entities, facts, relations, input.BudgetTokens)
	selectedMemories, selectedEntities, selectedFacts, selectedRelations = enforceRenderedBudget(selectedMemories, selectedEntities, selectedFacts, selectedRelations, input.BudgetTokens)
	contextText := fmcontext.BuildText(selectedMemories, selectedEntities, selectedFacts, selectedRelations)
	estimatedTokens := fmcontext.EstimateTokens(contextText)
	trace = append(trace, "built context text within budget")

	if !input.IncludeTrace {
		trace = nil
	}

	return ContextResult{
		Memories:        selectedMemories,
		Entities:        selectedEntities,
		Facts:           selectedFacts,
		Relations:       selectedRelations,
		ContextText:     contextText,
		EstimatedTokens: estimatedTokens,
		Trace:           trace,
	}, nil
}

func orderedUniqueMemories(hits []RecallHit) []memory.Memory {
	memories := make([]memory.Memory, 0, len(hits))
	seen := map[string]struct{}{}
	for _, hit := range hits {
		if _, ok := seen[hit.Memory.ID]; ok {
			continue
		}
		seen[hit.Memory.ID] = struct{}{}
		memories = append(memories, hit.Memory)
	}
	return memories
}

func (e *Base) expandGraphMemories(seed []memory.Memory) ([]memory.Memory, error) {
	expanded := make([]memory.Memory, 0)
	seen := map[string]struct{}{}
	for _, mem := range seed {
		seen[mem.ID] = struct{}{}
	}

	for _, mem := range seed {
		neighbors, err := e.Graph().Related(mem.ID, 1)
		if err != nil {
			if err == storage.ErrNotFound {
				continue
			}
			// graph package returns its own ErrNodeNotFound; ignore missing nodes.
			continue
		}

		for _, node := range neighbors {
			if _, ok := seen[node.ID]; ok {
				continue
			}

			relatedMemory, err := e.loadMemory(node.ID)
			if err != nil {
				if err == storage.ErrNotFound {
					continue
				}
				continue
			}

			seen[node.ID] = struct{}{}
			expanded = append(expanded, relatedMemory)
		}
	}

	return expanded, nil
}

func mergeMemories(primary, extra []memory.Memory) []memory.Memory {
	out := append([]memory.Memory{}, primary...)
	seen := map[string]struct{}{}
	for _, mem := range out {
		seen[mem.ID] = struct{}{}
	}
	for _, mem := range extra {
		if _, ok := seen[mem.ID]; ok {
			continue
		}
		seen[mem.ID] = struct{}{}
		out = append(out, mem)
	}
	return out
}

func (e *Base) loadEntitiesForMemories(memories []memory.Memory) ([]memory.Entity, error) {
	records, err := e.Store().List(entitiesNamespace, "", 0)
	if err != nil {
		return nil, err
	}

	memoryIDs := memoryIDSet(memories)
	entities := make([]memory.Entity, 0)
	for _, record := range records {
		var entity memory.Entity
		if err := json.Unmarshal(record.Value, &entity); err != nil {
			return nil, err
		}
		if _, ok := memoryIDs[entity.SourceMemoryID]; ok {
			entities = append(entities, entity)
		}
	}

	sort.Slice(entities, func(i, j int) bool {
		if entities[i].SourceMemoryID == entities[j].SourceMemoryID {
			return entities[i].Name < entities[j].Name
		}
		return entities[i].SourceMemoryID < entities[j].SourceMemoryID
	})
	return entities, nil
}

func (e *Base) loadFactsForMemories(memories []memory.Memory) ([]memory.Fact, error) {
	records, err := e.Store().List(factsNamespace, "", 0)
	if err != nil {
		return nil, err
	}

	memoryIDs := memoryIDSet(memories)
	facts := make([]memory.Fact, 0)
	for _, record := range records {
		var fact memory.Fact
		if err := json.Unmarshal(record.Value, &fact); err != nil {
			return nil, err
		}
		if _, ok := memoryIDs[fact.SourceMemoryID]; ok {
			facts = append(facts, fact)
		}
	}

	sort.Slice(facts, func(i, j int) bool {
		if facts[i].SourceMemoryID == facts[j].SourceMemoryID {
			return facts[i].Predicate < facts[j].Predicate
		}
		return facts[i].SourceMemoryID < facts[j].SourceMemoryID
	})
	return facts, nil
}

func (e *Base) loadRelationsForMemories(memories []memory.Memory) ([]memory.Relation, error) {
	records, err := e.Store().List(relationsNamespace, "", 0)
	if err != nil {
		return nil, err
	}

	memoryIDs := memoryIDSet(memories)
	relations := make([]memory.Relation, 0)
	for _, record := range records {
		var relation memory.Relation
		if err := json.Unmarshal(record.Value, &relation); err != nil {
			return nil, err
		}
		if _, left := memoryIDs[relation.FromID]; left {
			relations = append(relations, relation)
			continue
		}
		if _, right := memoryIDs[relation.ToID]; right {
			relations = append(relations, relation)
		}
	}

	sort.Slice(relations, func(i, j int) bool {
		if relations[i].FromID == relations[j].FromID {
			if relations[i].ToID == relations[j].ToID {
				return relations[i].Type < relations[j].Type
			}
			return relations[i].ToID < relations[j].ToID
		}
		return relations[i].FromID < relations[j].FromID
	})
	return relations, nil
}

func memoryIDSet(memories []memory.Memory) map[string]struct{} {
	set := make(map[string]struct{}, len(memories))
	for _, mem := range memories {
		set[mem.ID] = struct{}{}
	}
	return set
}

func fitContextBudget(memories []memory.Memory, entities []memory.Entity, facts []memory.Fact, relations []memory.Relation, budget int) ([]memory.Memory, []memory.Entity, []memory.Fact, []memory.Relation) {
	if budget <= 0 {
		return memories, entities, facts, relations
	}

	selectedMemories := make([]memory.Memory, 0, len(memories))
	selectedEntities := make([]memory.Entity, 0, len(entities))
	selectedFacts := make([]memory.Fact, 0, len(facts))
	selectedRelations := make([]memory.Relation, 0, len(relations))

	used := 0
	for _, mem := range memories {
		cost := fmcontext.EstimateTokens(mem.Content)
		if used+cost > budget && len(selectedMemories) > 0 {
			break
		}
		selectedMemories = append(selectedMemories, mem)
		used += cost
	}

	selectedMemoryIDs := memoryIDSet(selectedMemories)

	for _, entity := range entities {
		if _, ok := selectedMemoryIDs[entity.SourceMemoryID]; !ok {
			continue
		}
		cost := fmcontext.EstimateTokens(entity.Name + " " + entity.Type)
		if used+cost > budget {
			break
		}
		selectedEntities = append(selectedEntities, entity)
		used += cost
	}

	for _, fact := range facts {
		if _, ok := selectedMemoryIDs[fact.SourceMemoryID]; !ok {
			continue
		}
		cost := fmcontext.EstimateTokens(fact.Subject + " " + fact.Predicate + " " + fact.Object)
		if used+cost > budget {
			break
		}
		selectedFacts = append(selectedFacts, fact)
		used += cost
	}

	for _, relation := range relations {
		_, left := selectedMemoryIDs[relation.FromID]
		_, right := selectedMemoryIDs[relation.ToID]
		if !left && !right {
			continue
		}
		cost := fmcontext.EstimateTokens(relation.FromID + " " + string(relation.Type) + " " + relation.ToID)
		if used+cost > budget {
			break
		}
		selectedRelations = append(selectedRelations, relation)
		used += cost
	}

	return selectedMemories, selectedEntities, selectedFacts, selectedRelations
}

func enforceRenderedBudget(memories []memory.Memory, entities []memory.Entity, facts []memory.Fact, relations []memory.Relation, budget int) ([]memory.Memory, []memory.Entity, []memory.Fact, []memory.Relation) {
	if budget <= 0 {
		return memories, entities, facts, relations
	}

	for {
		rendered := fmcontext.BuildText(memories, entities, facts, relations)
		if fmcontext.EstimateTokens(rendered) <= budget {
			return memories, entities, facts, relations
		}

		switch {
		case len(relations) > 0:
			relations = relations[:len(relations)-1]
		case len(facts) > 0:
			facts = facts[:len(facts)-1]
		case len(entities) > 0:
			entities = entities[:len(entities)-1]
		case len(memories) > 1:
			memories = memories[:len(memories)-1]
		default:
			if len(memories) == 1 {
				memories[0].Content = truncateToBudget(memories[0].Content, budget-2)
			}
			return memories, nil, nil, nil
		}
	}
}

func truncateToBudget(text string, budget int) string {
	if budget <= 0 {
		return ""
	}

	words := strings.Fields(text)
	if len(words) <= budget {
		return text
	}

	return strings.Join(words[:budget], " ")
}
