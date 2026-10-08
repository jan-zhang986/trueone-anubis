package handler

import (
	"github.com/gin-gonic/gin"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type TestCaseHandler struct {
	caseService *service.TestCaseService
}

func NewTestCaseHandler() *TestCaseHandler {
	return &TestCaseHandler{
		caseService: service.NewTestCaseService(),
	}
}

func (h *TestCaseHandler) GetCasePage(c *gin.Context) {
	var params service.CasePageParams
	_ = c.ShouldBindJSON(&params)
	if params.ProjectID == "" {
		params.ProjectID = c.GetHeader("PROJECT")
	}

	list, total, err := h.caseService.GetCasePage(params)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *TestCaseHandler) GetCaseDetail(c *gin.Context) {
	var body struct {
		ID string `json:"id"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.ID == "" {
		body.ID = c.Query("id")
	}

	tc, err := h.caseService.GetCaseDetail(body.ID)
	if err != nil {
		response.Fail(c, 404, "用例不存在")
		return
	}
	response.Success(c, tc)
}

func (h *TestCaseHandler) AddCase(c *gin.Context) {
	var tc model.FunctionalCase
	if err := c.ShouldBindJSON(&tc); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	if tc.ProjectID == "" {
		tc.ProjectID = c.GetHeader("PROJECT")
	}

	err := h.caseService.AddCase(&tc, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleCase, service.OpTypeAdd, tc.ID, "创建用例: "+tc.Name)
	response.Success(c, tc)
}

func (h *TestCaseHandler) UpdateCase(c *gin.Context) {
	var tc model.FunctionalCase
	if err := c.ShouldBindJSON(&tc); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.caseService.UpdateCase(&tc, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleCase, service.OpTypeUpdate, tc.ID, "修改用例: "+tc.Name)
	response.Success(c, "更新成功")
}

func (h *TestCaseHandler) DeleteCase(c *gin.Context) {
	var body struct {
		ID string `json:"id"`
	}
	_ = c.ShouldBindJSON(&body)

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.caseService.DeleteCase(body.ID, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleCase, service.OpTypeDelete, body.ID, "删除用例: "+body.ID)
	response.Success(c, "删除成功")
}

func (h *TestCaseHandler) GetModuleTree(c *gin.Context) {
	projectId := c.Query("projectId")
	if projectId == "" {
		projectId = c.GetHeader("PROJECT")
	}

	tree, err := h.caseService.GetModuleTree(projectId)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, tree)
}

func (h *TestCaseHandler) AddModule(c *gin.Context) {
	var m model.FunctionalCaseModule
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	if m.ProjectID == "" {
		m.ProjectID = c.GetHeader("PROJECT")
	}

	err := h.caseService.AddModule(&m, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleCase, service.OpTypeAdd, m.ID, "创建模块: "+m.Name)
	response.Success(c, m)
}

func (h *TestCaseHandler) UpdateModule(c *gin.Context) {
	var m model.FunctionalCaseModule
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.caseService.UpdateModule(&m, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleCase, service.OpTypeUpdate, m.ID, "修改模块: "+m.Name)
	response.Success(c, "更新成功")
}

func (h *TestCaseHandler) DeleteModule(c *gin.Context) {
	var body struct {
		ID string `json:"id"`
	}
	_ = c.ShouldBindJSON(&body)

	err := h.caseService.DeleteModule(body.ID)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleCase, service.OpTypeDelete, body.ID, "删除模块: "+body.ID)
	response.Success(c, "删除成功")
}

func (h *TestCaseHandler) CustomField(c *gin.Context) {
	response.Success(c, []interface{}{})
}

func (h *TestCaseHandler) DefaultTemplateField(c *gin.Context) {
	response.Success(c, []interface{}{})
}

func (h *TestCaseHandler) ModuleCount(c *gin.Context) {
	var body struct {
		ProjectID string `json:"projectId"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.ProjectID == "" {
		body.ProjectID = c.GetHeader("PROJECT")
	}

	type countResult struct {
		ModuleID string
		Count    int64
	}
	var counts []countResult
	config.DB.Model(&model.FunctionalCase{}).
		Select("module_id, count(*) as count").
		Where("project_id = ? AND deleted = 0", body.ProjectID).
		Group("module_id").
		Scan(&counts)

	var total int64
	config.DB.Model(&model.FunctionalCase{}).
		Where("project_id = ? AND deleted = 0", body.ProjectID).
		Count(&total)

	res := make(map[string]int64)
	res["all"] = total
	for _, cnt := range counts {
		res[cnt.ModuleID] = cnt.Count
	}
	response.Success(c, res)
}

func (h *TestCaseHandler) TrashModuleCount(c *gin.Context) {
	response.Success(c, gin.H{
		"all": 0,
	})
}

func (h *TestCaseHandler) GetModuleTreeByParam(c *gin.Context) {
	projectId := c.Param("projectId")
	if projectId == "" {
		projectId = c.GetHeader("PROJECT")
	}
	tree, err := h.caseService.GetModuleTree(projectId)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, tree)
}

