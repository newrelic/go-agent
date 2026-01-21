package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nroci"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/nosql"
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

	usr, _ := user.Current()
	configPath := filepath.Join(usr.HomeDir, ".oci/config")
	configProvider, err := common.ConfigurationProviderFromFile(configPath, "")

	configWrapper, err := nroci.NRNewNoSQLClientWithConfigurationProvider(configProvider)
	if err != nil {
		panic(err)
	}

	txn := app.StartTransaction("OCI NoSQL Transaction")

	ctx := newrelic.NewContext(context.Background(), txn)

	compartmentID := os.Getenv("COMPARTMENT_OCID")
	statement := "SELECT * FROM audienceData1"
	tableName := "audienceData1"

	putReq := nosql.UpdateRowRequest{
		TableNameOrId: &tableName,
		UpdateRowDetails: nosql.UpdateRowDetails{
			Value: map[string]interface{}{
				"cookie_id": 123,
				"audience_data": map[string]interface{}{
					"ipaddr": "10.0.0.3",
					"audience_segment": map[string]interface{}{
						"sports_lover": "2018-11-30",
						"book_reader":  "2018-12-01",
					},
				},
			},
			CompartmentId: &compartmentID,
		},
	}
	queryReq := nosql.QueryRequest{
		QueryDetails: nosql.QueryDetails{
			CompartmentId: &compartmentID,
			Statement:     &statement,
		},
	}
	putRes, err := configWrapper.UpdateRow(ctx, putReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("UpdateRow row: %v\nresult\n", putRes)

	queryRes, err := configWrapper.Query(ctx, queryReq)

	if err != nil {
		panic(err)
	}
	fmt.Printf("QueryRow row: %v\nresult\n", queryRes)

	txn.End()
}
