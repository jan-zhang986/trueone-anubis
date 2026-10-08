package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
)

type LogHandler struct{}

func NewLogHandler() *LogHandler {
	return &LogHandler{}
}

type LogQueryParams struct {
	Keyword         string   `json:"keyword"`
	Current         int      `json:"current"`
	PageSize        int      `json:"pageSize"`
	OperUser        string   `json:"operUser"`
	StartTime       int64    `json:"startTime"`
	EndTime         int64    `json:"endTime"`
	ProjectIDs      []string `json:"projectIds"`
	OrganizationIDs []string `json:"organizationIds"`
	Type            string   `json:"type"`
	Module          string   `json:"module"`
	Content         string   `json:"content"`
}

type LogItemDTO struct {
	ID               string `json:"id"`
	CreateUser       string `json:"createUser"`
	UserName         string `json:"userName"`
	ProjectID        string `json:"projectId"`
	ProjectName      string `json:"projectName"`
	OrganizationID   string `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	Module           string `json:"module"`
	Type             string `json:"type"`
	Content          string `json:"content"`
	CreatedAt        int64  `json:"createdAt"`
	SourceID         string `json:"sourceId"`
	Status           string `json:"status"`
}

func (h *LogHandler) queryLogs(c *gin.Context, defaultOrgID, defaultProjID string) {
	var params LogQueryParams
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

	query := config.DB.Model(&model.OperationLog{})

	if defaultOrgID != "" {
		query = query.Where("organization_id = ?", defaultOrgID)
	}
	if defaultProjID != "" {
		query = query.Where("project_id = ?", defaultProjID)
	}
	if params.Keyword != "" {
		query = query.Where("content LIKE ? OR module LIKE ? OR type LIKE ? OR create_user LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}
	if params.OperUser != "" {
		query = query.Where("create_user = ?", params.OperUser)
	}
	if params.StartTime > 0 {
		query = query.Where("created_at >= ?", params.StartTime)
	}
	if params.EndTime > 0 {
		query = query.Where("created_at <= ?", params.EndTime)
	}
	if len(params.ProjectIDs) > 0 {
		query = query.Where("project_id IN ?", params.ProjectIDs)
	}
	if len(params.OrganizationIDs) > 0 {
		query = query.Where("organization_id IN ?", params.OrganizationIDs)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.Module != "" {
		query = query.Where("module = ?", params.Module)
	}

	var total int64
	query.Count(&total)

	offset := (params.Current - 1) * params.PageSize
	var logs []model.OperationLog
	query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&logs)

	// 获取用户表、组织表、项目表用于名称映射
	var users []model.User
	config.DB.Find(&users)
	userMap := make(map[string]string)
	for _, u := range users {
		userMap[u.ID] = u.Name
	}

	var orgs []model.Organization
	config.DB.Find(&orgs)
	orgMap := make(map[string]string)
	for _, o := range orgs {
		orgMap[o.ID] = o.Name
	}

	var projs []model.Project
	config.DB.Find(&projs)
	projMap := make(map[string]string)
	for _, p := range projs {
		projMap[p.ID] = p.Name
	}

	list := make([]LogItemDTO, 0, len(logs))
	for _, l := range logs {
		uName := l.CreateUser
		if name, ok := userMap[l.CreateUser]; ok && name != "" {
			uName = name
		}
		oName := l.OrganizationID
		if name, ok := orgMap[l.OrganizationID]; ok && name != "" {
			oName = name
		} else if l.OrganizationID == "SYSTEM" {
			oName = "系统"
		}
		pName := l.ProjectID
		if name, ok := projMap[l.ProjectID]; ok && name != "" {
			pName = name
		} else if l.ProjectID == "SYSTEM" {
			pName = "系统"
		}

		list = append(list, LogItemDTO{
			ID:               fmt.Sprintf("%d", l.ID),
			CreateUser:       l.CreateUser,
			UserName:         uName,
			ProjectID:        l.ProjectID,
			ProjectName:      pName,
			OrganizationID:   l.OrganizationID,
			OrganizationName: oName,
			Module:           l.Module,
			Type:             l.Type,
			Content:          l.Content,
			CreatedAt:        l.CreatedAt,
			SourceID:         l.SourceID,
			Status:           "SUCCESS",
		})
	}

	response.SuccessPage(c, list, total, params.Current, params.PageSize)
}

// 1. 系统日志
func (h *LogHandler) SystemLogList(c *gin.Context) {
	h.queryLogs(c, "", "")
}

func (h *LogHandler) SystemLogOptions(c *gin.Context) {
	var orgs []model.Organization
	config.DB.Where("deleted = 0").Find(&orgs)
	orgList := make([]map[string]interface{}, 0, len(orgs))
	for _, o := range orgs {
		orgList = append(orgList, map[string]interface{}{
			"id":   o.ID,
			"name": o.Name,
		})
	}

	var projs []model.Project
	config.DB.Where("deleted = 0").Find(&projs)
	projList := make([]map[string]interface{}, 0, len(projs))
	for _, p := range projs {
		projList = append(projList, map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		})
	}

	response.Success(c, map[string]interface{}{
		"organizationList": orgList,
		"projectList":      projList,
	})
}

func (h *LogHandler) SystemLogUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	var users []model.User
	query := config.DB.Where("deleted = 0")
	if keyword != "" {
		query = query.Where("name LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	query.Find(&users)

	res := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		res = append(res, map[string]interface{}{
			"id":    u.ID,
			"name":  u.Name,
			"email": u.Email,
		})
	}
	response.Success(c, res)
}

// 2. 组织日志
func (h *LogHandler) OrgLogList(c *gin.Context) {
	orgID := c.GetHeader("ORGANIZATION")
	h.queryLogs(c, orgID, "")
}

func (h *LogHandler) OrgLogOptions(c *gin.Context) {
	orgID := c.Param("organizationId")
	var projs []model.Project
	config.DB.Where("organization_id = ? AND deleted = 0", orgID).Find(&projs)
	projList := make([]map[string]interface{}, 0, len(projs))
	for _, p := range projs {
		projList = append(projList, map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		})
	}

	response.Success(c, map[string]interface{}{
		"organizationList": []map[string]interface{}{},
		"projectList":      projList,
	})
}

func (h *LogHandler) OrgLogUsers(c *gin.Context) {
	h.SystemLogUsers(c)
}

// 3. 项目日志
func (h *LogHandler) ProjectLogList(c *gin.Context) {
	projID := c.GetHeader("PROJECT")
	h.queryLogs(c, "", projID)
}

func (h *LogHandler) ProjectLogUsers(c *gin.Context) {
	h.SystemLogUsers(c)
}
