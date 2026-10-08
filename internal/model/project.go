package model

type Organization struct {
	ID          string  `gorm:"column:id;primaryKey;size:50" json:"id"`
	Num         int64   `gorm:"column:num" json:"num"`
	Name        string  `gorm:"column:name;size:255" json:"name"`
	Description string  `gorm:"column:description;size:1000" json:"description"`
	CreatedAt   int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   int64   `gorm:"column:updated_at" json:"updatedAt"`
	CreateUser  string  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser  string  `gorm:"column:update_user;size:50" json:"updateUser"`
	Deleted     BitBool `gorm:"column:deleted" json:"deleted"`
	Enable      BitBool `gorm:"column:enable" json:"enable"`
}

func (Organization) TableName() string {
	return "organization"
}

type Project struct {
	ID              string  `gorm:"column:id;primaryKey;size:50" json:"id"`
	Num             int64   `gorm:"column:num" json:"num"`
	OrganizationID  string  `gorm:"column:organization_id;size:50" json:"organizationId"`
	Name            string  `gorm:"column:name;size:255" json:"name"`
	Description     string  `gorm:"column:description;size:1000" json:"description"`
	CreatedAt       int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt       int64   `gorm:"column:updated_at" json:"updatedAt"`
	CreateUser      string  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser      string  `gorm:"column:update_user;size:50" json:"updateUser"`
	Deleted         BitBool `gorm:"column:deleted" json:"deleted"`
	Enable          BitBool `gorm:"column:enable" json:"enable"`
	ModuleSetting   string  `gorm:"column:module_setting;size:255" json:"moduleSetting"`
	AllResourcePool BitBool `gorm:"column:all_resource_pool" json:"allResourcePool"`
}

func (Project) TableName() string {
	return "project"
}

type ProjectSimple struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
