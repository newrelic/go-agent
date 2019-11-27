module github.com/newrelic/go-agent/v3/integrations/nrmongo

go 1.13

require (
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/newrelic/go-agent/v3 v3.0.0
	// mongo-driver does not support modules as of Nov 2019.
	go.mongodb.org/mongo-driver v1.0.0
)
