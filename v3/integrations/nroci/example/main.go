package main

import (
	"context"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nroci"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/oracle/nosql-go-sdk/nosqldb"
	"github.com/oracle/nosql-go-sdk/nosqldb/auth/iam"
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
	app.WaitForConnection(10 * time.Second) // for short lived processes in apps
	defer app.Shutdown(10 * time.Second)

	sp, err := iam.NewSignatureProviderFromFile("", "", "", "")
	if err != nil {
		panic(err)
	}
	cfg := &nosqldb.Config{
		Mode:                  "cloud",
		AuthorizationProvider: sp,
	}
	cfgWrapper, err := nroci.NRConfigCloud(cfg, sp, "") // should work for cloud
	if err != nil {
		panic(err)
	}
	// cfgWrapper := nrociNRConfigCloudSim()

	clientWrapper, err := nroci.NRCreateClient(cfgWrapper)
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
