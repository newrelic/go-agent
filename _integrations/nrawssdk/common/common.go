package common

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

	v := reflect.ValueOf(params).Elem()
	n := v.FieldByName("TableName")
	if name, ok := n.Interface().(string); ok {
		tableName = name
	}

	return tableName
}

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

func EndSegment(ctx context.Context) {
	if segment, ok := ctx.Value(segmentContextKey).(endable); ok {
		segment.End()
	}
}
