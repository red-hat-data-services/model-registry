package service

import (
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentTemplateArtifactRepository(t *testing.T) {
	sharedDB, cleanup := testutils.SetupPostgresWithMigrations(t, testDatastoreSpec())
	defer cleanup()

	agentTypeID := getAgentTypeID(t, sharedDB)
	templateTypeID := getAgentTemplateArtifactTypeID(t, sharedDB)

	agentRepo := NewAgentRepository(sharedDB, agentTypeID)
	templateRepo := NewAgentTemplateArtifactRepository(sharedDB, templateTypeID)

	saveAgent := func(name string) int32 {
		saved, err := agentRepo.Save(&models.AgentImpl{Attributes: &models.AgentAttributes{Name: &name}})
		require.NoError(t, err)
		return *saved.GetID()
	}

	saveTemplate := func(name, content string, parentID int32) models.AgentTemplateArtifact {
		saved, err := templateRepo.Save(&models.AgentTemplateArtifactImpl{
			Attributes: &models.AgentTemplateArtifactAttributes{
				Name:    &name,
				Content: &content,
			},
		}, &parentID)
		require.NoError(t, err)
		return saved
	}

	t.Run("TestDeleteByParentID_RemovesOnlyOwnParentAndType", func(t *testing.T) {
		parent1ID := saveAgent("source:agent-one")
		parent2ID := saveAgent("source:agent-two")

		saveTemplate("source:agent-one:agent.yaml", "content-1", parent1ID)
		tmpl2 := saveTemplate("source:agent-one:extra.yaml", "content-2", parent1ID)
		parent2Template := saveTemplate("source:agent-two:agent.yaml", "content-3", parent2ID)

		// Attach an artifact of a different type to parent1, to confirm
		// DeleteByParentID scopes its deletion to its own artifact type.
		otherTypeID := getTypeIDByName(t, sharedDB, "kf.OtherArtifact")
		otherArtifact := schema.Artifact{TypeID: otherTypeID, Name: new("unrelated-artifact")}
		require.NoError(t, sharedDB.Create(&otherArtifact).Error)
		require.NoError(t, sharedDB.Create(&schema.Attribution{ContextID: parent1ID, ArtifactID: otherArtifact.ID}).Error)

		require.NoError(t, templateRepo.DeleteByParentID(parent1ID))

		list1, err := templateRepo.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parent1ID})
		require.NoError(t, err)
		assert.Empty(t, list1.Items, "parent1's template artifacts should be deleted")

		_, err = templateRepo.GetByID(*tmpl2.GetID())
		assert.ErrorIs(t, err, ErrAgentTemplateArtifactNotFound)

		list2, err := templateRepo.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parent2ID})
		require.NoError(t, err)
		require.Len(t, list2.Items, 1, "parent2's template artifact should be untouched")
		require.NotNil(t, list2.Items[0].GetAttributes().Name)
		assert.Equal(t, *parent2Template.GetAttributes().Name, *list2.Items[0].GetAttributes().Name)

		var remaining schema.Artifact
		err = sharedDB.Where("id = ?", otherArtifact.ID).First(&remaining).Error
		require.NoError(t, err, "an artifact of a different type attached to the same parent should not be deleted")
	})

	t.Run("TestSave_StoresQualifiedName", func(t *testing.T) {
		parentID := saveAgent("test-source:qualified-name-agent")
		qualifiedName := "test-source:qualified-name-agent:deploy.yaml"
		content := "apiVersion: apps/v1"

		saved := saveTemplate(qualifiedName, content, parentID)
		require.NotNil(t, saved.GetAttributes().Name)
		assert.Equal(t, qualifiedName, *saved.GetAttributes().Name,
			"repository should store the fully qualified name")

		// Verify the DB has the qualified name
		var dbArtifact schema.Artifact
		err := sharedDB.Where("id = ?", *saved.GetID()).First(&dbArtifact).Error
		require.NoError(t, err)
		require.NotNil(t, dbArtifact.Name)
		assert.Equal(t, qualifiedName, *dbArtifact.Name,
			"DB should store the fully qualified name source_id:agent_name:template_name")

		// Verify list returns the artifact with the qualified name
		list, err := templateRepo.List(models.AgentTemplateArtifactListOptions{ParentResourceID: &parentID})
		require.NoError(t, err)
		require.Len(t, list.Items, 1)
		assert.Equal(t, qualifiedName, *list.Items[0].GetAttributes().Name,
			"repository List should return the fully qualified name")
	})

	t.Run("TestList_FilterByName", func(t *testing.T) {
		parentID := saveAgent("test-source:name-filter-agent")
		saveTemplate("test-source:name-filter-agent:agent.yaml", "content-a", parentID)
		saveTemplate("test-source:name-filter-agent:deploy.yaml", "content-b", parentID)

		nameFilter := "agent.yaml"
		list, err := templateRepo.List(models.AgentTemplateArtifactListOptions{
			ParentResourceID: &parentID,
			Name:             &nameFilter,
		})
		require.NoError(t, err)
		require.Len(t, list.Items, 1)
		assert.Contains(t, *list.Items[0].GetAttributes().Name, "agent.yaml")
	})

	t.Run("TestDeleteByParentID_NoArtifacts", func(t *testing.T) {
		agentID := saveAgent("source:agent-empty")

		// A parent with no template artifacts should be idempotent, not an error.
		require.NoError(t, templateRepo.DeleteByParentID(agentID))
	})
}
