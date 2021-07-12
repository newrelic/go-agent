// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package nrawssdk instruments requests made by the
// https://github.com/aws/aws-sdk-go-v2 library.
//
// For most operations, external segments and spans are automatically created
// for display in the New Relic UI on the External services section. For
// DynamoDB operations, datastore segements and spans are created and will be
// displayed on the Databases page. All operations will also be displayed on
// transaction traces and distributed traces.
//
// To use this integration, simply apply the AppendMiddlewares fuction to the apiOptions in
// your AWS Config object before performing any AWS operations. See
// example/main.go for a working sample.
package nrawssdk

import (
	"context"
	"strconv"

	awsmiddle "github.com/aws/aws-sdk-go-v2/aws/middleware"
	smithymiddle "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type nrMiddleware struct {
	txn *newrelic.Transaction
}

type endable interface{ End() }

// See https://aws.github.io/aws-sdk-go-v2/docs/middleware/ for a description of
// AWS SDK V2 middleware.
func (m nrMiddleware) deserializeMiddleware(stack *smithymiddle.Stack) error {
	return stack.Deserialize.Add(smithymiddle.DeserializeMiddlewareFunc("NRDeserializeMiddleware", func(
		ctx context.Context, in smithymiddle.DeserializeInput, next smithymiddle.DeserializeHandler) (
		out smithymiddle.DeserializeOutput, metadata smithymiddle.Metadata, err error) {

		txn := m.txn
		if txn == nil {
			txn = newrelic.FromContext(ctx)
		}

		smithyRequest := in.Request.(*smithyhttp.Request)

		// The actual http.Request is inside the smithyhttp.Request
		httpRequest := smithyRequest.Request
		serviceName := awsmiddle.GetServiceID(ctx)
		operation := awsmiddle.GetOperationName(ctx)
		region := awsmiddle.GetRegion(ctx)

		var segment endable
		// Service name capitalization is different for v1 and v2.
		if serviceName == "dynamodb" || serviceName == "DynamoDB" {
			segment = &newrelic.DatastoreSegment{
				Product:            newrelic.DatastoreDynamoDB,
				Collection:         "", // AWS SDK V2 doesn't expose TableName
				Operation:          operation,
				ParameterizedQuery: "",
				QueryParameters:    nil,
				Host:               httpRequest.URL.Host,
				PortPathOrID:       httpRequest.URL.Port(),
				DatabaseName:       "",
				StartTime:          txn.StartSegmentNow(),
			}
		} else {
			segment = newrelic.StartExternalSegment(txn, httpRequest)
		}

		// Hand off execution to other middlewares and then perform the request
		out, metadata, err = next.HandleDeserialize(ctx, in)

		// After the request
		response, ok := out.RawResponse.(*smithyhttp.Response)

		if ok {
			// Set additional span attributes
			integrationsupport.AddAgentSpanAttribute(txn,
				newrelic.AttributeResponseCode, strconv.Itoa(response.StatusCode))
			integrationsupport.AddAgentSpanAttribute(txn,
				newrelic.SpanAttributeAWSOperation, operation)
			integrationsupport.AddAgentSpanAttribute(txn,
				newrelic.SpanAttributeAWSRegion, region)
			requestID, ok := awsmiddle.GetRequestIDMetadata(metadata)
			if ok {
				integrationsupport.AddAgentSpanAttribute(txn,
					newrelic.AttributeAWSRequestID, requestID)
			}
		}
		segment.End()
		return out, metadata, err
	}),
		smithymiddle.Before)
}

// AppendMiddlewares inserts New Relic middleware in the given `apiOptions` for
// the AWS SDK V2 for Go. It must be called only once per AWS configuration.
//
// If `txn` is provided as nil, the New Relic transaction will be retrieved
// using `newrelic.FromContext`.
//
// Additional attributes will be added to transaction trace segments and span
// events: aws.region, aws.requestId, and aws.operation. In addition,
// http.statusCode will be added to span events.
//
// To see segments and spans for all AWS invocations, call AppendMiddlewares
// with the AWS Config `apiOptions` and provide nil for `txn`. For example:
//
//  awsConfig, err := config.LoadDefaultConfig(ctx)
//  if err != nil {
//      log.Fatal(err)
//  }
//  nraws.AppendMiddlewares(&awsConfig.APIOptions, nil)
//
// If do not want the transaction to be retrived from the context, you can
// explicitly set `txn`. For example:
//
//  awsConfig, err := config.LoadDefaultConfig(ctx)
//  if err != nil {
//      log.Fatal(err)
//  }
//
//  ...
//
//  txn := loadNewRelicTransaction()
//  nraws.AppendMiddlewares(&awsConfig.APIOptions, txn)
func AppendMiddlewares(apiOptions *[]func(*smithymiddle.Stack) error, txn *newrelic.Transaction) {
	m := nrMiddleware{txn: txn}
	*apiOptions = append(*apiOptions, m.deserializeMiddleware)
}
