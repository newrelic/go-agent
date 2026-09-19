module github.com/newrelic/go-agent/v3/integrations/nrgorm

go 1.21

require (
	github.com/newrelic/go-agent/v3 v3.36.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

replace github.com/newrelic/go-agent/v3 => ../..
