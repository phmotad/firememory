package main

import (
	"context"
	"fmt"
	"os"

	fqmcp "github.com/phmotad/firememory/internal/firequery/mcp"
	"github.com/phmotad/firememory/internal/firequeryapp"
)

func main() {
	service, err := firequeryapp.BuildService(func(key string) string {
		return os.Getenv(key)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	server := fqmcp.NewStdioServer(service.MCP(), "firequery", "0.1.0")
	if err := server.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
