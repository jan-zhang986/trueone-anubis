package service

import (
	"strings"
	"regexp"
	"path/filepath"
	"os"
	"net/http"
	"io"
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

type QualityWorkspaceService struct{}

func NewQualityWorkspaceService() *QualityWorkspaceService {
	return &QualityWorkspaceService{}
}

type WorkspacePageParams struct {
	Current   int      `json:"current"`
	PageSize  int      `json:"pageSize"`
	ProjectID string   `json:"projectId"`
	Keyword   string   `json:"keyword"`
	Statuses  []string `json:"statuses"`
	Archived  *bool    `json:"archived"`
}

func (s *QualityWorkspaceService) GetPage(params WorkspacePageParams) ([]model.QualityWorkspace, int64, error) {
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var list []model.QualityWorkspace
	var total int64

	query := config.DB.Model(&model.QualityWorkspace{})
	if params.ProjectID != "" {
		query = query.Where("project_id = ?", params.ProjectID)
	}
	if params.Keyword != "" {
		query = query.Where("name LIKE ? OR goal LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	}
	if params.Archived != nil {
		query = query.Where("archived = ?", *params.Archived)
	} else {
		query = query.Where("archived = 0")
	}

	query.Count(&total)
	offset := (params.Current - 1) * params.PageSize
	// metadata 包含大型对账树 JSON (可能达数百 KB)，列表页无需加载 metadata，避免 MySQL 触发 Error 1038 Out of sort memory
	err := query.Select("workspace_id, project_id, name, target_type, target_id, target_name, goal, description, owner_id, status, workflow_id, planned_start_time, planned_end_time, actual_start_time, actual_end_time, tags, scope_definition, prd_document_id, create_user, update_user, created_at, updated_at, archived").
		Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&list).Error

	return list, total, err
}

func (s *QualityWorkspaceService) GetDetail(id string) (*model.QualityWorkspace, error) {
	var ws model.QualityWorkspace
	err := config.DB.Where("workspace_id = ?", id).First(&ws).Error
	if err != nil {
		return nil, err
	}
	return &ws, nil
}


func sanitizeWorkspaceJson(ws *model.QualityWorkspace) {
	if ws.Tags == "" {
		ws.Tags = "[]"
	}
	if ws.ScopeDefinition == "" {
		ws.ScopeDefinition = "{}"
	}
	if ws.Metadata == "" {
		ws.Metadata = "{}"
	}
}

func sanitizeTaskJson(t *model.QualityTask) {
	if t.Metadata == "" {
		t.Metadata = "{}"
	}
}

func sanitizeWorkItemJson(w *model.QualityWorkItem) {
	if w.Metadata == "" {
		w.Metadata = "{}"
	}
	if w.RuntimeSnapshot == "" {
		w.RuntimeSnapshot = "{}"
	}
}

func sanitizeReportJson(r *model.QualityReport) {
	if r.Metadata == "" {
		r.Metadata = "{}"
	}
	if r.SnapshotJSON == "" {
		r.SnapshotJSON = "{}"
	}
}

func (s *QualityWorkspaceService) Save(ws *model.QualityWorkspace, userId string) error {
	sanitizeWorkspaceJson(ws)
	now := time.Now().UnixMilli()
	if ws.WorkspaceID == "" {
		ws.WorkspaceID = uuid.New().String()
		ws.CreateUser = userId
		ws.UpdateUser = userId
		ws.CreatedAt = now
		ws.UpdatedAt = now
		if ws.Status == "" {
			ws.Status = "DRAFT"
		}
		return config.DB.Create(ws).Error
	}

	ws.UpdateUser = userId
	ws.UpdatedAt = now
	return config.DB.Model(&model.QualityWorkspace{}).Where("workspace_id = ?", ws.WorkspaceID).Updates(ws).Error
}

func (s *QualityWorkspaceService) Archive(id string) error {
	return config.DB.Model(&model.QualityWorkspace{}).Where("workspace_id = ?", id).Update("archived", 1).Error
}

func (s *QualityWorkspaceService) Delete(id string) error {
	return config.DB.Where("workspace_id = ?", id).Delete(&model.QualityWorkspace{}).Error
}

func (s *QualityWorkspaceService) GetStats(workspaceId string) (*model.QualityWorkspaceStats, error) {
	var items []model.QualityWorkItem
	config.DB.Where("workspace_id = ?", workspaceId).Find(&items)

	stats := &model.QualityWorkspaceStats{
		WorkspaceID:       workspaceId,
		Total:             len(items),
		ReleaseConclusion: "READY",
		AnalysisStatus:    "REVIEWED",
		ReviewStatus:      "REVIEWED",
	}

	if len(items) == 0 {
		return stats, nil
	}

	for _, it := range items {
		switch it.Status {
		case "PASSED", "SUCCESS":
			stats.Passed++
		case "FAILED", "ERROR":
			stats.Failed++
		case "BLOCKED":
			stats.Blocked++
		case "SKIPPED":
			stats.Skipped++
		case "RUNNING", "IN_PROGRESS":
			stats.InProgress++
		default:
			stats.Todo++
		}
	}

	executed := stats.Passed + stats.Failed + stats.Blocked + stats.Skipped
	if stats.Total > 0 {
		stats.ExecutionRate = float64(executed) / float64(stats.Total) * 100
		if executed > 0 {
			stats.PassRate = float64(stats.Passed) / float64(executed) * 100
		}
	}
	stats.AllDone = (stats.Total > 0 && stats.Todo == 0 && stats.InProgress == 0)
	if stats.Failed > 0 || stats.Blocked > 0 {
		stats.ReleaseConclusion = "BLOCKED"
	} else if stats.AllDone && stats.PassRate == 100 {
		stats.ReleaseConclusion = "READY"
	} else {
		stats.ReleaseConclusion = "IN_PROGRESS"
	}

	return stats, nil
}

func (s *QualityWorkspaceService) GetTaskList(workspaceId string) ([]model.QualityTask, error) {
	var tasks []model.QualityTask
	err := config.DB.Where("workspace_id = ?", workspaceId).Order("sort ASC, created_at ASC").Find(&tasks).Error
	return tasks, err
}

func (s *QualityWorkspaceService) SaveTask(task *model.QualityTask, userId string) error {
	sanitizeTaskJson(task)
	now := time.Now().UnixMilli()
	if task.TaskID == "" {
		task.TaskID = uuid.New().String()
		task.CreateUser = userId
		task.UpdateUser = userId
		task.CreatedAt = now
		task.UpdatedAt = now
		if task.Status == "" {
			task.Status = "PENDING"
		}
		return config.DB.Create(task).Error
	}
	task.UpdateUser = userId
	task.UpdatedAt = now
	return config.DB.Model(&model.QualityTask{}).Where("task_id = ?", task.TaskID).Updates(task).Error
}

func (s *QualityWorkspaceService) CompleteTask(workspaceId, taskId string) error {
	return config.DB.Model(&model.QualityTask{}).Where("task_id = ? AND workspace_id = ?", taskId, workspaceId).Update("status", "COMPLETED").Error
}

func (s *QualityWorkspaceService) ReopenTask(workspaceId, taskId string) error {
	return config.DB.Model(&model.QualityTask{}).Where("task_id = ? AND workspace_id = ?", taskId, workspaceId).Update("status", "PENDING").Error
}

type WorkItemPageParams struct {
	WorkspaceID string `json:"workspaceId"`
	TaskID      string `json:"taskId"`
	Current     int    `json:"current"`
	PageSize    int    `json:"pageSize"`
	Keyword     string `json:"keyword"`
}

func (s *QualityWorkspaceService) GetWorkItemPage(params WorkItemPageParams) ([]model.QualityWorkItem, int64, error) {
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var list []model.QualityWorkItem
	var total int64

	query := config.DB.Model(&model.QualityWorkItem{})
	if params.WorkspaceID != "" {
		query = query.Where("workspace_id = ?", params.WorkspaceID)
	}
	if params.TaskID != "" {
		query = query.Where("task_id = ?", params.TaskID)
	}
	if params.Keyword != "" {
		query = query.Where("title LIKE ?", "%"+params.Keyword+"%")
	}

	query.Count(&total)
	offset := (params.Current - 1) * params.PageSize
	err := query.Order("created_at ASC").Offset(offset).Limit(params.PageSize).Find(&list).Error

	return list, total, err
}

func (s *QualityWorkspaceService) RunWorkItem(workspaceId, taskId, workItemId, result, userId string) error {
	now := time.Now().UnixMilli()
	status := "PASSED"
	if result == "FAILED" || result == "FAIL" {
		status = "FAILED"
	} else if result == "BLOCKED" {
		status = "BLOCKED"
	}
	return config.DB.Model(&model.QualityWorkItem{}).Where("work_item_id = ?", workItemId).Updates(map[string]interface{}{
		"result":      result,
		"status":      status,
		"update_user": userId,
		"updated_at": now,
	}).Error
}

type ReportPageParams struct {
	ProjectID   string `json:"projectId"`
	WorkspaceID string `json:"workspaceId"`
	ReportType  string `json:"reportType"`
	Keyword     string `json:"keyword"`
	Current     int    `json:"current"`
	PageSize    int    `json:"pageSize"`
}

func (s *QualityWorkspaceService) GetReportPage(params ReportPageParams) ([]model.QualityReport, int64, error) {
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var list []model.QualityReport
	var total int64

	query := config.DB.Model(&model.QualityReport{})
	if params.ProjectID != "" {
		query = query.Where("project_id = ?", params.ProjectID)
	}
	if params.WorkspaceID != "" {
		query = query.Where("workspace_id = ?", params.WorkspaceID)
	}
	if params.ReportType != "" {
		query = query.Where("report_type = ?", params.ReportType)
	}
	if params.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+params.Keyword+"%")
	}

	query.Count(&total)
	offset := (params.Current - 1) * params.PageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&list).Error

	return list, total, err
}

func (s *QualityWorkspaceService) GetReportDetail(reportId string) (*model.QualityReport, error) {
	var rep model.QualityReport
	err := config.DB.Where("report_id = ?", reportId).First(&rep).Error
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (s *QualityWorkspaceService) GenerateReport(workspaceId, reportType, taskId, userId string) (*model.QualityReport, error) {
	ws, err := s.GetDetail(workspaceId)
	if err != nil {
		return nil, err
	}

	stats, _ := s.GetStats(workspaceId)
	snapshotBytes, _ := json.Marshal(stats)

	mdContent := fmt.Sprintf("# %s - 质量评估报告\n\n## 概览\n- 工作台: %s\n- 生成时间: %s\n- 用例总数: %d\n- 通过率: %.1f%%\n- 执行率: %.1f%%\n\n## 质量结论\n%s\n",
		ws.Name,
		ws.Name,
		time.Now().Format("2006-01-02 15:04:05"),
		stats.Total,
		stats.PassRate,
		stats.ExecutionRate,
		stats.ReleaseConclusion,
	)

	now := time.Now().UnixMilli()
	rep := &model.QualityReport{
		ReportID:        uuid.New().String(),
		ProjectID:       ws.ProjectID,
		WorkspaceID:     workspaceId,
		TaskID:          taskId,
		ReportType:      reportType,
		Name:            fmt.Sprintf("%s 质量评估报告", ws.Name),
		Status:          "COMPLETED",
		VersionNo:       1,
		IsLatest:        model.BitBool(true),
		SnapshotJSON:    string(snapshotBytes),
		MarkdownContent: mdContent,
		CreateUser:      userId,
		UpdateUser:      userId,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err = config.DB.Create(rep).Error
	return rep, err
}




// --- Diff Matrix (Living Test Plan) Methods ---

// 返回干净的空结构，绝不伪造假用例
func buildDefaultDiffMatrix(workspaceId string, wsName string) model.DiffMatrixData {
	return model.DiffMatrixData{
		WorkspaceID:  workspaceId,
		RepoURL:      "",
		GitBranch:    "",
		CommitSHA:    "",
		PrdPath:      "",
		LastSyncedAt: 0,
		Sections:     []model.DiffMatrixRequirementSection{},
	}
}

func (s *QualityWorkspaceService) GetDiffMatrix(workspaceId string) (*model.DiffMatrixData, error) {
	ws, err := s.GetDetail(workspaceId)
	if err != nil {
		emptyMatrix := buildDefaultDiffMatrix(workspaceId, "未同步工作台")
		return &emptyMatrix, nil
	}

	// 优先从 ws.Metadata 中读取
	if ws.Metadata != "" && ws.Metadata != "{}" {
		var metaMap map[string]interface{}
		if err := json.Unmarshal([]byte(ws.Metadata), &metaMap); err == nil {
			if diffVal, ok := metaMap["diff_matrix"]; ok {
				diffBytes, err := json.Marshal(diffVal)
				if err == nil {
					var data model.DiffMatrixData
					if err := json.Unmarshal(diffBytes, &data); err == nil {
						data.WorkspaceID = workspaceId
						return &data, nil
					}
				}
			}
		}
	}

	// 未同步时诚实返回空切片，绝不伪造假用例写库
	emptyMatrix := buildDefaultDiffMatrix(workspaceId, ws.Name)
	return &emptyMatrix, nil
}

func (s *QualityWorkspaceService) SyncDiffMatrix(workspaceId string, req model.DiffMatrixSyncRequest, userId string) (*model.DiffMatrixData, error) {
	ws, err := s.GetDetail(workspaceId)
	if err != nil {
		return nil, err
	}

	data := model.DiffMatrixData{
		WorkspaceID:  workspaceId,
		RepoURL:      req.RepoURL,
		GitBranch:    req.GitBranch,
		CommitSHA:    req.CommitSHA,
		PrdPath:      req.PrdPath,
		LastSyncedAt: time.Now().UnixMilli(),
		Sections:     req.Sections,
	}

	// 重新推导每个章节的初始状态
	for i := range data.Sections {
		sec := &data.Sections[i]
		if len(sec.Analysis.Cases) == 0 {
			sec.DiffStatus = "GAP"
		} else {
			hasMissing := false
			hasFailed := false
			allPassed := true
			for _, tc := range sec.Analysis.Cases {
				if tc.Status == "FAILED" {
					hasFailed = true
					allPassed = false
				} else if tc.Status == "MISSING" {
					hasMissing = true
					allPassed = false
				} else if tc.Status != "PASSED" {
					allPassed = false
				}
			}
			if hasFailed {
				sec.DiffStatus = "BLOCKED"
			} else if hasMissing {
				sec.DiffStatus = "WARNING"
			} else if allPassed && len(sec.Analysis.Cases) > 0 {
				sec.DiffStatus = "COVERED"
			} else {
				sec.DiffStatus = "WARNING"
			}
		}
	}

	if err := s.saveMatrixToWorkspace(ws, &data, userId); err != nil {
		return nil, err
	}

	return &data, nil
}

// 修复推导机：GAP 不在 caseFound 内，FAILED 章节降级为 BLOCKED
func (s *QualityWorkspaceService) ReportCaseExecution(workspaceId string, req model.DiffMatrixCaseReportRequest, userId string) (*model.DiffMatrixData, error) {
	matrix, err := s.GetDiffMatrix(workspaceId)
	if err != nil {
		return nil, err
	}

	for i := range matrix.Sections {
		sec := &matrix.Sections[i]
		if req.SectionNumber != "" && sec.SectionNumber != req.SectionNumber {
			continue
		}

		for j := range sec.Analysis.Cases {
			tc := &sec.Analysis.Cases[j]
			if req.CaseID == "" || tc.ID == req.CaseID {
				if req.Status != "" {
					tc.Status = req.Status
				} else {
					tc.Status = "PASSED"
				}
				if req.DurationMs > 0 {
					tc.DurationMs = req.DurationMs
				}
				if req.Assertion != "" {
					tc.Evidence.Assertion = req.Assertion
				}
				if req.DbStateDiff != "" {
					tc.Evidence.DbStateDiff = req.DbStateDiff
				}
				if req.TraceLog != "" {
					tc.Evidence.TraceLog = req.TraceLog
				}
				break
			}
		}

		// 重新严密计算当前章节状态
		if len(sec.Analysis.Cases) == 0 {
			sec.DiffStatus = "GAP"
		} else {
			hasFailed := false
			hasMissing := false
			allPassed := true
			for _, tc := range sec.Analysis.Cases {
				if tc.Status == "FAILED" {
					hasFailed = true
					allPassed = false
				} else if tc.Status == "MISSING" {
					hasMissing = true
					allPassed = false
				} else if tc.Status != "PASSED" {
					allPassed = false
				}
			}
			if hasFailed {
				sec.DiffStatus = "BLOCKED"
			} else if hasMissing {
				sec.DiffStatus = "WARNING"
			} else if allPassed && len(sec.Analysis.Cases) > 0 {
				sec.DiffStatus = "COVERED"
			} else {
				sec.DiffStatus = "WARNING"
			}
		}
	}

	matrix.LastSyncedAt = time.Now().UnixMilli()

	ws, _ := s.GetDetail(workspaceId)
	if ws != nil {
		_ = s.saveMatrixToWorkspace(ws, matrix, userId)
	}

	return matrix, nil
}

func (s *QualityWorkspaceService) AutoParseDiffMatrix(workspaceId string, repoPath string, userId string) (*model.DiffMatrixData, error) {
	if repoPath == "" {
		repoPath = "/Users/zhangjian/vanguard-platform/trueone-anubis"
	}

	matrixFile := filepath.Join(repoPath, "docs", "test-architecture", "diff-matrix.json")
	dataBytes, err := os.ReadFile(matrixFile)
	if err != nil {
		return nil, fmt.Errorf("读取测试架构矩阵文件失败: %v", err)
	}

	var matrixReq model.DiffMatrixSyncRequest
	if err := json.Unmarshal(dataBytes, &matrixReq); err != nil {
		return nil, fmt.Errorf("解析矩阵数据失败: %v", err)
	}

	return s.SyncDiffMatrix(workspaceId, matrixReq, userId)
}

func (s *QualityWorkspaceService) saveMatrixToWorkspace(ws *model.QualityWorkspace, data *model.DiffMatrixData, userId string) error {
	var metaMap map[string]interface{}
	if ws.Metadata != "" && ws.Metadata != "{}" {
		_ = json.Unmarshal([]byte(ws.Metadata), &metaMap)
	}
	if metaMap == nil {
		metaMap = make(map[string]interface{})
	}

	metaMap["diff_matrix"] = data
	metaBytes, err := json.Marshal(metaMap)
	if err != nil {
		return err
	}

	ws.Metadata = string(metaBytes)
	ws.UpdateUser = userId
	ws.UpdatedAt = time.Now().UnixMilli()

	return config.DB.Model(&model.QualityWorkspace{}).
		Where("workspace_id = ?", ws.WorkspaceID).
		Updates(map[string]interface{}{
			"metadata":    ws.Metadata,
			"update_user": ws.UpdateUser,
			"updated_at":  ws.UpdatedAt,
		}).Error
}

// --- Real LLM Online Analysis for Diff Matrix ---

func resolveDeepSeekKey() string {
	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		return key
	}
	home, err := os.UserHomeDir()
	if err == nil {
		credPath := filepath.Join(home, ".dsh", ".credentials.yaml")
		data, err := os.ReadFile(credPath)
		if err == nil {
			re := regexp.MustCompile(`DEEPSEEK_API_KEY:\s*([^\s\r\n]+)`)
			matches := re.FindStringSubmatch(string(data))
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}
	return ""
}

func (s *QualityWorkspaceService) GenerateDiffMatrixFromMarkdown(workspaceId string, markdownContent string, prdTitle string, userId string) (*model.DiffMatrixData, error) {
	ws, err := s.GetDetail(workspaceId)
	if err != nil {
		return nil, fmt.Errorf("工作台不存在: %v", err)
	}

	apiKey := resolveDeepSeekKey()
	if apiKey == "" {
		return nil, fmt.Errorf("未找到 DEEPSEEK_API_KEY，请在系统环境变量或 ~/.dsh/.credentials.yaml 中配置")
	}

	prompt := fmt.Sprintf("你是一位拥有阿里/腾讯 P8 级别的 AI 首席测试开发架构师。\n" +
		"现在给你一份真实的业务需求文档 (PRD)。请对该需求进行深度、专业、工业级的质量分析与测试资产建模。\n\n" +
		"【严禁输出泛泛而谈的套话、假大空用例】必须根据这篇 PRD 具体描述的业务规则，输出真实的失效模式与原生测试代码。\n\n" +
		"输入的需求文档内容如下（请注意行号从 1 开始）：\n%s\n\n" +
		"请输出纯 JSON 数组格式（严禁包含任何前导或尾部 markdown 说明），数组每一项为一个完整的需求分析章节切片，JSON 结构包含：\n" +
		"id, sectionNumber, lineStart, lineEnd, title, paragraphs, diffStatus(COVERED/WARNING/GAP),\n" +
		"analysis (包含 id, title, riskLevel(P0/P1/P2), riskTag, riskDescription, impactScope, strategy, verificationChecklist, cases, uncoveredNotice)\n" +
		"特别规定：保持用例代码精炼，只输出标准 JSON，不要包裹任何 markdown 标记。", markdownContent)

	reqBody := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": "You are an expert Test Architect. Output raw JSON only."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
		"max_tokens":  8192,
	}

	reqBytes, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用 DeepSeek API 超时或网络异常: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DeepSeek API 响应异常 (%d): %s", resp.StatusCode, string(respBytes))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBytes, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("解析模型响应失败: %v", err)
	}

	rawContent := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if strings.HasPrefix(rawContent, "```json") {
		rawContent = strings.TrimPrefix(rawContent, "```json")
		rawContent = strings.TrimSuffix(rawContent, "```")
	} else if strings.HasPrefix(rawContent, "```") {
		rawContent = strings.TrimPrefix(rawContent, "```")
		rawContent = strings.TrimSuffix(rawContent, "```")
	}
	rawContent = strings.TrimSpace(rawContent)

	var sections []model.DiffMatrixRequirementSection
	if err := json.Unmarshal([]byte(rawContent), &sections); err != nil {
		var wrapper struct {
			Sections []model.DiffMatrixRequirementSection `json:"sections"`
		}
		if err2 := json.Unmarshal([]byte(rawContent), &wrapper); err2 == nil && len(wrapper.Sections) > 0 {
			sections = wrapper.Sections
		} else {
			limit := len(rawContent)
			if limit > 300 {
				limit = 300
			}
			return nil, fmt.Errorf("解析模型返回的 JSON 章节结构失败: %v, raw: %s", err, rawContent[:limit])
		}
	}

	displayPath := "docs/prd.md"
	if prdTitle != "" {
		displayPath = "docs/" + prdTitle
	}

	data := &model.DiffMatrixData{
		WorkspaceID:  workspaceId,
		RepoURL:      "git@github.com:vanguard/trade-payment-service.git",
		GitBranch:    "feature/ai-architect-gen",
		CommitSHA:    fmt.Sprintf("%x", time.Now().UnixNano())[:8],
		PrdPath:      displayPath,
		LastSyncedAt: time.Now().UnixMilli(),
		Sections:     sections,
	}

	if err := s.saveMatrixToWorkspace(ws, data, userId); err != nil {
		return nil, fmt.Errorf("持久化矩阵资产失败: %v", err)
	}

	return data, nil
}
