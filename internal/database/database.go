package database

import "fmt"

type Database interface {
	InsertCheck(host, port, protocol, status string, elapsed int64) error
	Close() error
}

//nolint:gocritic,revive
func NewDatabase(dbType, dbPath string) (Database, error) {
	switch dbType {
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
