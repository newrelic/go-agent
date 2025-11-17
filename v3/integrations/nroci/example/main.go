package main

import (
	"context"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nroci"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/oracle/nosql-go-sdk/nosqldb"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Basic NOSQLOCI App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		panic(err)
	}
	app.WaitForConnection(10 * time.Second)
	defer app.Shutdown(10 * time.Second)

	cfg := nroci.NRDefaultConfig()

	clientWrapper, err := nroci.NRCreateClient(cfg)
	if err != nil {
		panic(err)
	}
	defer clientWrapper.Client.Close()

	txn := app.StartTransaction("OCI NoSQL Transaction")

	ctx := newrelic.NewContext(context.Background(), txn)

	_, err = nroci.NRDoTableRequest(clientWrapper, ctx, &nosqldb.TableRequest{})
	if err != nil {
		panic(err)
	}
	txn.End()
}
