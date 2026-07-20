package main

import (
	"log"
	"os"
	"strings"

	"github.com/kubeflow/model-registry/internal/csi/modelregistry"
	"github.com/kubeflow/model-registry/internal/csi/storage"
	"github.com/kubeflow/model-registry/pkg/openapi"
)

const (
	modelRegistryBaseUrlEnv       = "MODEL_REGISTRY_BASE_URL"
	modelRegistrySchemeEnv        = "MODEL_REGISTRY_SCHEME"
	allowedArtifactURIPrefixesEnv = "ALLOWED_ARTIFACT_URI_PREFIXES"
	modelRegistryBaseUrlDefault   = "localhost:8080"
	modelRegistrySchemeDefault    = "http"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: ./mr-storage-initializer <src-uri> <dest-path>")
	}

	sourceUri := os.Args[1]
	destPath := os.Args[2]

	log.Printf("Initializing, args: src_uri [%s] dest_path[ [%s]\n", sourceUri, destPath)

	baseUrl, ok := os.LookupEnv(modelRegistryBaseUrlEnv)
	if !ok || baseUrl == "" {
		baseUrl = modelRegistryBaseUrlDefault
	}

	scheme, ok := os.LookupEnv(modelRegistrySchemeEnv)
	if !ok || scheme == "" {
		scheme = modelRegistrySchemeDefault
	}

	var allowedPrefixes []string
	if prefixesStr, ok := os.LookupEnv(allowedArtifactURIPrefixesEnv); ok && prefixesStr != "" {
		for _, p := range strings.Split(prefixesStr, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				allowedPrefixes = append(allowedPrefixes, trimmed)
			}
		}
		log.Printf("Artifact URI allowlist configured with %d prefix(es): %v", len(allowedPrefixes), allowedPrefixes)
	} else {
		log.Printf("No artifact URI allowlist configured (%s not set), all URIs will be accepted", allowedArtifactURIPrefixesEnv)
	}

	cfg := openapi.NewConfiguration()
	cfg.Host = baseUrl
	cfg.Scheme = scheme

	apiClient := modelregistry.NewAPIClient(cfg, sourceUri)

	provider, err := storage.NewModelRegistryProvider(apiClient, allowedPrefixes)
	if err != nil {
		log.Fatalf("Error initiliazing model registry provider: %v", err)
	}

	if err := provider.DownloadModel(destPath, "", sourceUri); err != nil {
		log.Fatalf("Error downloading the model: %s", err.Error())
	}
}
