package model

type QualityWorkspace struct {
	WorkspaceID      string  `gorm:"column:workspace_id;primaryKey;size:50" json:"workspaceId"`
	ProjectID        string  `gorm:"column:project_id;size:50" json:"projectId"`
	Name             string  `gorm:"column:name;size:255" json:"name"`
	TargetType       string  `gorm:"column:target_type;size:32" json:"targetType"`
	TargetID         string  `gorm:"column:target_id;size:100" json:"targetId"`
	TargetName       string  `gorm:"column:target_name;size:255" json:"targetName"`
	Goal             string  `gorm:"column:goal;size:2000" json:"goal"`
	Description      string  `gorm:"column:description;size:4000" json:"description"`
	OwnerID          string  `gorm:"column:owner_id;size:50" json:"ownerId"`
	Status           string  `gorm:"column:status;size:32" json:"status"`
	WorkflowID       string  `gorm:"column:workflow_id;size:50" json:"workflowId"`
	PlannedStartTime *int64  `gorm:"column:planned_start_time" json:"plannedStartTime"`
	PlannedEndTime   *int64  `gorm:"column:planned_end_time" json:"plannedEndTime"`
	ActualStartTime  *int64  `gorm:"column:actual_start_time" json:"actualStartTime"`
	ActualEndTime    *int64  `gorm:"column:actual_end_time" json:"actualEndTime"`
	Tags             string  `gorm:"column:tags;type:json" json:"tags"`
	ScopeDefinition  string  `gorm:"column:scope_definition;type:json" json:"scopeDefinition"`
	Metadata         string  `gorm:"column:metadata;type:json" json:"metadata"`
	PrdDocumentID    string  `gorm:"column:prd_document_id;size:50" json:"prdDocumentId"`
	CreateUser       string  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser       string  `gorm:"column:update_user;size:50" json:"updateUser"`
	CreatedAt        int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        int64   `gorm:"column:updated_at" json:"updatedAt"`
	Archived         BitBool `gorm:"column:archived" json:"archived"`
}

func (QualityWorkspace) TableName() string {
	return "quality_workspace"
}

type QualityTask struct {
	TaskID      string `gorm:"column:task_id;primaryKey;size:50" json:"taskId"`
	WorkspaceID string `gorm:"column:workspace_id;size:50" json:"workspaceId"`
	ProjectID   string `gorm:"column:project_id;size:50" json:"projectId"`
	TaskType    string `gorm:"column:task_type;size:32" json:"taskType"`
	Title       string `gorm:"column:title;size:255" json:"title"`
	Description string `gorm:"column:description;size:2000" json:"description"`
	Status      string `gorm:"column:status;size:32" json:"status"`
	OwnerID     string `gorm:"column:owner_id;size:50" json:"ownerId"`
	SuiteID     string `gorm:"column:suite_id;size:50" json:"suiteId"`
	SourceType  string `gorm:"column:source_type;size:32" json:"sourceType"`
	SourceID    string `gorm:"column:source_id;size:50" json:"sourceId"`
	Sort        int64  `gorm:"column:sort" json:"sort"`
	Metadata    string `gorm:"column:metadata;type:json" json:"metadata"`
	CreateUser  string `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser  string `gorm:"column:update_user;size:50" json:"updateUser"`
	CreatedAt   int64  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt   int64  `gorm:"column:updated_at" json:"updatedAt"`
}

func (QualityTask) TableName() string {
	return "quality_task"
}

type QualityWorkItem struct {
	WorkItemID       string `gorm:"column:work_item_id;primaryKey;size:50" json:"workItemId"`
	WorkspaceID      string `gorm:"column:workspace_id;size:50" json:"workspaceId"`
	TaskID           string `gorm:"column:task_id;size:50" json:"taskId"`
	ProjectID        string `gorm:"column:project_id;size:50" json:"projectId"`
	CaseID           string `gorm:"column:case_id;size:50" json:"caseId"`
	ImplementationID string `gorm:"column:implementation_id;size:50" json:"implementationId"`
	SourceSpaceID    string `gorm:"column:source_space_id;size:50" json:"sourceSpaceId"`
	Title            string `gorm:"column:title;size:255" json:"title"`
	Status           string `gorm:"column:status;size:32" json:"status"`
	AssigneeID       string `gorm:"column:assignee_id;size:50" json:"assigneeId"`
	WorkflowID       string `gorm:"column:workflow_id;size:50" json:"workflowId"`
	Result           string `gorm:"column:result;size:32" json:"result"`
	EvidenceCount    int    `gorm:"column:evidence_count" json:"evidenceCount"`
	RuntimeSnapshot  string `gorm:"column:runtime_snapshot;type:json" json:"runtimeSnapshot"`
	Metadata         string `gorm:"column:metadata;type:json" json:"metadata"`
	CreateUser       string `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser       string `gorm:"column:update_user;size:50" json:"updateUser"`
	CreatedAt        int64  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        int64  `gorm:"column:updated_at" json:"updatedAt"`
}

func (QualityWorkItem) TableName() string {
	return "quality_work_item"
}

type QualityReport struct {
	ReportID        string  `gorm:"column:report_id;primaryKey;size:50" json:"reportId"`
	ProjectID       string  `gorm:"column:project_id;size:50" json:"projectId"`
	WorkspaceID     string  `gorm:"column:workspace_id;size:50" json:"workspaceId"`
	TaskID          string  `gorm:"column:task_id;size:50" json:"taskId"`
	ReportType      string  `gorm:"column:report_type;size:32" json:"reportType"`
	Name            string  `gorm:"column:name;size:255" json:"name"`
	Status          string  `gorm:"column:status;size:32" json:"status"`
	VersionNo       int     `gorm:"column:version_no" json:"versionNo"`
	IsLatest        BitBool `gorm:"column:is_latest" json:"isLatest"`
	SnapshotJSON    string  `gorm:"column:snapshot_json;type:json" json:"snapshotJson"`
	MarkdownContent string  `gorm:"column:markdown_content;type:mediumtext" json:"markdownContent"`
	Metadata        string  `gorm:"column:metadata;type:json" json:"metadata"`
	CreateUser      string  `gorm:"column:create_user;size:50" json:"createUser"`
	UpdateUser      string  `gorm:"column:update_user;size:50" json:"updateUser"`
	CreatedAt       int64   `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt       int64   `gorm:"column:updated_at" json:"updatedAt"`
}

func (QualityReport) TableName() string {
	return "quality_report"
}

type QualityWorkspaceStats struct {
	WorkspaceID       string  `json:"workspaceId"`
	Total             int     `json:"total"`
	Todo              int     `json:"todo"`
	InProgress        int     `json:"inProgress"`
	Passed            int     `json:"passed"`
	Failed            int     `json:"failed"`
	Blocked           int     `json:"blocked"`
	Skipped           int     `json:"skipped"`
	PassRate          float64 `json:"passRate"`
	ExecutionRate     float64 `json:"executionRate"`
	ActualStartTime   *int64  `json:"actualStartTime,omitempty"`
	AllDone           bool    `json:"allDone"`
	AnalysisStatus    string  `json:"analysisStatus"`
	ReviewStatus      string  `json:"reviewStatus"`
	CheckItemTotal    int     `json:"checkItemTotal"`
	RiskCount         int     `json:"riskCount"`
	BlockedCount      int     `json:"blockedCount"`
	ReleaseConclusion string  `json:"releaseConclusion"`
}


// --- Test Plan Living Diff Matrix Models ---

type DiffMatrixEvidence struct {
	Assertion   string `json:"assertion"`
	DbStateDiff string `json:"dbStateDiff,omitempty"`
	TraceLog    string `json:"traceLog,omitempty"`
}

type DiffMatrixTestCase struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Lang        string             `json:"lang"`
	FilePath    string             `json:"filePath"`
	LineNo      int                `json:"lineNo"`
	Status      string             `json:"status"` // PASSED, FAILED, MISSING
	DurationMs  int                `json:"durationMs,omitempty"`
	CodeSnippet string             `json:"codeSnippet"`
	Evidence    DiffMatrixEvidence `json:"evidence"`
}

type DiffMatrixUncoveredNotice struct {
	Reason           string `json:"reason"`
	OnlineMonitoring string `json:"onlineMonitoring"`
}

type DiffMatrixAnalysisSection struct {
	ID                    string                     `json:"id"`
	Title                 string                     `json:"title"`
	RiskLevel             string                     `json:"riskLevel"` // P0, P1, P2
	RiskTag               string                     `json:"riskTag"`
	RiskDescription       string                     `json:"riskDescription"`
	ImpactScope           []string                   `json:"impactScope"`
	Strategy              string                     `json:"strategy"`
	VerificationChecklist []string                   `json:"verificationChecklist"`
	Cases                 []DiffMatrixTestCase       `json:"cases"`
	UncoveredNotice       *DiffMatrixUncoveredNotice `json:"uncoveredNotice,omitempty"`
}

type DiffMatrixRequirementSection struct {
	ID            string                    `json:"id"`
	SectionNumber string                    `json:"sectionNumber"`
	LineStart     int                       `json:"lineStart"`
	LineEnd       int                       `json:"lineEnd"`
	Title         string                    `json:"title"`
	Paragraphs    []string                  `json:"paragraphs"`
	DiffStatus    string                    `json:"diffStatus"` // COVERED, WARNING, GAP
	Analysis      DiffMatrixAnalysisSection `json:"analysis"`
}

type DiffMatrixData struct {
	WorkspaceID  string                         `json:"workspaceId"`
	RepoURL      string                         `json:"repoUrl"`
	GitBranch    string                         `json:"gitBranch"`
	CommitSHA    string                         `json:"commitSha"`
	PrdPath      string                         `json:"prdPath"`
	LastSyncedAt int64                          `json:"lastSyncedAt"`
	Sections     []DiffMatrixRequirementSection `json:"sections"`
}

type DiffMatrixSyncRequest struct {
	RepoURL   string                         `json:"repoUrl"`
	GitBranch string                         `json:"gitBranch"`
	CommitSHA string                         `json:"commitSha"`
	PrdPath   string                         `json:"prdPath"`
	Sections  []DiffMatrixRequirementSection `json:"sections"`
}

type DiffMatrixCaseReportRequest struct {
	CaseID        string `json:"caseId"`
	SectionNumber string `json:"sectionNumber"`
	Status        string `json:"status"`
	DurationMs    int    `json:"durationMs"`
	Assertion     string `json:"assertion"`
	DbStateDiff   string `json:"dbStateDiff"`
	TraceLog      string `json:"traceLog"`
}
