package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/phmotad/firememory/internal/defaultbrain"
	"github.com/phmotad/firememory/internal/storage"
	fqui "github.com/phmotad/firememory/internal/ui"
)

const defaultUIPort = 8765

func runUI(args []string, stdout, stderr io.Writer) error {
	brainPath, port, err := parseUIArgs(args)
	if err != nil {
		return err
	}

	if brainPath == "" {
		brainPath, err = defaultbrain.EnsureExists()
		if err != nil {
			return fmt.Errorf("ui: resolve brain path: %w", err)
		}
	}

	store, err := storage.OpenBboltStore(brainPath)
	if err != nil {
		return fmt.Errorf("ui: open brain %s: %w", brainPath, err)
	}
	defer store.Close()

	srv := fqui.New(store, port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Fprintf(stdout, "FireMemory UI — %s\n", url)
	fmt.Fprintf(stdout, "Brain: %s\n", brainPath)
	fmt.Fprintf(stderr, "Press Ctrl+C to stop.\n")

	return srv.Start(ctx)
}

func parseUIArgs(args []string) (brainPath string, port int, err error) {
	port = defaultUIPort
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 >= len(args) {
				return "", 0, fmt.Errorf("--port requires a value")
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n <= 0 {
				return "", 0, fmt.Errorf("invalid port %q", args[i])
			}
			port = n
		default:
			if brainPath != "" {
				return "", 0, fmt.Errorf("unexpected argument %q", args[i])
			}
			brainPath = args[i]
		}
	}
	return brainPath, port, nil
}
