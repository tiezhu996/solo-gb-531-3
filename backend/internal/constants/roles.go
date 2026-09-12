package constants

type Role string

const (
	RoleAdmin           Role = "admin"
	RoleProcessEngineer Role = "process_engineer"
	RoleSafetyReviewer  Role = "safety_reviewer"
	RoleAuditor         Role = "auditor"
)

const (
	PermissionRead       = "read"
	PermissionNodeWrite  = "node:write"
	PermissionScenario   = "scenario:write"
	PermissionReview     = "scenario:review"
	PermissionSafeguard  = "safeguard:write"
	PermissionEvaluation = "evaluation:run"
	PermissionConfirm    = "evaluation:confirm"
)

var rolePermissions = map[Role]map[string]struct{}{
	RoleAdmin: {
		PermissionRead: {}, PermissionNodeWrite: {}, PermissionScenario: {},
		PermissionReview: {}, PermissionSafeguard: {}, PermissionEvaluation: {}, PermissionConfirm: {},
	},
	RoleProcessEngineer: {
		PermissionRead: {}, PermissionNodeWrite: {}, PermissionScenario: {},
		PermissionSafeguard: {}, PermissionEvaluation: {},
	},
	RoleSafetyReviewer: {
		PermissionRead: {}, PermissionReview: {}, PermissionSafeguard: {}, PermissionEvaluation: {}, PermissionConfirm: {},
	},
	RoleAuditor: {PermissionRead: {}},
}

func (r Role) Valid() bool {
	_, ok := rolePermissions[r]
	return ok
}

func HasPermission(role Role, permission string) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = permissions[permission]
	return ok
}

func RoleValues() []string {
	return []string{string(RoleAdmin), string(RoleProcessEngineer), string(RoleSafetyReviewer), string(RoleAuditor)}
}
