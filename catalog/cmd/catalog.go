package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/catalog/internal/catalog"
	"github.com/kubeflow/model-registry/catalog/internal/server/openapi"
	"github.com/kubeflow/model-registry/internal/server/middleware"
	"github.com/spf13/cobra"
)

var catalogCfg = struct {
	ListenAddress      string
	ConfigPath         string
	CORSAllowedOrigins []string
}{
	ListenAddress: "0.0.0.0:8080",
	ConfigPath:    "sources.yaml",
}

var CatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Catalog API server",
	Long:  `Launch the API server for the model catalog`,
	RunE:  runCatalogServer,
}

func init() {
	CatalogCmd.Flags().StringVarP(&catalogCfg.ListenAddress, "listen", "l", catalogCfg.ListenAddress, "Address to listen on")
	CatalogCmd.Flags().StringVar(&catalogCfg.ConfigPath, "catalogs-path", catalogCfg.ConfigPath, "Path to catalog source configuration file")
	CatalogCmd.Flags().StringSliceVar(&catalogCfg.CORSAllowedOrigins, "cors-allowed-origins", nil,
		"Comma-separated list of allowed CORS origins. If empty (default), CORS is disabled. Can also be set via CATALOG_CORS_ALLOWED_ORIGINS environment variable.")
}

func runCatalogServer(cmd *cobra.Command, args []string) error {
	if !cmd.Flags().Changed("cors-allowed-origins") {
		if envVal := os.Getenv("CATALOG_CORS_ALLOWED_ORIGINS"); envVal != "" {
			for _, origin := range strings.Split(envVal, ",") {
				if o := strings.TrimSpace(origin); o != "" {
					catalogCfg.CORSAllowedOrigins = append(catalogCfg.CORSAllowedOrigins, o)
				}
			}
		}
	}

	sources, err := catalog.LoadCatalogSources(catalogCfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("error loading catalog sources: %v", err)
	}

	svc := openapi.NewModelCatalogServiceAPIService(sources)
	ctrl := openapi.NewModelCatalogServiceAPIController(svc)

	glog.Infof("Catalog API server listening on %s", catalogCfg.ListenAddress)
	return http.ListenAndServe(catalogCfg.ListenAddress, middleware.CORSMiddleware(catalogCfg.CORSAllowedOrigins)(openapi.NewRouter(ctrl)))
}
