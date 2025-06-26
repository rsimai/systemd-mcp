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

	userConn, err := systemd.NewUser(context.Background())
	if err != nil {
		panic(err.Error())
	}
	systemConn, err := systemd.NewSystem(context.Background())
	// add tool handler
	s.AddTool(mcp.NewTool("list_systemd_host_units",
		mcp.WithDescription("List the requested systemd units and services running on the host."),
		mcp.WithArray("states",
			mcp.Description("List units with the given states."),
			mcp.Enum(systemd.ValidStates()...),
		),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	),
		mcp.NewTypedToolHandler(systemConn.ListUnitHandler))
	s.AddTool(mcp.NewTool("list_systemd_user_units",
		mcp.WithDescription("List the requested systemd units and services of the user."),
		mcp.WithArray("states",
			mcp.Description("List units with the given states."),
			mcp.Enum(systemd.ValidStates()...),
		),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	),
		mcp.NewTypedToolHandler(systemConn.ListUnitHandler))
	// add ressource handler
	// s.AddResourceTemplate(systemd.UnitRessource(), conn.UnitResourceListState)
	for _, state := range systemd.ValidStates() {
		s.AddResource(
			systemd.UnitRessource(state),
			systemConn.CreateResHandler(state),
		)
	}

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
	userConn.Close()
}
