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
				mcp.Property("states", mcp.Description("List of the states. The keyword 'all' can be used to get all available units on the system."), mcp.Enum("all")),
				mcp.Property("verbose", mcp.Description("The verbose flag should only used for debugging and only if the without verbosed too less information was provided.")),
			)),
		)
		server.AddTools(mcp.NewServerTool("list_systemd_units_by_name",
			"List the requested systemd unit. The output is a json formated with all available non empty fields.",
			systemConn.ListUnitHandlerNameState,
			mcp.Input(
				mcp.Property("names", mcp.Description("List units with the given by it's exact name. Regular expressions should be used. The request foo* expands to foo.service.")),
				mcp.Property("debug", mcp.Description("The debug flag should only used for debugging and only if without the verbose flag too less information was provided.")),
			)),
		)
		server.AddTools(mcp.NewServerTool("restart_reload_unit",
			"Reload or restart a unit or service.",
			systemConn.RestartReloadUnit,
			mcp.Input(
				mcp.Property("name", mcp.Description("Exact name of unit to restart"), mcp.Required(true)),
				mcp.Property("timeout", mcp.Description("Time to wait for the restart or reload to finish. After the timeout the function will return and restart and reload will run in the background and the result can be retreived with a separate function.")),
				mcp.Property("forcerestart", mcp.Description("Enforce a restart instead of a reload.")),
				mcp.Property("modes", mcp.Description("mode of the operation. 'replace' should be used per default and replace allready queued jobs. With 'fail' the operation will fail if other operations are in progress."), mcp.Enum("all")),
			)),
		)
		server.AddTools(mcp.NewServerTool("stop_unit",
			"Stop a unit or service.",
			systemConn.StopUnit,
			mcp.Input(
				mcp.Property("name", mcp.Description("Exact name of unit to stop"), mcp.Required(true)),
				mcp.Property("timeout", mcp.Description("Time to wait for the stop to finish. After the timeout the function will return and stop run in the background and the result can be retreived with a separate function.")),
				mcp.Property("kill", mcp.Description("Kill the unit instead of shutting down cleanly. Use this option only if the unit doesn't shut down, even after waiting.")),
				mcp.Property("modes", mcp.Description("mode of the operation. 'replace' should be used per default and replace allready queued jobs. With 'fail' the operation will fail if other operations are in progress."), mcp.Enum("all")),
			)),
		)
		server.AddTools(mcp.NewServerTool("check_restart_reload",
			"Check the reload or restart status of a unit. Can only be called if the restart or reload job had a timeout.",
			systemConn.CheckForRestartReloadRunning,
			mcp.Input(
				mcp.Property("timeout", mcp.Description("Time to wait for the restart or reload to finish. After the timeout the function will return and restart and reload will run in the background and the result can be retreived with a separate function.")),
			)),
		)
		server.AddTools(mcp.NewServerTool("enable_or_disable_unit",
			"Enable an unit or service for the next startup of the system.",
			systemConn.EnableDisableUnit,
			mcp.Input(mcp.Property("file", mcp.Description("Name of the service or unit if the unit is in the standard location. Takes the absolute path if the unit or service is not placed under '/etc/' or '/usr/lib/systemd'. Does not take wildcards. For the service foo, this would be 'foo.service' if foo is installed by a package."), mcp.Required(true))),
			mcp.Input(mcp.Property("disable"), mcp.Description("Set to true to disable the unit instead of enable.")),
		),
		)
		server.AddTools(mcp.NewServerTool("list_unit_files",
			"Returns a list of all the unit files known to systemd. This tool can be used to determine the correct names for all the other correct unit/service names for the other calls.",
			systemConn.ListUnitFiles,
			mcp.Input(mcp.Property("types", mcp.Description("List of the type which should be returned."), mcp.Required(false))),
		),
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
			log.ListLogTimeout,
			mcp.Input(
				mcp.Property("count", mcp.Description("Number of log lines to output"), mcp.Required(true)),
				mcp.Property("unit", mcp.Description("Exact name of the service/unit from which to get the logs. Without an unit name the entries of all units are returned. This parameter is optional."), mcp.Required(false)),
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
