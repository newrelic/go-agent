// Package nrecho introduces to support for the echo framework.
//
// Example: https://github.com/newrelic/go-agent/tree/master/_integrations/nrecho/v4/example/main.go
package nrecho

import (
	"net/http"
	"reflect"

	"github.com/labstack/echo/v4"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func init() { internal.TrackUsage("integration", "framework", "echo", "v4") }

// FromContext returns the Transaction from the context if present, and nil
// otherwise.
func FromContext(c echo.Context) newrelic.Transaction {
	return newrelic.FromContext(c.Request().Context())
}

func handlerPointer(handler echo.HandlerFunc) uintptr {
	return reflect.ValueOf(handler).Pointer()
}

func transactionName(c echo.Context) string {
	ptr := handlerPointer(c.Handler())
	if ptr == handlerPointer(echo.NotFoundHandler) {
		return "NotFoundHandler"
	}
	if ptr == handlerPointer(echo.MethodNotAllowedHandler) {
		return "MethodNotAllowedHandler"
	}
	return c.Path()
}

// Middleware creates Echo middleware that instruments requests.
//
//  e := echo.New()
//  e.Use(nrecho.Middleware(app))
//
func Middleware(app newrelic.Application) func(echo.HandlerFunc) echo.HandlerFunc {

	if nil == app {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			rw := c.Response().Writer
			txn := app.StartTransaction(transactionName(c), rw, c.Request())
			defer txn.End()

			c.Response().Writer = txn

			// Add txn to c.Request().Context()
			c.SetRequest(c.Request().WithContext(newrelic.NewContext(c.Request().Context(), txn)))

			err = next(c)

			// Record the response code. The response headers are not captured
			// in this case because they are set after this middleware returns.
			// Designed to mimic the logic in echo.DefaultHTTPErrorHandler.
			if nil != err && !c.Response().Committed {

				txn.SetWebResponse(nil)
				c.Response().Writer = rw

				if httperr, ok := err.(*echo.HTTPError); ok {
					txn.WriteHeader(httperr.Code)
				} else {
					txn.WriteHeader(http.StatusInternalServerError)
				}
			}

			return
		}
	}
}
