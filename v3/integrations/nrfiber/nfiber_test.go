package nrfiber

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
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

// Test_FastHeaderResponseWriter test the different header operations
func Test_FastHeaderResponseWriter(t *testing.T) {
	// Setup
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	t.Run("implements http.ResponseWriter", func(t *testing.T) {
		// Verify that our implementation satisfies the http.ResponseWriter interface
		var _ http.ResponseWriter = &fastHeaderResponseWriter{}
	})

	t.Run("header operations", func(t *testing.T) {
		writer := newFastHeaderResponseWriter(ctx.Response())

		// Test adding headers
		writer.Header().Add("X-Test-Header", "value1")
		writer.Header().Set("X-Single-Header", "single-value")

		// Verify headers were stored in the wrapper
		assert.Equal(t, []string{"value1"}, writer.header["X-Test-Header"])
		assert.Equal(t, []string{"single-value"}, writer.header["X-Single-Header"])

		// Verify headers aren't yet in the actual response
		assert.Equal(t, "", string(ctx.Response().Header.Peek("X-Test-Header")))

		// Apply headers and verify they're in the response
		writer.applyHeaders()
		assert.Equal(t, "value1", string(ctx.Response().Header.Peek("X-Test-Header")))
		assert.Equal(t, "single-value", string(ctx.Response().Header.Peek("X-Single-Header")))
	})

	t.Run("status code handling", func(t *testing.T) {
		writer := newFastHeaderResponseWriter(ctx.Response())

		// Default status should be 200 OK
		assert.Equal(t, http.StatusOK, writer.statusCode)

		// Set status code
		writer.WriteHeader(http.StatusNotFound)

		// Check internal tracking
		assert.Equal(t, http.StatusNotFound, writer.statusCode)

		// Check fiber response status
		assert.Equal(t, http.StatusNotFound, ctx.Response().StatusCode())
	})

	t.Run("write operation", func(t *testing.T) {
		writer := newFastHeaderResponseWriter(ctx.Response())

		// The Write method is a no-op but we should test it returns expected values
		n, err := writer.Write([]byte("test"))
		assert.Equal(t, 0, n)
		assert.NoError(t, err)
	})

	t.Run("integration with fiber response", func(t *testing.T) {
		writer := newFastHeaderResponseWriter(ctx.Response())

		// Set headers via our wrapper
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-API-Key", "secret-key")

		// Set status
		writer.WriteHeader(http.StatusCreated)

		// Apply headers
		writer.applyHeaders()

		// Check that fiber response has the correct values
		assert.Equal(t, http.StatusCreated, ctx.Response().StatusCode())
		assert.Equal(t, "application/json", string(ctx.Response().Header.Peek("Content-Type")))
		assert.Equal(t, "secret-key", string(ctx.Response().Header.Peek("X-API-Key")))
	})
}

// This test specifically verifies the behavior when multiple values are set for a header
func Test_MultiValueHeaders(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	writer := newFastHeaderResponseWriter(ctx.Response())

	// Add multiple Set-Cookie headers (common use case for multi-value headers)
	writer.Header().Add("Set-Cookie", "cookie1=value1; Path=/")
	writer.Header().Add("Set-Cookie", "cookie2=value2; Path=/")

	// Apply headers
	writer.applyHeaders()

	// Fiber should handle multiple values properly for Set-Cookie
	// Get all Set-Cookie headers
	cookies := []string{}
	ctx.Response().Header.VisitAllCookie(func(key, value []byte) {
		cookies = append(cookies, string(value))
	})

	// Should have two cookies
	assert.Len(t, cookies, 2)
	assert.Contains(t, cookies, "cookie1=value1; Path=/")
	assert.Contains(t, cookies, "cookie2=value2; Path=/")
}

// This test verifies that our response writer correctly interacts with a New Relic transaction
func Test_FastHeaderResponseWriterWithNRTransaction(t *testing.T) {
	// This is a mock test to demonstrate interaction with NR transaction
	// In a real test, you would use a mock for the New Relic transaction

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	writer := newFastHeaderResponseWriter(ctx.Response())

	// Set custom headers
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("X-Custom-Header", "custom-value")

	// Set a non-200 status code
	writer.WriteHeader(http.StatusBadRequest)

	// Apply the headers
	writer.applyHeaders()

	// Verify response state
	assert.Equal(t, http.StatusBadRequest, writer.statusCode)
	assert.Equal(t, http.StatusBadRequest, ctx.Response().StatusCode())
	assert.Equal(t, "application/json", string(ctx.Response().Header.Peek("Content-Type")))
	assert.Equal(t, "custom-value", string(ctx.Response().Header.Peek("X-Custom-Header")))
}

// This test verifies that header manipulations after WriteHeader still work
func Test_HeadersAfterWriteHeader(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	writer := newFastHeaderResponseWriter(ctx.Response())

	// Set status code first
	writer.WriteHeader(http.StatusAccepted)

	// Then manipulate headers
	writer.Header().Set("X-Late-Header", "late-value")

	// Apply headers
	writer.applyHeaders()

	// Verify everything was set correctly
	assert.Equal(t, http.StatusAccepted, ctx.Response().StatusCode())
	assert.Equal(t, "late-value", string(ctx.Response().Header.Peek("X-Late-Header")))
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
