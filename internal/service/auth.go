package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"

	"time"

	"github.com/google/uuid"
	"trueone-anubis/config"
	"trueone-anubis/internal/middleware"
	"trueone-anubis/internal/model"
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func md5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *AuthService) Login(req model.LoginRequest) (*model.SessionUser, error) {
	var user model.User
	err := config.DB.Where("(id = ? OR email = ?) AND deleted = 0", req.Username, req.Username).First(&user).Error
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	if !user.Enable {
		return nil, errors.New("账号已被禁用")
	}

	// Verify password: support MD5 hash or direct match
	inputMD5 := md5Hash(req.Password)
	if user.Password != "" && user.Password != req.Password && user.Password != inputMD5 {
		return nil, errors.New("用户名或密码错误")
	}

	sessionId := uuid.New().String()
	csrfToken := uuid.New().String()

	middleware.Store.Set(sessionId, user.ID)

	var roles []model.UserRole
	config.DB.Where("create_user = ? OR scope_id = ?", user.ID, user.LastOrganizationID).Find(&roles)

	var relations []model.UserRoleRelation
	config.DB.Where("user_id = ?", user.ID).Find(&relations)

	// Ensure last project exists and is valid
	if user.LastProjectID != "" {
		var p model.Project
		if err := config.DB.Where("id = ? AND deleted = 0", user.LastProjectID).First(&p).Error; err != nil {
			// Find first project for user
			var firstP model.Project
			if err2 := config.DB.Where("organization_id = ? AND deleted = 0", user.LastOrganizationID).First(&firstP).Error; err2 == nil {
				user.LastProjectID = firstP.ID
			} else {
				user.LastProjectID = "no_such_project"
			}
		}
	} else {
		var firstP model.Project
		if err := config.DB.Where("organization_id = ? AND deleted = 0", user.LastOrganizationID).First(&firstP).Error; err == nil {
			user.LastProjectID = firstP.ID
		}
	}

	// Record operation log (like original system)
	logRecord := model.OperationLog{
		ProjectID:      "SYSTEM",
		OrganizationID: "SYSTEM",
		CreatedAt:      time.Now().UnixMilli(),
		CreateUser:     user.ID,
		SourceID:       "SYSTEM",
		Method:         "POST",
		Type:           "LOGIN",
		Module:         "SYSTEM",
		Content:        user.Name + "登录成功",
		Path:           "/login",
	}
	config.DB.Create(&logRecord)

	permissions := s.GetUserPermissions(user.ID, relations)

	sessionUser := &model.SessionUser{
		ID:                 user.ID,
		Name:               user.Name,
		Email:              user.Email,
		Phone:              user.Phone,
		LastOrganizationID: user.LastOrganizationID,
		LastProjectID:      user.LastProjectID,
		Language:           user.Language,
		SessionID:          sessionId,
		CsrfToken:          csrfToken,
		UserRoles:          roles,
		UserRoleRelations:  relations,
		Permissions:        permissions,
	}

	return sessionUser, nil
}

func (s *AuthService) GetUserPermissions(userId string, relations []model.UserRoleRelation) []string {
	// If admin, return all permissions
	isAdmin := userId == "admin"
	for _, r := range relations {
		if r.RoleID == "admin" {
			isAdmin = true
			break
		}
	}
	if isAdmin {
		return []string{
			"*",
			"WORKSPACE:READ",
			"QUALITY_WORKSPACE:READ",
			"QUALITY:READ",
			"QUALITY:WRITE",
			"QUALITY:EXECUTE",
			"CASE:READ",
			"CASE:WRITE",
			"CASE:DELETE",
			"FUNCTIONAL_CASE:READ",
			"BUG:READ",
			"BUG:WRITE",
			"BUG:DELETE",
			"COV:READ",
			"COV:EXECUTE",
			"PROJECT_MANAGEMENT:READ",
			"SYSTEM:READ",
			"SYSTEM:WRITE",
			"SYSTEM_SETTING:READ",
			"ORGANIZATION:READ",
		}
	}

	roleIDs := make([]string, 0, len(relations))
	for _, r := range relations {
		roleIDs = append(roleIDs, r.RoleID)
	}
	if len(roleIDs) == 0 {
		roleIDs = append(roleIDs, "project_member")
	}

	var rolePerms []model.UserRolePermission
	config.DB.Where("role_id IN ?", roleIDs).Find(&rolePerms)

	permMap := make(map[string]bool)
	for _, rp := range rolePerms {
		permMap[rp.PermissionID] = true
		switch rp.PermissionID {
		case "PROJECT_BUG:READ":
			permMap["BUG:READ"] = true
		case "PROJECT_BUG:READ+ADD":
			permMap["BUG:WRITE"] = true
		case "PROJECT_BUG:READ+DELETE":
			permMap["BUG:DELETE"] = true
		case "FUNCTIONAL_CASE:READ":
			permMap["CASE:READ"] = true
		case "QUALITY_WORKSPACE:READ":
			permMap["QUALITY:READ"] = true
			permMap["WORKSPACE:READ"] = true
		}
	}

	perms := make([]string, 0, len(permMap))
	for p := range permMap {
		perms = append(perms, p)
	}
	return perms
}

func (s *AuthService) GetCurrentUser(userId string) (*model.SessionUser, error) {
	var user model.User
	err := config.DB.Where("id = ? AND deleted = 0", userId).First(&user).Error
	if err != nil {
		// If not found by ID, try admin fallback
		err = config.DB.Where("deleted = 0").First(&user).Error
		if err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}
	}

	var roles []model.UserRole
	config.DB.Where("create_user = ? OR scope_id = ?", user.ID, user.LastOrganizationID).Find(&roles)

	var relations []model.UserRoleRelation
	config.DB.Where("user_id = ?", user.ID).Find(&relations)

	sessionId := uuid.New().String()
	csrfToken := uuid.New().String()

	middleware.Store.Set(sessionId, user.ID)

	permissions := s.GetUserPermissions(user.ID, relations)

	return &model.SessionUser{
		ID:                 user.ID,
		Name:               user.Name,
		Email:              user.Email,
		Phone:              user.Phone,
		LastOrganizationID: user.LastOrganizationID,
		LastProjectID:      user.LastProjectID,
		Language:           user.Language,
		SessionID:          sessionId,
		CsrfToken:          csrfToken,
		UserRoles:          roles,
		UserRoleRelations:  relations,
		Permissions:        permissions,
	}, nil
}

func (s *AuthService) SwitchProject(userId, projectId string) error {
	return config.DB.Model(&model.User{}).Where("id = ?", userId).Update("last_project_id", projectId).Error
}

func (s *AuthService) Logout(sessionId string) {
	if userId, exists := middleware.Store.Get(sessionId); exists {
		var user model.User
		if err := config.DB.Where("id = ?", userId).First(&user).Error; err == nil {
			logRecord := model.OperationLog{
				ProjectID:      "SYSTEM",
				OrganizationID: "SYSTEM",
				CreatedAt:      time.Now().UnixMilli(),
				CreateUser:     user.ID,
				SourceID:       "SYSTEM",
				Method:         "GET",
				Type:           "LOGOUT",
				Module:         "SYSTEM",
				Content:        user.Name + "登出成功",
				Path:           "/signout",
			}
			config.DB.Create(&logRecord)
		}
	}
	middleware.Store.Delete(sessionId)
}

func (s *AuthService) UpdateCurrentPassword(userId, oldPassword, newPassword string) error {
	var user model.User
	if err := config.DB.Where("id = ? AND deleted = 0", userId).First(&user).Error; err != nil {
		return errors.New("用户不存在")
	}
	oldMD5 := md5Hash(oldPassword)
	if user.Password != "" && user.Password != oldPassword && user.Password != oldMD5 {
		return errors.New("原密码错误")
	}
	newMD5 := md5Hash(newPassword)
	return config.DB.Model(&model.User{}).Where("id = ?", userId).Updates(map[string]interface{}{
		"password":   newMD5,
		"updated_at": time.Now().UnixMilli(),
	}).Error
}

func (s *AuthService) ResetUserPassword(userId, newPassword string) error {
	newMD5 := md5Hash(newPassword)
	return config.DB.Model(&model.User{}).Where("id = ?", userId).Updates(map[string]interface{}{
		"password":   newMD5,
		"updated_at": time.Now().UnixMilli(),
	}).Error
}
