package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/phmotad/firememory/internal/firequery/doctor"
	fqonnx "github.com/phmotad/firememory/internal/firequery/onnx"
	fqruntime "github.com/phmotad/firememory/internal/firequery/runtime"
	"github.com/phmotad/firememory/internal/firequeryapp"
	"github.com/phmotad/firememory/internal/initcfg"
	"github.com/phmotad/firememory/internal/modelcache"
	"github.com/phmotad/firememory/internal/util"
	"github.com/phmotad/firememory/internal/version"
)

func main() {
	args, jsonOutput := util.ExtractJSONFlag(os.Args[1:])
	if err := run(args, os.Stdout, os.Stderr, func(key string) string {
		return os.Getenv(key)
	}, jsonOutput); err != nil {
		writeError(os.Stderr, err, jsonOutput)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, lookupEnv func(string) string, jsonOutput bool) error {
	if len(args) == 0 {
		writeUsage(stderr)
		return fmt.Errorf("command is required")
	}

	switch args[0] {
	case "devices":
		manager := firequeryapp.BuildRuntimeManager(lookupEnv)
		return runDevices(stdout, manager, jsonOutput)
	case "doctor":
		manager := firequeryapp.BuildRuntimeManager(lookupEnv)
		return runDoctor(stdout, doctor.RuntimeReporter{Runtime: manager}, jsonOutput)
	case "mcp":
		return runMCP(stdout, stderr, lookupEnv)
	case "models":
		return runModels(args[1:], stdout, stderr, lookupEnv, jsonOutput)
	case "init-mcp":
		return runInitMCP(args[1:], stdout, jsonOutput)
	case "version":
		if jsonOutput {
			return util.WriteJSON(stdout, map[string]any{"ok": true, "version": version.Version})
		}
		fmt.Fprintln(stdout, version.Version)
		return nil
	default:
		writeUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runMCP(stdout, stderr io.Writer, lookupEnv func(string) string) error {
	modelsDir := fqonnx.DefaultModelsDir()
	if v := lookupEnv("FIREMEMORY_MODELS_DIR"); v != "" {
		modelsDir = v
	}

	// Auto-download models on first run. Progress goes to stderr (stdout is MCP JSON-RPC).
	if err := modelcache.EnsureAll(context.Background(), modelsDir, stderr); err != nil {
		return fmt.Errorf("model setup: %w", err)
	}

	service, err := firequeryapp.BuildService(lookupEnv)
	if err != nil {
		return err
	}
	transport := firequeryappTransport(service)
	return transport.Serve(context.Background(), os.Stdin, stdout)
}

func runModels(args []string, stdout, stderr io.Writer, lookupEnv func(string) string, jsonOutput bool) error {
	modelsDir := fqonnx.DefaultModelsDir()
	if v := lookupEnv("FIREMEMORY_MODELS_DIR"); v != "" {
		modelsDir = v
	}

	sub := "list"
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "list":
		return runModelsList(stdout, modelsDir, jsonOutput)
	case "pull":
		force := len(args) > 1 && args[1] == "--force"
		return runModelsPull(stdout, stderr, modelsDir, force)
	case "gc":
		return runModelsGC(stdout, modelsDir)
	default:
		fmt.Fprintln(stderr, "usage: fquery models <list|pull|gc>")
		return fmt.Errorf("unknown models subcommand %q", sub)
	}
}

func runModelsList(stdout io.Writer, cacheDir string, jsonOutput bool) error {
	statuses, err := modelcache.Status(cacheDir)
	if err != nil {
		return err
	}
	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":       true,
			"command":  "models list",
			"cache":    cacheDir,
			"models":   statuses,
		})
	}
	fmt.Fprintf(stdout, "cache: %s\n\n", cacheDir)
	for _, s := range statuses {
		status := "missing"
		if s.Present && s.Verified {
			status = "ok"
		} else if s.Present {
			status = "unverified"
		}
		extra := ""
		if s.Error != "" {
			extra = " (" + s.Error + ")"
		}
		fmt.Fprintf(stdout, "  %-32s [%s]%s\n", s.ID, status, extra)
	}
	return nil
}

func runModelsPull(stdout, stderr io.Writer, cacheDir string, force bool) error {
	if force {
		return modelcache.PullAll(context.Background(), cacheDir, stdout)
	}
	return modelcache.EnsureAll(context.Background(), cacheDir, stdout)
}

func runModelsGC(stdout io.Writer, cacheDir string) error {
	if err := modelcache.Remove(cacheDir); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "models removed from %s\n", cacheDir)
	return nil
}

func runDevices(stdout io.Writer, manager fqruntime.Manager, jsonOutput bool) error {
	devices, err := manager.Devices(context.Background())
	if err != nil {
		return err
	}
	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":      true,
			"command": "devices",
			"devices": devices,
		})
	}
	for _, device := range devices {
		fmt.Fprintf(stdout, "%s available=%t backend=%s\n", device.Name, device.Available, device.Supports())
	}
	return nil
}

func runDoctor(stdout io.Writer, reporter doctor.Reporter, jsonOutput bool) error {
	report, err := reporter.Run(context.Background())
	if err != nil {
		return err
	}
	if jsonOutput {
		return util.WriteJSON(stdout, report)
	}
	fmt.Fprintf(stdout, "ready: %t\n", report.Ready)
	for _, check := range report.Checks {
		fmt.Fprintf(stdout, "%s [%s] %s\n", check.Name, check.Status, check.Detail)
	}
	return nil
}

func runInitMCP(args []string, stdout io.Writer, jsonOutput bool) error {
	printOnly := false
	var clientName string
	var configOverride string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--print":
			printOnly = true
		case "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a path argument")
			}
			i++
			configOverride = args[i]
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown flag %q", args[i])
			}
			clientName = args[i]
		}
	}

	if clientName == "" {
		names := make([]string, 0)
		for _, c := range initcfg.Supported() {
			names = append(names, c.Name)
		}
		return fmt.Errorf("usage: fquery init-mcp <client> [--print] [--config <path>]\nclients: %s",
			strings.Join(names, ", "))
	}

	client, ok := initcfg.Find(clientName)
	if !ok {
		names := make([]string, 0)
		for _, c := range initcfg.Supported() {
			names = append(names, c.Name)
		}
		return fmt.Errorf("unknown client %q — supported: %s", clientName, strings.Join(names, ", "))
	}

	configPath := configOverride
	if configPath == "" {
		configPath = client.ConfigPath()
	}
	if configPath == "" {
		return fmt.Errorf("could not determine config path for %s", client.Name)
	}

	// Resolve fquery binary path (absolute, so the config stays valid from any cwd).
	fqueryExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve fquery path: %w", err)
	}

	entry := initcfg.MCPEntry{
		Command: fqueryExe,
		Args:    []string{"mcp"},
	}

	if printOnly {
		out, err := initcfg.Print(configPath, entry)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "# %s\n", configPath)
		fmt.Fprintln(stdout, string(out))
		return nil
	}

	result, err := initcfg.Patch(configPath, entry)
	if err != nil {
		return err
	}

	if jsonOutput {
		return util.WriteJSON(stdout, map[string]any{
			"ok":          true,
			"operation":   "init-mcp",
			"client":      client.Name,
			"config_path": result.ConfigPath,
			"created":     result.Created,
			"updated":     result.Updated,
		})
	}

	action := "added"
	if result.Updated {
		action = "updated"
	} else if result.Created {
		action = "created"
	}
	fmt.Fprintf(stdout, "%s: firequery entry %s in %s\n", client.Description, action, result.ConfigPath)
	fmt.Fprintln(stdout, "Restart your editor to pick up the change.")
	return nil
}

func writeUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: fquery <command>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  mcp                          start MCP server over stdio (auto-downloads models on first run)")
	fmt.Fprintln(w, "  init-mcp <client>            configure an AI coding client to use FireQuery")
	fmt.Fprintln(w, "    clients: claude-code, cursor, windsurf, zed")
	fmt.Fprintln(w, "    --print        dry-run: print the config that would be written")
	fmt.Fprintln(w, "    --config <p>   override config file path")
	fmt.Fprintln(w, "  models list                  show model download status")
	fmt.Fprintln(w, "  models pull                  download missing models")
	fmt.Fprintln(w, "  models pull --force           re-download all models")
	fmt.Fprintln(w, "  models gc                    remove all cached models")
	fmt.Fprintln(w, "  devices                      list available compute devices")
	fmt.Fprintln(w, "  doctor                       run diagnostics")
	fmt.Fprintln(w, "  version                      print version")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "env:")
	fmt.Fprintln(w, "  FIREMEMORY_MODELS_DIR              override model cache directory")
	fmt.Fprintln(w, "  FIREQUERY_REQUIRE_REAL_MODELS=1    fail if models not available")
	fmt.Fprintln(w, "  FIREQUERY_ENABLE_CUDA=1")
	fmt.Fprintln(w, "  FIREQUERY_ENABLE_DIRECTML=1")
	fmt.Fprintln(w, "  FIREQUERY_ENABLE_COREML=1")
	fmt.Fprintln(w, "  FIREQUERY_ENABLE_OPENVINO=1")
}

func containsLine(output, want string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, want) {
			return true
		}
	}
	return false
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
