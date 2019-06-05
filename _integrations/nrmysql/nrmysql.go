// +build go1.10

package nrmysql

import (
	"database/sql"
	"database/sql/driver"
	"net"

	"github.com/go-sql-driver/mysql"
	newrelic "github.com/newrelic/go-agent"
)

var (
	baseBuilder = newrelic.DriverSegmentBuilder{
		BaseSegment: newrelic.DatastoreSegment{
			Product: newrelic.DatastoreMySQL,
		},
		ParseQuery: nil, // TODO
		ParseDSN:   parseDSN,
	}
	// Driver can be used in place of mysql.MySQLDriver{} for instrumented
	// MySQL communication
	Driver = newrelic.InstrumentDriver(mysql.MySQLDriver{}, baseBuilder)
)

func init() {
	sql.Register("nrmysql", Driver)
}

// NewConnector can be used in place of mysql.NewConnector for instrumented
// MySQL communication.
func NewConnector(cfg *mysql.Config) (driver.Connector, error) {
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return connector, err
	}
	bld := baseBuilder
	parseConfig(&bld.BaseSegment, cfg)
	return newrelic.InstrumentConnector(connector, bld), nil
}

func parseDSN(s *newrelic.DatastoreSegment, dsn string) {
	cfg, err := mysql.ParseDSN(dsn)
	if nil != err {
		return
	}
	parseConfig(s, cfg)
}

func parseConfig(s *newrelic.DatastoreSegment, cfg *mysql.Config) {
	s.DatabaseName = cfg.DBName

	var host, ppoid string
	switch cfg.Net {
	case "unix", "unixgram", "unixpacket":
		host = "localhost"
		ppoid = cfg.Addr
	case "cloudsql":
		host = cfg.Addr
	default:
		var err error
		host, ppoid, err = net.SplitHostPort(cfg.Addr)
		if nil != err {
			host = cfg.Addr
		} else if host == "" {
			host = "localhost"
		}
	}

	s.Host = host
	s.PortPathOrID = ppoid
}
