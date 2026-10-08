package handler

import (
	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type BugHandler struct {
	bugService *service.BugService
}

func NewBugHandler() *BugHandler {
	return &BugHandler{
		bugService: service.NewBugService(),
	}
}

func (h *BugHandler) GetBugPage(c *gin.Context) {
	var params service.BugPageParams
	_ = c.ShouldBindJSON(&params)
	if params.ProjectID == "" {
		params.ProjectID = c.GetHeader("PROJECT")
	}

	list, total, err := h.bugService.GetBugPage(params)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *BugHandler) GetBugDetail(c *gin.Context) {
	id := c.Param("id")
	bug, err := h.bugService.GetBugDetail(id)
	if err != nil {
		response.Fail(c, 404, "缺陷不存在")
		return
	}
	response.Success(c, bug)
}

func (h *BugHandler) AddBug(c *gin.Context) {
	var bug model.Bug
	if err := c.ShouldBindJSON(&bug); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	if bug.ProjectID == "" {
		bug.ProjectID = c.GetHeader("PROJECT")
	}

	err := h.bugService.AddBug(&bug, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleBug, service.OpTypeAdd, bug.ID, "创建缺陷: "+bug.Title)
	response.Success(c, bug)
}

func (h *BugHandler) UpdateBug(c *gin.Context) {
	var bug model.Bug
	if err := c.ShouldBindJSON(&bug); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.bugService.UpdateBug(&bug, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleBug, service.OpTypeUpdate, bug.ID, "修改缺陷: "+bug.Title)
	response.Success(c, "更新成功")
}

func (h *BugHandler) DeleteBug(c *gin.Context) {
	id := c.Param("id")
	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.bugService.DeleteBug(id, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleBug, service.OpTypeDelete, id, "删除缺陷: "+id)
	response.Success(c, "删除成功")
}

func (h *BugHandler) CurrentPlatform(c *gin.Context) {
	response.Success(c, "LOCAL")
}

func (h *BugHandler) CustomFieldHeader(c *gin.Context) {
	response.Success(c, []interface{}{})
}

func (h *BugHandler) TemplateOption(c *gin.Context) {
	response.Success(c, []interface{}{})
}

func (h *BugHandler) ColumnsOption(c *gin.Context) {
	response.Success(c, []interface{}{})
}

func (h *BugHandler) FeishuOptions(c *gin.Context) {
	response.Success(c, []interface{}{})
}

