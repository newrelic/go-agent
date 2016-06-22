package datastore

// Segment contains the fields that should be provided when calling
// Transaction.EndDatastore.
type Segment struct {
	// Product is the datastore type.  See the constants below.
	Product Product
	// Collection is the table or group.
	Collection string
	// Operation is the relevant action, e.g. "SELECT" or "GET".
	Operation string
}

// Product encourages consistent metrics across New Relic agents.  You may
// create your own if your datastore is not listed below.
type Product string

// Datastore names used across New Relic agents:
const (
	Cassandra     Product = "Cassandra"
	Derby                 = "Derby"
	Elasticsearch         = "Elasticsearch"
	Firebird              = "Firebird"
	IBMDB2                = "IBMDB2"
	Informix              = "Informix"
	Memcached             = "Memcached"
	MongoDB               = "MongoDB"
	MySQL                 = "MySQL"
	MSSQL                 = "MSSQL"
	Oracle                = "Oracle"
	Postgres              = "Postgres"
	Redis                 = "Redis"
	Solr                  = "Solr"
	SQLite                = "SQLite"
	CouchDB               = "CouchDB"
	Riak                  = "Riak"
	VoltDB                = "VoltDB"
)
