package model

type CaseRepository struct {
	ID            string `gorm:"column:id;primaryKey;size:50" json:"id"`
	ProjectID     string `gorm:"column:project_id;size:50" json:"projectId"`
	Name          string `gorm:"column:name;size:255" json:"name"`
	Code          string `gorm:"column:code;size:100" json:"code"`
	DefaultBranch string `gorm:"column:default_branch;size:100" json:"defaultBranch"`
	Branches      string `gorm:"column:branches;type:json" json:"branches"`
	Description   string `gorm:"column:description;size:2000" json:"description"`
	GitURL        string `gorm:"column:git_url;size:500" json:"gitUrl"`
	GitPlatform   string `gorm:"column:git_platform;size:50" json:"gitPlatform"` // "github" | "gitlab" | "local"
	TestsDir      string `gorm:"column:tests_dir;size:255" json:"testsDir"`
	LocalPath     string `gorm:"column:local_path;size:500" json:"localPath"`
	GitToken      string `gorm:"column:git_token;size:500" json:"gitToken,omitempty"`
	CaseCount     int    `gorm:"column:case_count" json:"caseCount"`
	CreatedAt     int64  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt     int64  `gorm:"column:updated_at" json:"updatedAt"`
}

func (CaseRepository) TableName() string {
	return "case_repository"
}
