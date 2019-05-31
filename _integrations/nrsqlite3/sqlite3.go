package nrsqlite3

import (
	"path/filepath"
	"strings"

	newrelic "github.com/newrelic/go-agent"
)

func getPortPathOrID(dsn string) (ppoid string) {
	ppoid = strings.Split(dsn, "?")[0]
	ppoid = strings.TrimPrefix(ppoid, "file:")

	if ":memory:" != ppoid && "" != ppoid {
		if abs, err := filepath.Abs(ppoid); nil == err {
			ppoid = abs
		}
	}

	return
}

// ParseDSN accepts a DSN string and sets the Host, PortPathOrID, and
// DatabaseName fields on a newrelic.DatastoreSegment.
func ParseDSN(s *newrelic.DatastoreSegment, dsn string) {
	// See https://godoc.org/github.com/mattn/go-sqlite3#SQLiteDriver.Open
	s.Host = "localhost"
	s.PortPathOrID = getPortPathOrID(dsn)
	s.DatabaseName = ""
}
