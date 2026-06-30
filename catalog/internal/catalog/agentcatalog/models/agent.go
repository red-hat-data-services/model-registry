package models

import (
	"github.com/kubeflow/hub/internal/platform/db/filter"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
)

// AgentListOptions holds the options for listing Agent entities.
type AgentListOptions struct {
	dbmodels.Pagination
	Name        *string
	Query       *string
	SourceIDs   *[]string
	FilterQuery *string
}

// GetRestEntityType implements the FilterApplier interface.
func (o *AgentListOptions) GetRestEntityType() filter.RestEntityType {
	return filter.RestEntityType("agent")
}

// GetFilterQuery returns the filter query string for advanced filtering.
func (o *AgentListOptions) GetFilterQuery() string {
	if o.FilterQuery == nil {
		return ""
	}
	return *o.FilterQuery
}

// AgentAttributes holds the attributes for a Agent record.
type AgentAttributes struct {
	Name                     *string
	ExternalID               *string
	CreateTimeSinceEpoch     *int64
	LastUpdateTimeSinceEpoch *int64
}

// Agent represents a Agent stored in the database.
type Agent interface {
	dbmodels.Entity[AgentAttributes]
}

// AgentImpl is the concrete implementation of Agent.
type AgentImpl = dbmodels.BaseEntity[AgentAttributes]

// AgentRepository defines the interface for Agent persistence.
type AgentRepository interface {
	GetByID(id int32) (Agent, error)
	GetByName(name string) (Agent, error)
	List(listOptions *AgentListOptions) (*dbmodels.ListWrapper[Agent], error)
	Save(entity Agent) (Agent, error)
	DeleteBySource(sourceID string) error
	DeleteByID(id int32) error
	GetDistinctSourceIDs() ([]string, error)
	GetTypeID() int32
}
