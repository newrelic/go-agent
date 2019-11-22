package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/newrelic/go-agent/v3/integrations/nrecho-v4"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func getUser(c echo.Context) error {
	id := c.Param("id")

	if txn := nrecho.FromContext(c); nil != txn {
		txn.AddAttribute("userId", id)
	}

	return c.String(http.StatusOK, id)
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Echo App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// Echo instance
	e := echo.New()

	// The New Relic Middleware should be the first middleware registered
	e.Use(nrecho.Middleware(app))

	// Routes
	e.GET("/home", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	// Groups
	g := e.Group("/user")
	g.Use(middleware.Gzip())
	g.GET("/:id", getUser)

	// Start server
	e.Start(":8000")
}
