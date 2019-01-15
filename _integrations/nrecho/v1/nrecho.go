package nrecho

import (
	"github.com/labstack/echo"
	newrelic "github.com/newrelic/go-agent"
)

const echoContextKey = "newRelicTransaction"

// FromContext returns the Transaction from the context if present, and nil
// otherwise.
func FromContext(c echo.Context) newrelic.Transaction {
	txn, _ := c.Get(echoContextKey).(newrelic.Transaction)
	return txn
}

// Middleware creates Echo middleware that instruments requests.
//
//  e := echo.New()
//  e.Use(nrecho.Middleware(app))
//
func Middleware(app newrelic.Application) func(echo.HandlerFunc) echo.HandlerFunc {

	originalNotFoundHandler := echo.NotFoundHandler
	echo.NotFoundHandler = func(c echo.Context) error {
		txn := FromContext(c)
		if nil != txn {
			// Ignore all transactions that would hit the echo.NotFoundHandler
			// in order to prevent metric grouping issues.
			txn.Ignore()
		}
		return originalNotFoundHandler(c)
	}

	if nil == app {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			txn := app.StartTransaction(c.Path(), c.Response().Writer, c.Request())
			defer txn.End()

			c.Response().Writer = txn
			c.Set(echoContextKey, txn)

			err := next(c)
			if nil != err {
				txn.NoticeError(err)
			}

			return err
		}
	}
}
