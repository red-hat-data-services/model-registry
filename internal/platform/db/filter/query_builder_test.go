package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestArrayValuePropertyReference(t *testing.T) {
	for _, property := range []string{"architecture", "platform.architecture"} {
		t.Run(property, func(t *testing.T) {
			expr, err := Parse(property + ".array_value = 'ppc64le'")
			require.NoError(t, err)
			qb := &QueryBuilder{entityType: EntityTypeContext}
			ref := qb.BuildPropertyReference(expr)
			assert.Equal(t, property, ref.Name)
			assert.True(t, ref.IsCustom)
			assert.Equal(t, ArrayValueType, ref.ExplicitType)
			assert.Equal(t, ArrayValueType, ref.ValueType)
		})
	}
}

func TestArrayValueMembershipQuery(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=test dbname=test sslmode=disable",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	require.NoError(t, err)

	for _, query := range []string{
		"architecture.array_value = 'ppc64le'",
		"architecture.array_value = 'ppc64le' OR architecture.array_value = 'arm64'",
	} {
		t.Run(query, func(t *testing.T) {
			expr, err := Parse(query)
			require.NoError(t, err)
			qb := &QueryBuilder{entityType: EntityTypeContext, tablePrefix: "Context"}
			stmt := qb.BuildQuery(db.Table("Context"), expr).Find(&[]struct{ ID int32 }{}).Statement
			assert.Contains(t, stmt.SQL.String(), "IS JSON ARRAY")
			assert.Contains(t, stmt.SQL.String(), "::jsonb ?| array[")
			assert.Contains(t, stmt.Vars, "architecture")
			assert.NotContains(t, stmt.Vars, "architecture.array_value")
			assert.Contains(t, stmt.Vars, "ppc64le")
		})
	}
}
