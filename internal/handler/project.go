package handler

import (
	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type ProjectHandler struct {
	projectService *service.ProjectService
	envService     *service.EnvironmentService
}

func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{
		projectService: service.NewProjectService(),
		envService:     service.NewEnvironmentService(),
	}
}

func (h *ProjectHandler) GetProjectsByOrg(c *gin.Context) {
	orgId := c.Param("orgId")
	projects, err := h.projectService.GetProjectsByOrg(orgId)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, projects)
}

func (h *ProjectHandler) GetPublicProjects(c *gin.Context) {
	projects, err := h.projectService.GetPublicProjects()
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, projects)
}

func (h *ProjectHandler) GetProjectDetail(c *gin.Context) {
	id := c.Param("id")
	p, err := h.projectService.GetProjectDetail(id)
	if err != nil {
		response.Fail(c, 404, "项目不存在")
		return
	}
	response.Success(c, p)
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var body struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		OrganizationID string `json:"organizationId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	orgId := body.OrganizationID
	if orgId == "" {
		orgId = "100001"
	}

	p, err := h.projectService.CreateProject(body.Name, body.Description, orgId, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleProject, service.OpTypeAdd, p.ID, "创建项目: "+p.Name)
	response.Success(c, p)
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if id == "" {
		id = body.ID
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.projectService.UpdateProject(id, body.Name, body.Description, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleProject, service.OpTypeUpdate, id, "修改项目: "+body.Name)
	response.Success(c, "更新成功")
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	id := c.Param("id")
	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.projectService.DeleteProject(id, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleProject, service.OpTypeDelete, id, "删除项目: "+id)
	response.Success(c, "删除成功")
}

// Environments
func (h *ProjectHandler) GetEnvironmentList(c *gin.Context) {
	var body struct {
		ProjectID string `json:"projectId"`
		Keyword   string `json:"keyword"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.ProjectID == "" {
		body.ProjectID = c.Query("projectId")
	}

	envs, err := h.envService.GetList(body.ProjectID, body.Keyword)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, envs)
}

func (h *ProjectHandler) GetEnvironmentDetail(c *gin.Context) {
	id := c.Param("id")
	env, err := h.envService.GetDetail(id)
	if err != nil {
		response.Fail(c, 404, "环境不存在")
		return
	}
	response.Success(c, env)
}

func (h *ProjectHandler) AddEnvironment(c *gin.Context) {
	var env model.Environment
	if err := c.ShouldBindJSON(&env); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.envService.Add(&env, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleEnvironment, service.OpTypeAdd, env.ID, "添加运行环境: "+env.Name)
	response.Success(c, env)
}

func (h *ProjectHandler) UpdateEnvironment(c *gin.Context) {
	var env model.Environment
	if err := c.ShouldBindJSON(&env); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	userId, _ := c.Get("userId")
	uid := "admin"
	if userId != nil {
		uid = userId.(string)
	}

	err := h.envService.Update(&env, uid)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleEnvironment, service.OpTypeUpdate, env.ID, "更新运行环境: "+env.Name)
	response.Success(c, "更新成功")
}

func (h *ProjectHandler) DeleteEnvironment(c *gin.Context) {
	id := c.Param("id")
	err := h.envService.Delete(id)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleEnvironment, service.OpTypeDelete, id, "删除运行环境: "+id)
	response.Success(c, "删除成功")
}
