package nrmysql

import (
	"net"

	"github.com/go-sql-driver/mysql"
	newrelic "github.com/newrelic/go-agent"
)

func getHostPortPathOrID(cfg *mysql.Config) (host string, ppoid string) {
	switch cfg.Net {
	case "unix", "unixgram", "unixpacket":
		host = "localhost"
		ppoid = cfg.Addr
	case "cloudsql":
		host = cfg.Addr
	default:
		host, ppoid, _ = net.SplitHostPort(cfg.Addr)
	}
	return
}

func ParseDSN(s *newrelic.DatastoreSegment, dsn string) {
	cfg, err := mysql.ParseDSN(dsn)
	if nil != err {
		return
	}

	ParseConfig(s, cfg)
}

func ParseConfig(s *newrelic.DatastoreSegment, cfg *mysql.Config) {
	s.DatabaseName = cfg.DBName
	s.Host, s.PortPathOrID = getHostPortPathOrID(cfg)
}
