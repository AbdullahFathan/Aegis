package workflow_test

import (
	"testing"

	"aegis/internal/workflow"
	"aegis/pkg/database"

	"github.com/stretchr/testify/require"
)

func TestEscalationMatrix(t *testing.T) {
	cases := []struct {
		cat database.IncidentCategory
		sev database.Severity
		lvl database.EscalationLevel
	}{
		{database.CategoryNearMiss, database.SeverityLow, database.EscalationL1},
		{database.CategoryNearMiss, database.SeverityMedium, database.EscalationL1},
		{database.CategoryNearMiss, database.SeverityHigh, database.EscalationL2},
		{database.CategoryNearMiss, database.SeverityCritical, database.EscalationL2},
		{database.CategoryFirstAid, database.SeverityLow, database.EscalationL1},
		{database.CategoryFirstAid, database.SeverityMedium, database.EscalationL2},
		{database.CategoryFirstAid, database.SeverityHigh, database.EscalationL2},
		{database.CategoryFirstAid, database.SeverityCritical, database.EscalationL3},
		{database.CategoryMedicalTreatment, database.SeverityLow, database.EscalationL2},
		{database.CategoryMedicalTreatment, database.SeverityMedium, database.EscalationL2},
		{database.CategoryMedicalTreatment, database.SeverityHigh, database.EscalationL3},
		{database.CategoryMedicalTreatment, database.SeverityCritical, database.EscalationL3},
		{database.CategoryLTI, database.SeverityLow, database.EscalationL2},
		{database.CategoryLTI, database.SeverityMedium, database.EscalationL3},
		{database.CategoryLTI, database.SeverityHigh, database.EscalationL3},
		{database.CategoryLTI, database.SeverityCritical, database.EscalationL3},
		{database.CategoryFatality, database.SeverityLow, database.EscalationL3},
		{database.CategoryFatality, database.SeverityCritical, database.EscalationL3},
		{database.CategoryPropertyDamage, database.SeverityLow, database.EscalationL1},
		{database.CategoryEnvironmental, database.SeverityHigh, database.EscalationL2},
	}
	for _, tc := range cases {
		got, err := workflow.EscalationLevel(tc.cat, tc.sev)
		require.NoError(t, err, "%s %s", tc.cat, tc.sev)
		require.Equal(t, tc.lvl, got, "%s %s", tc.cat, tc.sev)
	}
}

func TestEscalationInvalidRejected(t *testing.T) {
	_, err := workflow.EscalationLevel("NOPE", database.SeverityLow)
	require.Error(t, err)
	_, err = workflow.EscalationLevel(database.CategoryNearMiss, "NOPE")
	require.Error(t, err)
}
