package main_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/newrelic/go-agent/v3/integrations/nrfiber-next"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
)

func setupTestApp(t *testing.T) *fiber.App {
	app := integrationsupport.NewBasicTestApp().Application
	if app == nil {
		t.Fatal("Failed to create New Relic application")
	}
	// Create a new Fiber app
	router := fiber.New()
	router.Use(nrfiber.Middleware(app))
	return router
}

func TestRoutes404(t *testing.T) {
	app := setupTestApp(t)
	app.Get("/404", func(c fiber.Ctx) error {
		c.SendStatus(404)
		return c.SendString("returning 404")
	})

	req := httptest.NewRequest("GET", "/404", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "returning 404" {
		t.Errorf("Expected body 'returning 404', got '%s'", string(body))
	}
}

func TestRouteStatusChange(t *testing.T) {
	app := setupTestApp(t)
	app.Get("/change", func(c fiber.Ctx) error {
		c.SendStatus(404)
		c.SendStatus(200)
		return c.SendString("actually ok!")
	})

	req := httptest.NewRequest("GET", "/change", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected final status code 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "actually ok!" {
		t.Errorf("Expected body 'actually ok!', got '%s'", string(body))
	}
}

func TestCustomHeaders(t *testing.T) {
	app := setupTestApp(t)
	app.Get("/headers", func(c fiber.Ctx) error {
		c.SendStatus(200)
		c.Response().Header.Set("X-Custom", "custom value")
		return c.SendString(`{"zip":"zap"}`)
	})

	req := httptest.NewRequest("GET", "/headers", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Header.Get("X-Custom") != "custom value" {
		t.Errorf("Expected X-Custom header 'custom value', got '%s'", resp.Header.Get("X-Custom"))
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"zip":"zap"}` {
		t.Errorf("Expected body '{\"zip\":\"zap\"}', got '%s'", string(body))
	}
}

func TestV1GroupRoutes(t *testing.T) {
	app := setupTestApp(t)

	v1 := app.Group("/v1")
	v1.Get("/login", func(c fiber.Ctx) error {
		return c.SendString("login")
	})
	v1.Get("/submit", func(c fiber.Ctx) error {
		return c.SendString("submit")
	})
	v1.Get("/read", func(c fiber.Ctx) error {
		return c.SendString("read")
	})

	paths := []string{"/v1/login", "/v1/submit", "/v1/read"}
	expected := []string{"login", "submit", "read"}

	for i, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status code 200 for %s, got %d", path, resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != expected[i] {
			t.Errorf("Expected body '%s' for %s, got '%s'", expected[i], path, string(body))
		}
	}
}
