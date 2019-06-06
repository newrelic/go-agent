package nrpq

import (
	"path"
	"regexp"
	"strings"

	"github.com/lib/pq"
	newrelic "github.com/newrelic/go-agent"
)

var dsnSplit = regexp.MustCompile(`(\w+)\s*=\s*('[^=]*'|[^'\s]+)`)

func getFirstHost(value string) string {
	host := strings.SplitN(value, ",", 2)[0]
	host = strings.Trim(host, "[]")
	return host
}

func parseDSN(getenv func(string) string) func(*newrelic.DatastoreSegment, string) {
	return func(s *newrelic.DatastoreSegment, dsn string) {
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			var err error
			dsn, err = pq.ParseURL(dsn)
			if nil != err {
				return
			}
		}

		host := getenv("PGHOST")
		hostaddr := ""
		ppoid := getenv("PGPORT")
		dbname := getenv("PGDATABASE")

		for _, split := range dsnSplit.FindAllStringSubmatch(dsn, -1) {
			if len(split) != 3 {
				continue
			}
			key := split[1]
			value := strings.Trim(split[2], `'`)

			switch key {
			case "dbname":
				dbname = value
			case "host":
				host = getFirstHost(value)
			case "hostaddr":
				hostaddr = getFirstHost(value)
			case "port":
				ppoid = strings.SplitN(value, ",", 2)[0]
			}
		}

		if "" != hostaddr {
			host = hostaddr
		} else if "" == host {
			host = "localhost"
		}
		if "" == ppoid {
			ppoid = "5432"
		}
		if strings.HasPrefix(host, "/") {
			// this is a unix socket
			ppoid = path.Join(host, ".s.PGSQL."+ppoid)
			host = "localhost"
		}

		s.Host = host
		s.PortPathOrID = ppoid
		s.DatabaseName = dbname
	}
}
