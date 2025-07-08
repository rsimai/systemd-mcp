package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suse/systemd-mcp/internal/pkg/systemd"
)

var httpAddr = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")

func main() {
	flag.Parse()
	server := mcp.NewServer("Systemd connection", "0.0.1", nil)

	userConn, err := systemd.NewUser(context.Background())
	if err != nil {
		panic(err.Error())
	}
	systemConn, err := systemd.NewSystem(context.Background())
	// add tool handler
	server.AddTools(mcp.NewServerTool("list_systemd_host_units_by_state",
		fmt.Sprintf("List the requested systemd units and services on the host with the given state. As result the unit name, descrition and name are listed as json. Valid states are: %v", systemd.ValidStates()),
		systemConn.ListUnitHandlerState,
		mcp.Input(
			mcp.Property("states", mcp.Description("List of the states. The keyword 'all' can be used to get all available units on the system.")),
			mcp.Property("verbose", mcp.Description("The verbose flag should only used for debugging and only if the without verbosed too less information was provided.")),
		)),
	)
	server.AddTools(mcp.NewServerTool("list_systemd_units_by_name",
		"List the requested systemd unit. The output is a json formated with all available non empty fields.",
		systemConn.ListUnitHandlerNameState,
		mcp.Input(
			mcp.Property("names", mcp.Description("List units with the given by it's exact name. Regular expressions should be used. The request foo* expands to foo.service.")),
			mcp.Property("verbose", mcp.Description("The verbose flag should only used for debugging and only if without the verbose flag too less information was provided.")),
		)),
	)
	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP handler listening at %s", *httpAddr)
		http.ListenAndServe(*httpAddr, handler)
	} else {
		t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
		if err := server.Run(context.Background(), t); err != nil {
			log.Printf("Server failed: %v", err)
		}
	}
	userConn.Close()
}
