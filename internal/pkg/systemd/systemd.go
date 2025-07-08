package systemd

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"
)

// DbusConnection is an interface that abstracts the dbus connection.
// This is primarily for testing purposes.
type DbusConnection interface {
	ListUnitsContext(ctx context.Context) ([]dbus.UnitStatus, error)
	ListUnitsFilteredContext(ctx context.Context, states []string) ([]dbus.UnitStatus, error)
	ListUnitsByPatternsContext(ctx context.Context, states []string, patterns []string) ([]dbus.UnitStatus, error)
	GetAllPropertiesContext(ctx context.Context, unitName string) (map[string]interface{}, error)
	Close()
}

type Connection struct {
	dbus DbusConnection
}

// opens a new user connection to the dbus
func NewUser(ctx context.Context) (conn *Connection, err error) {
	conn = new(Connection)
	conn.dbus, err = dbus.NewUserConnectionContext(ctx)
	if err != nil {
		return nil, err
	}
	return conn, err
}
func NewSystem(ctx context.Context) (conn *Connection, err error) {
	conn = new(Connection)
	conn.dbus, err = dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return nil, err
	}
	return conn, err
}

// close the connection
func (conn *Connection) Close() {
	conn.dbus.Close()
}
