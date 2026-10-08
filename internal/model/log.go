package model

type OperationLog struct {
	ID             int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ProjectID      string `gorm:"column:project_id;size:50" json:"projectId"`
	OrganizationID string `gorm:"column:organization_id;size:50" json:"organizationId"`
	CreatedAt      int64  `gorm:"column:created_at" json:"createdAt"`
	CreateUser     string `gorm:"column:create_user;size:50" json:"createUser"`
	SourceID       string `gorm:"column:source_id;size:50" json:"sourceId"`
	Method         string `gorm:"column:method;size:255" json:"method"`
	Type           string `gorm:"column:type;size:20" json:"type"`
	Module         string `gorm:"column:module;size:100" json:"module"`
	Content        string `gorm:"column:content;size:500" json:"content"`
	Path           string `gorm:"column:path;size:255" json:"path"`
}

func (OperationLog) TableName() string {
	return "operation_log"
}
