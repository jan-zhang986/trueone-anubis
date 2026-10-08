package model

type Environment struct {
	ID          string  `gorm:"column:id;primaryKey;size:50" json:"id"`
	Name        string  `gorm:"column:name;size:255" json:"name"`
	ProjectID   string  `gorm:"column:project_id;size:50" json:"projectId"`
	CreateUser  string  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser  string  `gorm:"column:update_user;size:50" json:"updateUser"`
	CreatedAt   int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   int64   `gorm:"column:updated_at" json:"updatedAt"`
	Mock        BitBool `gorm:"column:mock" json:"mock"`
	Description string  `gorm:"column:description;size:1000" json:"description"`
	Pos         int64   `gorm:"column:pos" json:"pos"`
}

func (Environment) TableName() string {
	return "environment"
}
