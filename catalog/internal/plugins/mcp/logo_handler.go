package mcp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog"
	"github.com/kubeflow/hub/catalog/internal/plugins/logo"
	"github.com/kubeflow/hub/pkg/api"
)

func LogoHandler(provider mcpcatalog.MCPCatalogProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverID := chi.URLParam(r, "server_id")
		if serverID == "" {
			http.Error(w, "missing server_id", http.StatusBadRequest)
			return
		}

		server, err := provider.GetMCPServer(r.Context(), serverID, false, 0)
		if err != nil {
			if errors.Is(err, api.ErrNotFound) {
				http.NotFound(w, r)
			} else if errors.Is(err, api.ErrBadRequest) {
				http.Error(w, "invalid server ID", http.StatusBadRequest)
			} else {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}
		if server == nil || !server.HasLogo() {
			http.NotFound(w, r)
			return
		}

		logo.Serve(w, r, server.GetLogo())
	}
}
