package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

func MD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// BootstrapSystem 负责系统启动时的自愈与数据初始化（开箱即用）
func BootstrapSystem() {
	now := time.Now().UnixMilli()

	// 1. 初始化默认组织
	var orgCount int64
	config.DB.Model(&model.Organization{}).Where("deleted = 0").Count(&orgCount)
	if orgCount == 0 {
		defaultOrg := model.Organization{
			ID:          "100001",
			Num:         100001,
			Name:        "默认组织",
			Description: "系统默认创建的组织",
			CreateUser:  "admin",
			UpdateUser:  "admin",
			CreatedAt:   now,
			UpdatedAt:   now,
			Enable:      model.BitBool(true),
			Deleted:     model.BitBool(false),
		}
		config.DB.Create(&defaultOrg)
		fmt.Println("🌱 [Bootstrap] 初始化默认组织: 100001 (默认组织)")
	}

	// 2. 初始化默认项目
	var projCount int64
	config.DB.Model(&model.Project{}).Where("deleted = 0").Count(&projCount)
	if projCount == 0 {
		defaultProj := model.Project{
			ID:             "100001100001",
			Num:            100001,
			OrganizationID: "100001",
			Name:           "示例项目",
			Description:    "系统默认创建的项目",
			CreateUser:     "admin",
			UpdateUser:     "admin",
			CreatedAt:      now,
			UpdatedAt:      now,
			Enable:         model.BitBool(true),
			Deleted:        model.BitBool(false),
			ModuleSetting:  `["bugManagement","caseManagement","apiTest","testPlan"]`,
		}
		config.DB.Create(&defaultProj)
		fmt.Println("🌱 [Bootstrap] 初始化默认项目: 100001100001 (示例项目)")
	}

	// 3. 初始化全局角色
	var roleCount int64
	config.DB.Model(&model.UserRole{}).Count(&roleCount)
	if roleCount == 0 {
		defaultRoles := []model.UserRole{
			{ID: "admin", Name: "系统管理员", Type: "SYSTEM", ScopeID: "global", Internal: model.BitBool(true), CreatedAt: now, UpdatedAt: now, CreateUser: "admin"},
			{ID: "member", Name: "系统成员", Type: "SYSTEM", ScopeID: "global", Internal: model.BitBool(true), CreatedAt: now, UpdatedAt: now, CreateUser: "admin"},
			{ID: "org_admin", Name: "组织管理员", Type: "ORGANIZATION", ScopeID: "global", Internal: model.BitBool(true), CreatedAt: now, UpdatedAt: now, CreateUser: "admin"},
			{ID: "org_member", Name: "组织成员", Type: "ORGANIZATION", ScopeID: "global", Internal: model.BitBool(true), CreatedAt: now, UpdatedAt: now, CreateUser: "admin"},
			{ID: "project_admin", Name: "项目管理员", Type: "PROJECT", ScopeID: "global", Internal: model.BitBool(true), CreatedAt: now, UpdatedAt: now, CreateUser: "admin"},
			{ID: "project_member", Name: "项目成员", Type: "PROJECT", ScopeID: "global", Internal: model.BitBool(true), CreatedAt: now, UpdatedAt: now, CreateUser: "admin"},
		}
		for _, r := range defaultRoles {
			config.DB.Create(&r)
		}
		fmt.Println("🌱 [Bootstrap] 初始化系统默认角色定义 (6个基础角色)")
	}

	// 4. 初始化/更新 admin 管理员账号及其密码（设置为 trueone）
	adminPwd := "trueone"
	if config.GlobalConfig != nil && config.GlobalConfig.Security.DefaultAdminPassword != "" {
		adminPwd = config.GlobalConfig.Security.DefaultAdminPassword
	}
	adminPwdHash := MD5Hash(adminPwd)

	var adminUser model.User
	err := config.DB.Where("id = 'admin'").First(&adminUser).Error
	if err != nil {
		// 不存在则创建
		newAdmin := model.User{
			ID:                 "admin",
			Name:               "Administrator",
			Email:              "admin@trueone.io",
			Password:           adminPwdHash,
			Enable:             model.BitBool(true),
			Deleted:            model.BitBool(false),
			CreatedAt:          now,
			UpdatedAt:          now,
			LastOrganizationID: "100001",
			LastProjectID:      "100001100001",
			CreateUser:         "admin",
			UpdateUser:         "admin",
			Source:             "LOCAL",
		}
		config.DB.Create(&newAdmin)
		fmt.Printf("🌱 [Bootstrap] 初始化默认管理员账号: admin (密码已设置为: %s)\n", adminPwd)
	} else {
		// 已存在则同步更新密码为配置的密码（trueone）
		config.DB.Model(&model.User{}).Where("id = 'admin'").Updates(map[string]interface{}{
			"password":   adminPwdHash,
			"updated_at": now,
		})
		fmt.Printf("🔒 [Bootstrap] 管理员账号 admin 密码已确认/同步为: %s (MD5: %s)\n", adminPwd, adminPwdHash)
	}

	// 5. 确保 admin 用户的三级管理员权限关联完整
	type RelInfo struct {
		RoleID   string
		SourceID string
	}
	requiredRels := []RelInfo{
		{RoleID: "admin", SourceID: "system"},
		{RoleID: "org_admin", SourceID: "100001"},
		{RoleID: "project_admin", SourceID: "100001100001"},
	}

	for _, rel := range requiredRels {
		var cnt int64
		config.DB.Model(&model.UserRoleRelation{}).Where("user_id = 'admin' AND role_id = ?", rel.RoleID).Count(&cnt)
		if cnt == 0 {
			r := model.UserRoleRelation{
				ID:         fmt.Sprintf("rel-%s-%d", rel.RoleID, now),
				UserID:     "admin",
				RoleID:     rel.RoleID,
				SourceID:   rel.SourceID,
				CreatedAt:  now,
				CreateUser: "admin",
			}
			config.DB.Create(&r)
		}
	}
}
