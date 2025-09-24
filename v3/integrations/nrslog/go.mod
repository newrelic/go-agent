module github.com/newrelic/go-agent/v3/integrations/nrslog

// The new log/slog package in Go 1.21 brings structured logging to the standard library.
go 1.24

require (
	github.com/newrelic/go-agent/v3 v3.41.0
	github.com/stretchr/testify v1.9.0
)

replace github.com/newrelic/go-agent/v3 => ../..
