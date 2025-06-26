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

func ValidStates() []string {
	return []string{"active", "dead", "inactive", "loaded", "mounted", "not-found", "plugged", "running", "all"}
}

// create a resource desription for getting the systemd states
func UnitRessource(state string) mcp.Resource {
	return mcp.NewResource(fmt.Sprintf("systemd://units/state/%s", state),
		fmt.Sprintf("systemd units and services on the host with the state %s", state),
		mcp.WithMIMEType("application/json"),
	)
}

// create a handler for to get the given state, some extra handing for
// the 'all' state, which is not implemted by the API
func (conn *Connection) CreateResHandler(state string) func(context.Context, mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context,
		request mcp.ReadResourceRequest,
	) (resources []mcp.ResourceContents, err error) {
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
}

func (conn *Connection) UnitResourceListState(ctx context.Context,
	request mcp.ReadResourceRequest,
) (resources []mcp.ResourceContents, err error) {
	uriSplit := strings.Split(request.Params.URI, "/")
	if len(uriSplit) == 0 {
		return nil, fmt.Errorf("malformed URI")
	}
	state := uriSplit[len(uriSplit)-1]
	if !slices.Contains(ValidStates(), state) {
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
			if !slices.Contains(ValidStates(), s) {
				return mcp.NewToolResultError(fmt.Sprintf("requsted state %s is not a valid state", s)), nil
			}
		}
	}
	var units []dbus.UnitStatus
	// route can't be taken as it confuses small modells
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
	type LightUnit struct {
		Name        string `json:"name"`
		State       string `json:"state"`
		Description string `json:"description"`
	}

	lightUnits := []LightUnit{}
	for _, u := range units {
		lightUnits = append(lightUnits, LightUnit{
			Name:        u.Name,
			State:       u.ActiveState,
			Description: u.Description,
		})
	}
	jsonByte, err := json.Marshal(&lightUnits)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), err
	}
	return mcp.NewToolResultText(string(jsonByte)), nil
}

// helper function to get the valid states
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
