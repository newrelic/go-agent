package nrfiber

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// TestMiddleware_NoNewRelicApp ensures requests proceed normally if no New Relic app is provided
func TestMiddleware_NoNewRelicApp(t *testing.T) {
	fiberApp := fiber.New()
	fiberApp.Use(Middleware(nil))

	fiberApp.Get("/no-nr", func(c *fiber.Ctx) error {
		return c.SendString("No NR App")
	})

	// Simulate a request
	req, err := http.NewRequest("GET", "/no-nr", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Make a test request
	resp, err := fiberApp.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestMiddleware_Success checks if the middleware correctly creates a New Relic transaction
func TestMiddleware_Success(t *testing.T) {
	// Create a test New Relic application
	app := integrationsupport.NewBasicTestApp()

	// Initialize Fiber app with New Relic middleware
	fiberApp := fiber.New()
	fiberApp.Use(Middleware(app.Application))

	// Define a sample route
	fiberApp.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// Simulate a request
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Make a test request
	resp, err := fiberApp.Test(req, -1)

	// Verify that no error occurred
	require.Nil(t, err)

	// Read the response body
	body, _ := io.ReadAll(resp.Body)

	if respBody := string(body); respBody != "Hello, World!" {
		t.Error("wrong response body", respBody)
	}

	// Assertions
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Check if the transaction was created
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /test",
		IsWeb:         true,
		UnknownCaller: true,
	})
}

// TestMiddleware_AnonymousFunctions checks if the middleware correctly handles anonymous functions
func TestMiddleware_AnonymousFunctions(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	fiberApp := fiber.New()
	fiberApp.Use(Middleware(app.Application))
	fiberApp.Get("/helloAnon", func(c *fiber.Ctx) error {
		return c.SendString("Hello, anon!")
	})

	req, err := http.NewRequest("GET", "/helloAnon", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Make a test request
	resp, err := fiberApp.Test(req, -1)
	// Verify that no error occurred
	require.Nil(t, err)

	// Read the response body
	body, _ := io.ReadAll(resp.Body)

	if respBody := string(body); respBody != "Hello, anon!" {
		t.Error("wrong response body", respBody)
	}

	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /helloAnon",
		IsWeb:         true,
		UnknownCaller: true,
	})

}

// TestMiddleware_ErrorHandling checks if the middleware captures errors correctly
func TestMiddleware_ErrorHandling(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	fiberApp := fiber.New()
	fiberApp.Use(Middleware(app.Application))

	// Define a route that returns an error
	fiberApp.Get("/error", func(c *fiber.Ctx) error {
		return fiber.ErrInternalServerError
	})

	// Simulate a request
	req, err := http.NewRequest("GET", "/error", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Make a test request
	resp, err := fiberApp.Test(req, -1)

	// Verify that no error occurred
	require.Nil(t, err)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Ensure the error is noticed in New Relic
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/GET /error",
		Msg:     "Internal Server Error",
	}})
}

// TestWrapHandler verifies that WrapHandler correctly wraps an existing handler
func TestWrapHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	fiberApp := fiber.New()

	wrappedHandler := WrapHandler(app.Application, "/wrapped", func(c *fiber.Ctx) error {
		return c.SendString("Wrapped Handler")
	})

	fiberApp.Get("/wrapped", wrappedHandler)

	// Simulate a request
	req, err := http.NewRequest("GET", "/wrapped", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Make a test request
	resp, err := fiberApp.Test(req, -1)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Ensure the transaction was correctly named
	// Check if the transaction was created
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /wrapped",
		IsWeb:         true,
		UnknownCaller: true,
	})
}

// Test_GetTransactionName tests the getTransactionName function
func Test_GetTransactionName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		method   string
		expected string
	}{
		{
			name:     "Root path",
			path:     "/",
			method:   "GET",
			expected: "GET /",
		},
		{
			name:     "API path",
			path:     "/api/users",
			method:   "POST",
			expected: "POST /api/users",
		},
		{
			name:     "Empty path defaults to root",
			path:     "",
			method:   "DELETE",
			expected: "DELETE /",
		},
		{
			name:     "Path with query parameters",
			path:     "/search?q=fiber",
			method:   "GET",
			expected: "GET /search",
		},
		{
			name:     "Path with parameters",
			path:     "/products/:id",
			method:   "PUT",
			expected: "PUT /products/:id",
		},
		{
			name:     "Path with multiple parameters",
			path:     "/users/:id/posts/:postID",
			method:   "GET",
			expected: "GET /users/:id/posts/:postID",
		},
		{
			name:     "Path with trailing slash",
			path:     "/products/",
			method:   "PUT",
			expected: "PUT /products/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new Fiber app and request for testing
			app := fiber.New()

			fctx := &fasthttp.RequestCtx{}
			fctx.Request.Header.SetMethod(tt.method)
			fctx.Request.SetRequestURI(tt.path)

			ctx := app.AcquireCtx(fctx)
			ctx.Request().SetRequestURI(tt.path)
			ctx.Request().Header.SetMethod(tt.method)

			// Get the transaction name
			result := getTransactionName(ctx)

			// Verify the result
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_ConvertHeaderToHTTP tests the convertHeaderToHTTP function
func Test_ConvertHeaderToHTTP(t *testing.T) {
	// Create a new Fiber app and request for testing
	app := fiber.New()

	fctx := &fasthttp.RequestCtx{}
	ctx := app.AcquireCtx(fctx)

	// Set test headers
	ctx.Request().Header.Set("Content-Type", "application/json")
	ctx.Request().Header.Set("X-Custom-Header", "test-value")

	// Convert the headers
	headers := convertHeaderToHTTP(ctx)

	fmt.Println(headers.Get("Content-Type"))

	// Verify the headers were correctly converted
	assert.Equal(t, "application/json", headers.Get("Content-Type"))
	assert.Equal(t, "test-value", headers.Get("X-Custom-Header"))

	// Release the context when done
	app.ReleaseCtx(ctx)
}

// Test_HeaderResponseWriter tests the headerResponseWriter implementation
func Test_HeaderResponseWriter(t *testing.T) {
	// Create a new Fiber app and response for testing
	app := fiber.New()
	fctx := &fasthttp.RequestCtx{}
	ctx := app.AcquireCtx(fctx)

	// Create a headerResponseWriter using the Fiber response
	writer := &headerResponseWriter{fiberResponse: ctx.Response()}

	// Set some headers on the Fiber response
	ctx.Response().Header.Set("Content-Type", "application/json")
	ctx.Response().Header.Set("X-Test-Header", "test-value")

	// Get the headers via the wrapper
	headers := writer.Header()

	// Verify the headers were correctly converted
	assert.Equal(t, "application/json", headers.Get("Content-Type"))
	assert.Equal(t, "test-value", headers.Get("X-Test-Header"))

	// Test WriteHeader
	writer.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, ctx.Response().StatusCode())
}

// Test_RouterGroup tests the router group functionality
func Test_RouterGroup(t *testing.T) {
	// Create a new Fiber app and router group for testing
	app := integrationsupport.NewBasicTestApp()
	router := fiber.New()
	router.Use(Middleware(app.Application))
	group := router.Group("/group")
	group.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendString("hello response")
	})

	// Simulate a request
	req, err := http.NewRequest("GET", "/group/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Make a test request
	resp, err := router.Test(req, -1)

	// Verify that no error occurred
	require.Nil(t, err)

	// Read the response body
	body, _ := io.ReadAll(resp.Body)

	if respBody := string(body); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	// Ensure the transaction was correctly named
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /group/hello",
		IsWeb:         true,
		UnknownCaller: true,
	})
}
