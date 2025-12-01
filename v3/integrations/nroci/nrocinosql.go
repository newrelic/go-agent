// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package nroci

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/oracle/nosql-go-sdk/nosqldb"
)

func init() {
	//add more here
}

type OCIClient interface {
	AddReplica(req *nosqldb.AddReplicaRequest) (*nosqldb.TableResult, error)
	Close() error
	Delete(req *nosqldb.DeleteRequest) (*nosqldb.DeleteResult, error)
	DoSystemRequest(req *nosqldb.SystemRequest) (*nosqldb.SystemResult, error)
	DoSystemRequestAndWait(statement string, timeout time.Duration, pollInterval time.Duration) (*nosqldb.SystemResult, error)
	DoTableRequest(req *nosqldb.TableRequest) (*nosqldb.TableResult, error)
	DoTableRequestAndWait(req *nosqldb.TableRequest, timeout time.Duration, pollInterval time.Duration) (*nosqldb.TableResult, error)
	DropReplica(req *nosqldb.DropReplicaRequest) (*nosqldb.TableResult, error)
	EnableRateLimiting(enable bool, usePercent float64)

	Get(req *nosqldb.GetRequest) (*nosqldb.GetResult, error)
	GetIndexes(req *nosqldb.GetIndexesRequest) (*nosqldb.GetIndexesResult, error)
	GetQueryVersion() int16
	GetReplicaStats(req *nosqldb.ReplicaStatsRequest) (*nosqldb.ReplicaStatsResult, error)
	GetSerialVersion() int16
	GetServerSerialVersion() int
	GetSystemStatus(req *nosqldb.SystemStatusRequest) (*nosqldb.SystemResult, error)
	GetTable(req *nosqldb.GetTableRequest) (*nosqldb.TableResult, error)
	GetTableUsage(req *nosqldb.TableUsageRequest) (*nosqldb.TableUsageResult, error)

	ListNamespaces() ([]string, error)
	ListRoles() ([]string, error)
	ListTables(req *nosqldb.ListTablesRequest) (*nosqldb.ListTablesResult, error)
	ListUsers() ([]nosqldb.UserInfo, error)

	MultiDelete(req *nosqldb.MultiDeleteRequest) (*nosqldb.MultiDeleteResult, error)
	Prepare(req *nosqldb.PrepareRequest) (*nosqldb.PrepareResult, error)
	Put(req *nosqldb.PutRequest) (*nosqldb.PutResult, error)
	Query(req *nosqldb.QueryRequest) (*nosqldb.QueryResult, error)
	ResetRateLimiters(tableName string)
	SetQueryVersion(qVer int16)
	SetSerialVersion(sVer int16)
	VerifyConnection() error
	WriteMultiple(req *nosqldb.WriteMultipleRequest) (*nosqldb.WriteMultipleResult, error)
}

type ConfigWrapper struct {
	Config *nosqldb.Config
}

type ClientWrapper struct {
	Client OCIClient
	Config *nosqldb.Config
}

type ClientRequestWrapper[R any] struct {
	ClientRequest R
}
type ClientResponseWrapper[T any] struct {
	ClientResponse T
}

func NRDefaultConfig() *ConfigWrapper {
	cfg := nosqldb.Config{}
	return &ConfigWrapper{
		Config: &cfg,
	}
}

func NRCreateClient(cfg *ConfigWrapper) (*ClientWrapper, error) {
	client, err := nosqldb.NewClient(*cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("error creating OCI Client: %s", err.Error())
	}
	return &ClientWrapper{
		Client: client,
		Config: cfg.Config,
	}, nil
}

// extractHostPort extracts host and port from an endpoint URL
func extractHostPort(endpoint string) (host, port string) {
	if endpoint == "" {
		return "", ""
	}

	// Parse the endpoint URL
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", ""
	}

	host = parsedURL.Hostname()
	port = parsedURL.Port()

	// Set default ports if not specified
	if port == "" {
		switch parsedURL.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}

	return host, port
}

func extractRequestFields(req any) (string, string, string) {
	var collection, statement, namespace string
	switch r := req.(type) {
	case *nosqldb.TableRequest:
		collection = r.TableName
		statement = r.Statement
		namespace = r.Namespace
	case *nosqldb.QueryRequest:
		collection = r.TableName
		statement = r.Statement
		namespace = r.Namespace
	case *nosqldb.PutRequest:
		collection = r.TableName
		namespace = r.Namespace
	default:
		// keep strings empty
	}
	return collection, statement, namespace
}

// executeWithDatastoreSegment is a generic helper function that executes a query with a given function from the
// OCI Client.  It takes a type parameter T as any because of the different response types that are used within the
// OCI Client.  This function will take the transaction from the context (if it exists) and create a Datastore Segment.
// It will then call whatever client function has been passed in.
func executeWithDatastoreSegment[T any, R any](
	cw *ClientWrapper,
	ctx context.Context,
	rw *ClientRequestWrapper[R],
	fn func() (T, error),
) (*ClientResponseWrapper[T], error) {

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil, fmt.Errorf("error executing DoTableRequest, no transaction")
	}

	// Extract host and port from config endpoint
	host, port := extractHostPort(cw.Config.Endpoint)
	collection, statement, namespace := extractRequestFields(rw.ClientRequest)
	sgmt := newrelic.DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            newrelic.DatastoreOracle,
		Collection:         collection,
		DatabaseName:       namespace, // using the namespace as the database name in this instance
		ParameterizedQuery: statement,
		Host:               host,
		PortPathOrID:       port,
	}

	responseWrapper := ClientResponseWrapper[T]{}
	res, err := fn() // call the client function
	responseWrapper.ClientResponse = res
	if err != nil {
		return &responseWrapper, fmt.Errorf("error making request: %s", err.Error())
	}

	sgmt.End()

	return &responseWrapper, nil
}

// Wrapper for nosqldb.Client.DoTableRequest.  Provide the ClientWrapper and Context as parameters in addition to the nosqldb.TableRequest.
// Returns a ClientResponseWrapper[*nosqldb.TableResult] and error.
func NRDoTableRequest(cw *ClientWrapper, ctx context.Context, req *nosqldb.TableRequest) (*ClientResponseWrapper[*nosqldb.TableResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.TableRequest]{ClientRequest: req}, func() (*nosqldb.TableResult, error) {
		return cw.Client.DoTableRequest(req)
	})
}

// Wrapper for nosqldb.Client.DoTableRequestWait.  Provide the ClientWrapper and Context as parameters in addition to the nosqldb.TableRequest,
// timeout, and pollInterval. Returns a ClientResponseWrapper[*nosqldb.TableResult] and error.
func NRDoTableRequestAndWait(cw *ClientWrapper, ctx context.Context, req *nosqldb.TableRequest, timeout time.Duration, pollInterval time.Duration) (*ClientResponseWrapper[*nosqldb.TableResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.TableRequest]{ClientRequest: req}, func() (*nosqldb.TableResult, error) {
		return cw.Client.DoTableRequestAndWait(req, timeout, pollInterval)
	})
}

// Wrapper for nosqldb.Client.Query. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.QueryRequest.  Returns a
// ClientResponseWrapper[*nosqldb.QueryResult] and error
func NRQuery(cw *ClientWrapper, ctx context.Context, req *nosqldb.QueryRequest) (*ClientResponseWrapper[*nosqldb.QueryResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.QueryRequest]{ClientRequest: req}, func() (*nosqldb.QueryResult, error) {
		return cw.Client.Query(req)
	})
}

// Wrapper for nosqldb.Client.Put. Provide the ClientWrapper and Context as parameters in addition to the nosqldb.PutRequest. Returns a
// ClientResponseWrapper[*nosqldb.PutResult] and error
func NRPut(cw *ClientWrapper, ctx context.Context, req *nosqldb.PutRequest) (*ClientResponseWrapper[*nosqldb.PutResult], error) {
	return executeWithDatastoreSegment(cw, ctx, &ClientRequestWrapper[*nosqldb.PutRequest]{ClientRequest: req}, func() (*nosqldb.PutResult, error) {
		return cw.Client.Put(req)
	})
}
