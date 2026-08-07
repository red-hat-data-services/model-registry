package converter_test

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/mcpcatalog/models"
	"github.com/kubeflow/hub/catalog/internal/converter"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDbMCPServerToOpenapi_IncludesDisplayName(t *testing.T) {
	baseName := "test-server"
	displayName := "Test Server"
	version := "1.0.0"
	server := &models.MCPServerImpl{
		Attributes: &models.MCPServerAttributes{
			Name: &baseName,
		},
		Properties: &[]dbmodels.Properties{
			{Name: "displayName", StringValue: &displayName},
			{Name: "version", StringValue: &version},
		},
	}

	result := converter.ConvertDbMCPServerToOpenapi(server)
	require.NotNil(t, result)
	require.NotNil(t, result.DisplayName)
	assert.Equal(t, displayName, *result.DisplayName)
}

func TestConvertDbMCPToolToOpenapi_StripsQualifiedPrefix(t *testing.T) {
	tests := []struct {
		name         string
		storedName   string
		expectedName string
	}{
		{
			name:         "strips server@version prefix",
			storedName:   "weather-api@1.0.0:get_current_weather",
			expectedName: "get_current_weather",
		},
		{
			name:         "strips server-only prefix (no version)",
			storedName:   "myserver:my_tool",
			expectedName: "my_tool",
		},
		{
			name:         "no prefix passes through unchanged",
			storedName:   "plain_tool_name",
			expectedName: "plain_tool_name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			accessType := "read_only"
			tool := &models.MCPServerToolImpl{
				Attributes: &models.MCPServerToolAttributes{
					Name: new(tc.storedName),
				},
				Properties: &[]dbmodels.Properties{
					{Name: "accessType", StringValue: new(accessType)},
				},
			}

			result := converter.ConvertDbMCPToolToOpenapi(tool)
			require.NotNil(t, result)
			assert.Equal(t, tc.expectedName, result.Name)
		})
	}
}

func TestConvertDbMCPServerToOpenapi_IncludesServerJson(t *testing.T) {
	baseName := "server-json-server"
	version := "1.0.0"
	serverJsonMap := map[string]any{
		"$schema": "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json",
		"name":    "com.example/test",
		"version": "1.0.0",
		"packages": []any{
			map[string]any{"registryType": "oci", "identifier": "registry.example.com/test:1.0"},
		},
	}
	jsonBytes, err := json.Marshal(serverJsonMap)
	require.NoError(t, err)
	serverJsonStr := string(jsonBytes)

	server := &models.MCPServerImpl{
		Attributes: &models.MCPServerAttributes{
			Name: &baseName,
		},
		Properties: &[]dbmodels.Properties{
			{Name: "version", StringValue: &version},
			{Name: "serverJson", StringValue: &serverJsonStr},
		},
	}

	result := converter.ConvertDbMCPServerToOpenapi(server)
	require.NotNil(t, result)
	require.NotNil(t, result.ServerJson)
	assert.Equal(t, "com.example/test", result.ServerJson["name"])
	assert.Equal(t, "1.0.0", result.ServerJson["version"])

	packages, ok := result.ServerJson["packages"].([]any)
	require.True(t, ok)
	require.Len(t, packages, 1)
}

func TestConvertDbMCPServerToOpenapi_NoServerJson(t *testing.T) {
	baseName := "no-server-json"
	version := "1.0.0"
	server := &models.MCPServerImpl{
		Attributes: &models.MCPServerAttributes{
			Name: &baseName,
		},
		Properties: &[]dbmodels.Properties{
			{Name: "version", StringValue: &version},
		},
	}

	result := converter.ConvertDbMCPServerToOpenapi(server)
	require.NotNil(t, result)
	assert.Nil(t, result.ServerJson)
}
