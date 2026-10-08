package model

type Bug struct {
	ID                  string  `gorm:"column:id;primaryKey;size:50" json:"id"`
	Num                 int     `gorm:"column:num" json:"num"`
	Title               string  `gorm:"column:title;size:255" json:"title"`
	HandleUsers         string  `gorm:"column:handle_users;size:1000" json:"handleUsers"`
	HandleUser          string  `gorm:"column:handle_user;size:50" json:"handleUser"`
	CreateUser          string  `gorm:"column:create_user;size:50" json:"createUser"`
	CreatedAt           int64   `gorm:"column:created_at" json:"createdAt"`
	UpdateUser          string  `gorm:"column:update_user;size:50" json:"updateUser"`
	UpdatedAt           int64   `gorm:"column:updated_at" json:"updatedAt"`
	ProjectID           string  `gorm:"column:project_id;size:50" json:"projectId"`
	TemplateID          string  `gorm:"column:template_id;size:50" json:"templateId"`
	Platform            string  `gorm:"column:platform;size:50" json:"platform"`
	Status              string  `gorm:"column:status;size:50" json:"status"`
	Tags                string  `gorm:"column:tags;size:1000" json:"tags"`
	PlatformBugID       string  `gorm:"column:platform_bug_id;size:50" json:"platformBugId"`
	DeleteUser          string  `gorm:"column:delete_user;size:50" json:"deleteUser"`
	DeleteTime          *int64  `gorm:"column:delete_time" json:"deleteTime"`
	Deleted             BitBool `gorm:"column:deleted" json:"deleted"`
	Pos                 int64   `gorm:"column:pos" json:"pos"`
	FeishuStoryID       string  `gorm:"column:feishu_story_id;size:100" json:"feishuStoryId"`
	DefectType          string  `gorm:"column:defect_type;size:50" json:"defectType"`
	DefectReason        string  `gorm:"column:defect_reason;size:255" json:"defectReason"`
	AppID               string  `gorm:"column:app_id;size:50" json:"appId"`
	AffectedAppIDs      string  `gorm:"column:affected_app_ids;size:500" json:"affectedAppIds"`
	DiscoveryPhase      string  `gorm:"column:discovery_phase;size:50" json:"discoveryPhase"`
	BusinessLine        string  `gorm:"column:business_line;size:50" json:"businessLine"`
	DiscoveryDifficulty string  `gorm:"column:discovery_difficulty;size:50" json:"discoveryDifficulty"`
	ReopenCount         int     `gorm:"column:reopen_count" json:"reopenCount"`
	Discoverer          string  `gorm:"column:discoverer;size:100" json:"discoverer"`
	ActualTime          *int64  `gorm:"column:actual_time" json:"actualTime"`
}

func (Bug) TableName() string {
	return "bug"
}
