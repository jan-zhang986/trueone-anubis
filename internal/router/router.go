package router

import (
	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/handler"
	"trueone-anubis/internal/middleware"
	"trueone-anubis/internal/response"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Global Middleware
	r.Use(middleware.Cors())
	r.Use(middleware.AuthMiddleware())

	authH := handler.NewAuthHandler()
	projH := handler.NewProjectHandler()
	wsH := handler.NewQualityWorkspaceHandler()
	caseH := handler.NewTestCaseHandler()
	bugH := handler.NewBugHandler()
	dashH := handler.NewDashboardHandler()
	sysH := handler.NewSystemHandler()
	msgH := handler.NewMessageHandler()
	orgH := handler.NewOrganizationHandler()
	logH := handler.NewLogHandler()
	menuH := handler.NewMenuHandler()
	wfH := handler.NewWorkflowHandler()
	repoCaseH := handler.NewRepoCaseHandler()

	// Helper to register routes on both root and /api prefixes
	register := func(rg *gin.RouterGroup) {
		// 1. Auth & Session
		rg.POST("/login", authH.Login)
		rg.GET("/is-login", authH.IsLogin)
		rg.GET("/signout", authH.Signout)
		rg.GET("/get-key", authH.GetKey)
		rg.GET("/lark/info", authH.LarkInfo)
		rg.POST("/project/switch", authH.SwitchProject)
		rg.POST("/user/update-current-password", authH.UpdateCurrentPassword)
		rg.POST("/system/user/password/reset", authH.ResetPassword)
		rg.POST("/system/user/edit-password", authH.ResetPassword)

		// 2. Project & Environment
		rg.GET("/project/list/options/:orgId", projH.GetProjectsByOrg)
		rg.GET("/project/list/:orgId", projH.GetProjectsByOrg)
		rg.GET("/project/list/public", projH.GetPublicProjects)
		rg.GET("/project/list", projH.GetPublicProjects)
		rg.GET("/project/get/:id", projH.GetProjectDetail)
		rg.GET("/projects/:id", projH.GetProjectDetail)
		rg.POST("/projects", projH.CreateProject)
		rg.POST("/project/update", projH.UpdateProject)
		rg.PUT("/projects/:id", projH.UpdateProject)
		rg.DELETE("/projects/:id", projH.DeleteProject)
		rg.GET("/project/version/options/:projectId", func(c *gin.Context) { response.Success(c, []interface{}{}) })

		// Case Repository (Test as Code Explorer)
		rg.GET("/case/repository/list", repoCaseH.ListRepositories)
		rg.POST("/case/repository/create", repoCaseH.CreateRepository)
		rg.GET("/case/repository/get-detail/:id", repoCaseH.GetRepositoryDetail)
		rg.POST("/case/repository/update", repoCaseH.UpdateRepository)
		rg.POST("/case/repository/delete/:id", repoCaseH.DeleteRepository)
		rg.GET("/case/repository/:id/tree", repoCaseH.GetRepoTree)
		rg.GET("/case/repository/:id/cases", repoCaseH.QueryCases)
		rg.GET("/case/repository/:id/case-detail", repoCaseH.GetCaseDetail)
		rg.GET("/case/repository/:id/branches", repoCaseH.GetRepositoryBranches)
		rg.POST("/case/repository/:id/branches", repoCaseH.CreateRepositoryBranch)
		rg.POST("/case/repository/:id/sync", repoCaseH.SyncRepository)
		rg.POST("/case/repository/:id/execute-case", repoCaseH.ExecuteCase)

		// Environment
		rg.POST("/project/environment/list", projH.GetEnvironmentList)
		rg.GET("/project/environment/:id", projH.GetEnvironmentDetail)
		rg.POST("/project/environment/add", projH.AddEnvironment)
		rg.POST("/project/environment/update", projH.UpdateEnvironment)
		rg.GET("/project/environment/delete/:id", projH.DeleteEnvironment)
		rg.POST("/project/environment/delete/:id", projH.DeleteEnvironment)

		// 3. Quality Workspace
		rg.POST("/quality-workspace/page", wsH.GetPage)
		rg.GET("/quality-workspace/:id", wsH.GetDetail)
		rg.POST("/quality-workspace/save", wsH.Save)
		rg.POST("/quality-workspace/:id/archive", wsH.Archive)
		rg.POST("/quality-workspace/:id/delete", wsH.Delete)
		rg.GET("/quality-workspace/:id/stats", wsH.GetStats)

		// Quality Tasks & WorkItems
		rg.GET("/quality-workspace/:id/task/list", wsH.GetTaskList)
		rg.POST("/quality-workspace/:id/task/save", wsH.SaveTask)
		rg.POST("/quality-workspace/:id/task/:taskId/complete", wsH.CompleteTask)
		rg.POST("/quality-workspace/:id/task/:taskId/reopen", wsH.ReopenTask)
		rg.POST("/quality-workspace/:id/task/:taskId/work-item/page", wsH.GetWorkItemPage)
		rg.POST("/quality-workspace/:id/task/:taskId/work-item/:workItemId/run", wsH.RunWorkItem)

		// Quality Reports
		rg.POST("/quality-workspace/report/page", wsH.GetReportPage)
		rg.GET("/quality-workspace/report/:reportId", wsH.GetReportDetail)
		rg.POST("/quality-workspace/:id/report/generate", wsH.GenerateReport)

		// Workflow DAG Engine
		rg.POST("/workflow/execute", wfH.ExecuteGraph)

		// Living Test Plan Diff Matrix (Code-as-Spec & Sync)
		rg.GET("/quality-workspace/:id/diff-matrix", wsH.GetDiffMatrix)
		rg.POST("/quality-workspace/:id/diff-matrix/sync", wsH.SyncDiffMatrix)
		rg.POST("/quality-workspace/:id/diff-matrix/case/report", wsH.ReportCaseExecution)
		rg.POST("/quality-workspace/:id/diff-matrix/generate", wsH.GenerateDiffMatrix)
		rg.POST("/quality-workspace/:id/diff-matrix/auto-parse", wsH.AutoParseDiffMatrix)

		// 4. Test Case (Functional Case)
		rg.POST("/functional/case/page", caseH.GetCasePage)
		rg.POST("/testcase/page", caseH.GetCasePage)
		rg.POST("/test-plan/page", func(c *gin.Context) { response.SuccessPage(c, []interface{}{}, 0, 1, 10) })
		rg.POST("/functional/case/detail", caseH.GetCaseDetail)
		rg.GET("/functional/case/detail", caseH.GetCaseDetail)
		rg.POST("/functional/case/add", caseH.AddCase)
		rg.POST("/functional/case/update", caseH.UpdateCase)
		rg.POST("/functional/case/delete", caseH.DeleteCase)
		rg.GET("/functional/case/module/tree", caseH.GetModuleTree)
		rg.POST("/functional/case/module/tree", caseH.GetModuleTree)
		rg.GET("/functional/case/module/tree/:projectId", caseH.GetModuleTreeByParam)
		rg.POST("/functional/case/module/tree/:projectId", caseH.GetModuleTreeByParam)
		rg.POST("/functional/case/module/add", caseH.AddModule)
		rg.POST("/functional/case/module/update", caseH.UpdateModule)
		rg.POST("/functional/case/module/delete", caseH.DeleteModule)
		rg.POST("/functional/case/custom/field", caseH.CustomField)
		rg.GET("/functional/case/default/template/field", caseH.DefaultTemplateField)
		rg.GET("/functional/case/default/template/field/:projectId", caseH.DefaultTemplateField)
		rg.POST("/functional/case/module/count", caseH.ModuleCount)
		rg.POST("/functional/case/trash/module/count", caseH.TrashModuleCount)

		// 5. Bug Management
		rg.POST("/bug/page", bugH.GetBugPage)
		rg.GET("/bug/get/:id", bugH.GetBugDetail)
		rg.POST("/bug/add", bugH.AddBug)
		rg.POST("/bug/update", bugH.UpdateBug)
		rg.GET("/bug/delete/:id", bugH.DeleteBug)
		rg.POST("/bug/delete/:id", bugH.DeleteBug)
		rg.GET("/bug/current-platform/:projectId", bugH.CurrentPlatform)
		rg.GET("/bug/header/custom-field/:projectId", bugH.CustomFieldHeader)
		rg.GET("/bug/header/columns-option/:projectId", bugH.ColumnsOption)
		rg.GET("/bug/template/option", bugH.TemplateOption)
		rg.GET("/bug/template/option/:projectId", bugH.TemplateOption)
		rg.POST("/bug/template/option", bugH.TemplateOption)
		rg.GET("/bug/feishu/*any", bugH.FeishuOptions)

		// 6. Dashboard & Metrics
		rg.POST("/metrics/efficiency/overview", dashH.EfficiencyOverview)
		rg.GET("/metrics/efficiency/overview", dashH.EfficiencyOverview)
		rg.POST("/metrics/efficiency/activity", dashH.EfficiencyActivity)
		rg.GET("/metrics/efficiency/activity", dashH.EfficiencyActivity)
		rg.GET("/metrics/requirement-quality", dashH.RequirementQualityMetrics)
		rg.POST("/metrics/requirement-quality", dashH.RequirementQualityMetrics)
		rg.POST("/metrics/requirement-quality/list", dashH.RequirementQualityList)
		rg.POST("/metrics/requirement-quality/overview", dashH.RequirementQualityOverview)
		rg.GET("/metrics/requirement-quality/filter-options", dashH.RequirementQualityFilterOptions)
		rg.GET("/metrics/requirement-quality/detail/:storyId", dashH.RequirementQualityDetail)
		rg.GET("/notification/un-read/:projectId", dashH.NotificationUnRead)

		// Precision Test Cov
		rg.GET("/cov/*any", dashH.PrecisionTestCov)
		rg.POST("/cov/*any", dashH.PrecisionTestCov)

		// 7. System Setting & Members
		rg.GET("/system/user", sysH.UserList)
		rg.POST("/system/user/list", sysH.UserPage)
		rg.POST("/system/user/page", sysH.UserPage)
		rg.GET("/system/user/get/global/system/role", sysH.SystemRoles)
		rg.GET("/user/role/global/list", sysH.GlobalUserGroupList)
		rg.POST("/user/role/relation/global/list", sysH.GlobalUserGroupMemberList)
		rg.GET("/user/role/global/permission/setting/:roleId", sysH.GlobalPermissionSetting)
		rg.POST("/user/role/global/permission/update", sysH.GlobalPermissionUpdate)
		rg.GET("/user/role/relation/global/user/option/:roleId", sysH.GlobalUserGroupMemberOption)
		rg.POST("/user/role/relation/global/add", sysH.GlobalUserGroupAddMember)
		rg.GET("/user/role/relation/global/delete/:relationId", sysH.GlobalUserGroupDeleteMember)

		rg.GET("/system/organization/total", sysH.OrganizationTotal)
		rg.POST("/system/organization/list", sysH.OrganizationList)
		rg.POST("/system/organization/page", sysH.OrganizationList)
		rg.POST("/system/organization/add", sysH.SystemOrgAdd)
		rg.POST("/system/organization/update", sysH.SystemOrgUpdate)
		rg.POST("/system/organization/rename", sysH.SystemOrgRename)
		rg.GET("/system/organization/delete/:id", sysH.SystemOrgDelete)
		rg.GET("/system/organization/enable/:id", sysH.SystemOrgEnable)
		rg.GET("/system/organization/disable/:id", sysH.SystemOrgDisable)
		rg.GET("/system/organization/recover/:id", sysH.SystemOrgRevoke)
		rg.POST("/system/organization/option/all", sysH.SystemOrgOptionsAll)
		rg.GET("/system/organization/option/all", sysH.SystemOrgOptionsAll)
		rg.POST("/system/organization/member-list", orgH.GetMemberList)
		rg.POST("/system/organization/add-member", orgH.AddMember)
		rg.POST("/system/organization/list-member", orgH.GetMemberList)
		rg.GET("/system/organization/remove-member/:organizationId/:userId", orgH.RemoveMember)

		rg.POST("/system/project/list", sysH.SystemProjectList)
		rg.POST("/system/project/page", sysH.SystemProjectList)
		rg.POST("/system/project/add", sysH.SystemProjectAdd)
		rg.POST("/system/project/update", sysH.SystemProjectUpdate)
		rg.POST("/system/project/rename", sysH.SystemProjectRename)
		rg.GET("/system/project/delete/:id", sysH.SystemProjectDelete)
		rg.GET("/system/project/enable/:id", sysH.SystemProjectEnable)
		rg.GET("/system/project/disable/:id", sysH.SystemProjectDisable)
		rg.GET("/system/project/revoke/:id", sysH.SystemProjectRevoke)
		rg.GET("/system/project/user-list", sysH.SystemProjectUserList)
		rg.POST("/system/project/member-list", sysH.ProjectMemberList)
		rg.POST("/system/project/add-member", orgH.AddMember)
		rg.GET("/system/project/remove-member/:projectId/:userId", orgH.RemoveMember)
		rg.GET("/system/parameter/get/base-info", sysH.BaseInfo)
		rg.GET("/system/parameter/get/email-info", sysH.EmailInfo)
		rg.POST("/system/parameter/edit/email-info", sysH.EmailInfoSave)
		rg.POST("/system/parameter/test/email", sysH.EmailTest)
		rg.GET("/system/parameter/get/clean-config", sysH.CleanConfig)
		rg.POST("/system/parameter/edit/clean-config", sysH.CleanConfigSave)

		rg.POST("/system/authsource/list", sysH.AuthSourceList)
		rg.GET("/setting/get/platform/info", sysH.PlatformInfo)
		rg.GET("/lark/info/with_detail", sysH.LarkInfoWithDetail)
		rg.POST("/lark/save", sysH.LarkSave)
		rg.POST("/lark/validate", sysH.LarkValidate)
		rg.POST("/lark/enable", sysH.LarkEnable)
		rg.GET("/display/info", sysH.DisplayInfo)

		// System Menu Management
		rg.GET("/system/menu/tree", menuH.GetMenuTree)
		rg.GET("/system/menu/user-tree", menuH.GetUserMenuTree)
		rg.POST("/system/menu/add", menuH.AddMenu)
		rg.POST("/system/menu/update", menuH.UpdateMenu)
		rg.POST("/system/menu/delete/:id", menuH.DeleteMenu)
		rg.GET("/system/menu/delete/:id", menuH.DeleteMenu)

		rg.GET("/user/profile", sysH.UserProfile)
		rg.POST("/user/profile/page", sysH.UserProfilePage)
		rg.POST("/project/member/list", sysH.ProjectMemberList)
		rg.GET("/project/member/get-member/option/:projectId", sysH.ProjectMemberOptions)
		rg.POST("/user/role/project/list", sysH.UserRoleProjectList)

		// 8. Message Management & Robots
		rg.GET("/notice/message/task/get/:projectId", msgH.MessageList)
		rg.POST("/notice/message/task/save", msgH.MessageSave)
		rg.GET("/notice/message/task/get/user", msgH.MessageUserList)
		rg.GET("/notice/message/template/detail", msgH.MessageDetail)
		rg.GET("/notice/template/get/fields", msgH.MessageFields)

		rg.GET("/project/robot/list/:projectId", msgH.RobotList)
		rg.GET("/project/robot/list", msgH.RobotList)
		rg.POST("/project/robot/add", msgH.RobotAdd)
		rg.POST("/project/robot/update", msgH.RobotUpdate)
		rg.GET("/project/robot/delete/:id", msgH.RobotDelete)
		rg.POST("/project/robot/delete/:id", msgH.RobotDelete)
		rg.POST("/project/robot/enable", msgH.RobotToggle)

		// 9. Organization Management
		// Organization Members
		rg.POST("/organization/member/list", orgH.GetMemberList)
		rg.GET("/organization/user/role/list/:organizationId", orgH.GetUserRoleOptions)
		rg.GET("/organization/project/list/:organizationId", orgH.GetProjectOptions)
		rg.GET("/organization/not-exist/user/list/:organizationId", orgH.GetAvailableUsers)
		rg.POST("/organization/add-member", orgH.AddMember)
		rg.POST("/organization/update-member", orgH.UpdateMember)
		rg.GET("/organization/remove-member/:organizationId/:userId", orgH.RemoveMember)
		rg.POST("/organization/user/invite", orgH.InviteMember)
		rg.POST("/organization/batch-add-project", orgH.BatchAddProject)

		// Organization Projects
		rg.POST("/organization/project/page", orgH.GetProjectPage)
		rg.POST("/organization/project/add", orgH.AddProject)
		rg.POST("/organization/project/update", orgH.UpdateProject)
		rg.POST("/organization/project/rename", orgH.RenameProject)
		rg.GET("/organization/project/delete/:id", orgH.DeleteProject)
		rg.GET("/organization/project/enable/:id", orgH.EnableProject)
		rg.GET("/organization/project/disable/:id", orgH.DisableProject)

		// Organization User Roles / Groups
		rg.GET("/user/role/organization/list/:organizationId", orgH.GetOrgUserRoleList)
		rg.POST("/user/role/organization/list-member", orgH.OrgUserGroupMemberList)
		rg.GET("/user/role/organization/get-member/option/:organizationId/:roleId", orgH.OrgUserGroupMemberOption)
		rg.POST("/user/role/organization/add", func(c *gin.Context) { response.Success(c, "ok") })
		rg.POST("/user/role/organization/update", func(c *gin.Context) { response.Success(c, "ok") })
		rg.GET("/user/role/organization/delete/:id", func(c *gin.Context) { response.Success(c, "ok") })
		rg.POST("/user/role/organization/add-member", orgH.OrgUserGroupAddMember)
		rg.GET("/user/role/organization/remove-member/:id", orgH.OrgUserGroupDeleteMember)
		rg.GET("/user/role/organization/permission/setting/:roleId", orgH.OrgPermissionSetting)
		rg.POST("/user/role/organization/permission/update", orgH.OrgPermissionUpdate)

		// Service Integration
		rg.GET("/service/integration/list/:organizationId", orgH.ServiceIntegrationList)
		rg.POST("/service/integration/add", orgH.ServiceIntegrationAdd)
		rg.POST("/service/integration/update", orgH.ServiceIntegrationUpdate)
		rg.GET("/service/integration/delete/:id", orgH.ServiceIntegrationDelete)
		rg.GET("/service/integration/validate/:id", orgH.ServiceIntegrationValidate)
		rg.POST("/service/integration/validate/*any", orgH.ServiceIntegrationValidate)
		rg.GET("/service/integration/script/*any", orgH.ServiceIntegrationScript)

		// Organization Template, Custom Field, Status Flow & Log
		rg.GET("/organization/template/list/:organizationId/:scene", func(c *gin.Context) { response.Success(c, []interface{}{}) })
		rg.GET("/organization/template/get/:id", func(c *gin.Context) { response.Success(c, map[string]interface{}{}) })
		rg.POST("/organization/template/add", func(c *gin.Context) { response.Success(c, "ok") })
		rg.POST("/organization/template/update", func(c *gin.Context) { response.Success(c, "ok") })
		rg.GET("/organization/template/delete/:id", func(c *gin.Context) { response.Success(c, "ok") })
		rg.GET("/organization/template/enable/config/:organizationId", func(c *gin.Context) { response.Success(c, map[string]interface{}{}) })
		rg.GET("/organization/template/disable/:organizationId/:scene", func(c *gin.Context) { response.Success(c, "ok") })

		rg.GET("/organization/custom/field/list/:scopeId/:scene", func(c *gin.Context) { response.Success(c, []interface{}{}) })
		rg.POST("/organization/custom/field/add", func(c *gin.Context) { response.Success(c, "ok") })
		rg.POST("/organization/custom/field/update", func(c *gin.Context) { response.Success(c, "ok") })
		rg.GET("/organization/custom/field/delete", func(c *gin.Context) { response.Success(c, "ok") })
		rg.GET("/organization/custom/field/get/:id", func(c *gin.Context) { response.Success(c, map[string]interface{}{}) })

		rg.GET("/organization/status/flow/setting/get/:scopeId/:scene", func(c *gin.Context) { response.Success(c, []interface{}{}) })
		rg.POST("/organization/status/flow/setting/status/add", func(c *gin.Context) { response.Success(c, []interface{}{}) })
		rg.POST("/organization/status/flow/setting/status/update", func(c *gin.Context) { response.Success(c, []interface{}{}) })
		rg.GET("/organization/status/flow/setting/status/delete", func(c *gin.Context) { response.Success(c, "ok") })
		rg.POST("/organization/status/flow/setting/status/definition/update", func(c *gin.Context) { response.Success(c, "ok") })
		rg.POST("/organization/status/flow/setting/status/sort/*any", func(c *gin.Context) { response.Success(c, "ok") })
		rg.POST("/organization/status/flow/setting/status/flow/update", func(c *gin.Context) { response.Success(c, "ok") })

		// Logs: System, Organization, Project
		rg.POST("/operation/log/list", logH.SystemLogList)
		rg.GET("/operation/log/get/options", logH.SystemLogOptions)
		rg.GET("/operation/log/user/list", logH.SystemLogUsers)

		rg.POST("/organization/log/list", logH.OrgLogList)
		rg.GET("/organization/log/get/options/:organizationId", logH.OrgLogOptions)
		rg.GET("/organization/log/user/list/:organizationId", logH.OrgLogUsers)

		rg.POST("/project/log/list", logH.ProjectLogList)
		rg.GET("/project/log/user/list/:projectId", logH.ProjectLogUsers)

		rg.POST("/organization/task-center/*any", func(c *gin.Context) { response.Success(c, []interface{}{}) })
	}

	rootGroup := r.Group("")
	register(rootGroup)

	apiGroup := r.Group("/api")
	register(apiGroup)

	return r
}
