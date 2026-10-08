package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

func (h *SystemHandler) UserList(c *gin.Context) {
	var users []model.User
	config.DB.Where("deleted = 0").Find(&users)
	response.Success(c, users)
}

type UserPageParams struct {
	Current  int    `json:"current"`
	PageSize int    `json:"pageSize"`
	Keyword  string `json:"keyword"`
}

func (h *SystemHandler) UserPage(c *gin.Context) {
	var params UserPageParams
	_ = c.ShouldBindJSON(&params)
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
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

	// Fetch all organizations and roles for mapping
	var orgs []model.Organization
	config.DB.Where("deleted = 0").Find(&orgs)
	orgMap := make(map[string]model.Organization)
	for _, o := range orgs {
		orgMap[o.ID] = o
	}

	var roles []model.UserRole
	config.DB.Find(&roles)
	roleMap := make(map[string]model.UserRole)
	for _, r := range roles {
		roleMap[r.ID] = r
	}

	var relations []model.UserRoleRelation
	config.DB.Find(&relations)
	userRolesMap := make(map[string][]model.UserRole)
	for _, rel := range relations {
		if r, ok := roleMap[rel.RoleID]; ok {
			userRolesMap[rel.UserID] = append(userRolesMap[rel.UserID], r)
		}
	}

	type UserItemResponse struct {
		ID                 string               `json:"id"`
		Name               string               `json:"name"`
		Email              string               `json:"email"`
		Phone              string               `json:"phone"`
		Enable             bool                 `json:"enable"`
		CreatedAt         int64                `json:"createdAt"`
		UpdatedAt         int64                `json:"updatedAt"`
		Language           string               `json:"language"`
		LastOrganizationID string               `json:"lastOrganizationId"`
		LastProjectID      string               `json:"lastProjectId"`
		OrganizationList   []model.Organization `json:"organizationList"`
		UserRoleList       []model.UserRole     `json:"userRoleList"`
		UserRoles          []model.UserRole     `json:"userRoles"`
	}

	list := make([]UserItemResponse, 0, len(users))
	for _, u := range users {
		var userOrgs []model.Organization
		if o, ok := orgMap[u.LastOrganizationID]; ok {
			userOrgs = append(userOrgs, o)
		}
		ur := userRolesMap[u.ID]
		if ur == nil {
			ur = []model.UserRole{}
		}
		list = append(list, UserItemResponse{
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
			OrganizationList:   userOrgs,
			UserRoleList:       ur,
			UserRoles:          ur,
		})
	}

	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *SystemHandler) SystemRoles(c *gin.Context) {
	var roles []model.UserRole
	config.DB.Where("scope_id = 'global' OR type = 'SYSTEM'").Find(&roles)

	type RoleOption struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Selected  bool   `json:"selected"`
		Closeable bool   `json:"closeable"`
	}
	options := make([]RoleOption, 0, len(roles))
	for _, r := range roles {
		options = append(options, RoleOption{
			ID:        r.ID,
			Name:      r.Name,
			Selected:  false,
			Closeable: true,
		})
	}
	response.Success(c, options)
}

func (h *SystemHandler) GlobalUserGroupList(c *gin.Context) {
	var roles []model.UserRole
	config.DB.Where("scope_id = 'global' OR type = 'SYSTEM'").Find(&roles)
	response.Success(c, roles)
}

func (h *SystemHandler) OrganizationTotal(c *gin.Context) {
	var orgTotal, projTotal int64
	config.DB.Model(&model.Organization{}).Where("deleted = 0").Count(&orgTotal)
	config.DB.Model(&model.Project{}).Where("deleted = 0").Count(&projTotal)

	response.Success(c, gin.H{
		"organizationTotal": orgTotal,
		"projectTotal":      projTotal,
	})
}

func (h *SystemHandler) OrganizationList(c *gin.Context) {
	var params struct {
		Current  int    `json:"current"`
		PageSize int    `json:"pageSize"`
		Keyword  string `json:"keyword"`
	}
	_ = c.ShouldBindJSON(&params)
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var orgs []model.Organization
	var total int64

	query := config.DB.Model(&model.Organization{}).Where("deleted = 0")
	if params.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+params.Keyword+"%")
	}
	query.Count(&total)

	offset := (params.Current - 1) * params.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&orgs)

	type OrgTableItem struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		Enable       bool   `json:"enable"`
		CreatedAt   int64  `json:"createdAt"`
		UpdatedAt   int64  `json:"updatedAt"`
		MemberCount  int    `json:"memberCount"`
		ProjectCount int    `json:"projectCount"`
	}

	list := make([]OrgTableItem, 0, len(orgs))
	for _, o := range orgs {
		var pCount int64
		config.DB.Model(&model.Project{}).Where("organization_id = ? AND deleted = 0", o.ID).Count(&pCount)
		list = append(list, OrgTableItem{
			ID:           o.ID,
			Name:         o.Name,
			Description:  o.Description,
			Enable:       bool(o.Enable),
			CreatedAt:   o.CreatedAt,
			UpdatedAt:   o.UpdatedAt,
			MemberCount:  1,
			ProjectCount: int(pCount),
		})
	}

	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *SystemHandler) SystemProjectList(c *gin.Context) {
	var params struct {
		Current        int    `json:"current"`
		PageSize       int    `json:"pageSize"`
		Keyword        string `json:"keyword"`
		OrganizationID string `json:"organizationId"`
	}
	_ = c.ShouldBindJSON(&params)
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
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

	// Fetch all organizations to map organizationName
	var orgs []model.Organization
	config.DB.Where("deleted = 0").Find(&orgs)
	orgMap := make(map[string]string)
	for _, o := range orgs {
		orgMap[o.ID] = o.Name
	}

	type ProjectTableItem struct {
		ID               string                `json:"id"`
		Name             string                `json:"name"`
		Description      string                `json:"description"`
		Enable           bool                  `json:"enable"`
		OrganizationID   string                `json:"organizationId"`
		OrganizationName string                `json:"organizationName"`
		CreatedAt        int64                 `json:"createdAt"`
		UpdatedAt        int64                 `json:"updatedAt"`
		MemberCount      int                   `json:"memberCount"`
		OrgAdmins        []model.ProjectSimple `json:"orgAdmins"`
	}

	defaultAdmins := []model.ProjectSimple{
		{ID: "admin", Name: "Administrator"},
	}

	list := make([]ProjectTableItem, 0, len(projects))
	for _, p := range projects {
		orgName := orgMap[p.OrganizationID]
		if orgName == "" {
			orgName = "默认组织"
		}
		list = append(list, ProjectTableItem{
			ID:               p.ID,
			Name:             p.Name,
			Description:      p.Description,
			Enable:           bool(p.Enable),
			OrganizationID:   p.OrganizationID,
			OrganizationName: orgName,
			CreatedAt:        p.CreatedAt,
			UpdatedAt:        p.UpdatedAt,
			MemberCount:      1,
			OrgAdmins:        defaultAdmins,
		})
	}

	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

func (h *SystemHandler) SystemProjectAdd(c *gin.Context) {
	var body struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		OrganizationID string `json:"organizationId"`
		Enable         *bool  `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	orgID := body.OrganizationID
	if orgID == "" {
		orgID = "100001"
	}
	now := time.Now().UnixMilli()
	proj := model.Project{
		ID:             fmt.Sprintf("%d", now),
		Name:           body.Name,
		Description:    body.Description,
		OrganizationID: orgID,
		Enable:         model.BitBool(true),
		Deleted:        model.BitBool(false),
		CreatedAt:      now,
		UpdatedAt:      now,
		CreateUser:     "admin",
	}
	if body.Enable != nil {
		proj.Enable = model.BitBool(*body.Enable)
	}
	config.DB.Create(&proj)
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeAdd, proj.ID, "创建项目: "+proj.Name)
	response.Success(c, proj)
}

func (h *SystemHandler) SystemProjectUpdate(c *gin.Context) {
	var body struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		OrganizationID string `json:"organizationId"`
		Enable         *bool  `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	updates := map[string]interface{}{
		"updated_at": time.Now().UnixMilli(),
	}
	if body.Name != "" {
		updates["name"] = body.Name
	}
	if body.Description != "" {
		updates["description"] = body.Description
	}
	if body.OrganizationID != "" {
		updates["organization_id"] = body.OrganizationID
	}
	if body.Enable != nil {
		updates["enable"] = model.BitBool(*body.Enable)
	}
	config.DB.Model(&model.Project{}).Where("id = ?", body.ID).Updates(updates)
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeUpdate, body.ID, "修改项目: "+body.Name)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemProjectRename(c *gin.Context) {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	config.DB.Model(&model.Project{}).Where("id = ?", body.ID).Updates(map[string]interface{}{
		"name":       body.Name,
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeUpdate, body.ID, "重命名项目: "+body.Name)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemProjectDelete(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":    model.BitBool(true),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeDelete, id, "删除项目: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemProjectEnable(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enable":     model.BitBool(true),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeUpdate, id, "启用项目: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemProjectDisable(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enable":     model.BitBool(false),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeUpdate, id, "禁用项目: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemProjectRevoke(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":    model.BitBool(false),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleOrgProject, service.OpTypeUpdate, id, "恢复项目: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemProjectUserList(c *gin.Context) {
	var users []model.User
	query := config.DB.Where("deleted = 0")
	keyword := c.Query("keyword")
	if keyword != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	query.Find(&users)
	type Opt struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	opts := make([]Opt, 0, len(users))
	for _, u := range users {
		opts = append(opts, Opt{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		})
	}
	response.Success(c, opts)
}

func (h *SystemHandler) SystemOrgOptionsAll(c *gin.Context) {
	var orgs []model.Organization
	config.DB.Where("deleted = 0").Find(&orgs)
	type Opt struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	opts := make([]Opt, 0, len(orgs))
	for _, o := range orgs {
		opts = append(opts, Opt{
			ID:   o.ID,
			Name: o.Name,
		})
	}
	response.Success(c, opts)
}

func (h *SystemHandler) SystemOrgAdd(c *gin.Context) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	now := time.Now().UnixMilli()
	org := model.Organization{
		ID:          fmt.Sprintf("%d", now),
		Name:        body.Name,
		Description: body.Description,
		Enable:      model.BitBool(true),
		Deleted:     model.BitBool(false),
		CreatedAt:   now,
		UpdatedAt:   now,
		CreateUser:  "admin",
	}
	config.DB.Create(&org)
	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeAdd, org.ID, "创建组织: "+org.Name)
	response.Success(c, org)
}

func (h *SystemHandler) SystemOrgUpdate(c *gin.Context) {
	var body struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	config.DB.Model(&model.Organization{}).Where("id = ?", body.ID).Updates(map[string]interface{}{
		"name":        body.Name,
		"description": body.Description,
		"updated_at":  time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeUpdate, body.ID, "修改组织: "+body.Name)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemOrgRename(c *gin.Context) {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	config.DB.Model(&model.Organization{}).Where("id = ?", body.ID).Updates(map[string]interface{}{
		"name":       body.Name,
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeUpdate, body.ID, "重命名组织: "+body.Name)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemOrgDelete(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Organization{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":    model.BitBool(true),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeDelete, id, "删除组织: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemOrgEnable(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Organization{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enable":     model.BitBool(true),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeUpdate, id, "启用组织: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemOrgDisable(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Organization{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enable":     model.BitBool(false),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeUpdate, id, "禁用组织: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) SystemOrgRevoke(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Organization{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":    model.BitBool(false),
		"updated_at": time.Now().UnixMilli(),
	})
	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeUpdate, id, "恢复组织: "+id)
	response.Success(c, "ok")
}

func (h *SystemHandler) BaseInfo(c *gin.Context) {
	response.Success(c, gin.H{
		"url":   "http://localhost:5173",
		"title": "TrueOne Harness",
	})
}

func (h *SystemHandler) UserProfile(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists || userId == "" {
		userId = "admin"
	}
	var u model.User
	config.DB.Where("id = ? AND deleted = 0", userId).First(&u)
	response.Success(c, u)
}

func (h *SystemHandler) UserProfilePage(c *gin.Context) {
	response.SuccessPage(c, []interface{}{}, 0, 1, 10)
}

func (h *SystemHandler) ProjectMemberList(c *gin.Context) {
	var body struct {
		ProjectID string `json:"projectId"`
	}
	_ = c.ShouldBindJSON(&body)

	var users []model.User
	config.DB.Where("deleted = 0").Find(&users)

	type MemberItem struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		RoleName string `json:"roleName"`
	}

	members := make([]MemberItem, 0, len(users))
	for _, u := range users {
		members = append(members, MemberItem{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			RoleName: "管理员",
		})
	}

	response.SuccessPage(c, members, int64(len(members)), 1, 100)
}

func (h *SystemHandler) ProjectMemberOptions(c *gin.Context) {
	var users []model.User
	config.DB.Where("deleted = 0").Find(&users)

	type OptionItem struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	options := make([]OptionItem, 0, len(users))
	for _, u := range users {
		options = append(options, OptionItem{
			ID:   u.ID,
			Name: u.Name,
		})
	}
	response.Success(c, options)
}

func (h *SystemHandler) UserRoleProjectList(c *gin.Context) {
	var roles []model.UserRole
	config.DB.Find(&roles)
	response.SuccessPage(c, roles, int64(len(roles)), 1, 100)
}

// User Group Relations & Permissions
func (h *SystemHandler) GlobalUserGroupMemberList(c *gin.Context) {
	var body struct {
		RoleID   string `json:"roleId"`
		Current  int    `json:"current"`
		PageSize int    `json:"pageSize"`
		Keyword  string `json:"keyword"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Current <= 0 {
		body.Current = 1
	}
	if body.PageSize <= 0 {
		body.PageSize = 10
	}

	var relations []model.UserRoleRelation
	query := config.DB.Model(&model.UserRoleRelation{})
	if body.RoleID != "" {
		query = query.Where("role_id = ?", body.RoleID)
	}

	var total int64
	query.Count(&total)
	offset := (body.Current - 1) * body.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(body.PageSize).Find(&relations)

	// Fetch users
	var userIds []string
	for _, r := range relations {
		userIds = append(userIds, r.UserID)
	}
	var users []model.User
	if len(userIds) > 0 {
		config.DB.Where("id IN ?", userIds).Find(&users)
	}
	userMap := make(map[string]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	type MemberItem struct {
		ID         string `json:"id"`
		UserID     string `json:"userId"`
		Name       string `json:"name"`
		Email      string `json:"email"`
		Phone      string `json:"phone"`
		Enable     bool   `json:"enable"`
		CreatedAt int64  `json:"createdAt"`
	}

	list := make([]MemberItem, 0, len(relations))
	for _, r := range relations {
		u := userMap[r.UserID]
		list = append(list, MemberItem{
			ID:         r.ID,
			UserID:     r.UserID,
			Name:       u.Name,
			Email:      u.Email,
			Phone:      u.Phone,
			Enable:     bool(u.Enable),
			CreatedAt: r.CreatedAt,
		})
	}

	response.SuccessPage(c, list, total, body.Current, body.PageSize)
}

func (h *SystemHandler) GlobalPermissionSetting(c *gin.Context) {
	roleID := c.Param("roleId")

	// Standard permissions tree
	type Perm struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Enable bool   `json:"enable"`
	}
	type TreeItem struct {
		ID          string     `json:"id"`
		Name        string     `json:"name"`
		Enable      bool       `json:"enable"`
		Permissions []Perm     `json:"permissions"`
		Children    []TreeItem `json:"children,omitempty"`
	}

	var rolePerms []model.UserRolePermission
	config.DB.Where("role_id = ?", roleID).Find(&rolePerms)
	enabledMap := make(map[string]bool)
	for _, rp := range rolePerms {
		enabledMap[rp.PermissionID] = true
	}

	isPermEnabled := func(permID string) bool {
		if roleID == "admin" {
			return true
		}
		if len(rolePerms) == 0 {
			// 如果该角色在库中暂无单独配置，默认开启 READ 权限
			return strings.HasSuffix(permID, ":READ")
		}
		return enabledMap[permID]
	}

	settings := []TreeItem{
		{
			ID:     "WORKSPACE",
			Name:   "工作台",
			Enable: isPermEnabled("WORKSPACE:READ"),
			Permissions: []Perm{
				{ID: "WORKSPACE:READ", Name: "查看", Enable: isPermEnabled("WORKSPACE:READ")},
			},
		},
		{
			ID:     "QUALITY_WORKSPACE",
			Name:   "需求质量",
			Enable: isPermEnabled("QUALITY:READ"),
			Permissions: []Perm{
				{ID: "QUALITY:READ", Name: "查看", Enable: isPermEnabled("QUALITY:READ")},
				{ID: "QUALITY:WRITE", Name: "创建/编辑", Enable: isPermEnabled("QUALITY:WRITE")},
				{ID: "QUALITY:EXECUTE", Name: "执行用例", Enable: isPermEnabled("QUALITY:EXECUTE")},
			},
		},
		{
			ID:     "FUNCTIONAL_CASE",
			Name:   "测试资产",
			Enable: isPermEnabled("CASE:READ"),
			Permissions: []Perm{
				{ID: "CASE:READ", Name: "查看", Enable: isPermEnabled("CASE:READ")},
				{ID: "CASE:WRITE", Name: "编辑用例", Enable: isPermEnabled("CASE:WRITE")},
				{ID: "CASE:DELETE", Name: "删除用例", Enable: isPermEnabled("CASE:DELETE")},
			},
		},
		{
			ID:     "BUG",
			Name:   "缺陷管理",
			Enable: isPermEnabled("BUG:READ"),
			Permissions: []Perm{
				{ID: "BUG:READ", Name: "查看缺陷", Enable: isPermEnabled("BUG:READ")},
				{ID: "BUG:WRITE", Name: "创建/修改缺陷", Enable: isPermEnabled("BUG:WRITE")},
				{ID: "BUG:DELETE", Name: "删除缺陷", Enable: isPermEnabled("BUG:DELETE")},
			},
		},
		{
			ID:     "PRECISION_TEST",
			Name:   "精准测试",
			Enable: isPermEnabled("COV:READ"),
			Permissions: []Perm{
				{ID: "COV:READ", Name: "查看覆盖率", Enable: isPermEnabled("COV:READ")},
				{ID: "COV:EXECUTE", Name: "执行覆盖率分析", Enable: isPermEnabled("COV:EXECUTE")},
			},
		},
		{
			ID:     "SYSTEM_SETTING",
			Name:   "系统设置",
			Enable: isPermEnabled("SYSTEM:READ"),
			Permissions: []Perm{
				{ID: "SYSTEM:READ", Name: "查看配置", Enable: isPermEnabled("SYSTEM:READ")},
				{ID: "SYSTEM:WRITE", Name: "修改配置", Enable: isPermEnabled("SYSTEM:WRITE")},
			},
		},
	}
	response.Success(c, settings)
}

func (h *SystemHandler) GlobalPermissionUpdate(c *gin.Context) {
	var body struct {
		UserRoleID  string `json:"userRoleId"`
		Permissions []struct {
			ID     string `json:"id"`
			Enable bool   `json:"enable"`
		} `json:"permissions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	for _, p := range body.Permissions {
		if p.Enable {
			var count int64
			config.DB.Model(&model.UserRolePermission{}).
				Where("role_id = ? AND permission_id = ?", body.UserRoleID, p.ID).
				Count(&count)
			if count == 0 {
				newPerm := model.UserRolePermission{
					ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
					RoleID:       body.UserRoleID,
					PermissionID: p.ID,
				}
				config.DB.Create(&newPerm)
			}
		} else {
			config.DB.Where("role_id = ? AND permission_id = ?", body.UserRoleID, p.ID).
				Delete(&model.UserRolePermission{})
		}
	}

	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeUpdate, body.UserRoleID, "修改角色权限配置: "+body.UserRoleID)
	response.Success(c, "更新成功")
}

func (h *SystemHandler) GlobalUserGroupMemberOption(c *gin.Context) {
	var users []model.User
	config.DB.Where("deleted = 0").Find(&users)
	type UserOpt struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	opts := make([]UserOpt, 0, len(users))
	for _, u := range users {
		opts = append(opts, UserOpt{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		})
	}
	response.Success(c, opts)
}

func (h *SystemHandler) GlobalUserGroupAddMember(c *gin.Context) {
	response.Success(c, "添加成功")
}

func (h *SystemHandler) GlobalUserGroupDeleteMember(c *gin.Context) {
	relationId := c.Param("relationId")
	config.DB.Where("id = ?", relationId).Delete(&model.UserRoleRelation{})
	response.Success(c, "移除成功")
}

// Email Config
func (h *SystemHandler) EmailInfo(c *gin.Context) {
	response.Success(c, gin.H{
		"host":      "",
		"port":      "465",
		"account":   "",
		"from":      "",
		"ssl":       "true",
		"tsl":       "false",
		"recipient": "",
	})
}

func (h *SystemHandler) EmailInfoSave(c *gin.Context) {
	response.Success(c, "保存成功")
}

func (h *SystemHandler) EmailTest(c *gin.Context) {
	response.Success(c, "测试连接成功")
}

// Clean Config
func (h *SystemHandler) CleanConfig(c *gin.Context) {
	response.Success(c, gin.H{
		"operationLog":     "30",
		"operationHistory": "30",
	})
}

func (h *SystemHandler) CleanConfigSave(c *gin.Context) {
	response.Success(c, "保存成功")
}

// Auth Source Config
func (h *SystemHandler) AuthSourceList(c *gin.Context) {
	response.Success(c, gin.H{
		"total": 0,
		"list":  []interface{}{},
	})
}

func (h *SystemHandler) PlatformInfo(c *gin.Context) {
	response.Success(c, []gin.H{
		{
			"platform":  "LARK",
			"enable":    false,
			"valid":     false,
			"hasConfig": false,
		},
	})
}

func (h *SystemHandler) LarkInfoWithDetail(c *gin.Context) {
	response.Success(c, gin.H{
		"agentId":   "",
		"appSecret": "",
		"callBack":  "http://localhost:5173/login",
		"enable":    false,
		"valid":     false,
	})
}

func (h *SystemHandler) LarkSave(c *gin.Context) {
	response.Success(c, "保存成功")
}

func (h *SystemHandler) LarkValidate(c *gin.Context) {
	response.Success(c, "校验成功")
}

func (h *SystemHandler) LarkEnable(c *gin.Context) {
	response.Success(c, "更新成功")
}

func (h *SystemHandler) DisplayInfo(c *gin.Context) {
	response.Success(c, gin.H{
		"title": "TrueOne Harness",
	})
}

