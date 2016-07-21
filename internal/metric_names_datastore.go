package internal

import "github.com/newrelic/go-agent/datastore"

var (
	datastoreProductMetricsCache = map[datastore.Product]datastoreProductMetrics{
		datastore.Cassandra: datastoreProductMetrics{
			All:   "Datastore/Cassandra/all",
			Web:   "Datastore/Cassandra/allWeb",
			Other: "Datastore/Cassandra/allOther",
		},
		datastore.Derby: datastoreProductMetrics{
			All:   "Datastore/Derby/all",
			Web:   "Datastore/Derby/allWeb",
			Other: "Datastore/Derby/allOther",
		},
		datastore.Elasticsearch: datastoreProductMetrics{
			All:   "Datastore/Elasticsearch/all",
			Web:   "Datastore/Elasticsearch/allWeb",
			Other: "Datastore/Elasticsearch/allOther",
		},
		datastore.Firebird: datastoreProductMetrics{
			All:   "Datastore/Firebird/all",
			Web:   "Datastore/Firebird/allWeb",
			Other: "Datastore/Firebird/allOther",
		},
		datastore.IBMDB2: datastoreProductMetrics{
			All:   "Datastore/IBMDB2/all",
			Web:   "Datastore/IBMDB2/allWeb",
			Other: "Datastore/IBMDB2/allOther",
		},
		datastore.Informix: datastoreProductMetrics{
			All:   "Datastore/Informix/all",
			Web:   "Datastore/Informix/allWeb",
			Other: "Datastore/Informix/allOther",
		},
		datastore.Memcached: datastoreProductMetrics{
			All:   "Datastore/Memcached/all",
			Web:   "Datastore/Memcached/allWeb",
			Other: "Datastore/Memcached/allOther",
		},
		datastore.MongoDB: datastoreProductMetrics{
			All:   "Datastore/MongoDB/all",
			Web:   "Datastore/MongoDB/allWeb",
			Other: "Datastore/MongoDB/allOther",
		},
		datastore.MySQL: datastoreProductMetrics{
			All:   "Datastore/MySQL/all",
			Web:   "Datastore/MySQL/allWeb",
			Other: "Datastore/MySQL/allOther",
		},
		datastore.MSSQL: datastoreProductMetrics{
			All:   "Datastore/MSSQL/all",
			Web:   "Datastore/MSSQL/allWeb",
			Other: "Datastore/MSSQL/allOther",
		},
		datastore.Oracle: datastoreProductMetrics{
			All:   "Datastore/Oracle/all",
			Web:   "Datastore/Oracle/allWeb",
			Other: "Datastore/Oracle/allOther",
		},
		datastore.Postgres: datastoreProductMetrics{
			All:   "Datastore/Postgres/all",
			Web:   "Datastore/Postgres/allWeb",
			Other: "Datastore/Postgres/allOther",
		},
		datastore.Redis: datastoreProductMetrics{
			All:   "Datastore/Redis/all",
			Web:   "Datastore/Redis/allWeb",
			Other: "Datastore/Redis/allOther",
		},
		datastore.Solr: datastoreProductMetrics{
			All:   "Datastore/Solr/all",
			Web:   "Datastore/Solr/allWeb",
			Other: "Datastore/Solr/allOther",
		},
		datastore.SQLite: datastoreProductMetrics{
			All:   "Datastore/SQLite/all",
			Web:   "Datastore/SQLite/allWeb",
			Other: "Datastore/SQLite/allOther",
		},
		datastore.CouchDB: datastoreProductMetrics{
			All:   "Datastore/CouchDB/all",
			Web:   "Datastore/CouchDB/allWeb",
			Other: "Datastore/CouchDB/allOther",
		},
		datastore.Riak: datastoreProductMetrics{
			All:   "Datastore/Riak/all",
			Web:   "Datastore/Riak/allWeb",
			Other: "Datastore/Riak/allOther",
		},
		datastore.VoltDB: datastoreProductMetrics{
			All:   "Datastore/VoltDB/all",
			Web:   "Datastore/VoltDB/allWeb",
			Other: "Datastore/VoltDB/allOther",
		},
	}
)
