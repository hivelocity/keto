package keto

import "gitlab.host1plus.com/linas/keto/sdk/go/keto/swagger"

type RoleSDK interface {
	AddMembersToRole(id string, body swagger.RoleMembers) (*swagger.APIResponse, error)
	DeleteRole(id string) (*swagger.APIResponse, error)
	CreateRole(body swagger.Role) (*swagger.Role, *swagger.APIResponse, error)
	GetRole(id string) (*swagger.Role, *swagger.APIResponse, error)
	ListRoles(member string, limit int64, offset int64) ([]swagger.Role, *swagger.APIResponse, error)
	RemoveMembersFromRole(id string, body swagger.RoleMembers) (*swagger.APIResponse, error)
}
