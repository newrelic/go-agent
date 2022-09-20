module github.com/newrelic/go-agent/v3/integrations/nrmongo

// As of Dec 2019, 1.10 is the mongo-driver requirement:
// https://github.com/mongodb/mongo-go-driver#requirements
go 1.17

require (
	github.com/newrelic/go-agent/v3 v3.18.2
	// mongo-driver does not support modules as of Nov 2019.
	go.mongodb.org/mongo-driver v1.10.2
)
