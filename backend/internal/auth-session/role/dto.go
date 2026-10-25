package role

type RoleRequest struct {
	Code          string   `json:"code"           validate:"required,max=50"`
	Name          string   `json:"name"           validate:"required,max=100"`
	Description   *string  `json:"description"    validate:"omitempty,max=1000"`
	PermissionIDs []string `json:"permission_ids" validate:"required,dive,uuid"`
}

type RoleResponse struct {
	ID            string   `json:"id"`
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	Description   *string  `json:"description"`
	PermissionIDs []string `json:"permission_ids"`
}

type LookupResponse struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name,omitempty"`
	Module string `json:"module,omitempty"`
	Action string `json:"action,omitempty"`
}
