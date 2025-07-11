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

func (sj *HostLog) seekAndSkip(count uint64) (uint64, error) {
	if err := sj.journal.SeekTail(); err != nil {
		return 0, fmt.Errorf("failed to seek to end: %w", err)
	}
	if skip, err := sj.journal.PreviousSkip(count); err != nil {
		return 0, fmt.Errorf("failed to move back entries: %w", err)
	} else {
		return skip, nil
	}
}

// get the lat log entries for a given unit, else just the last messages
func (sj *HostLog) ListLog(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListLogParams]) (*mcp.CallToolResultFor[any], error) {
	if params.Arguments.Unit != "" {
		if err := sj.journal.AddMatch("SYSLOG_IDENTIFIER=" + params.Arguments.Unit); err != nil {
			return nil, fmt.Errorf("failed to add unit filter: %w", err)
		}
		seek, err := sj.seekAndSkip(uint64(params.Arguments.Count))
		if err != nil {
			return nil, err
		}
		if seek == 0 {
			if err := sj.journal.AddMatch("_SYSTEMD_USER_UNIT=" + params.Arguments.Unit); err != nil {
				return nil, fmt.Errorf("failed to add unit filter: %w", err)
			}
			seek, err := sj.seekAndSkip(uint64(params.Arguments.Count))
			if err != nil {
				return nil, err
			}
			if seek == 0 {
				sj.journal.FlushMatches()
				_, err := sj.seekAndSkip(uint64(params.Arguments.Count))
				if err != nil {
					return nil, err
				}

			}
		}
	} else {
		_, err := sj.seekAndSkip(uint64(params.Arguments.Count))
		if err != nil {
			return nil, err
		}

	}
	txtContentList := []mcp.Content{}
	isFirst := true
	for {
		ret, err := sj.journal.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to read next entry: %w", err)
		}
		if ret == 0 && isFirst {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("couldn't match unit: '%s'", params.Arguments.Unit)}},
			}, nil
		}
		isFirst = false
		entry, err := sj.journal.GetEntry()
		if err != nil {
			return nil, fmt.Errorf("failed to get entry: %w", err)
		}

		timestamp := time.Unix(0, int64(entry.RealtimeTimestamp)*int64(time.Microsecond))

		structEntr := struct {
			Time time.Time         `json:"time"`
			Unit string            `json:"unit"`
			Host string            `json:"host"`
			Msg  string            `json:"message"`
			Full map[string]string `json:"full"`
		}{
			Unit: entry.Fields["SYSLOG_IDENTIFIER"],
			Time: timestamp,
			Host: entry.Fields["_HOSTNAME"],
			Msg:  entry.Fields["MESSAGE"],
			Full: entry.Fields,
		}
		if structEntr.Unit == "" {
			structEntr.Unit = fmt.Sprintf("%s:%s", entry.Fields["_SYSTEMD_UNIT"], entry.Fields["_SYSTEMD_USER_UNIT"])
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
