// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

// DatastoreProduct is used to identify your datastore type in New Relic.  It
// is used in the DatastoreSegment Product field.  See
// https://github.com/newrelic/go-agent/blob/master/datastore.go for the full
// list of available DatastoreProducts.
type DatastoreProduct string

// Datastore names used across New Relic agents:
const (
	DatastoreCassandra     DatastoreProduct = "Cassandra"
	DatastoreDerby         DatastoreProduct = "Derby"
	DatastoreElasticsearch DatastoreProduct = "Elasticsearch"
	DatastoreFirebird      DatastoreProduct = "Firebird"
	DatastoreIBMDB2        DatastoreProduct = "IBMDB2"
	DatastoreInformix      DatastoreProduct = "Informix"
	DatastoreMemcached     DatastoreProduct = "Memcached"
	DatastoreMongoDB       DatastoreProduct = "MongoDB"
	DatastoreMySQL         DatastoreProduct = "MySQL"
	DatastoreMSSQL         DatastoreProduct = "MSSQL"
	DatastoreNeptune       DatastoreProduct = "Neptune"
	DatastoreOracle        DatastoreProduct = "Oracle"
	DatastorePostgres      DatastoreProduct = "Postgres"
	DatastoreRedis         DatastoreProduct = "Redis"
	DatastoreSolr          DatastoreProduct = "Solr"
	DatastoreSQLite        DatastoreProduct = "SQLite"
	DatastoreCouchDB       DatastoreProduct = "CouchDB"
	DatastoreRiak          DatastoreProduct = "Riak"
	DatastoreVoltDB        DatastoreProduct = "VoltDB"
	DatastoreDynamoDB      DatastoreProduct = "DynamoDB"
)
