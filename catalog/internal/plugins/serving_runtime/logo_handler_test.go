package serving_runtime

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

type logoRuntimeProvider struct {
	runtime *openapi.ServingRuntime
	err     error
}

func (p logoRuntimeProvider) GetServingRuntime(_ context.Context, _ string) (*openapi.ServingRuntime, error) {
	return p.runtime, p.err
}

func requestRuntimeLogo(provider logoRuntimeProvider, id string) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.Get("/api/serving_runtime_catalog/v1/serving_runtimes/{id}/logo", LogoHandler(provider))
	req := httptest.NewRequest(http.MethodGet, "/api/serving_runtime_catalog/v1/serving_runtimes/"+id+"/logo", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func runtimeWithLogo(logo string) *openapi.ServingRuntime {
	runtime := &openapi.ServingRuntime{}
	runtime.SetLogo(logo)
	return runtime
}

func TestLogoHandlerServesDataURI(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G'}
	w := requestRuntimeLogo(logoRuntimeProvider{
		runtime: runtimeWithLogo("data:image/png;base64," + base64.StdEncoding.EncodeToString(png)),
	}, "1")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, png, w.Body.Bytes())
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, "inline", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
}

func TestLogoHandlerSecuresSVG(t *testing.T) {
	w := requestRuntimeLogo(logoRuntimeProvider{
		runtime: runtimeWithLogo("DATA:image/svg+xml,%3Csvg%3E%3C/svg%3E"),
	}, "1")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "<svg></svg>", w.Body.String())
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'none'")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "sandbox")
}

func TestLogoHandlerRedirectsHTTPURL(t *testing.T) {
	logoURL := "https://example.com/runtime.png"
	w := requestRuntimeLogo(logoRuntimeProvider{runtime: runtimeWithLogo(logoURL)}, "1")
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, logoURL, w.Header().Get("Location"))
}

func TestLogoHandlerMissingRuntimeOrLogo(t *testing.T) {
	for _, provider := range []logoRuntimeProvider{{}, {runtime: &openapi.ServingRuntime{}}, {err: fmt.Errorf("missing: %w", api.ErrNotFound)}} {
		w := requestRuntimeLogo(provider, "999")
		assert.Equal(t, http.StatusNotFound, w.Code)
	}
}

func TestLogoHandlerProviderErrors(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want int
	}{
		{fmt.Errorf("invalid ID: %w", api.ErrBadRequest), http.StatusBadRequest},
		{errors.New("database failure"), http.StatusInternalServerError},
	} {
		w := requestRuntimeLogo(logoRuntimeProvider{err: tc.err}, "bad")
		assert.Equal(t, tc.want, w.Code)
	}
}

func TestLogoHandlerRejectsInvalidLogo(t *testing.T) {
	for _, tc := range []struct {
		logo string
		want int
	}{
		{"data:image/png;base64", http.StatusBadRequest},
		{"data:image/png;base64,!!!", http.StatusBadRequest},
		{"data:text/html,%3Cscript%3E", http.StatusUnsupportedMediaType},
		{"javascript:alert(1)", http.StatusBadRequest},
		{"//example.com/logo.png", http.StatusBadRequest},
		{"data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("A", 1<<20+1))), http.StatusRequestEntityTooLarge},
	} {
		w := requestRuntimeLogo(logoRuntimeProvider{runtime: runtimeWithLogo(tc.logo)}, "1")
		assert.Equal(t, tc.want, w.Code, "logo %q", tc.logo[:min(len(tc.logo), 40)])
	}
}
