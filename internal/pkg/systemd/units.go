package systemd

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

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
	States  []string `json:"states"`
	Verbose bool     `json:"verbose"`
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
		var jsonByte []byte
		if params.Arguments.Verbose {
			jsonByte, _ = json.Marshal(&u)
		} else {
			lightUnit := LightUnit{
				Name:        u.Name,
				State:       u.ActiveState,
				Description: u.Description,
			}
			jsonByte, _ = json.Marshal(&lightUnit)
		}
		txtContenList = append(txtContenList, &mcp.TextContent{
			Text: string(jsonByte),
		})

	}

	return &mcp.CallToolResultFor[any]{
		Content: txtContenList,
	}, nil
}

type ListUnitNameParams struct {
	Names   []string `json:"names"`
	Verbose bool     `json:"debug"`
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
		jsonByte, err := json.Marshal(&props)
		if err != nil {
			return nil, err
		}
		if params.Arguments.Verbose {

			txtContentList = append(txtContentList, &mcp.TextContent{
				Text: string(jsonByte),
			})
		} else {
			prop := struct {
				Id          string `json:"Id"`
				Description string `json:"Description"`

				// Load state info
				LoadState      string `json:"LoadState"`
				FragmentPath   string `json:"FragmentPath"`
				UnitFileState  string `json:"UnitFileState"`
				UnitFilePreset string `json:"UnitFilePreset"`

				// Active state info
				ActiveState          string `json:"ActiveState"`
				SubState             string `json:"SubState"`
				ActiveEnterTimestamp uint64 `json:"ActiveEnterTimestamp"`

				// Process info
				InvocationID   string `json:"InvocationID"`
				MainPID        int    `json:"MainPID"`
				ExecMainPID    int    `json:"ExecMainPID"`
				ExecMainStatus int    `json:"ExecMainStatus"`

				// Resource usage
				TasksCurrent int    `json:"TasksCurrent"`
				TasksMax     uint64 `json:"TasksMax"`
				CPUUsageNSec uint64 `json:"CPUUsageNSec"`

				// Control group
				ControlGroup string `json:"ControlGroup"`

				// Exec commands (simplified - would need additional processing)
				ExecStartPre [][]interface{} `json:"ExecStartPre"`
				ExecStart    [][]interface{} `json:"ExecStart"`

				// Additional fields that might be useful
				Restart       string `json:"Restart"`
				MemoryCurrent uint64 `json:"MemoryCurrent"`
			}{}
			err = json.Unmarshal(jsonByte, &prop)
			if err != nil {
				return nil, err
			}
			jsonByte, err = json.Marshal(&prop)
			if err != nil {
				return nil, err
			}
			txtContentList = append(txtContentList, &mcp.TextContent{
				Text: string(jsonByte),
			})
		}

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

type RestartReloadParams struct {
	Name         string `json:"name"`
	TimeOut      uint   `json:"timeout"`
	Mode         string `json:"mode"`
	Forcerestart bool   `json:"forcerestart"`
}

// return which are define in the upstream documentation as:
// The mode needs to be one of
// replace, fail, isolate, ignore-dependencies, ignore-requirements. If
// "replace" the call will start the unit and its dependencies, possibly
// replacing already queued jobs that conflict with this. If "fail" the call
// will start the unit and its dependencies, but will fail if this would change
// an already queued job. If "isolate" the call will start the unit in question
// and terminate all units that aren't dependencies of it. If
// "ignore-dependencies" it will start a unit but ignore all its dependencies.
// If "ignore-requirements" it will start a unit but only ignore the
// requirement dependencies. It is not recommended to make use of the latter
// two options.
func ValidRestartModes() []string {
	return []string{"replace", "fail", "isolate", "ignore-dependencies", "ignore-requirements"}
}

func ValidRestartModesEnum() (ret []mcp.SchemaOption) {
	for _, m := range ValidRestartModes() {
		ret = append(ret, mcp.Enum(m))
	}
	return
}

const MaxTimeOut uint = 60

// restart or reload a service
func (conn *Connection) RestartReloadUnit(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[RestartReloadParams]) (res *mcp.CallToolResultFor[any], err error) {
	if params.Arguments.Mode == "" {
		params.Arguments.Mode = "replace"
	}
	if !slices.Contains(ValidRestartModes(), params.Arguments.Mode) {
		return nil, fmt.Errorf("invalid mode for restart or reload: %s", params.Arguments.Mode)
	}
	if params.Arguments.TimeOut > MaxTimeOut {
		return nil, fmt.Errorf("not waiting longer than MaxTimeOut(%d), longer operation will run in the background and result can be gathered with separate function.", MaxTimeOut)
	}
	if params.Arguments.Forcerestart {
		_, err = conn.dbus.RestartUnitContext(ctx, params.Arguments.Name, params.Arguments.Mode, conn.rchannel)
	} else {
		_, err = conn.dbus.ReloadOrRestartUnitContext(ctx, params.Arguments.Name, params.Arguments.Mode, conn.rchannel)
	}
	if err != nil {
		return nil, err
	}
	return conn.CheckForRestartReloadRunning(ctx, cc, &mcp.CallToolParamsFor[CheckReloadRestartParams]{
		Arguments: CheckReloadRestartParams{TimeOut: params.Arguments.TimeOut},
	})
}

type CheckReloadRestartParams struct {
	TimeOut uint `json:"timeout"`
}

// check status of reload or restart
func (conn *Connection) CheckForRestartReloadRunning(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[CheckReloadRestartParams]) (res *mcp.CallToolResultFor[any], err error) {
	select {
	case result := <-conn.rchannel:
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: result,
				},
			},
		}, nil
	case <-time.After(3 * time.Second):
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "Reload or restart still in progress.",
				},
			},
		}, nil
	default:
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "Finished",
				},
			},
		}, nil
	}
}

type StopParams struct {
	Name    string `json:"name"`
	TimeOut uint   `json:"timeout"`
	Mode    string `json:"mode"`
	Kill    bool   `json:"kill"`
}

// Stop or kill the given unit
func (conn *Connection) StopUnit(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[StopParams]) (res *mcp.CallToolResultFor[any], err error) {
	if params.Arguments.Mode == "" {
		params.Arguments.Mode = "replace"
	}
	if !slices.Contains(ValidRestartModes(), params.Arguments.Mode) {
		return nil, fmt.Errorf("invalid mode for restart or reload: %s", params.Arguments.Mode)
	}
	if params.Arguments.TimeOut > MaxTimeOut {
		return nil, fmt.Errorf("not waiting longer than MaxTimeOut(%d), longer operation will run in the background and result can be gathered with separate function.", MaxTimeOut)
	}
	if params.Arguments.Kill {
		conn.dbus.KillUnitContext(ctx, params.Arguments.Name, int32(9))
	} else {
		_, err = conn.dbus.StopUnitContext(ctx, params.Arguments.Name, params.Arguments.Mode, conn.rchannel)
	}
	if err != nil {
		return nil, err
	}
	return conn.CheckForRestartReloadRunning(ctx, cc, &mcp.CallToolParamsFor[CheckReloadRestartParams]{
		Arguments: CheckReloadRestartParams{TimeOut: params.Arguments.TimeOut},
	})
}

type EnableParams struct {
	File string `json:"file"`
}

func (conn *Connection) EnableUnit(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[EnableParams]) (res *mcp.CallToolResultFor[any], err error) {
	_, enabledRes, err := conn.dbus.EnableUnitFilesContext(ctx, []string{params.Arguments.File}, false, true)
	if err != nil {
		return nil, fmt.Errorf("error when enabling: %w", err)
	}
	if len(enabledRes) == 0 {
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("nothing changed for %s", params.Arguments.File),
				},
			},
		}, nil
	} else {
		txtContentList := []mcp.Content{}
		for _, res := range enabledRes {
			resJson := struct {
				Type        string `json:"type"`
				Filename    string `json:"filename"`
				Destination string `json:"destination"`
			}{
				Type:        res.Type,
				Filename:    res.Filename,
				Destination: res.Destination,
			}
			jsonByte, err := json.Marshal(resJson)
			if err != nil {
				return nil, fmt.Errorf("could not unmarshall result: %w", err)
			}
			txtContentList = append(txtContentList, &mcp.TextContent{
				Text: string(jsonByte),
			})
		}
		return &mcp.CallToolResultFor[any]{
			Content: txtContentList,
		}, nil
	}
}
