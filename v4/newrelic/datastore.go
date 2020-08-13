// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

// DatastoreProduct is used to identify your datastore type in New Relic.  It
// is used in the DatastoreSegment Product field.
type DatastoreProduct string

// Datastore names used across New Relic agents:
const (
	DatastoreCassandra     DatastoreProduct = "cassandra"
	DatastoreCouchDB       DatastoreProduct = "couchdb"
	DatastoreDerby         DatastoreProduct = "derby"
	DatastoreDynamoDB      DatastoreProduct = "dynamodb"
	DatastoreElasticsearch DatastoreProduct = "elasticsearch"
	DatastoreFirebird      DatastoreProduct = "firebird"
	DatastoreIBMDB2        DatastoreProduct = "db2"
	DatastoreInformix      DatastoreProduct = "informix"
	DatastoreMSSQL         DatastoreProduct = "mssql"
	DatastoreMemcached     DatastoreProduct = "memcached"
	DatastoreMongoDB       DatastoreProduct = "mongodb"
	DatastoreMySQL         DatastoreProduct = "mysql"
	DatastoreNeptune       DatastoreProduct = "neptune"
	DatastoreOracle        DatastoreProduct = "oracle"
	DatastorePostgres      DatastoreProduct = "postgresql"
	DatastoreRedis         DatastoreProduct = "redis"
	DatastoreRiak          DatastoreProduct = "riak"
	DatastoreSQLite        DatastoreProduct = "sqlite"
	DatastoreSnowflake     DatastoreProduct = "snowflake"
	DatastoreSolr          DatastoreProduct = "solr"
	DatastoreTarantool     DatastoreProduct = "tarantool"
	DatastoreVoltDB        DatastoreProduct = "voltDB"
)
