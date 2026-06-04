package mapper_test

import (
	"testing"

	"github.com/kubeflow/hub/internal/db/models"
	"github.com/kubeflow/hub/internal/defaults"
	"github.com/kubeflow/hub/internal/mapper"
	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants for type IDs
const (
	testRegisteredModelTypeId    = int32(1)
	testModelVersionTypeId       = int32(2)
	testDocArtifactTypeId        = int32(3)
	testModelArtifactTypeId      = int32(4)
	testServingEnvironmentTypeId = int32(5)
	testInferenceServiceTypeId   = int32(6)
	testServeModelTypeId         = int32(7)
)

var testTypesMap = map[string]int32{
	defaults.RegisteredModelTypeName:    testRegisteredModelTypeId,
	defaults.ModelVersionTypeName:       testModelVersionTypeId,
	defaults.DocArtifactTypeName:        testDocArtifactTypeId,
	defaults.ModelArtifactTypeName:      testModelArtifactTypeId,
	defaults.ServingEnvironmentTypeName: testServingEnvironmentTypeId,
	defaults.InferenceServiceTypeName:   testInferenceServiceTypeId,
	defaults.ServeModelTypeName:         testServeModelTypeId,
}

func setupEmbedMDMapper(t *testing.T) (*assert.Assertions, *mapper.EmbedMDMapper) {
	return assert.New(t), mapper.NewEmbedMDMapper(testTypesMap)
}

// Tests for OpenAPI --> EmbedMD mapping

func TestEmbedMDMapFromRegisteredModel(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	openAPIModel := &openapi.RegisteredModel{
		Name:        "test-registered-model",
		Description: new("Test description"),
		Owner:       new("test-owner"),
		ExternalId:  new("ext-123"),
		State:       new(openapi.REGISTEREDMODELSTATE_LIVE),
	}

	result, err := mapper.MapFromRegisteredModel(openAPIModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify type ID
	assertion.Equal(int32(testRegisteredModelTypeId), *result.GetTypeID())

	// Verify attributes
	attrs := result.GetAttributes()
	assertion.NotNil(attrs)
	assertion.Equal("test-registered-model", *attrs.Name)
	assertion.Equal("ext-123", *attrs.ExternalID)

	// Verify properties
	props := result.GetProperties()
	assertion.NotNil(props)

	// Check for description property
	var foundDescription, foundOwner, foundState bool
	for _, prop := range *props {
		switch prop.Name {
		case "description":
			foundDescription = true
			assertion.Equal("Test description", *prop.StringValue)
		case "owner":
			foundOwner = true
			assertion.Equal("test-owner", *prop.StringValue)
		case "state":
			foundState = true
			assertion.Equal("LIVE", *prop.StringValue)
		}
	}
	assertion.True(foundDescription, "Should find description property")
	assertion.True(foundOwner, "Should find owner property")
	assertion.True(foundState, "Should find state property")
}

func TestEmbedMDMapFromModelVersion(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	openAPIModel := &openapi.ModelVersion{
		Name:              "test-model-version",
		Description:       new("Test version description"),
		Author:            new("test-author"),
		ExternalId:        new("version-ext-123"),
		State:             new(openapi.MODELVERSIONSTATE_LIVE),
		RegisteredModelId: "1",
	}

	result, err := mapper.MapFromModelVersion(openAPIModel, &openAPIModel.RegisteredModelId)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify type ID
	assertion.Equal(int32(testModelVersionTypeId), *result.GetTypeID())

	// Verify attributes
	attrs := result.GetAttributes()
	assertion.NotNil(attrs)
	assertion.Equal("1:test-model-version", *attrs.Name) // Now expects prefixed name
	assertion.Equal("version-ext-123", *attrs.ExternalID)

	// Verify properties
	props := result.GetProperties()
	assertion.NotNil(props)

	var foundDescription, foundAuthor, foundState, foundRegisteredModelId bool
	for _, prop := range *props {
		switch prop.Name {
		case "description":
			foundDescription = true
			assertion.Equal("Test version description", *prop.StringValue)
		case "author":
			foundAuthor = true
			assertion.Equal("test-author", *prop.StringValue)
		case "state":
			foundState = true
			assertion.Equal("LIVE", *prop.StringValue)
		case "registered_model_id":
			foundRegisteredModelId = true
			assertion.Equal(int32(1), *prop.IntValue)
		}
	}
	assertion.True(foundDescription, "Should find description property")
	assertion.True(foundAuthor, "Should find author property")
	assertion.True(foundState, "Should find state property")
	assertion.True(foundRegisteredModelId, "Should find registered_model_id property")
}

func TestEmbedMDMapFromServingEnvironment(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	openAPIModel := &openapi.ServingEnvironment{
		Name:        "test-serving-env",
		Description: new("Test serving environment"),
		ExternalId:  new("env-ext-123"),
	}

	result, err := mapper.MapFromServingEnvironment(openAPIModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify type ID
	assertion.Equal(int32(testServingEnvironmentTypeId), *result.GetTypeID())

	// Verify attributes
	attrs := result.GetAttributes()
	assertion.NotNil(attrs)
	assertion.Equal("test-serving-env", *attrs.Name)
	assertion.Equal("env-ext-123", *attrs.ExternalID)
}

func TestEmbedMDMapFromInferenceService(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	openAPIModel := &openapi.InferenceService{
		Name:                 new("test-inference-service"),
		Description:          new("Test inference service"),
		ExternalId:           new("inf-ext-123"),
		ServingEnvironmentId: "5",
		RegisteredModelId:    "1",
		ModelVersionId:       new("2"),
		Runtime:              new("tensorflow"),
		DesiredState:         new(openapi.INFERENCESERVICESTATE_DEPLOYED),
	}

	result, err := mapper.MapFromInferenceService(openAPIModel, "5")
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify type ID
	assertion.Equal(int32(testInferenceServiceTypeId), *result.GetTypeID())

	// Verify attributes
	attrs := result.GetAttributes()
	assertion.NotNil(attrs)
	assertion.Equal("5:test-inference-service", *attrs.Name)
	assertion.Equal("inf-ext-123", *attrs.ExternalID)

	// Verify properties
	props := result.GetProperties()
	assertion.NotNil(props)

	var foundServingEnvId, foundRegisteredModelId, foundModelVersionId, foundRuntime, foundDesiredState bool
	for _, prop := range *props {
		switch prop.Name {
		case "serving_environment_id":
			foundServingEnvId = true
			assertion.Equal(int32(5), *prop.IntValue)
		case "registered_model_id":
			foundRegisteredModelId = true
			assertion.Equal(int32(1), *prop.IntValue)
		case "model_version_id":
			foundModelVersionId = true
			assertion.Equal(int32(2), *prop.IntValue)
		case "runtime":
			foundRuntime = true
			assertion.Equal("tensorflow", *prop.StringValue)
		case "desired_state":
			foundDesiredState = true
			assertion.Equal("DEPLOYED", *prop.StringValue)
		}
	}
	assertion.True(foundServingEnvId, "Should find serving_environment_id property")
	assertion.True(foundRegisteredModelId, "Should find registered_model_id property")
	assertion.True(foundModelVersionId, "Should find model_version_id property")
	assertion.True(foundRuntime, "Should find runtime property")
	assertion.True(foundDesiredState, "Should find desired_state property")
}

func TestEmbedMDMapFromModelArtifact(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	openAPIModel := &openapi.ModelArtifact{
		Name:               new("test-model-artifact"),
		Description:        new("Test model artifact"),
		ExternalId:         new("model-art-ext-123"),
		Uri:                new("s3://bucket/model.pkl"),
		State:              new(openapi.ARTIFACTSTATE_LIVE),
		ModelFormatName:    new("pickle"),
		ModelFormatVersion: new("1.0"),
		StorageKey:         new("storage-key"),
		StoragePath:        new("/path/to/model"),
	}

	t.Run("with parent ID", func(t *testing.T) {
		testParentId := "test-parent-123"
		result, err := mapper.MapFromModelArtifact(openAPIModel, &testParentId)
		assertion.Nil(err)
		assertion.NotNil(result)

		// Verify type ID
		assertion.Equal(int32(testModelArtifactTypeId), *result.GetTypeID())

		// Verify attributes
		attrs := result.GetAttributes()
		assertion.NotNil(attrs)
		assertion.Equal("test-parent-123:test-model-artifact", *attrs.Name)
		assertion.Equal("model-art-ext-123", *attrs.ExternalID)
		assertion.Equal("s3://bucket/model.pkl", *attrs.URI)
		assertion.Equal("LIVE", *attrs.State)
		// Add nil check for ArtifactType
		if attrs.ArtifactType != nil {
			assertion.Equal("model-artifact", *attrs.ArtifactType)
		}
	})

	t.Run("without parent ID (standalone)", func(t *testing.T) {
		result, err := mapper.MapFromModelArtifact(openAPIModel, nil)
		assertion.Nil(err)
		assertion.NotNil(result)

		// Verify type ID
		assertion.Equal(int32(testModelArtifactTypeId), *result.GetTypeID())

		// Verify attributes
		attrs := result.GetAttributes()
		assertion.NotNil(attrs)
		// For standalone artifacts, name will be UUID-prefixed
		assertion.Contains(*attrs.Name, ":test-model-artifact")
		assertion.True(len(*attrs.Name) > len("test-model-artifact"), "Name should be longer due to UUID prefix")
		assertion.Equal("model-art-ext-123", *attrs.ExternalID)
		assertion.Equal("s3://bucket/model.pkl", *attrs.URI)
		assertion.Equal("LIVE", *attrs.State)
		// Add nil check for ArtifactType
		if attrs.ArtifactType != nil {
			assertion.Equal("model-artifact", *attrs.ArtifactType)
		}
	})
}

func TestEmbedMDMapFromDocArtifact(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	openAPIModel := &openapi.DocArtifact{
		Name:        new("test-doc-artifact"),
		Description: new("Test doc artifact"),
		ExternalId:  new("doc-art-ext-123"),
		Uri:         new("s3://bucket/doc.pdf"),
		State:       new(openapi.ARTIFACTSTATE_LIVE),
	}

	t.Run("with parent ID", func(t *testing.T) {
		testParentId := "test-parent-456"
		result, err := mapper.MapFromDocArtifact(openAPIModel, &testParentId)
		assertion.Nil(err)
		assertion.NotNil(result)

		// Verify type ID
		assertion.Equal(int32(testDocArtifactTypeId), *result.GetTypeID())

		// Verify attributes
		attrs := result.GetAttributes()
		assertion.NotNil(attrs)
		assertion.Equal("test-parent-456:test-doc-artifact", *attrs.Name)
		assertion.Equal("doc-art-ext-123", *attrs.ExternalID)
		assertion.Equal("s3://bucket/doc.pdf", *attrs.URI)
		assertion.Equal("LIVE", *attrs.State)
		// Add nil check for ArtifactType
		if attrs.ArtifactType != nil {
			assertion.Equal("doc-artifact", *attrs.ArtifactType)
		}
	})

	t.Run("without parent ID (standalone)", func(t *testing.T) {
		result, err := mapper.MapFromDocArtifact(openAPIModel, nil)
		assertion.Nil(err)
		assertion.NotNil(result)

		// Verify type ID
		assertion.Equal(int32(testDocArtifactTypeId), *result.GetTypeID())

		// Verify attributes
		attrs := result.GetAttributes()
		assertion.NotNil(attrs)
		// For standalone artifacts, name will be UUID-prefixed
		assertion.Contains(*attrs.Name, ":test-doc-artifact")
		assertion.True(len(*attrs.Name) > len("test-doc-artifact"), "Name should be longer due to UUID prefix")
		assertion.Equal("doc-art-ext-123", *attrs.ExternalID)
		assertion.Equal("s3://bucket/doc.pdf", *attrs.URI)
		assertion.Equal("LIVE", *attrs.State)
		// Add nil check for ArtifactType
		if attrs.ArtifactType != nil {
			assertion.Equal("doc-artifact", *attrs.ArtifactType)
		}
	})
}

func TestEmbedMDMapFromServeModel(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	openAPIModel := &openapi.ServeModel{
		Name:           new("test-serve-model"),
		Description:    new("Test serve model"),
		ExternalId:     new("serve-ext-123"),
		ModelVersionId: "2",
		LastKnownState: new(openapi.EXECUTIONSTATE_RUNNING),
	}

	// ServeModel always requires a parent ID (InferenceService)
	// It does not support standalone operation according to the API design
	testParentId := "test-parent-789"
	result, err := mapper.MapFromServeModel(openAPIModel, &testParentId)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify type ID
	assertion.Equal(int32(testServeModelTypeId), *result.GetTypeID())

	// Verify attributes
	attrs := result.GetAttributes()
	assertion.NotNil(attrs)
	assertion.Equal("test-parent-789:test-serve-model", *attrs.Name)
	assertion.Equal("serve-ext-123", *attrs.ExternalID)
	// Add nil check for LastKnownState
	if attrs.LastKnownState != nil {
		assertion.Equal("RUNNING", *attrs.LastKnownState)
	}

	// Verify properties
	props := result.GetProperties()
	assertion.NotNil(props)

	var foundModelVersionId bool
	for _, prop := range *props {
		if prop.Name == "model_version_id" {
			foundModelVersionId = true
			assertion.Equal(int32(2), *prop.IntValue)
		}
	}
	assertion.True(foundModelVersionId, "Should find model_version_id property")
}

// Tests for EmbedMD --> OpenAPI mapping

func TestEmbedMDMapToRegisteredModel(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	embedMDModel := &models.RegisteredModelImpl{
		ID:     new(int32(1)),
		TypeID: new(int32(testRegisteredModelTypeId)),
		Attributes: &models.RegisteredModelAttributes{
			Name:                     new("test-registered-model"),
			ExternalID:               new("ext-123"),
			CreateTimeSinceEpoch:     new(int64(1234567890)),
			LastUpdateTimeSinceEpoch: new(int64(1234567891)),
		},
		Properties: &[]models.Properties{
			{
				Name:        "description",
				StringValue: new("Test description"),
			},
			{
				Name:        "owner",
				StringValue: new("test-owner"),
			},
			{
				Name:        "state",
				StringValue: new("LIVE"),
			},
		},
		CustomProperties: &[]models.Properties{
			{
				Name:             "custom-prop",
				StringValue:      new("custom-value"),
				IsCustomProperty: true,
			},
		},
	}

	result, err := mapper.MapToRegisteredModel(embedMDModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify basic fields
	assertion.Equal("1", *result.Id)
	assertion.Equal("test-registered-model", result.Name)
	assertion.Equal("ext-123", *result.ExternalId)
	assertion.Equal("1234567890", *result.CreateTimeSinceEpoch)
	assertion.Equal("1234567891", *result.LastUpdateTimeSinceEpoch)

	// Verify mapped properties
	assertion.Equal("Test description", *result.Description)
	assertion.Equal("test-owner", *result.Owner)
	assertion.Equal(openapi.REGISTEREDMODELSTATE_LIVE, *result.State)

	// Verify custom properties
	assertion.NotNil(result.CustomProperties)
	customProps := result.CustomProperties
	assertion.Contains(customProps, "custom-prop")
	assertion.Equal("custom-value", customProps["custom-prop"].MetadataStringValue.StringValue)
}

func TestEmbedMDMapToRegisteredModelNil(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	result, err := mapper.MapToRegisteredModel(nil)
	assertion.NotNil(err)
	assertion.Nil(result)
	assertion.Equal("registered model is nil", err.Error())
}

func TestEmbedMDMapToModelVersion(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	embedMDModel := &models.ModelVersionImpl{
		ID:     new(int32(2)),
		TypeID: new(int32(testModelVersionTypeId)),
		Attributes: &models.ModelVersionAttributes{
			Name:                     new("test-model-version"),
			ExternalID:               new("version-ext-123"),
			CreateTimeSinceEpoch:     new(int64(1234567890)),
			LastUpdateTimeSinceEpoch: new(int64(1234567891)),
		},
		Properties: &[]models.Properties{
			{
				Name:        "description",
				StringValue: new("Test version description"),
			},
			{
				Name:        "author",
				StringValue: new("test-author"),
			},
			{
				Name:        "state",
				StringValue: new("LIVE"),
			},
			{
				Name:     "registered_model_id",
				IntValue: new(int32(1)),
			},
		},
	}

	result, err := mapper.MapToModelVersion(embedMDModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify basic fields
	assertion.Equal("2", *result.Id)
	assertion.Equal("test-model-version", result.Name)
	assertion.Equal("version-ext-123", *result.ExternalId)
	assertion.Equal("1234567890", *result.CreateTimeSinceEpoch)
	assertion.Equal("1234567891", *result.LastUpdateTimeSinceEpoch)

	// Verify mapped properties
	assertion.Equal("Test version description", *result.Description)
	assertion.Equal("test-author", *result.Author)
	assertion.Equal(openapi.MODELVERSIONSTATE_LIVE, *result.State)
	assertion.Equal("1", result.RegisteredModelId)
}

func TestEmbedMDMapToServingEnvironment(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	embedMDModel := &models.ServingEnvironmentImpl{
		ID:     new(int32(5)),
		TypeID: new(int32(testServingEnvironmentTypeId)),
		Attributes: &models.ServingEnvironmentAttributes{
			Name:                     new("test-serving-env"),
			ExternalID:               new("env-ext-123"),
			CreateTimeSinceEpoch:     new(int64(1234567890)),
			LastUpdateTimeSinceEpoch: new(int64(1234567891)),
		},
		Properties: &[]models.Properties{
			{
				Name:        "description",
				StringValue: new("Test serving environment"),
			},
		},
	}

	result, err := mapper.MapToServingEnvironment(embedMDModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify basic fields
	assertion.Equal("5", *result.Id)
	assertion.Equal("test-serving-env", result.Name)
	assertion.Equal("env-ext-123", *result.ExternalId)
	assertion.Equal("1234567890", *result.CreateTimeSinceEpoch)
	assertion.Equal("1234567891", *result.LastUpdateTimeSinceEpoch)

	// Verify mapped properties
	assertion.Equal("Test serving environment", *result.Description)
}

func TestEmbedMDMapToInferenceService(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	embedMDModel := &models.InferenceServiceImpl{
		ID:     new(int32(6)),
		TypeID: new(int32(testInferenceServiceTypeId)),
		Attributes: &models.InferenceServiceAttributes{
			Name:                     new("test-inference-service"),
			ExternalID:               new("inf-ext-123"),
			CreateTimeSinceEpoch:     new(int64(1234567890)),
			LastUpdateTimeSinceEpoch: new(int64(1234567891)),
		},
		Properties: &[]models.Properties{
			{
				Name:        "description",
				StringValue: new("Test inference service"),
			},
			{
				Name:     "serving_environment_id",
				IntValue: new(int32(5)),
			},
			{
				Name:     "registered_model_id",
				IntValue: new(int32(1)),
			},
			{
				Name:     "model_version_id",
				IntValue: new(int32(2)),
			},
			{
				Name:        "runtime",
				StringValue: new("tensorflow"),
			},
			{
				Name:        "desired_state",
				StringValue: new("DEPLOYED"),
			},
		},
	}

	result, err := mapper.MapToInferenceService(embedMDModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify basic fields
	assertion.Equal("6", *result.Id)
	assertion.Equal("test-inference-service", *result.Name)
	assertion.Equal("inf-ext-123", *result.ExternalId)
	assertion.Equal("1234567890", *result.CreateTimeSinceEpoch)
	assertion.Equal("1234567891", *result.LastUpdateTimeSinceEpoch)

	// Verify mapped properties
	assertion.Equal("Test inference service", *result.Description)
	assertion.Equal("5", result.ServingEnvironmentId)
	assertion.Equal("1", result.RegisteredModelId)
	assertion.Equal("2", *result.ModelVersionId)
	assertion.Equal("tensorflow", *result.Runtime)
	assertion.Equal(openapi.INFERENCESERVICESTATE_DEPLOYED, *result.DesiredState)
}

func TestEmbedMDMapToModelArtifact(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	embedMDModel := &models.ModelArtifactImpl{
		ID:     new(int32(3)),
		TypeID: new(int32(testModelArtifactTypeId)),
		Attributes: &models.ModelArtifactAttributes{
			Name:                     new("test-model-artifact"),
			ExternalID:               new("model-art-ext-123"),
			URI:                      new("s3://bucket/model.pkl"),
			State:                    new("LIVE"),
			ArtifactType:             new("model-artifact"),
			CreateTimeSinceEpoch:     new(int64(1234567890)),
			LastUpdateTimeSinceEpoch: new(int64(1234567891)),
		},
		Properties: &[]models.Properties{
			{
				Name:        "description",
				StringValue: new("Test model artifact"),
			},
			{
				Name:        "model_format_name",
				StringValue: new("pickle"),
			},
			{
				Name:        "model_format_version",
				StringValue: new("1.0"),
			},
		},
	}

	result, err := mapper.MapToModelArtifact(embedMDModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify basic fields
	assertion.Equal("3", *result.Id)
	assertion.Equal("test-model-artifact", *result.Name)
	assertion.Equal("model-art-ext-123", *result.ExternalId)
	assertion.Equal("s3://bucket/model.pkl", *result.Uri)
	assertion.Equal(openapi.ARTIFACTSTATE_LIVE, *result.State)
	assertion.Equal("model-artifact", *result.ArtifactType)
	assertion.Equal("1234567890", *result.CreateTimeSinceEpoch)
	assertion.Equal("1234567891", *result.LastUpdateTimeSinceEpoch)

	// Verify mapped properties
	assertion.Equal("Test model artifact", *result.Description)
	assertion.Equal("pickle", *result.ModelFormatName)
	assertion.Equal("1.0", *result.ModelFormatVersion)
}

func TestEmbedMDMapToDocArtifact(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	embedMDModel := &models.DocArtifactImpl{
		ID:     new(int32(4)),
		TypeID: new(int32(testDocArtifactTypeId)),
		Attributes: &models.DocArtifactAttributes{
			Name:                     new("test-doc-artifact"),
			ExternalID:               new("doc-art-ext-123"),
			URI:                      new("s3://bucket/doc.pdf"),
			State:                    new("LIVE"),
			ArtifactType:             new("doc-artifact"),
			CreateTimeSinceEpoch:     new(int64(1234567890)),
			LastUpdateTimeSinceEpoch: new(int64(1234567891)),
		},
		Properties: &[]models.Properties{
			{
				Name:        "description",
				StringValue: new("Test doc artifact"),
			},
		},
	}

	result, err := mapper.MapToDocArtifact(embedMDModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify basic fields
	assertion.Equal("4", *result.Id)
	assertion.Equal("test-doc-artifact", *result.Name)
	assertion.Equal("doc-art-ext-123", *result.ExternalId)
	assertion.Equal("s3://bucket/doc.pdf", *result.Uri)
	assertion.Equal(openapi.ARTIFACTSTATE_LIVE, *result.State)
	assertion.Equal("doc-artifact", *result.ArtifactType)
	assertion.Equal("1234567890", *result.CreateTimeSinceEpoch)
	assertion.Equal("1234567891", *result.LastUpdateTimeSinceEpoch)

	// Verify mapped properties
	assertion.Equal("Test doc artifact", *result.Description)
}

func TestEmbedMDMapToServeModel(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	embedMDModel := &models.ServeModelImpl{
		ID:     new(int32(7)),
		TypeID: new(int32(testServeModelTypeId)),
		Attributes: &models.ServeModelAttributes{
			Name:                     new("test-serve-model"),
			ExternalID:               new("serve-ext-123"),
			LastKnownState:           new("RUNNING"),
			CreateTimeSinceEpoch:     new(int64(1234567890)),
			LastUpdateTimeSinceEpoch: new(int64(1234567891)),
		},
		Properties: &[]models.Properties{
			{
				Name:        "description",
				StringValue: new("Test serve model"),
			},
			{
				Name:     "model_version_id",
				IntValue: new(int32(2)),
			},
		},
	}

	result, err := mapper.MapToServeModel(embedMDModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify basic fields
	assertion.Equal("7", *result.Id)
	assertion.Equal("test-serve-model", *result.Name)
	assertion.Equal("serve-ext-123", *result.ExternalId)
	assertion.Equal(openapi.EXECUTIONSTATE_RUNNING, *result.LastKnownState)
	assertion.Equal("1234567890", *result.CreateTimeSinceEpoch)
	assertion.Equal("1234567891", *result.LastUpdateTimeSinceEpoch)

	// Verify mapped properties
	assertion.Equal("Test serve model", *result.Description)
	assertion.Equal("2", result.ModelVersionId)
}

// Test edge cases and error conditions

func TestEmbedMDMapFromWithCustomProperties(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	customProps := map[string]openapi.MetadataValue{
		"string-prop": {
			MetadataStringValue: &openapi.MetadataStringValue{
				StringValue: "string-value",
			},
		},
		"int-prop": {
			MetadataIntValue: &openapi.MetadataIntValue{
				IntValue: "42",
			},
		},
		"bool-prop": {
			MetadataBoolValue: &openapi.MetadataBoolValue{
				BoolValue: true,
			},
		},
		"double-prop": {
			MetadataDoubleValue: &openapi.MetadataDoubleValue{
				DoubleValue: 3.14,
			},
		},
	}

	openAPIModel := &openapi.RegisteredModel{
		Name:             "test-with-custom-props",
		CustomProperties: customProps,
	}

	result, err := mapper.MapFromRegisteredModel(openAPIModel)
	assertion.Nil(err)
	assertion.NotNil(result)

	// Verify custom properties were converted
	customPropsResult := result.GetCustomProperties()
	assertion.NotNil(customPropsResult)
	assertion.Len(*customPropsResult, 4)

	// Check each custom property
	propMap := make(map[string]models.Properties)
	for _, prop := range *customPropsResult {
		propMap[prop.Name] = prop
	}

	assertion.Contains(propMap, "string-prop")
	assertion.Equal("string-value", *propMap["string-prop"].StringValue)
	assertion.True(propMap["string-prop"].IsCustomProperty)

	assertion.Contains(propMap, "int-prop")
	assertion.Equal(int32(42), *propMap["int-prop"].IntValue)
	assertion.True(propMap["int-prop"].IsCustomProperty)

	assertion.Contains(propMap, "bool-prop")
	assertion.Equal(true, *propMap["bool-prop"].BoolValue)
	assertion.True(propMap["bool-prop"].IsCustomProperty)

	assertion.Contains(propMap, "double-prop")
	assertion.Equal(3.14, *propMap["double-prop"].DoubleValue)
	assertion.True(propMap["double-prop"].IsCustomProperty)
}

func TestEmbedMDMapperCreation(t *testing.T) {
	assertion := assert.New(t)

	mapper := mapper.NewEmbedMDMapper(testTypesMap)
	assertion.NotNil(mapper)
	// Note: Cannot test unexported fields from external package
}

func TestEmbedMDMapFromWithMinimalData(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	// Test with minimal required data
	openAPIModel := &openapi.RegisteredModel{
		Name: "minimal-model",
	}

	result, err := mapper.MapFromRegisteredModel(openAPIModel)
	assertion.Nil(err)
	assertion.NotNil(result)
	assertion.Equal("minimal-model", *result.GetAttributes().Name)
	assertion.Equal(int32(testRegisteredModelTypeId), *result.GetTypeID())
}

func TestEmbedMDRoundTripConversion(t *testing.T) {
	assertion, mapper := setupEmbedMDMapper(t)

	// Create an OpenAPI model
	originalOpenAPI := &openapi.RegisteredModel{
		Name:        "roundtrip-test",
		Description: new("Test roundtrip conversion"),
		Owner:       new("test-owner"),
		ExternalId:  new("roundtrip-ext-123"),
		State:       new(openapi.REGISTEREDMODELSTATE_LIVE),
	}

	// Convert to EmbedMD
	embedMDModel, err := mapper.MapFromRegisteredModel(originalOpenAPI)
	require.NoError(t, err)

	// Set ID for the conversion back (simulating saved model)
	embedMDModel.(*models.RegisteredModelImpl).ID = new(int32(1))
	embedMDModel.(*models.RegisteredModelImpl).Attributes.CreateTimeSinceEpoch = new(int64(1234567890))
	embedMDModel.(*models.RegisteredModelImpl).Attributes.LastUpdateTimeSinceEpoch = new(int64(1234567891))

	// Convert back to OpenAPI
	resultOpenAPI, err := mapper.MapToRegisteredModel(embedMDModel)
	require.NoError(t, err)

	// Verify the roundtrip preserved the data
	assertion.Equal(originalOpenAPI.Name, resultOpenAPI.Name)
	assertion.Equal(*originalOpenAPI.Description, *resultOpenAPI.Description)
	assertion.Equal(*originalOpenAPI.Owner, *resultOpenAPI.Owner)
	assertion.Equal(*originalOpenAPI.ExternalId, *resultOpenAPI.ExternalId)
	assertion.Equal(*originalOpenAPI.State, *resultOpenAPI.State)
}
