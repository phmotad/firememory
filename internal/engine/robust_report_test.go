package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/phmotad/firememory/internal/embedder"
)

func TestRobustMemoryFlowGeneratesMarkdownReport(t *testing.T) {
	reportPath := filepath.Clean(filepath.Join("..", "..", "docs", "reports", "robust-memory-flow-report.md"))
	var report strings.Builder
	failures := make([]string, 0)

	defer func() {
		if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
			t.Fatalf("create reports dir: %v", err)
		}
		if err := os.WriteFile(reportPath, []byte(report.String()), 0o644); err != nil {
			t.Fatalf("write report: %v", err)
		}
	}()

	writeReportHeader(&report, "Robust Memory Flow Report")
	fmt.Fprintf(&report, "Generated at: `%s`\n\n", time.Now().UTC().Format(time.RFC3339))

	path := filepath.Join(t.TempDir(), "robust-agent.fbrain")
	custom := &recordingEmbedder{
		dimension: 6,
		vectors: map[string]embedder.Vector{
			"Cliente Joao usa Firebird 2.5 no servidor fiscal principal": {1.0, 0.4, 0, 0, 0, 0},
			"Joao reportou erro fiscal na NF-e apos atualizacao 3.2":     {0.4, 1.0, 0, 0, 0, 0},
			" Joao reportou erro fiscal na NF-e apos atualizacao 3.2 ":   {0.4, 1.0, 0, 0, 0, 0},
			"Maria opera PostgreSQL 15 no ambiente de analytics":         {0, 0, 1.0, 0, 0, 0},
			"Backup noturno roda as 02:00 e salva em storage local":      {0, 0, 0, 1.0, 0, 0},
			"Contrato ACME vence em dezembro e requer renovacao anual":   {0, 0, 0, 0, 1.0, 0},
			"cliente joao firebird":                                      {1.0, 0, 0, 0, 0, 0},
			"problema fiscal nfe apos atualizacao":                       {0, 1.0, 0, 0, 0, 0},
			"maria postgresql analytics":                                 {0, 0, 1.0, 0, 0, 0},
			"responder Joao sobre erro fiscal apos atualizacao":          {1.0, 1.0, 0, 0, 0, 0},
		},
	}

	engine := openTestEngine(t, path, custom, 6)

	memories := []string{
		"Cliente Joao usa Firebird 2.5 no servidor fiscal principal",
		"Joao reportou erro fiscal na NF-e apos atualizacao 3.2",
		"Maria opera PostgreSQL 15 no ambiente de analytics",
		"Backup noturno roda as 02:00 e salva em storage local",
		"Contrato ACME vence em dezembro e requer renovacao anual",
		" Joao reportou erro fiscal na NF-e apos atualizacao 3.2 ",
	}

	fmt.Fprintf(&report, "## Stored Memories\n\n")
	fmt.Fprintf(&report, "| # | Content | Action | Memory ID |\n")
	fmt.Fprintf(&report, "|---|---------|--------|-----------|\n")

	var rememberedIDs []string
	for index, content := range memories {
		result, err := engine.Remember(RememberInput{
			BrainPath: engine.Path(),
			Content:   content,
		})
		if err != nil {
			fmt.Fprintf(&report, "\nFailure during remember step %d: `%v`\n", index+1, err)
			_ = engine.Close()
			t.Fatalf("remember %d: %v", index+1, err)
		}
		rememberedIDs = append(rememberedIDs, result.Memory.ID)
		fmt.Fprintf(&report, "| %d | %s | `%s` | `%s` |\n", index+1, escapeTable(content), result.DedupAction, result.Memory.ID)
	}

	syncResult, err := engine.Sync(SyncInput{BrainPath: engine.Path()})
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during sync: `%v`\n", err)
		_ = engine.Close()
		t.Fatalf("sync: %v", err)
	}

	snapshotBeforeClose, err := engine.Store().Snapshot()
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during snapshot: `%v`\n", err)
		_ = engine.Close()
		t.Fatalf("snapshot: %v", err)
	}

	if err := engine.Close(); err != nil {
		fmt.Fprintf(&report, "\nFailure while closing engine before reopen: `%v`\n", err)
		t.Fatalf("close before reopen: %v", err)
	}

	reopened, err := Open(Options{Path: path, Embedder: custom})
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during reopen: `%v`\n", err)
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	recallJoao, err := reopened.Recall(RecallInput{
		BrainPath: reopened.Path(),
		Query:     "cliente joao firebird",
		TopK:      3,
	})
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during recallJoao: `%v`\n", err)
		t.Fatalf("recallJoao: %v", err)
	}

	recallFiscal, err := reopened.Recall(RecallInput{
		BrainPath:    reopened.Path(),
		Query:        "problema fiscal nfe apos atualizacao",
		TopK:         3,
		IncludeTrace: true,
	})
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during recallFiscal: `%v`\n", err)
		t.Fatalf("recallFiscal: %v", err)
	}

	recallMaria, err := reopened.Recall(RecallInput{
		BrainPath: reopened.Path(),
		Query:     "maria postgresql analytics",
		TopK:      3,
	})
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during recallMaria: `%v`\n", err)
		t.Fatalf("recallMaria: %v", err)
	}

	contextResult, err := reopened.Context(ContextInput{
		BrainPath:    reopened.Path(),
		Query:        "responder Joao sobre erro fiscal apos atualizacao",
		TopK:         3,
		BudgetTokens: 300,
		IncludeGraph: true,
		IncludeTrace: true,
	})
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during context: `%v`\n", err)
		t.Fatalf("context: %v", err)
	}

	explainResult, err := reopened.Explain(ExplainInput{
		BrainPath: reopened.Path(),
		Operation: "recall",
		MemoryID:  firstHitID(recallFiscal),
		Trace:     recallFiscal.Trace,
	})
	if err != nil {
		fmt.Fprintf(&report, "\nFailure during explain: `%v`\n", err)
		t.Fatalf("explain: %v", err)
	}

	assertReport := func(name string, ok bool, detail string) {
		status := "PASS"
		if !ok {
			status = "FAIL"
			failures = append(failures, fmt.Sprintf("%s: %s", name, detail))
		}
		fmt.Fprintf(&report, "| %s | %s | %s |\n", name, status, escapeTable(detail))
	}

	fmt.Fprintf(&report, "\n## Assertions\n\n")
	fmt.Fprintf(&report, "| Check | Status | Detail |\n")
	fmt.Fprintf(&report, "|-------|--------|--------|\n")

	assertReport("dedup reinforcement", len(snapshotBeforeClose.Namespaces["memories"]) == 5, fmt.Sprintf("memory_count=%d expected=5", len(snapshotBeforeClose.Namespaces["memories"])))
	assertReport("sync processed entries", syncResult.Processed >= 5, fmt.Sprintf("processed=%d expected>=5", syncResult.Processed))
	assertReport("entities extracted", len(snapshotBeforeClose.Namespaces["entities"]) > 0, fmt.Sprintf("entity_count=%d expected>0", len(snapshotBeforeClose.Namespaces["entities"])))
	assertReport("facts extracted", len(snapshotBeforeClose.Namespaces["facts"]) > 0, fmt.Sprintf("fact_count=%d expected>0", len(snapshotBeforeClose.Namespaces["facts"])))
	assertReport("relations created", len(snapshotBeforeClose.Namespaces["relations"]) > 0, fmt.Sprintf("relation_count=%d expected>0", len(snapshotBeforeClose.Namespaces["relations"])))
	assertReport("recall joao finds firebird", containsHit(recallJoao, "Firebird 2.5"), summarizeHits(recallJoao))
	assertReport("recall fiscal finds error memory", containsHit(recallFiscal, "erro fiscal na NF-e"), summarizeHits(recallFiscal))
	assertReport("recall maria finds postgresql", containsHit(recallMaria, "PostgreSQL 15"), summarizeHits(recallMaria))
	assertReport("context includes Joao", strings.Contains(contextResult.ContextText, "Joao"), trimForReport(contextResult.ContextText, 180))
	assertReport("context includes Firebird", strings.Contains(contextResult.ContextText, "Firebird 2.5"), trimForReport(contextResult.ContextText, 180))
	assertReport("context includes fiscal issue", strings.Contains(strings.ToLower(contextResult.ContextText), "fiscal"), trimForReport(contextResult.ContextText, 180))
	assertReport("explain summary present", strings.Contains(explainResult.Summary, "Recall explanation"), explainResult.Summary)
	assertReport("reopen preserved data", len(recallFiscal.Hits) > 0 && firstHitID(recallFiscal) != "", fmt.Sprintf("top_hit=%s", firstHitID(recallFiscal)))

	fmt.Fprintf(&report, "\n## Recall Outputs\n\n")
	writeRecallSection(&report, "Joao Query", recallJoao)
	writeRecallSection(&report, "Fiscal Query", recallFiscal)
	writeRecallSection(&report, "Maria Query", recallMaria)

	fmt.Fprintf(&report, "\n## Context Output\n\n")
	fmt.Fprintf(&report, "Estimated tokens: `%d`\n\n", contextResult.EstimatedTokens)
	fmt.Fprintf(&report, "```text\n%s\n```\n", contextResult.ContextText)

	fmt.Fprintf(&report, "\n## Explain Output\n\n")
	fmt.Fprintf(&report, "Summary: %s\n\n", explainResult.Summary)
	fmt.Fprintf(&report, "Trace steps: `%d`\n\n", len(explainResult.Trace))

	fmt.Fprintf(&report, "## Snapshot Summary\n\n")
	fmt.Fprintf(&report, "- memories: `%d`\n", len(snapshotBeforeClose.Namespaces["memories"]))
	fmt.Fprintf(&report, "- entities: `%d`\n", len(snapshotBeforeClose.Namespaces["entities"]))
	fmt.Fprintf(&report, "- facts: `%d`\n", len(snapshotBeforeClose.Namespaces["facts"]))
	fmt.Fprintf(&report, "- relations: `%d`\n", len(snapshotBeforeClose.Namespaces["relations"]))
	fmt.Fprintf(&report, "- traces: `%d`\n", len(snapshotBeforeClose.Namespaces["traces"]))
	fmt.Fprintf(&report, "- sync_queue: `%d`\n", len(snapshotBeforeClose.Namespaces["sync_queue"]))

	fmt.Fprintf(&report, "\n## Verdict\n\n")
	if len(failures) == 0 {
		fmt.Fprintf(&report, "All assertions passed.\n")
	} else {
		for _, failure := range failures {
			fmt.Fprintf(&report, "- %s\n", failure)
		}
	}

	if len(failures) > 0 {
		t.Fatalf("robust memory flow failures: %s", strings.Join(failures, "; "))
	}
}

func writeReportHeader(report *strings.Builder, title string) {
	fmt.Fprintf(report, "# %s\n\n", title)
}

func writeRecallSection(report *strings.Builder, title string, result RecallResult) {
	fmt.Fprintf(report, "### %s\n\n", title)
	if len(result.Hits) == 0 {
		fmt.Fprintf(report, "No hits.\n\n")
		return
	}
	fmt.Fprintf(report, "| Rank | Score | Memory ID | Content |\n")
	fmt.Fprintf(report, "|------|-------|-----------|---------|\n")
	for index, hit := range result.Hits {
		fmt.Fprintf(report, "| %d | %.3f | `%s` | %s |\n", index+1, hit.Score, hit.Memory.ID, escapeTable(hit.Memory.Content))
	}
	fmt.Fprintf(report, "\n")
}

func summarizeHits(result RecallResult) string {
	parts := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		parts = append(parts, hit.Memory.Content)
	}
	return strings.Join(parts, " || ")
}

func containsHit(result RecallResult, fragment string) bool {
	for _, hit := range result.Hits {
		if strings.Contains(hit.Memory.Content, fragment) {
			return true
		}
	}
	return false
}

func firstHitID(result RecallResult) string {
	if len(result.Hits) == 0 {
		return ""
	}
	return result.Hits[0].Memory.ID
}

func trimForReport(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func escapeTable(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "|", "\\|")
	return text
}
