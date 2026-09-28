package rbac_test

import (
	"testing"

	"aegis/pkg/database"
	"aegis/pkg/rbac"

	"github.com/stretchr/testify/require"
)

func TestHasPermissionMatrix(t *testing.T) {
	cases := []struct {
		role database.Role
		perm rbac.Permission
		want bool
	}{
		{database.RoleReporter, rbac.UsersWrite, false},
		{database.RoleReporter, rbac.LocationsWrite, false},
		{database.RoleReporter, rbac.LocationsRead, true},
		{database.RoleReporter, rbac.IncidentsRead, true},
		{database.RoleReporter, rbac.IncidentsWrite, true},
		{database.RoleReporter, rbac.IncidentsVerify, false},
		{database.RoleReporter, rbac.IncidentsClose, false},
		{database.RoleAdmin, rbac.UsersWrite, true},
		{database.RoleAdmin, rbac.LocationsWrite, true},
		{database.RoleAdmin, rbac.IncidentsClose, false},
		{database.RoleSupervisor, rbac.IncidentsVerify, true},
		{database.RoleHSEOfficer, rbac.IncidentsReject, true},
		{database.RoleHSEOfficer, rbac.IncidentsVerify, false},
		{database.RoleHSEOfficer, rbac.RCAWrite, true},
		{database.RoleHSEOfficer, rbac.CAWrite, true},
		{database.RoleHSEOfficer, rbac.CAVerify, true},
		{database.RoleReporter, rbac.RCAWrite, false},
		{database.RoleSupervisor, rbac.RCAWrite, false},
		{database.RoleAdmin, rbac.RCAWrite, false},
		{database.RoleHSEManager, rbac.IncidentsClose, true},
		{database.RoleHSEManager, rbac.UsersWrite, false},
		{database.RoleSuperAdmin, rbac.UsersWrite, true},
		{database.RoleSuperAdmin, rbac.LocationsWrite, true},
		{database.RoleSuperAdmin, rbac.IncidentsClose, true},
		{database.RoleSuperAdmin, rbac.FilesWrite, true},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, rbac.Has(tc.role, tc.perm), "%s %s", tc.role, tc.perm)
	}

	for _, perm := range rbac.AllPermissions() {
		require.True(t, rbac.Has(database.RoleSuperAdmin, perm), "super admin missing %s", perm)
	}
}
