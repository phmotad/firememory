package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/phmotad/firememory/internal/brainfile"
	"github.com/phmotad/firememory/internal/defaultbrain"
	"github.com/phmotad/firememory/internal/engine"
	"github.com/phmotad/firememory/internal/storage"
	"github.com/phmotad/firememory/internal/util"
	"github.com/phmotad/firememory/internal/version"
)

func main() {
	args, jsonOutput := util.ExtractJSONFlag(os.Args[1:])
	if err := run(args, os.Stdout, os.Stderr, jsonOutput); err != nil {
		writeError(os.Stderr, err, jsonOutput)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, jsonOutput bool) error {
	if len(args) == 0 {
		writeUsage(stderr)
		return fmt.Errorf("command is required")
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, jsonOutput)
	case "remember":
		return runRemember(args[1:], stdout, jsonOutput)
	case "recall":
		return runRecall(args[1:], stdout, jsonOutput)
	case "sync":
		return runSync(args[1:], stdout, jsonOutput)
	case "context":
		return runContext(args[1:], stdout, jsonOutput)
	case "inspect":
		return runInspect(args[1:], stdout, jsonOutput)
	case "snapshot":
		return runSnapshot(args[1:], stdout, jsonOutput)
	case "backup":
		return runBackup(args[1:], stdout, jsonOutput)
	case "restore":
		return runRestore(args[1:], stdout, jsonOutput)
	case "compact":
		return runCompact(args[1:], stdout, jsonOutput)
	case "version":
		return runVersion(stdout, jsonOutput)
	case "stats":
		return runStats(args[1:], stdout, jsonOutput)
	case "default":
		return runDefault(stdout, jsonOutput)
	default:
		writeUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runInit(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: fmem init <brainfile.fbrain>")
	}

	handle, err := brainfile.Create(args[0], brainfile.CreateOptions{})
	if err != nil {
		return err
	}
	defer handle.Close()

	manifest := handle.Manifest()
	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":              true,
			"operation":       "init",
			"brainfile":       handle.Path(),
			"name":            manifest.Name,
			"embedding_model": manifest.EmbeddingModel,
			"embedding_dim":   manifest.EmbeddingDim,
		})
	}
	fmt.Fprintf(stdout, "initialized brainfile: %s\n", handle.Path())
	fmt.Fprintf(stdout, "name: %s\n", manifest.Name)
	fmt.Fprintf(stdout, "embedding: %s (%d)\n", manifest.EmbeddingModel, manifest.EmbeddingDim)
	return nil
}

func runRemember(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fmem remember <brainfile.fbrain> <content>")
	}

	path := args[0]
	content := strings.Join(args[1:], " ")

	eng, err := engine.Open(engine.Options{Path: path})
	if err != nil {
		return err
	}
	defer eng.Close()

	result, err := eng.Remember(engine.RememberInput{
		BrainPath: path,
		Content:   content,
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":                   true,
			"operation":            "remember",
			"dedup_action":         result.DedupAction,
			"memory_id":            result.Memory.ID,
			"reinforced_memory_id": result.ReinforcedMemoryID,
			"status":               result.Memory.Status,
			"trace":                util.StructuredTrace("firememory.engine", result.Trace),
		})
	}

	fmt.Fprintf(stdout, "action: %s\n", result.DedupAction)
	fmt.Fprintf(stdout, "memory_id: %s\n", result.Memory.ID)
	if result.ReinforcedMemoryID != "" {
		fmt.Fprintf(stdout, "reinforced_memory_id: %s\n", result.ReinforcedMemoryID)
	}
	fmt.Fprintf(stdout, "status: %s\n", result.Memory.Status)
	return nil
}

func runRecall(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fmem recall <brainfile.fbrain> <query>")
	}

	path := args[0]
	query := strings.Join(args[1:], " ")

	eng, err := engine.Open(engine.Options{Path: path})
	if err != nil {
		return err
	}
	defer eng.Close()

	result, err := eng.Recall(engine.RecallInput{
		BrainPath: path,
		Query:     query,
		TopK:      5,
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		hits := make([]map[string]any, 0, len(result.Hits))
		for _, hit := range result.Hits {
			hits = append(hits, map[string]any{
				"memory_id": hit.Memory.ID,
				"content":   hit.Memory.Content,
				"score":     hit.Score,
				"reasons":   hit.Reasons,
			})
		}
		return util.WriteJSON(stdout, map[string]any{
			"ok":        true,
			"operation": "recall",
			"hits":      hits,
			"trace":     util.StructuredTrace("firememory.engine", result.Trace),
		})
	}

	if len(result.Hits) == 0 {
		fmt.Fprintln(stdout, "no results")
		return nil
	}

	for i, hit := range result.Hits {
		fmt.Fprintf(stdout, "%d. [%0.3f] %s\n", i+1, hit.Score, hit.Memory.Content)
	}
	return nil
}

func runSync(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: fmem sync <brainfile.fbrain>")
	}

	path := args[0]
	eng, err := engine.Open(engine.Options{Path: path})
	if err != nil {
		return err
	}
	defer eng.Close()

	result, err := eng.Sync(engine.SyncInput{BrainPath: path})
	if err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":         true,
			"operation":  "sync",
			"processed":  result.Processed,
			"synced_ids": result.SyncedIDs,
			"trace":      util.StructuredTrace("firememory.engine", result.Trace),
		})
	}

	fmt.Fprintf(stdout, "processed: %d\n", result.Processed)
	if len(result.SyncedIDs) > 0 {
		fmt.Fprintf(stdout, "synced_ids: %s\n", strings.Join(result.SyncedIDs, ", "))
	}
	return nil
}

func runContext(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fmem context <brainfile.fbrain> <query>")
	}

	path := args[0]
	query := strings.Join(args[1:], " ")

	eng, err := engine.Open(engine.Options{Path: path})
	if err != nil {
		return err
	}
	defer eng.Close()

	result, err := eng.Context(engine.ContextInput{
		BrainPath:    path,
		Query:        query,
		TopK:         5,
		BudgetTokens: 2000,
		IncludeGraph: true,
	})
	if err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":               true,
			"operation":        "context",
			"context":          result.ContextText,
			"estimated_tokens": result.EstimatedTokens,
			"trace":            util.StructuredTrace("firememory.engine", result.Trace),
		})
	}

	fmt.Fprintln(stdout, result.ContextText)
	fmt.Fprintf(stdout, "\nEstimated tokens: %d\n", result.EstimatedTokens)
	return nil
}

func runInspect(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: fmem inspect <brainfile.fbrain>")
	}

	info, err := brainfile.Inspect(args[0])
	if err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, info)
	}

	fmt.Fprintf(stdout, "brainfile: %s\n", info.Path)
	fmt.Fprintf(stdout, "name: %s\n", info.Manifest.Name)
	fmt.Fprintf(stdout, "format_version: %s\n", info.Manifest.FormatVersion)
	fmt.Fprintf(stdout, "embedding: %s (%d)\n", info.Manifest.EmbeddingModel, info.Manifest.EmbeddingDim)
	fmt.Fprintln(stdout, "namespaces:")
	for _, namespace := range info.Namespaces {
		fmt.Fprintf(stdout, "- %s: %d\n", namespace, info.NamespaceCounts[namespace])
	}
	return nil
}

func runSnapshot(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: fmem snapshot <brainfile.fbrain>")
	}

	if err := brainfile.ValidatePath(args[0]); err != nil {
		return err
	}

	store, err := storage.OpenBboltStore(args[0])
	if err != nil {
		return err
	}
	defer store.Close()

	snapshot, err := store.Snapshot()
	if err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, snapshot)
	}

	namespaces := make([]string, 0, len(snapshot.Namespaces))
	for namespace := range snapshot.Namespaces {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)

	fmt.Fprintf(stdout, "snapshot taken at: %s\n", snapshot.TakenAt.Format("2006-01-02T15:04:05Z"))
	for _, namespace := range namespaces {
		fmt.Fprintf(stdout, "- %s: %d\n", namespace, len(snapshot.Namespaces[namespace]))
	}
	return nil
}

func runCompact(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: fmem compact <brainfile.fbrain>")
	}

	if err := brainfile.ValidatePath(args[0]); err != nil {
		return err
	}

	store, err := storage.OpenBboltStore(args[0])
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.Compact(); err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":        true,
			"operation": "compact",
			"brainfile": args[0],
		})
	}

	fmt.Fprintln(stdout, "compact complete")
	return nil
}

func runBackup(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: fmem backup <brainfile.fbrain> <backup-path>")
	}

	if err := brainfile.Backup(args[0], args[1]); err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":        true,
			"operation": "backup",
			"brainfile": args[0],
			"backup":    args[1],
		})
	}

	fmt.Fprintf(stdout, "backup created: %s\n", args[1])
	return nil
}

func runRestore(args []string, stdout io.Writer, jsonOutput bool) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: fmem restore <backup-path> <brainfile.fbrain>")
	}

	if err := brainfile.Restore(args[0], args[1]); err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":        true,
			"operation": "restore",
			"backup":    args[0],
			"brainfile": args[1],
		})
	}

	fmt.Fprintf(stdout, "brainfile restored: %s\n", args[1])
	return nil
}

func runVersion(stdout io.Writer, jsonOutput bool) error {
	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":      true,
			"version": version.Version,
		})
	}
	fmt.Fprintln(stdout, version.Version)
	return nil
}

func runDefault(stdout io.Writer, jsonOutput bool) error {
	path, err := defaultbrain.EnsureExists()
	if err != nil {
		return err
	}
	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":   true,
			"path": path,
		})
	}
	fmt.Fprintln(stdout, path)
	return nil
}

func runStats(args []string, stdout io.Writer, jsonOutput bool) error {
	var path string
	if len(args) == 1 {
		path = args[0]
	} else {
		var err error
		path, err = defaultbrain.EnsureExists()
		if err != nil {
			return err
		}
	}

	info, err := brainfile.Inspect(path)
	if err != nil {
		return err
	}

	total := 0
	for _, n := range info.NamespaceCounts {
		total += n
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":         true,
			"brainfile":  info.Path,
			"memories":   total,
			"namespaces": info.NamespaceCounts,
		})
	}
	fmt.Fprintf(stdout, "brainfile: %s\n", info.Path)
	fmt.Fprintf(stdout, "memories:  %d\n", total)
	for _, ns := range info.Namespaces {
		fmt.Fprintf(stdout, "  %s: %d\n", ns, info.NamespaceCounts[ns])
	}
	return nil
}

func writeError(w io.Writer, err error, jsonOutput bool) {
	if jsonOutput {
		_ = util.WriteJSON(w, map[string]any{
			"ok":    false,
			"error": util.ErrorDiagnostic(err),
		})
		return
	}
	fmt.Fprintf(w, "error [%s]: %s\n", util.ErrorCode(err), err)
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: fmem <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  init <brainfile.fbrain>")
	fmt.Fprintln(w, "  remember <brainfile.fbrain> <content>")
	fmt.Fprintln(w, "  recall <brainfile.fbrain> <query>")
	fmt.Fprintln(w, "  sync <brainfile.fbrain>")
	fmt.Fprintln(w, "  context <brainfile.fbrain> <query>")
	fmt.Fprintln(w, "  inspect <brainfile.fbrain>")
	fmt.Fprintln(w, "  snapshot <brainfile.fbrain>")
	fmt.Fprintln(w, "  backup <brainfile.fbrain> <backup-path>")
	fmt.Fprintln(w, "  restore <backup-path> <brainfile.fbrain>")
	fmt.Fprintln(w, "  compact <brainfile.fbrain>")
	fmt.Fprintln(w, "  stats [<brainfile.fbrain>]   (defaults to ~/.firememory/default.fbrain)")
	fmt.Fprintln(w, "  default                      print default brainfile path (creates if missing)")
	fmt.Fprintln(w, "  version")
}
