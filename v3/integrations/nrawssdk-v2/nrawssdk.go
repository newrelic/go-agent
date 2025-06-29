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
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddle "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/middleware"
	smithymiddle "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

type nrMiddleware struct {
	txn *newrelic.Transaction
}

type contextKey string

const (
	dynamodbInputKey contextKey = "DynamoDBInput"
	queueURLKey      contextKey = "QueueURL"
)

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

		if serviceName == "dynamodb" || serviceName == "DynamoDB" {
			input, _ := ctx.Value(dynamodbInputKey).(dynamodbInput)
			collection := input.tableName
			if input.indexName != "" {
				collection += "." + input.indexName
			}

			segment = &newrelic.DatastoreSegment{
				Product:            newrelic.DatastoreDynamoDB,
				Collection:         collection,
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
			if serviceName == "sqs" || serviceName == "SQS" {
				if queueURL, ok := ctx.Value(queueURLKey).(string); ok {
					parsedURL, err := url.Parse(queueURL)
					if err == nil {
						// Example URL: https://sqs.{region}.amazonaws.com/{account.id}/{queue.name}
						pathParts := strings.Split(parsedURL.Path, "/")
						if len(pathParts) >= 3 {
							accountID := pathParts[1]
							queueName := pathParts[2]
							integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeCloudAccountID, accountID)
							integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeCloudRegion, region)
							integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeMessageSystem, "aws_sqs")
							integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeMessageDestinationName, queueName)
						}
					}

				}
			}
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
	}), smithymiddle.Before)
}

func (m nrMiddleware) serializeMiddleware(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("NRSerializeMiddleware", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

		serviceName := awsmiddle.GetServiceID(ctx)
		switch serviceName {
		case "dynamodb", "DynamoDB":
			ctx = context.WithValue(ctx, dynamodbInputKey, dynamoDBInputFromMiddlewareInput(in))
		case "sqs", "SQS":
			ctx = context.WithValue(ctx, queueURLKey, sqsQueueURFromMiddlewareInput(in))
		}
		return next.HandleInitialize(ctx, in)
	}), middleware.After)
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
//	awsConfig, err := config.LoadDefaultConfig(ctx, func(o *config.LoadOptions) error {
//		// Instrument all new AWS clients with New Relic
//		nrawssdk.AppendMiddlewares(&o.APIOptions, nil)
//		return nil
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
// If do not want the transaction to be retrieved from the context, you can
// explicitly set `txn`. For example:
//
//	txn := loadNewRelicTransaction()
//	awsConfig, err := config.LoadDefaultConfig(ctx, func(o *config.LoadOptions) error {
//		// Instrument all new AWS clients with New Relic
//		nrawssdk.AppendMiddlewares(&o.APIOptions, txn)
//		return nil
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
// The middleware can also be added later, per AWS service call using
// the `optFns` parameter. For example:
//
//	awsConfig, err := config.LoadDefaultConfig(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	...
//
//	s3Client := s3.NewFromConfig(awsConfig)
//
//	...
//
//	txn := loadNewRelicTransaction()
//	output, err := s3Client.ListBuckets(ctx, nil, func(o *s3.Options) error {
//		nrawssdk.AppendMiddlewares(&o.APIOptions, txn)
//		return nil
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
func AppendMiddlewares(apiOptions *[]func(*smithymiddle.Stack) error, txn *newrelic.Transaction) {
	m := nrMiddleware{txn: txn}
	*apiOptions = append(*apiOptions, m.deserializeMiddleware)
	*apiOptions = append(*apiOptions, m.serializeMiddleware)
}

func sqsQueueURFromMiddlewareInput(in middleware.InitializeInput) string {
	switch params := in.Parameters.(type) {
	case *sqs.SendMessageInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.DeleteQueueInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.ReceiveMessageInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.DeleteMessageInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.ChangeMessageVisibilityInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.ChangeMessageVisibilityBatchInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.DeleteMessageBatchInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.SendMessageBatchInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.PurgeQueueInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.GetQueueAttributesInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.SetQueueAttributesInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.TagQueueInput:
		return aws.ToString(params.QueueUrl)
	case *sqs.UntagQueueInput:
		return aws.ToString(params.QueueUrl)
	default:
		return ""
	}
}

type dynamodbInput struct {
	tableName string
	indexName string
}

func dynamoDBInputFromMiddlewareInput(in middleware.InitializeInput) dynamodbInput {
	switch params := in.Parameters.(type) {
	case *dynamodb.DeleteItemInput:
		return dynamodbInput{tableName: aws.ToString(params.TableName)}
	case *dynamodb.GetItemInput:
		return dynamodbInput{tableName: aws.ToString(params.TableName)}
	case *dynamodb.PutItemInput:
		return dynamodbInput{tableName: aws.ToString(params.TableName)}
	case *dynamodb.QueryInput:
		return dynamodbInput{tableName: aws.ToString(params.TableName), indexName: aws.ToString(params.IndexName)}
	case *dynamodb.ScanInput:
		return dynamodbInput{tableName: aws.ToString(params.TableName), indexName: aws.ToString(params.IndexName)}
	case *dynamodb.UpdateItemInput:
		return dynamodbInput{tableName: aws.ToString(params.TableName)}
	default:
		return dynamodbInput{}
	}
}
