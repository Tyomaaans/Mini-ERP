package role

import "mini-erp/internal/auth-session/domain"

func ToRoleResponse(e *domain.RoleEntity) *RoleResponse {
	return &RoleResponse{
		ID:            e.ID,
		Code:          e.Code,
		Name:          e.Name,
		Description:   e.Description,
		PermissionIDs: e.PermissionIDs,
	}
}

func ToLookupResponse(e domain.LookupEntity) LookupResponse {
	return LookupResponse(e)
}
