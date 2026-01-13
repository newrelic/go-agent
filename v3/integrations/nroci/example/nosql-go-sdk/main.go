package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nroci"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/oracle/nosql-go-sdk/nosqldb"
	"github.com/oracle/nosql-go-sdk/nosqldb/auth/iam"
	"github.com/oracle/nosql-go-sdk/nosqldb/types"
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
	// 1. Create Config Wrapper
	cfgWrapper, err := nroci.NRConfig("cloud") // create config wrapper
	if err != nil {
		panic(err)
	}

	// 2. Get new SignatureProvider.  Function automatically sets signatureProvider in configWrapper
	provider, err := iam.NewSignatureProviderFromFile("<insert-configfile-path>", "<insert-profile-name>", "<insert-private-key-passphrase>", "<insert-compartment-id>")
	if err != nil {
		panic(err)
	}

	cfg := nosqldb.Config{
		Mode:                  "cloud",
		AuthorizationProvider: provider,
		Region:                "<insert-region>",
	}
	cfgWrapper.Config = &cfg

	// EXAMPLE for cloudsim
	// csp := cloudsim.AccessTokenProvider{TenantID: "<replace-tenant-id>"} // Get access token for cloudsim
	// cfg := &nosqldb.Config{
	// 	Mode: "cloudsim",
	// }
	// cfgWrapper, err := nroci.NRConfig(cfg)
	// if err != nil {
	// 	panic(err)
	// }

	clientWrapper, err := nroci.NRCreateClient(cfgWrapper)
	if err != nil {
		panic(err)
	}

	defer clientWrapper.Client.Close()

	tableName := "audienceData"
	val := map[string]interface{}{
		"cookie_id": 123,
		"audience_data": map[string]interface{}{
			"ipaddr": "10.0.0.3",
			"audience_segment": map[string]interface{}{
				"sports_lover": "2018-11-30",
				"book_reader":  "2018-12-01",
			},
		},
	}
	putReq := &nosqldb.PutRequest{
		TableName: tableName,
		Value:     types.NewMapValue(val),
	}
	queryReq := &nosqldb.QueryRequest{
		Statement: "SELECT * FROM audienceData1",
	}

	txn := app.StartTransaction("OCI NoSQL Transaction")

	ctx := newrelic.NewContext(context.Background(), txn)

	stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ("+
		"cookie_id LONG, "+
		"audience_data JSON, "+
		"PRIMARY KEY(cookie_id))",
		tableName)
	tableReq := &nosqldb.TableRequest{
		Statement: stmt,
		TableLimits: &nosqldb.TableLimits{
			ReadUnits:  50,
			WriteUnits: 50,
			StorageGB:  1,
		},
	}
	_, err = nroci.NRDoTableRequestAndWait(clientWrapper, ctx, tableReq, time.Second*10, time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println("Created table", tableName)

	_, err = nroci.NRPut(clientWrapper, ctx, putReq)
	if err != nil {
		panic(err)
	}
	fmt.Println("Wrote row to table")

	queryRes, err := nroci.NRQuery(clientWrapper, ctx, queryReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Query Result: %v\n", queryRes.ClientResponse)

	txn.End()
}
