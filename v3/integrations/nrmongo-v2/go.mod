module github.com/newrelic/go-agent/v3/integrations/nrmongo-v2

// https://github.com/mongodb/mongo-go-driver#requirements
go 1.22

require (
	github.com/newrelic/go-agent/v3 v3.40.1
	// mongo-driver does not support modules as of Nov 2019.
	go.mongodb.org/mongo-driver/v2 v2.2.2
)

replace github.com/newrelic/go-agent/v3 => ../..
