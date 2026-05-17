package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/phmotad/firememory/internal/daemon"
	"github.com/phmotad/firememory/internal/defaultbrain"
	"github.com/phmotad/firememory/internal/engine"
	fqonnx "github.com/phmotad/firememory/internal/firequery/onnx"
	"github.com/phmotad/firememory/internal/firequeryapp"
	"github.com/phmotad/firememory/internal/modelcache"
)

func runDaemon(args []string, stderr io.Writer, lookupEnv func(string) string) error {
	brainPath, err := resolveDaemonBrainPath(args, lookupEnv)
	if err != nil {
		return fmt.Errorf("daemon: resolve brain path: %w", err)
	}

	port := daemonPort(lookupEnv)

	modelsDir := fqonnx.DefaultModelsDir()
	if v := lookupEnv("FIREMEMORY_MODELS_DIR"); v != "" {
		modelsDir = v
	}

	// Ensure models are present before opening the engine.
	if err := modelcache.EnsureAll(context.Background(), modelsDir, stderr); err != nil {
		return fmt.Errorf("daemon: model setup: %w", err)
	}

	// Open the engine — this is the sole bbolt connection for this brain.
	eng, err := engine.Open(engine.Options{Path: brainPath})
	if err != nil {
		return fmt.Errorf("daemon: open brain %s: %w", brainPath, err)
	}

	// Build the ML pipeline wired to the persistent engine.
	handleFn, err := firequeryapp.BuildDaemonHandler(lookupEnv, eng)
	if err != nil {
		eng.Close()
		return fmt.Errorf("daemon: build handler: %w", err)
	}

	shutdownFn := func() {
		if err := eng.Close(); err != nil {
			fmt.Fprintf(stderr, "daemon: close engine: %v\n", err)
		}
	}

	srv := daemon.New(daemon.Config{
		HandleFn:   handleFn,
		ErrLog:     stderr,
		Port:       port,
		ShutdownFn: shutdownFn,
	})

	return srv.Start()
}

// resolveDaemonBrainPath resolves the brain path for the daemon from, in order:
//  1. --brain <path> flag in args
//  2. FIREMEMORY_DEFAULT_BRAIN environment variable
//  3. The default brain path (~/.firememory/default.fbrain), auto-created if absent
func resolveDaemonBrainPath(args []string, lookupEnv func(string) string) (string, error) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--brain" {
			return args[i+1], nil
		}
	}
	if v := lookupEnv("FIREMEMORY_DEFAULT_BRAIN"); v != "" {
		return v, nil
	}
	return defaultbrain.EnsureExists()
}

// daemonPort returns the TCP port the daemon should listen on.
// Reads FIREMEMORY_DAEMON_PORT; falls back to daemon.DefaultPort.
func daemonPort(lookupEnv func(string) string) int {
	if v := lookupEnv("FIREMEMORY_DAEMON_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return daemon.DefaultPort
}

// isDaemonAvailable returns true if a daemon is already listening on port.
func isDaemonAvailable(ctx context.Context, port int) bool {
	c := daemon.NewClient(port)
	return c.Ping(ctx) == nil
}

// startDaemonProcess forks a background daemon process.
// The child inherits all environment variables so it picks up
// FIREMEMORY_DEFAULT_BRAIN and FIREMEMORY_DAEMON_PORT automatically.
func startDaemonProcess() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	proc, err := os.StartProcess(exe, []string{exe, "daemon"}, &os.ProcAttr{
		Env:   os.Environ(),
		Files: []*os.File{nil, nil, os.Stderr},
	})
	if err != nil {
		return fmt.Errorf("start daemon process: %w", err)
	}
	// Detach — we don't wait for it.
	_ = proc.Release()
	return nil
}
