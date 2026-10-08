package handler

import (
	"strings"
	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type QualityWorkspaceHandler struct {
	wsService *service.QualityWorkspaceService
}

func NewQualityWorkspaceHandler() *QualityWorkspaceHandler {
	return &QualityWorkspaceHandler{
		wsService: service.NewQualityWorkspaceService(),
	}
}

func (h *QualityWorkspaceHandler) GetPage(c *gin.Context) {
	var params service.WorkspacePageParams
	_ = c.ShouldBindJSON(&params)
	if params.ProjectID == "" {
		params.ProjectID = c.GetHeader("PROJECT")
	}

	list, total, err := h.wsService.GetPage(params)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *QualityWorkspaceHandler) GetDetail(c *gin.Context) {
	id := c.Param("id")
	ws, err := h.wsService.GetDetail(id)
	if err != nil {
		response.Fail(c, 404, "工作台不存在")
		return
	}
	response.Success(c, ws)
}

func (h *QualityWorkspaceHandler) Save(c *gin.Context) {
	var ws model.QualityWorkspace
	if err := c.ShouldBindJSON(&ws); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	if ws.ProjectID == "" {
		if pid := c.GetHeader("PROJECT"); pid != "" {
			ws.ProjectID = pid
		}
	}

	err := h.wsService.Save(&ws, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleQualityWorkspace, service.OpTypeAdd, ws.WorkspaceID, "保存工作空间: "+ws.Name)
	response.Success(c, ws)
}

func (h *QualityWorkspaceHandler) Archive(c *gin.Context) {
	id := c.Param("id")
	err := h.wsService.Archive(id)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleQualityWorkspace, service.OpTypeUpdate, id, "归档工作空间: "+id)
	response.Success(c, "已归档")
}

func (h *QualityWorkspaceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.wsService.Delete(id)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleQualityWorkspace, service.OpTypeDelete, id, "删除工作空间: "+id)
	response.Success(c, "已删除")
}

func (h *QualityWorkspaceHandler) GetStats(c *gin.Context) {
	id := c.Param("id")
	stats, err := h.wsService.GetStats(id)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, stats)
}

func (h *QualityWorkspaceHandler) GetTaskList(c *gin.Context) {
	id := c.Param("id")
	tasks, err := h.wsService.GetTaskList(id)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, tasks)
}

func (h *QualityWorkspaceHandler) SaveTask(c *gin.Context) {
	workspaceId := c.Param("id")
	var task model.QualityTask
	if err := c.ShouldBindJSON(&task); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	task.WorkspaceID = workspaceId

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.wsService.SaveTask(&task, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, task)
}

func (h *QualityWorkspaceHandler) CompleteTask(c *gin.Context) {
	workspaceId := c.Param("id")
	taskId := c.Param("taskId")
	err := h.wsService.CompleteTask(workspaceId, taskId)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, "任务已完成")
}

func (h *QualityWorkspaceHandler) ReopenTask(c *gin.Context) {
	workspaceId := c.Param("id")
	taskId := c.Param("taskId")
	err := h.wsService.ReopenTask(workspaceId, taskId)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, "任务已重开")
}

func (h *QualityWorkspaceHandler) GetWorkItemPage(c *gin.Context) {
	workspaceId := c.Param("id")
	taskId := c.Param("taskId")

	var params service.WorkItemPageParams
	_ = c.ShouldBindJSON(&params)
	params.WorkspaceID = workspaceId
	params.TaskID = taskId

	list, total, err := h.wsService.GetWorkItemPage(params)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *QualityWorkspaceHandler) RunWorkItem(c *gin.Context) {
	workspaceId := c.Param("id")
	taskId := c.Param("taskId")
	workItemId := c.Param("workItemId")

	var body struct {
		Result string `json:"result"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Result == "" {
		body.Result = "PASSED"
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.wsService.RunWorkItem(workspaceId, taskId, workItemId, body.Result, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, "执行记录已更新")
}

func (h *QualityWorkspaceHandler) GetReportPage(c *gin.Context) {
	var params service.ReportPageParams
	_ = c.ShouldBindJSON(&params)
	if params.ProjectID == "" {
		params.ProjectID = c.GetHeader("PROJECT")
	}

	list, total, err := h.wsService.GetReportPage(params)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *QualityWorkspaceHandler) GetReportDetail(c *gin.Context) {
	reportId := c.Param("reportId")
	rep, err := h.wsService.GetReportDetail(reportId)
	if err != nil {
		response.Fail(c, 404, "报告不存在")
		return
	}
	response.Success(c, rep)
}

func (h *QualityWorkspaceHandler) GenerateReport(c *gin.Context) {
	workspaceId := c.Param("id")
	var body struct {
		ReportType string `json:"reportType"`
		TaskID     string `json:"taskId"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.ReportType == "" {
		body.ReportType = "OVERVIEW"
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	rep, err := h.wsService.GenerateReport(workspaceId, body.ReportType, body.TaskID, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, rep)
}


func (h *QualityWorkspaceHandler) GetDiffMatrix(c *gin.Context) {
	workspaceId := c.Param("id")
	matrix, err := h.wsService.GetDiffMatrix(workspaceId)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, matrix)
}

func (h *QualityWorkspaceHandler) SyncDiffMatrix(c *gin.Context) {
	workspaceId := c.Param("id")
	var req model.DiffMatrixSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数格式错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	matrix, err := h.wsService.SyncDiffMatrix(workspaceId, req, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleQualityWorkspace, service.OpTypeUpdate, workspaceId, "同步测试计划对账矩阵资产")
	response.Success(c, matrix)
}

func (h *QualityWorkspaceHandler) ReportCaseExecution(c *gin.Context) {
	workspaceId := c.Param("id")
	var req model.DiffMatrixCaseReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数格式错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	matrix, err := h.wsService.ReportCaseExecution(workspaceId, req, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, matrix)
}


func (h *QualityWorkspaceHandler) GenerateDiffMatrix(c *gin.Context) {
	workspaceId := c.Param("id")
	var req struct {
		MarkdownContent string `json:"markdownContent"`
		PrdTitle        string `json:"prdTitle"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.MarkdownContent) == "" {
		response.Fail(c, 400, "markdownContent 不能为空")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	matrix, err := h.wsService.GenerateDiffMatrixFromMarkdown(workspaceId, req.MarkdownContent, req.PrdTitle, uid)
	if err != nil {
		response.Fail(c, 500, "大模型分析生成失败: "+err.Error())
		return
	}

	service.RecordAuditLog(c, service.ModuleQualityWorkspace, service.OpTypeUpdate, workspaceId, "AI 架构师在线生成需求差分对账矩阵")
	response.Success(c, matrix)
}

func (h *QualityWorkspaceHandler) AutoParseDiffMatrix(c *gin.Context) {
	workspaceId := c.Param("id")
	var req struct {
		RepoPath string `json:"repoPath"`
	}
	_ = c.ShouldBindJSON(&req)

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	matrix, err := h.wsService.AutoParseDiffMatrix(workspaceId, req.RepoPath, uid)
	if err != nil {
		response.Fail(c, 500, "云端自动解析与同步失败: "+err.Error())
		return
	}

	service.RecordAuditLog(c, service.ModuleQualityWorkspace, service.OpTypeUpdate, workspaceId, "云端自主解析并同步代码AST与对账矩阵")
	response.Success(c, matrix)
}
