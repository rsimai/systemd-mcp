package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/suse/systemd-mcp/internal/pkg/systemd"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Systemd connection",
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	conn, err := systemd.New(context.Background())
	if err != nil {
		panic(err.Error())
	}
	// add tool handler
	s.AddTool(systemd.UnitTool(), mcp.NewTypedToolHandler(conn.ListUnitHandler))
	// add ressource handler
	// s.AddResourceTemplate(systemd.UnitRessource(), conn.UnitResourceListState)
	for _, state := range systemd.ValidStates() {
		s.AddResource(
			systemd.UnitRessource(state),
			conn.CreateResHandler(state),
		)
	}

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
	conn.Close()
}
