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

// NewApplication will return an Application on success, or nil and an error on
// failure.  Failure will occur if the config's Validate method returns
// false.  If this function is successful, goroutines will be spawned to manage
// the aggregation and harvesting of data.
//
// Applications returned by this function do not not use any global state (other
// than the shared log.Logger).  Therefore, it is safe to create multiple
// applications.
//
// The config is passed by value but it contains reference type fields (such as
// Labels).  These fields must not be modified during the NewApplication call.
func NewApplication(c api.Config) (Application, error) {
	return internal.NewApp(c)
}
