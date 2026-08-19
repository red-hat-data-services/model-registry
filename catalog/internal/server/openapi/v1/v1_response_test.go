package v1

import (
	"encoding/json"
	"testing"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV1ResponseNativeJSONFormatting(t *testing.T) {
	sourceID := "test-source-1"
	licenseLink := "https://example.com/license"

	t.Run("CatalogModel produces camelCase sourceId", func(t *testing.T) {
		m := model.CatalogModel{
			Name:     "test-model",
			SourceId: &sourceID,
		}

		data, err := json.Marshal(m)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
	})

	t.Run("CatalogModelList produces camelCase sourceId in items", func(t *testing.T) {
		m := model.CatalogModel{
			Name:     "test-model",
			SourceId: &sourceID,
		}
		list := model.CatalogModelList{
			Items: []model.CatalogModel{m},
			Size:  1,
		}

		data, err := json.Marshal(list)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
	})

	t.Run("MCPServer produces camelCase sourceId and licenseLink", func(t *testing.T) {
		s := model.MCPServer{
			Name:        "test-mcp",
			SourceId:    &sourceID,
			LicenseLink: &licenseLink,
		}

		data, err := json.Marshal(s)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.Contains(t, jsonStr, `"licenseLink":"https://example.com/license"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
		assert.NotContains(t, jsonStr, `"license_link"`)
	})

	t.Run("MCPServerList produces camelCase sourceId and licenseLink in items", func(t *testing.T) {
		s := model.MCPServer{
			Name:        "test-mcp",
			SourceId:    &sourceID,
			LicenseLink: &licenseLink,
		}
		list := model.MCPServerList{
			Items: []model.MCPServer{s},
			Size:  1,
		}

		data, err := json.Marshal(list)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.Contains(t, jsonStr, `"licenseLink":"https://example.com/license"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
		assert.NotContains(t, jsonStr, `"license_link"`)
	})

	t.Run("Agent produces camelCase sourceId", func(t *testing.T) {
		a := model.Agent{
			Name:     "test-agent",
			SourceId: &sourceID,
		}

		data, err := json.Marshal(a)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
	})

	t.Run("AgentList produces camelCase sourceId in items", func(t *testing.T) {
		a := model.Agent{
			Name:     "test-agent",
			SourceId: &sourceID,
		}
		list := model.AgentList{
			Items: []model.Agent{a},
			Size:  1,
		}

		data, err := json.Marshal(list)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
	})

	t.Run("Skill produces camelCase sourceId", func(t *testing.T) {
		sk := model.Skill{
			Name:     "test-skill",
			SourceId: &sourceID,
		}

		data, err := json.Marshal(sk)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
	})

	t.Run("SkillList produces camelCase sourceId in items", func(t *testing.T) {
		sk := model.Skill{
			Name:     "test-skill",
			SourceId: &sourceID,
		}
		list := model.SkillList{
			Items: []model.Skill{sk},
			Size:  1,
		}

		data, err := json.Marshal(list)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"sourceId":"test-source-1"`)
		assert.NotContains(t, jsonStr, `"source_id"`)
	})
}
