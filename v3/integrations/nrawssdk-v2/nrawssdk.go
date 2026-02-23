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
	"encoding/base32"
	"fmt"
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

type credentialsResolver interface {
	AWSAccountIdFromAWSAccessKey(creds aws.Credentials) (string, error)
}

type nrMiddleware struct {
	txn       *newrelic.Transaction
	accountID string
	resolver  credentialsResolver
}

type defaultResolver struct{}

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
		accountID := m.accountID
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
			if serviceName == "OpenSearch" || serviceName == "opensearch" {
				integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeAWSElastSearchDomainEndpoint, httpRequest.URL.String()) // this way I don't have to pull it out of context
			}
			// Set additional span attributes
			integrationsupport.AddAgentSpanAttribute(txn, newrelic.AttributeCloudAccountID, accountID) // setting account ID here, why do we only do this if it is an SQS service?

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

// Deprecated: Use InitializeMiddleware instead.  Please note the different parameters used.
func AppendMiddlewares(apiOptions *[]func(*smithymiddle.Stack) error, txn *newrelic.Transaction) {
	m := nrMiddleware{txn: txn}
	*apiOptions = append(*apiOptions, m.deserializeMiddleware)
	*apiOptions = append(*apiOptions, m.serializeMiddleware)
}

// InitializeMiddleware registers New Relic middleware in the AWS SDK V2 for Go service stack.
// It must be called only once per AWS configuration.
//
// The New Relic transaction, `txn`, is fetched from the ctx.  Make sure to add the txn to the ctx
// before passing it in in as a parameter. This can be done with:
// ctx := newrelic.NewContext(context.Background(), txn)
//
// Additional attributes will be added to transaction trace segments and span
// events: aws.accountId, aws.region, aws.requestId, and aws.operation. In addition,
// http.statusCode will be added to span events.
//
// To see segments and spans for all AWS invocations, call InitializeMiddleware
// with the aws.Config and provide ctx. For example:
//
//	awsConfig, err := config.LoadDefaultConfig(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	nrawssdk.InitializeMiddleware(awsConfig, nil, awsConfig.Credentials)
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
//		nrawssdk.AppendMiddlewares(&o.APIOptions, txn, o.Credentials)
//		return nil
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
func InitializeMiddleware(apiOptions *[]func(*smithymiddle.Stack) error, ctx context.Context, credentials aws.CredentialsProvider) {
	txn := newrelic.FromContext(ctx)

	m := nrMiddleware{txn: txn, resolver: &defaultResolver{}}

	m.initializeMiddleware(apiOptions, ctx, credentials)
}

func (m *nrMiddleware) initializeMiddleware(apiOptions *[]func(*smithymiddle.Stack) error, ctx context.Context, credentials aws.CredentialsProvider) {

	if m.txn == nil {
		m.txn = newrelic.FromContext(ctx)
	}
	cfg, ok := m.txn.Application().Config()
	// if the nr config has not been intialized yet, don't try to resolve the credentials
	if ok {
		creds, err := credentials.Retrieve(ctx)
		if err != nil {
			cfg.Logger.Error(err.Error(), map[string]any{})
		}

		err = m.ResolveAWSCredentials(cfg, creds)
		if err != nil {
			cfg.Logger.Error(err.Error(), map[string]any{})
		}
	}

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

func (m *nrMiddleware) ResolveAWSCredentials(cfg newrelic.Config, creds aws.Credentials) error {

	// use cfg accountID or use cfg accountID if account decoding is disabled
	if cfg.CloudAWS.AccountID != "" || !cfg.CloudAWS.AccountDecoding.Enabled {
		m.accountID = cfg.CloudAWS.AccountID
		return nil
	}

	if m.resolver == nil {
		m.resolver = &defaultResolver{}
	}
	accountID, err := m.resolver.AWSAccountIdFromAWSAccessKey(creds)
	if err != nil {
		// return err, aws account id remains empty
		return err
	}

	// Otherwise use the resolved accountID
	m.accountID = accountID
	return nil
}

func (m *defaultResolver) AWSAccountIdFromAWSAccessKey(creds aws.Credentials) (string, error) {
	if creds.AccountID != "" {
		return creds.AccountID, nil
	}
	if creds.AccessKeyID == "" {
		return "", fmt.Errorf("no access key id found")
	}
	if len(creds.AccessKeyID) < 16 {
		return "", fmt.Errorf("improper access key id format")
	}
	trimmedAccessKey := creds.AccessKeyID[4:]
	decoded, err := base32.StdEncoding.DecodeString(trimmedAccessKey)
	if err != nil {
		return "", fmt.Errorf("error decoding access keys")
	}
	var bigEndian uint64
	for i := 0; i < 6; i++ {
		bigEndian = bigEndian << 8      // shift 8 bits left.  Most significant byte read in first (decoded[i])
		bigEndian |= uint64(decoded[i]) // apply OR for current byte
	}

	mask := uint64(0x7fffffffff80)

	num := (bigEndian & mask) >> 7 // apply mask and get rid of last 7 bytes from mask

	return fmt.Sprintf("%d", num), nil
}
