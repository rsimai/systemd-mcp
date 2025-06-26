package systemd

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"
)

// DbusConnection is an interface that abstracts the dbus connection.
// This is primarily for testing purposes.
type DbusConnection interface {
	Close()
	ListUnitsContext(ctx context.Context) ([]dbus.UnitStatus, error)
	ListUnitsFilteredContext(ctx context.Context, filter []string) ([]dbus.UnitStatus, error)
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
