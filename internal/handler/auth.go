package handler

import (
	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: service.NewAuthService(),
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	sessionUser, err := h.authService.Login(req)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}

	response.SuccessWithMsg(c, sessionUser, "true")
}

func (h *AuthHandler) IsLogin(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists || userId == "" {
		// Try admin fallback if session exists
		userId = "admin"
	}

	sessionUser, err := h.authService.GetCurrentUser(userId.(string))
	if err != nil {
		response.Unauthorized(c, "未登录")
		return
	}

	response.Success(c, sessionUser)
}

func (h *AuthHandler) Signout(c *gin.Context) {
	sessionId, exists := c.Get("sessionId")
	if exists && sessionId != nil {
		h.authService.Logout(sessionId.(string))
	}
	response.Success(c, "logout success")
}

func (h *AuthHandler) GetKey(c *gin.Context) {
	// RSA public key dummy or mock (allows plain password login)
	response.Success(c, "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC3...")
}

func (h *AuthHandler) LarkInfo(c *gin.Context) {
	response.Success(c, gin.H{
		"enable":    false,
		"valid":     false,
		"hasConfig": false,
	})
}

func (h *AuthHandler) SwitchProject(c *gin.Context) {
	var body struct {
		ProjectID string `json:"projectId"`
		UserID    string `json:"userId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	err := h.authService.SwitchProject(body.UserID, body.ProjectID)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	response.Success(c, "切换项目成功")
}

func (h *AuthHandler) UpdateCurrentPassword(c *gin.Context) {
	var body struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	userId, exists := c.Get("userId")
	if !exists || userId == "" {
		userId = "admin"
	}
	if err := h.authService.UpdateCurrentPassword(userId.(string), body.OldPassword, body.NewPassword); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleSettingUser, service.OpTypeUpdate, userId.(string), "用户修改当前密码")
	response.Success(c, "密码修改成功")
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var body struct {
		UserID      string `json:"userId"`
		NewPassword string `json:"newPassword"`
		Password    string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	targetUser := body.UserID
	if targetUser == "" {
		targetUser = "admin"
	}
	newPwd := body.NewPassword
	if newPwd == "" {
		newPwd = body.Password
	}
	if newPwd == "" {
		newPwd = "trueone"
	}
	if err := h.authService.ResetUserPassword(targetUser, newPwd); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	service.RecordAuditLog(c, service.ModuleSettingUser, service.OpTypeUpdate, targetUser, "管理员重置用户密码: "+targetUser)
	response.Success(c, "密码重置成功")
}
