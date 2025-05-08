package main

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/newrelic/go-agent/v3/integrations/nrfiber"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func v1login(c *fiber.Ctx)  { c.WriteString("v1 login") }
func v1submit(c *fiber.Ctx) { c.WriteString("v1 submit") }
func v1read(c *fiber.Ctx)   { c.WriteString("v1 read") }

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("fiber App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	router := fiber.New()
	router.Use(nrfiber.Middleware(app))

	// 404 handler
	router.Get("/404", func(c *fiber.Ctx) error {
		c.SendStatus(404)
		c.WriteString("returning 404")
		return nil
	})

	//
	router.Get("/change", func(c *fiber.Ctx) error {
		c.SendStatus(404)
		c.SendStatus(200)
		c.WriteString("actually ok!")
		return nil
	})

	// Headers
	router.Get("/headers", func(c *fiber.Ctx) error {
		// Since fiber.Response buffers the response code, response headers
		// can be set afterwards.
		c.SendStatus(200)
		c.Response().Header.Set("X-Custom", "custom value")
		c.SendString(`{"zip":"zap"}`)
		return nil
	})

	router.Get("/txn", func(c *fiber.Ctx) error {
		txn := nrfiber.Transaction(c.Context())
		txn.SetName("custom-name")
		c.WriteString("changed the name of the transaction!")
		return nil
	})

	// Since the handler function name is used as the transaction name,
	// anonymous functions do not get usefully named.  We encourage
	// transforming anonymous functions into named functions.
	router.Get("/anon", func(c *fiber.Ctx) error {
		return c.SendString("anonymous function handler")
	})

	v1 := router.Group("/v1")
	v1.Get("/login", func(c *fiber.Ctx) error {
		v1login(c)
		return nil
	})
	v1.Get("/submit", func(c *fiber.Ctx) error {
		v1submit(c)
		return nil
	})
	v1.Get("/read", func(c *fiber.Ctx) error {
		v1read(c)
		return nil
	})

	router.Listen(":8000")
}
