package model

type User struct {
	ID                 string  `gorm:"column:id;primaryKey;size:50" json:"id"`
	Name               string  `gorm:"column:name;size:255" json:"name"`
	Email              string  `gorm:"column:email;size:64" json:"email"`
	Password           string  `gorm:"column:password;size:256" json:"-"`
	Enable             BitBool `gorm:"column:enable" json:"enable"`
	CreatedAt          int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt          int64   `gorm:"column:updated_at" json:"updatedAt"`
	Language           string  `gorm:"column:language;size:30" json:"language"`
	LastOrganizationID string  `gorm:"column:last_organization_id;size:50" json:"lastOrganizationId"`
	Phone              string  `gorm:"column:phone;size:50" json:"phone"`
	Source             string  `gorm:"column:source;size:50" json:"source"`
	LastProjectID      string  `gorm:"column:last_project_id;size:50" json:"lastProjectId"`
	CreateUser         string  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser         string  `gorm:"column:update_user;size:50" json:"updateUser"`
	Deleted            BitBool `gorm:"column:deleted" json:"deleted"`
	CftToken           string  `gorm:"column:cft_token;size:255" json:"cftToken"`
	LarkOpenID         string  `gorm:"column:lark_open_id;size:50" json:"larkOpenId"`
	LarkUnionID        string  `gorm:"column:lark_union_id;size:50" json:"larkUnionId"`
}

func (User) TableName() string {
	return "user"
}

type UserRole struct {
	ID         string  `gorm:"column:id;primaryKey;size:50" json:"id"`
	Name       string  `gorm:"column:name;size:255" json:"name"`
	Type       string  `gorm:"column:type;size:50" json:"type"`
	ScopeID    string  `gorm:"column:scope_id;size:50" json:"scopeId"`
	Internal   BitBool `gorm:"column:internal" json:"internal"`
	CreatedAt  int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt  int64   `gorm:"column:updated_at" json:"updatedAt"`
	CreateUser string  `gorm:"column:create_user;size:50" json:"createUser"`
}

func (UserRole) TableName() string {
	return "user_role"
}

type UserRoleRelation struct {
	ID         string `gorm:"column:id;primaryKey;size:50" json:"id"`
	UserID     string `gorm:"column:user_id;size:50" json:"userId"`
	RoleID     string `gorm:"column:role_id;size:50" json:"roleId"`
	SourceID   string `gorm:"column:source_id;size:50" json:"sourceId"`
	CreatedAt  int64  `gorm:"column:created_at" json:"createdAt"`
	CreateUser string `gorm:"column:create_user;size:50" json:"createUser"`
}

func (UserRoleRelation) TableName() string {
	return "user_role_relation"
}

type UserRolePermission struct {
	ID           string `gorm:"column:id;primaryKey;size:64" json:"id"`
	RoleID       string `gorm:"column:role_id;size:64" json:"roleId"`
	PermissionID string `gorm:"column:permission_id;size:128" json:"permissionId"`
}

func (UserRolePermission) TableName() string {
	return "user_role_permission"
}

type SessionUser struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Email              string             `json:"email"`
	Phone              string             `json:"phone"`
	LastOrganizationID string             `json:"lastOrganizationId"`
	LastProjectID      string             `json:"lastProjectId"`
	Language           string             `json:"language"`
	SessionID          string             `json:"sessionId"`
	CsrfToken          string             `json:"csrfToken"`
	UserRoles          []UserRole         `json:"userRoles"`
	UserRoleRelations  []UserRoleRelation `json:"userRoleRelations"`
	Permissions        []string           `json:"permissions"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
