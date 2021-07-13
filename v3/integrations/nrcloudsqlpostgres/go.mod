module github.com/newrelic/go-agent/v3/integrations/nrcloudsqlpostgress

go 1.13

require (
	github.com/GoogleCloudPlatform/cloudsql-proxy v1.19.1
	// v3.3.0 includes the new location of ParseQuery
	github.com/newrelic/go-agent/v3 v3.3.0
)
