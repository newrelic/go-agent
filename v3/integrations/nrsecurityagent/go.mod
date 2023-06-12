module github.com/newrelic/go-agent/v3/integrations/nrsecurityagent

go 1.18

require (
	github.com/newrelic/csec-go-agent v0.2.1
	github.com/newrelic/go-agent/v3 v3.22.0
	github.com/newrelic/go-agent/v3/integrations/nrsqlite3 v1.1.1
	gopkg.in/yaml.v2 v2.4.0
)

replace (
	github.com/newrelic/go-agent/v3 v3.22.0 => ../..
	github.com/newrelic/go-agent/v3/integrations/nrsqlite3 => ../nrsqlite3
)
