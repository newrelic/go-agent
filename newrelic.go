// Package newrelic performs application performance monitoring.
package newrelic

import (
	"github.com/newrelic/go-agent/api"
	"github.com/newrelic/go-agent/internal"
)

// NewConfig creates an api.Config populated with the given appname, license,
// and expected default values.  For descriptions of the fields, see
// api/config.go
func NewConfig(appname, license string) api.Config {
	return api.NewConfig(appname, license)
}

// NewApplication creates an Application and spawns goroutines to manage the
// aggregation and harvesting of data.  On success, a non-nil Application and a
// nil error will be returned. On failure, a nil Application and a non-nil error
// will be returned.  This function will fail if the config's Validate method
// returns an error.
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

// Application is described in api/application.go
type Application api.Application

// Transaction is described in api/transaction.go
type Transaction api.Transaction
