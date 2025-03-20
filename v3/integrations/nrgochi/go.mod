module github.com/newrelic/go-agent/v3/integrations/nrgochi

go 1.22

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/newrelic/go-agent/v3 v3.38.0
)

replace github.com/newrelic/go-agent/v3 => ../..
