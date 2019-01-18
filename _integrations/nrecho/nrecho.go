// Package nrecho introduces to support for the echo framework.
//
// Example: https://github.com/newrelic/go-agent/tree/master/_integrations/nrecho/example/main.go
package nrecho

import (
	"reflect"

	"github.com/labstack/echo"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func init() { internal.TrackUsage("integration", "framework", "echo") }

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
			txn := app.StartTransaction(transactionName(c), c.Response().Writer, c.Request())
			defer txn.End()

			c.Response().Writer = txn

			// Add txn to c.Request().Context()
			c.SetRequest(c.Request().WithContext(newrelic.NewContext(c.Request().Context(), txn)))

			err = next(c)

			if nil == err {
				return
			}

			if httperr, ok := err.(*echo.HTTPError); ok {
				if 404 == httperr.Code {
					return
				}
			}

			txn.NoticeError(err)
			return
		}
	}
}
