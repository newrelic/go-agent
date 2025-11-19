module github.com/newrelic/go-agent/v3/integrations/nrmongo

// As of Dec 2019, 1.10 is the mongo-driver requirement:
// https://github.com/mongodb/mongo-go-driver#requirements
go 1.24

require (
	github.com/newrelic/go-agent/v3 v3.42.0
	// mongo-driver does not support modules as of Nov 2019.
	go.mongodb.org/mongo-driver v1.17.4
)

replace github.com/newrelic/go-agent/v3 => ../..
