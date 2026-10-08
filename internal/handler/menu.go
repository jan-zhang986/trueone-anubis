package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type MenuHandler struct {
	authService *service.AuthService
}

func NewMenuHandler() *MenuHandler {
	return &MenuHandler{
		authService: service.NewAuthService(),
	}
}

// GetMenuTree 获取所有菜单的完整树形结构（供菜单管理配置使用）
func (h *MenuHandler) GetMenuTree(c *gin.Context) {
	var menus []model.SysMenu
	if err := config.DB.Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	tree := buildMenuTree(menus, 0)
	response.Success(c, tree)
}

// GetUserMenuTree 获取当前登录用户有权限的菜单树（供全局左侧侧边栏渲染）
func (h *MenuHandler) GetUserMenuTree(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists || userId == "" {
		userId = "admin"
	}

	var menus []model.SysMenu
	if err := config.DB.Where("status = ?", "ENABLE").Order("sort ASC, id ASC").Find(&menus).Error; err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	// 判断用户权限
	sessionUser, err := h.authService.GetCurrentUser(userId.(string))
	var userPerms []string
	isAdmin := false
	if err == nil && sessionUser != nil {
		userPerms = sessionUser.Permissions
		for _, p := range userPerms {
			if p == "*" || p == "ADMIN" || p == "SYSTEM:READ" {
				isAdmin = true
				break
			}
		}
	} else {
		isAdmin = true
	}

	permMap := make(map[string]bool)
	for _, p := range userPerms {
		permMap[p] = true
	}

	// 过滤有权限且未被隐藏的菜单
	var filtered []model.SysMenu
	for _, m := range menus {
		if m.IsHidden == 1 {
			continue
		}
		if isAdmin {
			filtered = append(filtered, m)
			continue
		}
		if m.Permission == "" {
			filtered = append(filtered, m)
			continue
		}
		// 校验权限码
		if permMap[m.Permission] {
			filtered = append(filtered, m)
		}
	}

	tree := buildMenuTree(filtered, 0)
	response.Success(c, tree)
}

// AddMenu 新增菜单
func (h *MenuHandler) AddMenu(c *gin.Context) {
	var m model.SysMenu
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Fail(c, 400, "参数错误: "+err.Error())
		return
	}

	if m.MenuKey == "" || m.Title == "" {
		response.Fail(c, 400, "菜单标识和菜单名称不能为空")
		return
	}

	var count int64
	config.DB.Model(&model.SysMenu{}).Where("menu_key = ?", m.MenuKey).Count(&count)
	if count > 0 {
		response.Fail(c, 400, "菜单唯一标识 ["+m.MenuKey+"] 已存在")
		return
	}

	now := time.Now().UnixMilli()
	m.CreateTime = now
	m.UpdateTime = now
	if m.Status == "" {
		m.Status = "ENABLE"
	}
	if m.MenuType == "" {
		m.MenuType = "MENU"
	}
	if m.Icon == "" {
		m.Icon = "Globe"
	}

	if err := config.DB.Create(&m).Error; err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeAdd, strconv.FormatInt(m.ID, 10), "新增系统菜单: "+m.Title)
	response.Success(c, m)
}

// UpdateMenu 更新菜单
func (h *MenuHandler) UpdateMenu(c *gin.Context) {
	var m model.SysMenu
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Fail(c, 400, "参数错误: "+err.Error())
		return
	}

	if m.ID == 0 {
		response.Fail(c, 400, "菜单ID不能为空")
		return
	}

	var existing model.SysMenu
	if err := config.DB.First(&existing, m.ID).Error; err != nil {
		response.Fail(c, 404, "菜单不存在")
		return
	}

	// 检查 menu_key 是否冲突
	if m.MenuKey != "" && m.MenuKey != existing.MenuKey {
		var count int64
		config.DB.Model(&model.SysMenu{}).Where("menu_key = ? AND id != ?", m.MenuKey, m.ID).Count(&count)
		if count > 0 {
			response.Fail(c, 400, "菜单唯一标识已存在")
			return
		}
		existing.MenuKey = m.MenuKey
	}

	existing.ParentID = m.ParentID
	existing.Title = m.Title
	existing.Path = m.Path
	existing.Icon = m.Icon
	existing.Permission = m.Permission
	existing.MenuType = m.MenuType
	existing.Sort = m.Sort
	existing.IsHidden = m.IsHidden
	existing.Status = m.Status
	existing.UpdateTime = time.Now().UnixMilli()

	if err := config.DB.Save(&existing).Error; err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeUpdate, strconv.FormatInt(m.ID, 10), "修改系统菜单: "+existing.Title)
	response.Success(c, existing)
}

// DeleteMenu 删除菜单
func (h *MenuHandler) DeleteMenu(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, 400, "无效的菜单ID")
		return
	}

	var childCount int64
	config.DB.Model(&model.SysMenu{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		response.Fail(c, 400, "存在子菜单，请先删除或移动子菜单")
		return
	}

	var m model.SysMenu
	if err := config.DB.First(&m, id).Error; err == nil {
		config.DB.Delete(&m)
		service.RecordAuditLog(c, service.ModuleSystem, service.OpTypeDelete, idStr, "删除系统菜单: "+m.Title)
	}

	response.Success(c, "删除成功")
}

func buildMenuTree(items []model.SysMenu, parentID int64) []*model.SysMenu {
	var result []*model.SysMenu
	for i := range items {
		if items[i].ParentID == parentID {
			itemCopy := items[i]
			itemCopy.Children = buildMenuTree(items, itemCopy.ID)
			result = append(result, &itemCopy)
		}
	}
	return result
}
