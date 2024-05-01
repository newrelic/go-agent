module github.com/newrelic/go-agent/v3/integrations/nrsecurityagent

go 1.20

require (
	github.com/newrelic/csec-go-agent v1.2.0
	github.com/newrelic/go-agent/v3 v3.32.0
	github.com/newrelic/go-agent/v3/integrations/nrsqlite3 v1.2.0
	gopkg.in/yaml.v2 v2.4.0
)


replace github.com/newrelic/go-agent/v3 => ../..
