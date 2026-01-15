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
	var collection, statement, compartmentID string
	switch r := req.(type) {
	case nosql.QueryRequest:
		// Could do FROM <table-name> type of parse?
		if r.QueryDetails.CompartmentId != nil {
			compartmentID = *r.QueryDetails.CompartmentId
		}
		if r.Statement != nil {
			statement = *r.Statement
		}
	case nosql.UpdateRowRequest:
		if r.TableNameOrId != nil {
			collection = *r.TableNameOrId
		}
		if r.UpdateRowDetails.CompartmentId != nil {
			compartmentID = *r.UpdateRowDetails.CompartmentId
		}
	default:
		// keep strings empty
	}
	return collection, statement, compartmentID
}

func executeWithDatastoreSegmentOCI[T any, R any](
	ctx context.Context,
	rw *NoSQLClientRequestWrapper[R],
	fn func() (T, error),
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

func (cw *NoSQLClientWrapper) Query(ctx context.Context, req *NoSQLClientRequestWrapper[nosql.QueryRequest]) (*NoSQLClientResponseWrapper[nosql.QueryResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, req, func() (nosql.QueryResponse, error) {
		return cw.Client.Query(ctx, req.ClientRequest)
	})
}

func (cw *NoSQLClientWrapper) UpdateRow(ctx context.Context, req *NoSQLClientRequestWrapper[nosql.UpdateRowRequest]) (*NoSQLClientResponseWrapper[nosql.UpdateRowResponse], error) {
	return executeWithDatastoreSegmentOCI(ctx, req, func() (nosql.UpdateRowResponse, error) {
		return cw.Client.UpdateRow(ctx, req.ClientRequest)
	})
}
