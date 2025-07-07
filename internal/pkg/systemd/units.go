package systemd

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/suse/systemd-mcp/internal/pkg/util"
)

func ValidStates() []string {
	return []string{"active", "dead", "inactive", "loaded", "mounted", "not-found", "plugged", "running", "all"}
}

// create a resource desription for getting the systemd states
/*
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
*/

type ListUnitParams struct {
	States []string `json:"states"`
}

func (conn *Connection) ListUnitHandlerState(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListUnitParams]) (*mcp.CallToolResultFor[any], error) {
	var err error
	reqStates := params.Arguments.States
	if len(reqStates) == 0 {
		reqStates = []string{"running"}
	} else {
		for _, s := range reqStates {
			if !slices.Contains(ValidStates(), s) {
				return nil, fmt.Errorf("requsted state %s is not a valid state", s)
			}
		}
	}
	var units []dbus.UnitStatus
	// route can't be taken as it confuses small modells
	if slices.Contains(reqStates, "all") {
		units, err = conn.dbus.ListUnitsContext(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		units, err = conn.dbus.ListUnitsFilteredContext(ctx, reqStates)
		if err != nil {
			return nil, err
		}
	}
	type LightUnit struct {
		Name        string `json:"name"`
		State       string `json:"state"`
		Description string `json:"description"`
	}

	txtContenList := []mcp.Content{}
	for _, u := range units {
		lightUnit := LightUnit{
			Name:        u.Name,
			State:       u.ActiveState,
			Description: u.Description,
		}
		jsonByte, _ := json.Marshal(&lightUnit)
		txtContenList = append(txtContenList, &mcp.TextContent{
			Text: string(jsonByte),
		})

	}

	return &mcp.CallToolResultFor[any]{
		Content: txtContenList,
	}, nil
}

type ListUnitNameParams struct {
	Names []string `json:"names"`
}

/*
Handler to list the unit by name
*/
func (conn *Connection) ListUnitHandlerNameState(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListUnitNameParams]) (*mcp.CallToolResultFor[any], error) {
	var err error
	reqNames := params.Arguments.Names
	// reqStates := request.GetStringSlice("states", []string{""})
	var units []dbus.UnitStatus
	units, err = conn.dbus.ListUnitsByPatternsContext(ctx, []string{}, reqNames)
	if err != nil {
		return nil, err
	}
	// unitProps := make([]map[string]interface{}, 1, 1)
	txtContentList := []mcp.Content{}
	for _, u := range units {
		props, err := conn.dbus.GetAllPropertiesContext(ctx, u.Name)
		if err != nil {
			return nil, err
		}
		props = util.ClearMap(props)
		jsonByte, _ := json.Marshal(&props)

		txtContentList = append(txtContentList, &mcp.TextContent{
			Text: string(jsonByte),
		})

	}
	if len(txtContentList) == 0 {
		return nil, fmt.Errorf("found no units with name pattern: %v", reqNames)
	}
	return &mcp.CallToolResultFor[any]{
		Content: txtContentList,
	}, nil
}

// helper function to get the valid states
func (conn *Connection) ListStatesHandler(ctx context.Context) (lst []string, err error) {
	units, err := conn.dbus.ListUnitsContext(ctx)
	if err != nil {
		return
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
	for key := range states {
		lst = append(lst, key)
	}
	return
}
