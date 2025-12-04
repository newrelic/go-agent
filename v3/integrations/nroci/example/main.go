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
	app.WaitForConnection(10 * time.Second) // for short lived processes in apps
	defer app.Shutdown(10 * time.Second)

	// EXAMPLE for cloud
	cfg := &nosqldb.Config{
		Mode: "cloud",
	}
	// 1. Create Config Wrapper
	cfgWrapper, err := nroci.NRConfig(cfg) // create config wrapper
	if err != nil {
		panic(err)
	}

	// 2. Get new SignatureProvider.  Function automatically sets signatureProvider in configWrapper
	_, err = nroci.NRNewSignatureProviderFromFile(cfgWrapper, "", "", "", "")
	if err != nil {
		panic(err)
	}

	// EXAMPLE for cloudsim
	// csp := cloudsim.AccessTokenProvider{TenantID: "<replace-tenant-id>"} // Get access token for cloudsim
	// cfg := &nosqldb.Config{
	// 	Mode: "cloudsim",
	// }
	// cfgWrapper, err := nroci.NRConfig(cfg)
	// if err != nil {
	// 	panic(err)
	// }

	// cfgWrapper.CompartmentID = csp.TenantID // Set tenant ID based on what comes back

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
