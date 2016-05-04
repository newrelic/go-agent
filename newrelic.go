package newrelic

import (
	"github.com/newrelic/go-sdk/api"
	"github.com/newrelic/go-sdk/internal"
)

func NewConfig(appname, license string) api.Config {
	return api.NewConfig(appname, license)
}

type Application api.Application
type Transaction api.Transaction

func NewApplication(c api.Config) (Application, error) {
	return internal.NewApp(c)
}
