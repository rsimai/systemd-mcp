package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suse/systemd-mcp/internal/pkg/journal"
	"github.com/suse/systemd-mcp/internal/pkg/systemd"
)

var httpAddr = flag.String("http", "", "if set, use streamable HTTP at this address, instead of stdin/stdout")

func main() {
	flag.Parse()
	server := mcp.NewServer("Systemd connection", "0.0.1", nil)
	systemConn, err := systemd.NewSystem(context.Background())
	if err != nil {
		slog.Warn("couldn't add systemd tools", slog.Any("error", err))
	} else {
		// add systend tool handler
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
				mcp.Property("debug", mcp.Description("The verbose flag should only used for debugging and only if without the verbose flag too less information was provided.")),
			)),
		)
	}
	descriptionJournal := "Get the last log entries for the given service or unit."
	if os.Geteuid() != 0 {
		descriptionJournal += "Please note that this tool is not running as root, so system ressources may not been listed correctly."
	}
	log, err := journal.NewLog()
	if err != nil {
		slog.Warn("couldn't open log, not adding journal tool", slog.Any("error", err))
	} else {
		server.AddTools(mcp.NewServerTool("list_log", descriptionJournal,
			log.ListLog,
			mcp.Input(
				mcp.Property("count", mcp.Description("Number of log lines to output")),
				mcp.Property("unit", mcp.Description("Exact name of the service/unit from which to get the logs. Without log entries of all units are returned.")),
			)),
		)
	}

	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		slog.Info("MCP handler listening at", slog.String("address", *httpAddr))
		http.ListenAndServe(*httpAddr, handler)
	} else {
		t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
		if err := server.Run(context.Background(), t); err != nil {
			slog.Error("Server failed", slog.Any("error", err))
		}
	}
	systemConn.Close()
}
