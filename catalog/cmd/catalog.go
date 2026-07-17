package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/model-registry/catalog/internal/catalog"
	"github.com/kubeflow/model-registry/catalog/internal/db/models"
	"github.com/kubeflow/model-registry/catalog/internal/db/service"
	"github.com/kubeflow/model-registry/catalog/internal/server/openapi"
	"github.com/kubeflow/model-registry/internal/datastore"
	"github.com/kubeflow/model-registry/internal/datastore/embedmd"
	mrmiddleware "github.com/kubeflow/model-registry/internal/server/middleware"
	"github.com/spf13/cobra"
)

var catalogCfg = struct {
	ListenAddress          string
	ConfigPath             []string
	PerformanceMetricsPath []string
	CORSAllowedOrigins     []string
}{
	ListenAddress:          "0.0.0.0:8080",
	ConfigPath:             []string{"sources.yaml"},
	PerformanceMetricsPath: []string{},
}

var CatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Catalog API server",
	Long: `Launch the API server for the model catalog. Use PostgreSQL's
	environment variables
	(https://www.postgresql.org/docs/current/libpq-envars.html) to
	configure the database connection.`,
	RunE: runCatalogServer,
}

func init() {
	fs := CatalogCmd.Flags()
	fs.StringVarP(&catalogCfg.ListenAddress, "listen", "l", catalogCfg.ListenAddress, "Address to listen on")
	fs.StringSliceVar(&catalogCfg.ConfigPath, "catalogs-path", catalogCfg.ConfigPath, "Path to catalog source configuration file")
	fs.StringSliceVar(&catalogCfg.PerformanceMetricsPath, "performance-metrics", catalogCfg.PerformanceMetricsPath, "Path to performance metrics data directory")
	fs.StringSliceVar(&catalogCfg.CORSAllowedOrigins, "cors-allowed-origins", nil,
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

	ds, err := datastore.NewConnector("embedmd", &embedmd.EmbedMDConfig{
		DatabaseType: "postgres", // We only support postgres right now
		DatabaseDSN:  "",         // Empty DSN, see https://www.postgresql.org/docs/current/libpq-envars.html
	})
	if err != nil {
		return fmt.Errorf("error creating datastore: %w", err)
	}

	repoSet, err := ds.Connect(service.DatastoreSpec())
	if err != nil {
		return fmt.Errorf("error initializing datastore: %v", err)
	}

	services := service.NewServices(
		getRepo[models.CatalogModelRepository](repoSet),
		getRepo[models.CatalogArtifactRepository](repoSet),
		getRepo[models.CatalogModelArtifactRepository](repoSet),
		getRepo[models.CatalogMetricsArtifactRepository](repoSet),
		getRepo[models.CatalogSourceRepository](repoSet),
		getRepo[models.PropertyOptionsRepository](repoSet),
	)

	loader := catalog.NewLoader(services, catalogCfg.ConfigPath)

	perfLoader, err := catalog.NewPerformanceMetricsLoader(catalogCfg.PerformanceMetricsPath, services.CatalogModelRepository, services.CatalogMetricsArtifactRepository, repoSet.TypeMap())
	if err != nil {
		return fmt.Errorf("error initializing performance metrics: %v", err)
	}
	loader.RegisterEventHandler(perfLoader.Load)

	poRefresher := models.NewPropertyOptionsRefresher(context.Background(), services.PropertyOptionsRepository, time.Second)
	loader.RegisterEventHandler(func(ctx context.Context, record catalog.ModelProviderRecord) error {
		poRefresher.Trigger()
		return nil
	})

	err = loader.Start(context.Background())
	if err != nil {
		return fmt.Errorf("error loading catalog sources: %v", err)
	}

	svc := openapi.NewModelCatalogServiceAPIService(
		catalog.NewDBCatalog(services, loader.Sources),
		loader.Sources,
		loader.Labels,
		services.CatalogSourceRepository,
	)
	ctrl := openapi.NewModelCatalogServiceAPIController(svc)

	glog.Infof("Catalog API server listening on %s", catalogCfg.ListenAddress)
	return http.ListenAndServe(catalogCfg.ListenAddress, mrmiddleware.CORSMiddleware(catalogCfg.CORSAllowedOrigins)(openapi.NewRouter(ctrl)))
}

func getRepo[T any](repoSet datastore.RepoSet) T {
	repo, err := repoSet.Repository(reflect.TypeFor[T]())
	if err != nil {
		panic(fmt.Sprintf("unable to get repository: %v", err))
	}

	return repo.(T)
}
