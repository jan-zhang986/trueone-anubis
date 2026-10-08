package model

type FunctionalCase struct {
	ID                string  `gorm:"column:id;primaryKey;size:50" json:"id"`
	Num               int64   `gorm:"column:num" json:"num"`
	ModuleID          string  `gorm:"column:module_id;size:50" json:"moduleId"`
	ProjectID         string  `gorm:"column:project_id;size:50" json:"projectId"`
	TemplateID        string  `gorm:"column:template_id;size:50" json:"templateId"`
	Name              string  `gorm:"column:name;size:255" json:"name"`
	ReviewStatus      string  `gorm:"column:review_status;size:64" json:"reviewStatus"`
	Tags              string  `gorm:"column:tags;size:1000" json:"tags"`
	CaseEditType      string  `gorm:"column:case_edit_type;size:50" json:"caseEditType"`
	Pos               int64   `gorm:"column:pos" json:"pos"`
	VersionID         string  `gorm:"column:version_id;size:50" json:"versionId"`
	RefID             string  `gorm:"column:ref_id;size:50" json:"refId"`
	SourceCaseID      string  `gorm:"column:source_case_id;size:50" json:"sourceCaseId"`
	ReuseType         string  `gorm:"column:reuse_type;size:32" json:"reuseType"`
	WorkflowID        string  `gorm:"column:workflow_id;size:50" json:"workflowId"`
	LastExecuteResult string  `gorm:"column:last_execute_result;size:64" json:"lastExecuteResult"`
	Deleted           BitBool `gorm:"column:deleted" json:"deleted"`
	PublicCase        BitBool `gorm:"column:public_case" json:"publicCase"`
	Latest            BitBool `gorm:"column:latest" json:"latest"`
	CreateUser        string  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser        string  `gorm:"column:update_user;size:50" json:"updateUser"`
	DeleteUser        string  `gorm:"column:delete_user;size:50" json:"deleteUser"`
	CreatedAt         int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt         int64   `gorm:"column:updated_at" json:"updatedAt"`
	DeleteTime        *int64  `gorm:"column:delete_time" json:"deleteTime"`
	AiCreate          BitBool `gorm:"column:ai_create" json:"aiCreate"`
}

func (FunctionalCase) TableName() string {
	return "functional_case"
}

type FunctionalCaseModule struct {
	ID         string                  `gorm:"column:id;primaryKey;size:50" json:"id"`
	ProjectID  string                  `gorm:"column:project_id;size:50" json:"projectId"`
	Name       string                  `gorm:"column:name;size:255" json:"name"`
	ParentID   string                  `gorm:"column:parent_id;size:50" json:"parentId"`
	Pos        int64                   `gorm:"column:pos" json:"pos"`
	CreatedAt  int64                   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt  int64                   `gorm:"column:updated_at" json:"updatedAt"`
	CreateUser string                  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser string                  `gorm:"column:update_user;size:50" json:"updateUser"`
	Children   []*FunctionalCaseModule `gorm:"-" json:"children,omitempty"`
	CaseCount  int64                   `gorm:"-" json:"caseCount"`
}

func (FunctionalCaseModule) TableName() string {
	return "functional_case_module"
}
