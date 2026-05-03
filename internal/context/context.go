package context

import (
	"strings"

	"github.com/phmotad/firememory/internal/memory"
)

func EstimateTokens(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}

	return len(strings.Fields(text))
}

func BuildText(memories []memory.Memory, entities []memory.Entity, facts []memory.Fact, relations []memory.Relation) string {
	sections := make([]string, 0, 4)

	if len(memories) > 0 {
		lines := make([]string, 0, len(memories)+1)
		lines = append(lines, "Memories:")
		for _, mem := range memories {
			lines = append(lines, "- "+mem.Content)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if len(entities) > 0 {
		lines := make([]string, 0, len(entities)+1)
		lines = append(lines, "Entities:")
		for _, entity := range entities {
			lines = append(lines, "- "+entity.Name+" ("+entity.Type+")")
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if len(facts) > 0 {
		lines := make([]string, 0, len(facts)+1)
		lines = append(lines, "Facts:")
		for _, fact := range facts {
			lines = append(lines, "- "+fact.Subject+" | "+fact.Predicate+" | "+fact.Object)
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if len(relations) > 0 {
		lines := make([]string, 0, len(relations)+1)
		lines = append(lines, "Relations:")
		for _, relation := range relations {
			lines = append(lines, "- "+relation.FromID+" -> "+relation.ToID+" ("+string(relation.Type)+")")
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	return strings.Join(sections, "\n\n")
}
