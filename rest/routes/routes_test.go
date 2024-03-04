package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func hasRoute(app *fiber.App, method, path string) bool {
	for _, route := range app.GetRoutes() {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func TestLoadRoutes(t *testing.T) {
	// Create an instance of fiber.App
	app := fiber.New()

	// Call the LoadRoutes function
	LoadRoutes(app)

	// Assert that the app has the expected routes
	assert.True(t, hasRoute(app, "POST", "/api/v1/deposit"))
	assert.True(t, hasRoute(app, "GET", "/api/v1/deposit/:id"))
	assert.True(t, hasRoute(app, "POST", "/api/v1/deposit/:id/receipt"))
}
