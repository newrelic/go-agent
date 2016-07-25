package internal

import "github.com/newrelic/go-agent/datastore"

var (
	datastoreProductMetricsCache = map[datastore.Product]datastoreProductMetrics{
		datastore.Cassandra: {
			All:   "Datastore/Cassandra/all",
			Web:   "Datastore/Cassandra/allWeb",
			Other: "Datastore/Cassandra/allOther",
		},
		datastore.Derby: {
			All:   "Datastore/Derby/all",
			Web:   "Datastore/Derby/allWeb",
			Other: "Datastore/Derby/allOther",
		},
		datastore.Elasticsearch: {
			All:   "Datastore/Elasticsearch/all",
			Web:   "Datastore/Elasticsearch/allWeb",
			Other: "Datastore/Elasticsearch/allOther",
		},
		datastore.Firebird: {
			All:   "Datastore/Firebird/all",
			Web:   "Datastore/Firebird/allWeb",
			Other: "Datastore/Firebird/allOther",
		},
		datastore.IBMDB2: {
			All:   "Datastore/IBMDB2/all",
			Web:   "Datastore/IBMDB2/allWeb",
			Other: "Datastore/IBMDB2/allOther",
		},
		datastore.Informix: {
			All:   "Datastore/Informix/all",
			Web:   "Datastore/Informix/allWeb",
			Other: "Datastore/Informix/allOther",
		},
		datastore.Memcached: {
			All:   "Datastore/Memcached/all",
			Web:   "Datastore/Memcached/allWeb",
			Other: "Datastore/Memcached/allOther",
		},
		datastore.MongoDB: {
			All:   "Datastore/MongoDB/all",
			Web:   "Datastore/MongoDB/allWeb",
			Other: "Datastore/MongoDB/allOther",
		},
		datastore.MySQL: {
			All:   "Datastore/MySQL/all",
			Web:   "Datastore/MySQL/allWeb",
			Other: "Datastore/MySQL/allOther",
		},
		datastore.MSSQL: {
			All:   "Datastore/MSSQL/all",
			Web:   "Datastore/MSSQL/allWeb",
			Other: "Datastore/MSSQL/allOther",
		},
		datastore.Oracle: {
			All:   "Datastore/Oracle/all",
			Web:   "Datastore/Oracle/allWeb",
			Other: "Datastore/Oracle/allOther",
		},
		datastore.Postgres: {
			All:   "Datastore/Postgres/all",
			Web:   "Datastore/Postgres/allWeb",
			Other: "Datastore/Postgres/allOther",
		},
		datastore.Redis: {
			All:   "Datastore/Redis/all",
			Web:   "Datastore/Redis/allWeb",
			Other: "Datastore/Redis/allOther",
		},
		datastore.Solr: {
			All:   "Datastore/Solr/all",
			Web:   "Datastore/Solr/allWeb",
			Other: "Datastore/Solr/allOther",
		},
		datastore.SQLite: {
			All:   "Datastore/SQLite/all",
			Web:   "Datastore/SQLite/allWeb",
			Other: "Datastore/SQLite/allOther",
		},
		datastore.CouchDB: {
			All:   "Datastore/CouchDB/all",
			Web:   "Datastore/CouchDB/allWeb",
			Other: "Datastore/CouchDB/allOther",
		},
		datastore.Riak: {
			All:   "Datastore/Riak/all",
			Web:   "Datastore/Riak/allWeb",
			Other: "Datastore/Riak/allOther",
		},
		datastore.VoltDB: {
			All:   "Datastore/VoltDB/all",
			Web:   "Datastore/VoltDB/allWeb",
			Other: "Datastore/VoltDB/allOther",
		},
	}
)
