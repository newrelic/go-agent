module github.com/newrelic/go-agent/v4/integrations/nrmongo

// As of Dec 2019, 1.10 is the mongo-driver requirement:
// https://github.com/mongodb/mongo-go-driver#requirements
go 1.10

require (
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/newrelic/go-agent/v4 v4.0.0
	// mongo-driver does not support modules as of Nov 2019.
	go.mongodb.org/mongo-driver v1.0.0
)
