# Model Context Proctocl (MCP) for systemd

The server directly connects to systemd via it's C API and so doesn't need systemctl to run.

# Usage

Compile direclty with
```
  go build systemd-mcp.go
```
or
```
  make build
```

# Functionality

Following tools are provided:
* `list_systemd_units_by_state` which list the unit in the given state, also all states can be listed
* `list_systemd_units_by_name` which list the unit given by their pattern
* `restart_reload_unit` which restarts or reloads a unit
* `start_unit` start a unit
* `stop_unit` stops a unit
* `check_restart_reload` check the state of reload or restart
* `enable_or_disable_unit` what enables or disables a unit
* `list_unit_files` which lists the unit files known to systemd
* `list_log` which has access to the system log, with various filters

# Testing

You can test the functions with [mcptools](https://github.com/f/mcptools), with e.g.
```
  mcptools shell go run systemd-mcp.go
```


