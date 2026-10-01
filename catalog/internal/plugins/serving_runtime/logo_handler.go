package serving_runtime

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubeflow/hub/catalog/internal/plugins/logo"
	openapi "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/pkg/api"
)

type runtimeLogoProvider interface {
	GetServingRuntime(context.Context, string) (*openapi.ServingRuntime, error)
}

func LogoHandler(provider runtimeLogoProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "missing runtime ID", http.StatusBadRequest)
			return
		}

		runtime, err := provider.GetServingRuntime(r.Context(), id)
		if err != nil {
			if errors.Is(err, api.ErrNotFound) {
				http.NotFound(w, r)
			} else if errors.Is(err, api.ErrBadRequest) {
				http.Error(w, "invalid runtime ID", http.StatusBadRequest)
			} else {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		if runtime == nil || !runtime.HasLogo() {
			http.NotFound(w, r)
			return
		}

		logo.Serve(w, r, runtime.GetLogo())
	}
}
