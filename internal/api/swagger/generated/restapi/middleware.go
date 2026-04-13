// This file is safe to edit.

package restapi

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

var (
	// SetupApiMiddlewares The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
	// The middleware executes after routing but before authentication, binding and validation.
	SetupApiMiddlewares middleware.Builder

	// SetupGlobalMiddleware The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
	// So this is a good place to plug in a panic handling middleware, logging and metrics.
	SetupGlobalMiddleware middleware.Builder
)

func setupMiddlewares(handler http.Handler) http.Handler {
	return SetupApiMiddlewares(handler)
}

func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return SetupGlobalMiddleware(handler)
}
