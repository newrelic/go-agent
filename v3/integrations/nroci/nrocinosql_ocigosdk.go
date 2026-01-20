package nroci

import (
	"context"
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/nosql"
)

func init() {

}

type NoSQLClient interface {
	Query(ctx context.Context, req nosql.QueryRequest) (nosql.QueryResponse, error)
	UpdateRow(ctx context.Context, req nosql.UpdateRowRequest) (nosql.UpdateRowResponse, error)
	CreateTable(ctx context.Context, req nosql.CreateTableRequest) (nosql.CreateTableResponse, error)
	DeleteRow(ctx context.Context, req nosql.DeleteRowRequest) (nosql.DeleteRowResponse, error)
	DeleteTable(ctx context.Context, req nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error)
	GetRow(ctx context.Context, req nosql.GetRowRequest) (nosql.GetRowResponse, error)
	GetTable(ctx context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error)
	UpdateTable(ctx context.Context, req nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error)
}

type NoSQLClientWrapper struct {
	Client NoSQLClient
}

type NoSQLClientRequestWrapper[R any] struct {
	ClientRequest R
}

type NoSQLClientResponseWrapper[T any] struct {
	ClientResponse T
}

func NRNewNoSQLClientWithConfigurationProvider(configProvider common.ConfigurationProvider) (*NoSQLClientWrapper, error) {
	ociNoSQLClient, err := nosql.NewNosqlClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}
	return &NoSQLClientWrapper{
		Client: ociNoSQLClient,
	}, nil
}

func extractRequestFieldsOCI(req any) (string, string, string) {

	requestNilCheck := func(value *string) string {
		if value != nil {
			return *value
		}
		return ""
	}

	var collection, statement, compartmentID string
	switch r := req.(type) {
	case nosql.QueryRequest:
		// Could do FROM <table-name> type of parse?
		compartmentID = requestNilCheck(r.QueryDetails.CompartmentId)
		statement = requestNilCheck(r.QueryDetails.Statement)
	case nosql.UpdateRowRequest:
		collection = requestNilCheck(r.TableNameOrId)
		compartmentID = requestNilCheck(r.UpdateRowDetails.CompartmentId)
	case nosql.CreateTableRequest:
		collection = requestNilCheck(r.CreateTableDetails.Name)
		compartmentID = requestNilCheck(r.CreateTableDetails.CompartmentId)
		statement = requestNilCheck(r.CreateTableDetails.DdlStatement)
	case nosql.DeleteRowRequest:
		collection = requestNilCheck(r.TableNameOrId)
		compartmentID = requestNilCheck(r.CompartmentId)
	case nosql.DeleteTableRequest:
		collection = requestNilCheck(r.TableNameOrId)
		compartmentID = requestNilCheck(r.CompartmentId)
	case nosql.GetRowRequest:
		collection = requestNilCheck(r.TableNameOrId)
		compartmentID = requestNilCheck(r.CompartmentId)
	case nosql.GetTableRequest:
		collection = requestNilCheck(r.TableNameOrId)
		compartmentID = requestNilCheck(r.CompartmentId)
	case nosql.UpdateTableRequest:
		collection = requestNilCheck(r.TableNameOrId)
		compartmentID = requestNilCheck(r.UpdateTableDetails.CompartmentId)
		statement = requestNilCheck(r.UpdateTableDetails.DdlStatement)
	default:
		// keep strings empty
	}
	return collection, statement, compartmentID
}

func executeWithDatastoreSegmentOCI[T any, R any](
	ctx context.Context,
	rw *NoSQLClientRequestWrapper[R],
	fn func() (T, error),
	operation string,
) (*NoSQLClientResponseWrapper[T], error) {

	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil, fmt.Errorf("error executing OCI request, no transaction")
	}

	collection, statement, compartmentID := extractRequestFieldsOCI(rw.ClientRequest)
	sgmt := newrelic.DatastoreSegment{
		StartTime:          txn.StartSegmentNow(),
		Product:            newrelic.DatastoreOracle,
		ParameterizedQuery: statement,
		Collection:         collection,
		DatabaseName:       compartmentID,
		Operation:          operation,
	}
	res, err := fn()
	if err != nil {
		return nil, fmt.Errorf("error executing OCI requestL %s", err.Error())
	}
	responseWrapper := NoSQLClientResponseWrapper[T]{
		ClientResponse: res,
	}

	sgmt.End()
	return &responseWrapper, nil
}

func (cw *NoSQLClientWrapper) Query(ctx context.Context, req nosql.QueryRequest) (*NoSQLClientResponseWrapper[nosql.QueryResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.QueryRequest]{
		ClientRequest: req,
	}, func() (nosql.QueryResponse, error) {
		return cw.Client.Query(ctx, req)
	}, "Query")
}

func (cw *NoSQLClientWrapper) UpdateRow(ctx context.Context, req nosql.UpdateRowRequest) (*NoSQLClientResponseWrapper[nosql.UpdateRowResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.UpdateRowRequest]{
		ClientRequest: req,
	}, func() (nosql.UpdateRowResponse, error) {
		return cw.Client.UpdateRow(ctx, req)
	}, "UpdateRow")
}

func (cw *NoSQLClientWrapper) CreateTable(ctx context.Context, req nosql.CreateTableRequest) (*NoSQLClientResponseWrapper[nosql.CreateTableResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.CreateTableRequest]{
		ClientRequest: req,
	}, func() (nosql.CreateTableResponse, error) {
		return cw.Client.CreateTable(ctx, req)
	}, "CreateTable")
}

func (cw *NoSQLClientWrapper) DeleteRow(ctx context.Context, req nosql.DeleteRowRequest) (*NoSQLClientResponseWrapper[nosql.DeleteRowResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.DeleteRowRequest]{
		ClientRequest: req,
	}, func() (nosql.DeleteRowResponse, error) {
		return cw.Client.DeleteRow(ctx, req)
	}, "DeleteRow")
}

func (cw *NoSQLClientWrapper) DeleteTable(ctx context.Context, req nosql.DeleteTableRequest) (*NoSQLClientResponseWrapper[nosql.DeleteTableResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.DeleteTableRequest]{
		ClientRequest: req,
	}, func() (nosql.DeleteTableResponse, error) {
		return cw.Client.DeleteTable(ctx, req)
	}, "DeleteTable")
}

func (cw *NoSQLClientWrapper) GetRow(ctx context.Context, req nosql.GetRowRequest) (*NoSQLClientResponseWrapper[nosql.GetRowResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.GetRowRequest]{
		ClientRequest: req,
	}, func() (nosql.GetRowResponse, error) {
		return cw.Client.GetRow(ctx, req)
	}, "GetRow")
}

func (cw *NoSQLClientWrapper) GetTable(ctx context.Context, req nosql.GetTableRequest) (*NoSQLClientResponseWrapper[nosql.GetTableResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.GetTableRequest]{
		ClientRequest: req,
	}, func() (nosql.GetTableResponse, error) {
		return cw.Client.GetTable(ctx, req)
	}, "GetTable")
}

func (cw *NoSQLClientWrapper) UpdateTable(ctx context.Context, req nosql.UpdateTableRequest) (*NoSQLClientResponseWrapper[nosql.UpdateTableResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, &NoSQLClientRequestWrapper[nosql.UpdateTableRequest]{
		ClientRequest: req,
	}, func() (nosql.UpdateTableResponse, error) {
		return cw.Client.UpdateTable(ctx, req)
	}, "UpdateTable")
}
