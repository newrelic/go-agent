package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/newrelic/go-agent/v3/integrations/nrawssdk-v2"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {

	// Create a New Relic application. This will look for your license key in an
	// environment variable called NEW_RELIC_LICENSE_KEY. This example turns on
	// Distributed Tracing, but that's not required.
	app, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
		// newrelic.ConfigCloudAWSAccountID("<insert-aws-account-id>"), // Set the AWS accountID.  Will override any previous config options
		// newrelic.ConfigCloudAWSAccountDecodingEnabled(false), // Enable/disable accountID decoding. Default is enabled
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// For demo purposes only. Don't use the app.WaitForConnection call in
	// production unless this is a very short-lived process and the caller
	// doesn't block or exit if there's an error.
	app.WaitForConnection(5 * time.Second)

	// Start recording a New Relic transaction
	txn := app.StartTransaction("My sample transaction")

	ctx := newrelic.NewContext(context.Background(), txn)

	// We need the values in the aws.Config for InitializeMiddleware, so
	// there is no longer a need for a config option function.
	awsConfig, err := config.LoadDefaultConfig(ctx)

	// nrawssdk.AppendMiddlewares(&awsConfig.APIOptions, txn) // LEGACY
	// AppendMiddlewares is DEPRECATED. Please use InitializeMiddleware shown below
	nrawssdk.InitializeMiddleware(&awsConfig.APIOptions, ctx, awsConfig.Credentials)
	if err != nil {
		log.Fatal(err)
	}

	s3Client := s3.NewFromConfig(awsConfig)
	// If you want to instrument per request pass InitializeMiddleware in an optional
	// function with the resolved awsConfig.Credentials
	// output, err := s3Client.ListBuckets(ctx, nil, func(o *s3.Options) {
	// 	nrawssdk.InitializeMiddleware(&o.APIOptions, ctx, awsConfig.Credentials)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// })
	output, err := s3Client.ListBuckets(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, object := range output.Buckets {
		log.Printf("Bucket name is %s\n", aws.ToString(object.Name))
	}

	// End the New Relic transaction
	txn.End()

	// Force all the harvests and shutdown. Like the app.WaitForConnection call
	// above, this is for the purposes of this demo only and can be safely
	// removed for longer-running processes.
	app.Shutdown(10 * time.Second)
}
