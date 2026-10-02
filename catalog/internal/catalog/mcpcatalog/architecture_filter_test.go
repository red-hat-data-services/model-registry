package mcpcatalog

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServerArchitectureFiltering(t *testing.T) {
	_, services, cleanup := setupMCPLoaderTest(t)
	defer cleanup()

	servers := []struct {
		name          string
		architectures []string
	}{
		{name: "power-server", architectures: []string{"ppc64le"}},
		{name: "multi-server", architectures: []string{"amd64", "ppc64le"}},
		{name: "amd-server", architectures: []string{"amd64"}},
		{name: "arm-server", architectures: []string{"arm64"}},
		{name: "empty-server", architectures: []string{}},
		{name: "missing-server"},
	}
	for _, server := range servers {
		record := &models.MCPServerImpl{
			Attributes: &models.MCPServerAttributes{Name: &server.name},
			Properties: &[]dbmodels.Properties{
				dbmodels.NewStringProperty("source_id", "architecture-test", false),
			},
		}
		if server.architectures != nil {
			encoded, err := json.Marshal(server.architectures)
			require.NoError(t, err)
			record.CustomProperties = &[]dbmodels.Properties{
				dbmodels.NewStringProperty("architecture", string(encoded), true),
			}
		}
		_, err := services.MCPServerRepository.Save(record)
		require.NoError(t, err)
	}

	catalog := NewDBMCPCatalog(services, nil, nil)
	require.NoError(t, services.PropertyOptionsRepository.Refresh(sharedmodels.ContextPropertyOptionType))
	options, err := catalog.GetFilterOptions(t.Context())
	require.NoError(t, err)
	require.NotNil(t, options.Filters)
	const architectureField = "architecture.array_value"
	require.Contains(t, *options.Filters, architectureField)
	architectureOption := (*options.Filters)[architectureField]
	assert.Equal(t, "string", architectureOption.Type)
	assert.Equal(t, []any{"amd64", "arm64", "ppc64le"}, architectureOption.Values)

	for _, tc := range []struct {
		name  string
		query string
		want  []string
	}{
		{name: "ppc64le membership", query: architectureField + " = 'ppc64le'", want: []string{"power-server", "multi-server"}},
		{name: "amd64 membership", query: architectureField + " = 'amd64'", want: []string{"amd-server", "multi-server"}},
		{name: "unsupported architecture", query: architectureField + " = 's390x'"},
		{name: "exact membership", query: architectureField + " = 'ppc64'"},
		{name: "combined conditions", query: architectureField + " = 'ppc64le' AND name = 'multi-server'", want: []string{"multi-server"}},
		{name: "alternative architectures", query: architectureField + " = 'ppc64le' OR " + architectureField + " = 'arm64'", want: []string{"power-server", "multi-server", "arm-server"}},
		{name: "excluded architecture in OR", query: architectureField + " != 'ppc64le' OR name = 'does-not-exist'", want: []string{"amd-server", "arm-server", "empty-server"}},
		{name: "excluded architecture or missing property", query: architectureField + " != 'ppc64le' OR name = 'missing-server'", want: []string{"amd-server", "arm-server", "empty-server", "missing-server"}},
		{name: "architecture list", query: architectureField + " IN ('ppc64le', 'arm64')", want: []string{"power-server", "multi-server", "arm-server"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := catalog.ListMCPServers(t.Context(), ListMCPServersParams{
				FilterQuery: tc.query,
				SourceIDs:   []string{"architecture-test"},
				PageSize:    100,
			})
			require.NoError(t, err)
			names := make([]string, 0, len(result.Items))
			for _, server := range result.Items {
				names = append(names, server.Name)
			}
			assert.ElementsMatch(t, tc.want, names)
		})
	}
}
