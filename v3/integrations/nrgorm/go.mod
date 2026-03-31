module github.com/newrelic/go-agent/v3/integrations/nrgorm

go 1.21

require (
	gorm.io/gorm v1.25.12
	github.com/newrelic/go-agent/v3 v3.36.0
)


replace github.com/newrelic/go-agent/v3 => ../..
