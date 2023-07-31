module github.com/newrelic/go-agent/v3/integrations/nrgorilla

// As of Dec 2019, the gorilla/mux go.mod file uses 1.12:
// https://github.com/gorilla/mux/blob/master/go.mod
go 1.12

require (
	// v1.7.0 is the earliest version of Gorilla using modules.
	github.com/gorilla/mux v1.7.0
	github.com/newrelic/go-agent/v3 v3.17.0
)
