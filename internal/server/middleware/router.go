package middleware

import (
	"net/http"

	"github.com/kubeflow/model-registry/internal/server/openapi"
)

// WrapWithValidation wraps the auto-generated router with CORS and validation middleware.
// If corsAllowedOrigins is empty, CORS is disabled (no cross-origin headers are added).
func WrapWithValidation(corsAllowedOrigins []string, routers ...openapi.Router) http.Handler {
	baseRouter := openapi.NewRouter(routers...)

	handler := CORSMiddleware(corsAllowedOrigins)(baseRouter)

	return ValidationMiddleware(handler)
}
