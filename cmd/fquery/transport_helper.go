package main

import (
	"github.com/phmotad/firememory/internal/firequery"
	fqmcp "github.com/phmotad/firememory/internal/firequery/mcp"
)

func firequeryappTransport(service *firequery.Service) *fqmcp.StdioServer {
	return fqmcp.NewStdioServer(service.MCP(), "firequery", "0.1.0")
}
