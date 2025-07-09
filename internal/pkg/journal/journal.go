package journal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HostLog struct {
	journal *sdjournal.Journal
}

// NewLog instance creates a new HostLog instance
func NewLog() (*HostLog, error) {
	j, err := sdjournal.NewJournal()
	if err != nil {
		return nil, fmt.Errorf("failed to open journal: %w", err)
	}
	return &HostLog{journal: j}, nil
}

// Close the log and underlying journal
func (log *HostLog) Close() error {
	return log.journal.Close()
}

type ListLogParams struct {
	Count int    `json:"count"`
	Unit  string `json:"unit"`
}

func (sj *HostLog) ListLog(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListLogParams]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.Unit != "" {
		if err := sj.journal.AddMatch("_SYSTEMD_UNIT=" + params.Arguments.Unit); err != nil {
			return nil, fmt.Errorf("failed to add unit filter: %w", err)
		}
	}
	if err := sj.journal.SeekTail(); err != nil {
		return nil, fmt.Errorf("failed to seek to end: %w", err)
	}
	_, err := sj.journal.PreviousSkip(uint64(params.Arguments.Count))
	if err != nil {
		return nil, fmt.Errorf("failed to move back entries: %w", err)
	}
	txtContentList := []mcp.Content{}
	isFirst := true
	for {
		ret, err := sj.journal.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to read next entry: %w", err)
		}
		if ret == 0 && isFirst {
			return nil, fmt.Errorf("couldn't get entry for unit: %s", params.Arguments.Unit)
		}
		isFirst = false
		entry, err := sj.journal.GetEntry()
		if err != nil {
			return nil, fmt.Errorf("failed to get entry: %w", err)
		}

		timestamp := time.Unix(0, int64(entry.RealtimeTimestamp)*int64(time.Microsecond))

		structEntr := struct {
			Time time.Time `json:"time"`
			Unit string    `json:"unit"`
			Host string    `json:"host"`
			Msg  string    `json:"message"`
		}{
			Unit: entry.Fields["_SYSTEMD_UNIT"],
			Time: timestamp,
			Host: entry.Fields["_HOSTNAME"],
			Msg:  entry.Fields["MESSAGE"],
		}
		jsonByte, err := json.Marshal(&structEntr)
		if err != nil {
			return nil, err
		}
		txtContentList = append(txtContentList, &mcp.TextContent{
			Text: string(jsonByte),
		})
		if ret == 0 {
			break
		}

	}
	return &mcp.CallToolResultFor[any]{
		Content: txtContentList,
	}, nil
}
