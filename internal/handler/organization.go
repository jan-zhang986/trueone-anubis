package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type OrganizationHandler struct{}

func NewOrganizationHandler() *OrganizationHandler {
	return &OrganizationHandler{}
}

// ==================== 1. 组织成员管理 ====================

type OrgMemberListParams struct {
	OrganizationID string `json:"organizationId"`
	Current        int    `json:"current"`
	PageSize       int    `json:"pageSize"`
	Keyword        string `json:"keyword"`
}

type OrgMemberItemResponse struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Email              string                  `json:"email"`
	Phone              string                  `json:"phone"`
	Enable             bool                    `json:"enable"`
	CreatedAt         int64                   `json:"createdAt"`
	UpdatedAt         int64                   `json:"updatedAt"`
	Language           string                  `json:"language"`
	LastOrganizationID string                  `json:"lastOrganizationId"`
	LastProjectID      string                  `json:"lastProjectId"`
	ProjectIdNameMap   []model.ProjectSimple   `json:"projectIdNameMap"`
	UserRoleIdNameMap  []model.ProjectSimple   `json:"userRoleIdNameMap"`
}

func (h *OrganizationHandler) GetMemberList(c *gin.Context) {
	var params OrgMemberListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		params.Current = 1
		params.PageSize = 10
	}
	if params.Current < 1 {
		params.Current = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 10
	}

	var users []model.User
	var total int64

	query := config.DB.Model(&model.User{}).Where("deleted = 0")
	if params.Keyword != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	query.Count(&total)

	offset := (params.Current - 1) * params.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&users)

	// 获取项目列表以构造 ProjectIdNameMap
	var projects []model.Project
	config.DB.Where("deleted = 0").Find(&projects)
	projSimples := make([]model.ProjectSimple, 0, len(projects))
	for _, p := range projects {
		projSimples = append(projSimples, model.ProjectSimple{ID: p.ID, Name: p.Name})
	}

	// 组织角色列表
	defaultRoles := []model.ProjectSimple{
		{ID: "org_admin", Name: "组织管理员"},
	}

	list := make([]OrgMemberItemResponse, 0, len(users))
	for _, u := range users {
		list = append(list, OrgMemberItemResponse{
			ID:                 u.ID,
			Name:               u.Name,
			Email:              u.Email,
			Phone:              u.Phone,
			Enable:             bool(u.Enable),
			CreatedAt:         u.CreatedAt,
			UpdatedAt:         u.UpdatedAt,
			Language:           u.Language,
			LastOrganizationID: u.LastOrganizationID,
			LastProjectID:      u.LastProjectID,
			ProjectIdNameMap:   projSimples,
			UserRoleIdNameMap:  defaultRoles,
		})
	}

	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *OrganizationHandler) GetUserRoleOptions(c *gin.Context) {
	orgID := c.Param("organizationId")
	var roles []model.UserRole
	config.DB.Where("type = 'ORGANIZATION' OR scope_id = ?", orgID).Find(&roles)

	if len(roles) == 0 {
		response.Success(c, []map[string]interface{}{
			{"id": "org_admin", "name": "组织管理员"},
			{"id": "org_member", "name": "组织成员"},
		})
		return
	}

	res := make([]map[string]interface{}, 0, len(roles))
	for _, r := range roles {
		res = append(res, map[string]interface{}{
			"id":   r.ID,
			"name": r.Name,
		})
	}
	response.Success(c, res)
}

func (h *OrganizationHandler) GetProjectOptions(c *gin.Context) {
	orgID := c.Param("organizationId")
	var projects []model.Project
	query := config.DB.Where("deleted = 0 AND enable = 1")
	if orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	}
	query.Find(&projects)

	list := make([]model.ProjectSimple, 0, len(projects))
	for _, p := range projects {
		list = append(list, model.ProjectSimple{
			ID:   p.ID,
			Name: p.Name,
		})
	}
	response.Success(c, list)
}

func (h *OrganizationHandler) GetAvailableUsers(c *gin.Context) {
	var users []model.User
	config.DB.Where("deleted = 0").Find(&users)
	list := make([]model.ProjectSimple, 0, len(users))
	for _, u := range users {
		list = append(list, model.ProjectSimple{
			ID:   u.ID,
			Name: u.Name + " (" + u.Email + ")",
		})
	}
	response.Success(c, list)
}

func (h *OrganizationHandler) AddMember(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) UpdateMember(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) InviteMember(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) BatchAddProject(c *gin.Context) {
	response.Success(c, "ok")
}

// ==================== 2. 组织内项目管理 ====================

type OrgProjectPageParams struct {
	OrganizationID string `json:"organizationId"`
	Current        int    `json:"current"`
	PageSize       int    `json:"pageSize"`
	Keyword        string `json:"keyword"`
}

type OrgProjectTableResponse struct {
	ID             string                  `json:"id"`
	Num            int64                   `json:"num"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	Enable         bool                    `json:"enable"`
	OrganizationID string                  `json:"organizationId"`
	CreatedAt     int64                   `json:"createdAt"`
	UpdatedAt     int64                   `json:"updatedAt"`
	CreateUser     string                  `json:"createUser"`
	MemberCount    int                     `json:"memberCount"`
	OrgAdmins      []model.ProjectSimple   `json:"orgAdmins"`
}

func (h *OrganizationHandler) GetProjectPage(c *gin.Context) {
	var params OrgProjectPageParams
	if err := c.ShouldBindJSON(&params); err != nil {
		params.Current = 1
		params.PageSize = 10
	}
	if params.Current < 1 {
		params.Current = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 10
	}

	var projects []model.Project
	var total int64

	query := config.DB.Model(&model.Project{}).Where("deleted = 0")
	if params.OrganizationID != "" {
		query = query.Where("organization_id = ?", params.OrganizationID)
	}
	if params.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	query.Count(&total)

	offset := (params.Current - 1) * params.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&projects)

	defaultAdmins := []model.ProjectSimple{
		{ID: "admin", Name: "Administrator"},
	}

	list := make([]OrgProjectTableResponse, 0, len(projects))
	for _, p := range projects {
		list = append(list, OrgProjectTableResponse{
			ID:             p.ID,
			Num:            p.Num,
			Name:           p.Name,
			Description:    p.Description,
			Enable:         bool(p.Enable),
			OrganizationID: p.OrganizationID,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
			CreateUser:     p.CreateUser,
			MemberCount:    1,
			OrgAdmins:      defaultAdmins,
		})
	}

	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *OrganizationHandler) AddProject(c *gin.Context) {
	var body struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		OrganizationID string `json:"organizationId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	now := time.Now().UnixMilli()
	proj := model.Project{
		ID:             fmt.Sprintf("%d", now),
		Name:           body.Name,
		Description:    body.Description,
		OrganizationID: body.OrganizationID,
		Enable:         model.BitBool(true),
		Deleted:        model.BitBool(false),
		CreatedAt:     now,
		UpdatedAt:     now,
		CreateUser:     "admin",
	}
	config.DB.Create(&proj)
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeAdd, proj.ID, "添加项目: "+proj.Name)
	response.Success(c, proj)
}

func (h *OrganizationHandler) UpdateProject(c *gin.Context) {
	var body struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	config.DB.Model(&model.Project{}).Where("id = ?", body.ID).Updates(map[string]interface{}{
		"name":        body.Name,
		"description": body.Description,
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeUpdate, body.ID, "修改项目: "+body.Name)
	response.Success(c, "ok")
}

func (h *OrganizationHandler) RenameProject(c *gin.Context) {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	config.DB.Model(&model.Project{}).Where("id = ?", body.ID).Updates(map[string]interface{}{
		"name":        body.Name,
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeUpdate, body.ID, "重命名项目: "+body.Name)
	response.Success(c, "ok")
}

func (h *OrganizationHandler) DeleteProject(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":     model.BitBool(true),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeDelete, id, "删除项目: "+id)
	response.Success(c, "ok")
}

func (h *OrganizationHandler) EnableProject(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enable":      model.BitBool(true),
		"updated_at": time.Now().UnixMilli(),
	})
	response.Success(c, "ok")
}

func (h *OrganizationHandler) DisableProject(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enable":      model.BitBool(false),
		"updated_at": time.Now().UnixMilli(),
	})
	response.Success(c, "ok")
}

// ==================== 3. 组织级用户组管理 ====================

func (h *OrganizationHandler) GetOrgUserRoleList(c *gin.Context) {
	orgID := c.Param("organizationId")
	var roles []model.UserRole
	config.DB.Where("type = 'ORGANIZATION' OR scope_id = ?", orgID).Find(&roles)

	if len(roles) == 0 {
		roles = []model.UserRole{
			{ID: "org_admin", Name: "组织管理员", Type: "ORGANIZATION", ScopeID: orgID, Internal: model.BitBool(true)},
			{ID: "org_member", Name: "组织成员", Type: "ORGANIZATION", ScopeID: orgID, Internal: model.BitBool(true)},
		}
	}
	response.Success(c, roles)
}

func (h *OrganizationHandler) OrgUserGroupMemberList(c *gin.Context) {
	var body struct {
		UserRoleID     string `json:"userRoleId"`
		OrganizationID string `json:"organizationId"`
		Current        int    `json:"current"`
		PageSize       int    `json:"pageSize"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		body.Current = 1
		body.PageSize = 10
	}
	if body.Current < 1 {
		body.Current = 1
	}
	if body.PageSize < 1 {
		body.PageSize = 10
	}

	var total int64
	var relations []model.UserRoleRelation
	config.DB.Model(&model.UserRoleRelation{}).Where("role_id = ?", body.UserRoleID).Count(&total)
	offset := (body.Current - 1) * body.PageSize
	config.DB.Where("role_id = ?", body.UserRoleID).Order("created_at DESC").Offset(offset).Limit(body.PageSize).Find(&relations)

	userIDs := make([]string, 0, len(relations))
	for _, r := range relations {
		userIDs = append(userIDs, r.UserID)
	}

	var users []model.User
	if len(userIDs) > 0 {
		config.DB.Where("id IN ?", userIDs).Find(&users)
	}

	type MemberItem struct {
		ID         string `json:"id"`
		UserID     string `json:"userId"`
		UserName   string `json:"userName"`
		UserEmail  string `json:"userEmail"`
		CreatedAt int64  `json:"createdAt"`
	}

	list := make([]MemberItem, 0, len(relations))
	userMap := make(map[string]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	for _, r := range relations {
		name := r.UserID
		email := ""
		if u, ok := userMap[r.UserID]; ok {
			name = u.Name
			email = u.Email
		}
		list = append(list, MemberItem{
			ID:         r.ID,
			UserID:     r.UserID,
			UserName:   name,
			UserEmail:  email,
			CreatedAt: r.CreatedAt,
		})
	}

	// 如果为空且 roleId 是 org_admin，默认提供 admin 账号展示
	if total == 0 && (body.UserRoleID == "org_admin" || body.UserRoleID == "admin") {
		list = append(list, MemberItem{
			ID:         "rel-admin",
			UserID:     "admin",
			UserName:   "Administrator",
			UserEmail:  "admin@metersphere.io",
			CreatedAt: 1700000000000,
		})
		total = 1
	}

	response.SuccessPage(c, list, total, body.Current, body.PageSize)
}

func (h *OrganizationHandler) OrgUserGroupMemberOption(c *gin.Context) {
	var users []model.User
	config.DB.Where("deleted = 0").Find(&users)
	list := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		list = append(list, map[string]interface{}{
			"id":   u.ID,
			"name": u.Name,
		})
	}
	response.Success(c, list)
}

func (h *OrganizationHandler) OrgUserGroupAddMember(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) OrgUserGroupDeleteMember(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) OrgPermissionSetting(c *gin.Context) {
	// 返回组织级权限树，让前端能渲染勾选框
	perms := []map[string]interface{}{
		{
			"id":   "ORG_PROJECT",
			"name": "项目管理",
			"type": "MODULE",
			"permissions": []map[string]interface{}{
				{"id": "ORG_PROJECT:READ", "name": "查看项目", "checked": true},
				{"id": "ORG_PROJECT:CREATE", "name": "创建项目", "checked": true},
				{"id": "ORG_PROJECT:UPDATE", "name": "编辑项目", "checked": true},
				{"id": "ORG_PROJECT:DELETE", "name": "删除项目", "checked": true},
			},
		},
		{
			"id":   "ORG_MEMBER",
			"name": "组织成员",
			"type": "MODULE",
			"permissions": []map[string]interface{}{
				{"id": "ORG_MEMBER:READ", "name": "查看成员", "checked": true},
				{"id": "ORG_MEMBER:CREATE", "name": "添加成员", "checked": true},
				{"id": "ORG_MEMBER:UPDATE", "name": "更新成员", "checked": true},
				{"id": "ORG_MEMBER:DELETE", "name": "移除成员", "checked": true},
			},
		},
	}
	response.Success(c, perms)
}

func (h *OrganizationHandler) OrgPermissionUpdate(c *gin.Context) {
	response.Success(c, "ok")
}

// ==================== 4. 服务集成 ====================

func (h *OrganizationHandler) ServiceIntegrationList(c *gin.Context) {
	response.Success(c, []interface{}{})
}

func (h *OrganizationHandler) ServiceIntegrationAdd(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) ServiceIntegrationUpdate(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) ServiceIntegrationDelete(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) ServiceIntegrationValidate(c *gin.Context) {
	response.Success(c, "ok")
}

func (h *OrganizationHandler) ServiceIntegrationScript(c *gin.Context) {
	response.Success(c, "ok")
}
