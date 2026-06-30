package agentcatalog

import (
	agentmodels "github.com/kubeflow/hub/catalog/internal/catalog/agentcatalog/models"
	sharedmodels "github.com/kubeflow/hub/catalog/internal/db/models"
)

type Services struct {
	AgentRepository agentmodels.AgentRepository
	CatalogSourceRepository   sharedmodels.CatalogSourceRepository
	PropertyOptionsRepository sharedmodels.PropertyOptionsRepository
}
