package systemd

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/mark3labs/mcp-go/mcp"
)

type ListUnitArgs struct {
	Names []string `json:"names"`
}

func validStates() []string {
	return []string{"active", "dead", "inactive", "loaded", "mounted", "not-found", "plugged", "running", "all"}
}

func UnitTool() mcp.Tool {
	return mcp.NewTool("list_units",
		mcp.WithDescription("List the requested systemd units on the systemd."),
		mcp.WithArray("states",
			mcp.Description("List units with the given states. Defaults to running units"),
			mcp.Enum(validStates()...),
		),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
}
func UnitRessource() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate("units://units/state/{state}",
		fmt.Sprintf("systemd units and services on the host, valid states are: (%s)", strings.Join(validStates(), ",")),
		mcp.WithTemplateDescription("list all the units with the requested state"),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

func (conn *Connection) UnitResourceListState(ctx context.Context,
	request mcp.ReadResourceRequest,
) (resources []mcp.ResourceContents, err error) {
	uriSplit := strings.Split(request.Params.URI, "/")
	if len(uriSplit) == 0 {
		return nil, fmt.Errorf("malformed URI")
	}
	state := uriSplit[len(uriSplit)-1]
	if !slices.Contains(validStates(), state) {
		return nil, fmt.Errorf("invalid unit state requested: %s", state)

	}
	var units []dbus.UnitStatus
	if strings.EqualFold(state, "all") {
		units, err = conn.dbus.ListUnitsContext(ctx)
		if err != nil {
			return resources, err
		}
	} else {
		units, err = conn.dbus.ListUnitsFilteredContext(ctx, []string{state})
		if err != nil {
			return nil, err
		}
	}
	jsonByte, err := json.Marshal(&units)
	if err != nil {
		return nil, err
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonByte),
		},
	}, nil
}

func (conn *Connection) ListUnitHandler(ctx context.Context, request mcp.CallToolRequest, args ListUnitArgs) (*mcp.CallToolResult, error) {
	var err error
	reqStates := request.GetStringSlice("states", []string{""})
	if len(reqStates) == 0 {
		reqStates = []string{"running"}
	} else {
		for _, s := range reqStates {
			if !slices.Contains(validStates(), s) {
				return mcp.NewToolResultError(fmt.Sprintf("requsted state %s is not a valid state", s)), nil
			}
		}
	}
	var units []dbus.UnitStatus
	if slices.Contains(reqStates, "all") {
		units, err = conn.dbus.ListUnitsContext(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	} else {
		units, err = conn.dbus.ListUnitsFilteredContext(ctx, reqStates)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	jsonByte, err := json.Marshal(&units)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), err
	}
	return mcp.NewToolResultText(string(jsonByte)), nil
}

func (conn *Connection) ListStatesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var err error
	units, err := conn.dbus.ListUnitsContext(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	states := make(map[string]bool)
	for _, u := range units {
		if _, ok := states[u.ActiveState]; !ok {
			states[u.ActiveState] = true
		}
		if _, ok := states[u.LoadState]; !ok {
			states[u.LoadState] = true
		}
		if _, ok := states[u.SubState]; !ok {
			states[u.SubState] = true
		}
	}
	stateSlc := []string{}
	for key := range states {
		stateSlc = append(stateSlc, key)
	}
	jsonBytes, err := json.Marshal(stateSlc)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), err
	}
	return mcp.NewToolResultText(fmt.Sprintf(`"valid_states": %s`, string(jsonBytes))), nil
}
