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

	// 6. 清理老旧示例用例，同步真实自动化测试用例资产到 functional_case 表
	syncRealCasesToDatabase()
}

func syncRealCasesToDatabase() {
	projectID := "100001100001"
	now := time.Now().UnixMilli()

	// 1. 清理非标准 TrueOne 用例及老旧模块数据
	config.DB.Model(&model.FunctionalCase{}).
		Where("project_id = ? AND id NOT LIKE 'TC-%'", projectID).
		Update("deleted", model.BitBool(true))

	config.DB.Where("project_id = ? AND id NOT IN ('mod-auth-001', 'mod-org-001', 'mod-prj-001')", projectID).
		Delete(&model.FunctionalCaseModule{})

	// 2. 初始化 3 个真实业务领域模块树节点
	type ModDef struct {
		ID   string
		Name string
		Pos  int64
	}
	modules := []ModDef{
		{ID: "mod-auth-001", Name: "用户认证与会话安全 (REQ-AUTH-001)", Pos: 1000},
		{ID: "mod-org-001", Name: "多租户组织管理 (REQ-ORG-001)", Pos: 2000},
		{ID: "mod-prj-001", Name: "项目生命周期管理 (REQ-PRJ-001)", Pos: 3000},
	}

	for _, m := range modules {
		var cnt int64
		config.DB.Model(&model.FunctionalCaseModule{}).Where("id = ?", m.ID).Count(&cnt)
		if cnt == 0 {
			mod := model.FunctionalCaseModule{
				ID:         m.ID,
				ProjectID:  projectID,
				Name:       m.Name,
				ParentID:   "NONE",
				Pos:        m.Pos,
				CreatedAt:  now,
				UpdatedAt:  now,
				CreateUser: "admin",
				UpdateUser: "admin",
			}
			config.DB.Create(&mod)
		}
	}

	// 3. 同步 12 个真实测试用例
	type CaseDef struct {
		ID        string
		Num       int64
		ModuleID  string
		Name      string
		Tags      string
	}

	cases := []CaseDef{
		{ID: "TC-AUTH-001", Num: 101, ModuleID: "mod-auth-001", Name: "验证管理员使用正确凭证成功登录并颁发有效会话令牌", Tags: `["P0", "smoke", "auth"]`},
		{ID: "TC-AUTH-002", Num: 102, ModuleID: "mod-auth-001", Name: "验证输入错误密码时登录被安全拒绝", Tags: `["P0", "security", "auth"]`},
		{ID: "TC-AUTH-003", Num: 103, ModuleID: "mod-auth-001", Name: "验证使用已认证会话检查登录态接口 /is-login", Tags: `["P1", "session"]`},
		{ID: "TC-AUTH-004", Num: 104, ModuleID: "mod-auth-001", Name: "验证注销接口 /signout 主动释放登录会话", Tags: `["P1", "signout"]`},
		{ID: "TC-ORG-001", Num: 201, ModuleID: "mod-org-001", Name: "验证创建新组织并在组织分页列表中准确检索", Tags: `["P0", "org", "crud"]`},
		{ID: "TC-ORG-002", Num: 202, ModuleID: "mod-org-001", Name: "验证组织信息更新与重命名能力", Tags: `["P1", "org", "update"]`},
		{ID: "TC-ORG-003", Num: 203, ModuleID: "mod-org-001", Name: "验证组织状态机 (禁用与重新启用) 流转", Tags: `["P1", "org", "lifecycle"]`},
		{ID: "TC-ORG-004", Num: 204, ModuleID: "mod-org-001", Name: "验证组织软删除隔离逻辑", Tags: `["P1", "org", "delete"]`},
		{ID: "TC-PRJ-001", Num: 301, ModuleID: "mod-prj-001", Name: "验证在指定组织下创建新项目并通过详情接口检索", Tags: `["P0", "project", "crud"]`},
		{ID: "TC-PRJ-002", Num: 302, ModuleID: "mod-prj-001", Name: "验证按组织维度获取项目列表", Tags: `["P1", "project", "filter"]`},
		{ID: "TC-PRJ-003", Num: 303, ModuleID: "mod-prj-001", Name: "验证项目名称与描述编辑更新", Tags: `["P1", "project", "update"]`},
		{ID: "TC-PRJ-004", Num: 304, ModuleID: "mod-prj-001", Name: "验证项目软删除后详情查询返回 404", Tags: `["P1", "project", "delete"]`},
	}

	for _, c := range cases {
		var cnt int64
		config.DB.Model(&model.FunctionalCase{}).Where("id = ?", c.ID).Count(&cnt)
		if cnt == 0 {
			fc := model.FunctionalCase{
				ID:                c.ID,
				Num:               c.Num,
				ModuleID:          c.ModuleID,
				ProjectID:         projectID,
				TemplateID:        "default-tpl",
				Name:              c.Name,
				ReviewStatus:      "PASS",
				Tags:              c.Tags,
				CaseEditType:      "STEP",
				Pos:               c.Num * 100,
				VersionID:         "100000000000001",
				RefID:             c.ID,
				LastExecuteResult: "SUCCESS",
				Deleted:           model.BitBool(false),
				PublicCase:        model.BitBool(true),
				Latest:            model.BitBool(true),
				CreateUser:        "admin",
				UpdateUser:        "admin",
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			config.DB.Create(&fc)
		}
	}
	fmt.Printf("📦 [Bootstrap] 成功同步/更新 12 个真实测试用例到项目 %s (淘汰老旧假数据)\n", projectID)
}
