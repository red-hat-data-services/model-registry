package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog"
	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelCatalogRoutesRegisterWithoutPanic guards against a regression where the
// GetAllModelArtifacts and GetAllModelPerformanceArtifacts route patterns embedded a
// chi wildcard ("*") in the middle of the path (".../models/*/artifacts") instead of
// only at the end. chi panics as soon as such a route is registered:
//
//	panic: chi: wildcard '*' must be the last value in a route. trim trailing text or
//	use a '{param}' instead
//
// This exercises the exact call made by plugin.RegisterRoutes (routing OrderedRoutes()
// into a chi.Router), which is what crashed the catalog service on startup.
func TestModelCatalogRoutesRegisterWithoutPanic(t *testing.T) {
	sources := catalog.NewSourceCollection()
	sources.Merge("", map[string]catalog.ModelSource{
		"test-source": {CatalogSource: model.CatalogSource{Id: "test-source", Name: "Test Source"}},
	})
	sourceLabels := catalog.NewLabelCollection()

	provider := &mockModelProvider{models: map[string]*model.CatalogModel{}}
	service := NewModelCatalogServiceAPIService(provider, sources, nil, nil, sourceLabels, nil)
	controller := NewModelCatalogServiceAPIController(service)

	var router http.Handler
	require.NotPanics(t, func() {
		router = NewRouter(controller)
	})
	require.NotNil(t, router)
}

// TestModelCatalogRoutesEndToEnd verifies the three model routes still behave correctly
// once registered: GetModel (including delegation for model names containing slashes,
// which is why it uses a chi wildcard), GetAllModelArtifacts, and
// GetAllModelPerformanceArtifacts (which must use a plain {model_name} path param).
func TestModelCatalogRoutesEndToEnd(t *testing.T) {
	sources := catalog.NewSourceCollection()
	sources.Merge("", map[string]catalog.ModelSource{
		"test-source": {CatalogSource: model.CatalogSource{Id: "test-source", Name: "Test Source"}},
	})
	sourceLabels := catalog.NewLabelCollection()

	artifactName := "artifact-1"
	artifact := model.CatalogArtifact{
		CatalogModelArtifact: &model.CatalogModelArtifact{
			Name:             &artifactName,
			ArtifactType:     "model-artifact",
			Uri:              "oci://example/model",
			CustomProperties: map[string]model.MetadataValue{},
		},
	}

	provider := &mockModelProvider{
		models: map[string]*model.CatalogModel{
			"simple-model":     {Name: "simple-model"},
			"org/nested-model": {Name: "org/nested-model"},
		},
		artifacts: map[string][]model.CatalogArtifact{
			"simple-model":     {artifact},
			"org/nested-model": {artifact},
		},
	}

	service := NewModelCatalogServiceAPIService(provider, sources, nil, nil, sourceLabels, nil)
	controller := NewModelCatalogServiceAPIController(service)
	router := NewRouter(controller)

	t.Run("GetModel with slash-containing model name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/model_catalog/v1/sources/test-source/models/org/nested-model", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogModel
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &result))
		assert.Equal(t, "org/nested-model", result.Name)
	})

	t.Run("GetAllModelArtifacts direct route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/model_catalog/v1/sources/test-source/models/simple-model/artifacts", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &result))
		assert.Equal(t, int32(1), result.Size)
	})

	t.Run("GetModel delegates /artifacts for slash-containing model name", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/model_catalog/v1/sources/test-source/models/org/nested-model/artifacts", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result model.CatalogArtifactList
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &result))
		assert.Equal(t, int32(1), result.Size)
	})

	t.Run("GetAllModelPerformanceArtifacts direct route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/model_catalog/v1/sources/test-source/models/simple-model/artifacts/performance", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}
