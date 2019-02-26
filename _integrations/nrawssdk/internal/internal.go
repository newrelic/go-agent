package internal

import (
	"context"
	"net/http"
	"reflect"

	newrelic "github.com/newrelic/go-agent"
)

type contextKeyType struct{}

var segmentContextKey = contextKeyType(struct{}{})

type endable interface{ End() error }

func getTableName(params interface{}) string {
	var tableName string

	v := reflect.ValueOf(params)
	if v.IsValid() && v.Kind() == reflect.Ptr {
		e := v.Elem()
		if e.Kind() == reflect.Struct {
			n := e.FieldByName("TableName")
			if n.IsValid() {
				if name, ok := n.Interface().(*string); ok {
					if nil != name {
						tableName = *name
					}
				}
			}
		}
	}

	return tableName
}

// StartSegment starts a segment of either type DatastoreSegment or
// ExternalSegment given the serviceName provided. The segment is then added to
// the request context.
func StartSegment(httpRequest *http.Request, serviceName, operation string,
	params interface{}) *http.Request {

	httpCtx := httpRequest.Context()
	txn := newrelic.FromContext(httpCtx)

	var segment endable
	if serviceName == "dynamodb" {
		segment = &newrelic.DatastoreSegment{
			Product:            newrelic.DatastoreDynamoDB,
			Collection:         getTableName(params),
			Operation:          operation,
			ParameterizedQuery: "",
			QueryParameters:    map[string]interface{}{},
			Host:               httpRequest.URL.Host,
			PortPathOrID:       httpRequest.URL.Port(),
			DatabaseName:       "",
			StartTime:          newrelic.StartSegmentNow(txn),
		}
	} else {
		segment = newrelic.StartExternalSegment(txn, httpRequest)
	}

	ctx := context.WithValue(httpCtx, segmentContextKey, segment)
	return httpRequest.WithContext(ctx)
}

// EndSegment will end any segment found in the given context.
func EndSegment(ctx context.Context) {
	if segment, ok := ctx.Value(segmentContextKey).(endable); ok {
		segment.End()
	}
}
