package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

// Constants aligning with OperationLogModule and OperationLogType in legacy system
const (
	ModuleSystem                 = "SYSTEM"
	ModuleProject                = "PROJECT_PROJECT_MANAGER"
	ModuleEnvironment            = "PROJECT_ENVIRONMENT_SETTING"
	ModuleCase                   = "CASE_MANAGEMENT_CASE_CASE"
	ModuleBug                    = "BUG_MANAGEMENT_INDEX"
	ModuleQualityWorkspace       = "QUALITY_WORKSPACE"
	ModuleSettingUser            = "SETTING_SYSTEM_USER_SINGLE"
	ModuleOrgMember              = "ORGANIZATION_MEMBER"
	ModuleOrgProject             = "ORGANIZATION_PROJECT"

	OpTypeAdd    = "ADD"
	OpTypeUpdate = "UPDATE"
	OpTypeDelete = "DELETE"
	OpTypeLogin  = "LOGIN"
	OpTypeLogout = "LOGOUT"
)

// RecordAuditLog 统一异步记录操作日志（完全对齐老系统 OperationLogAspect 规范）
func RecordAuditLog(c *gin.Context, module, opType, sourceID, content string) {
	// 1. 获取操作用户
	userID := "admin"
	if u, exists := c.Get("userId"); exists && u != nil && u.(string) != "" {
		userID = u.(string)
	} else if userHeader := c.GetHeader("X-User-Id"); userHeader != "" {
		userID = userHeader
	}

	// 2. 获取组织与项目上下文
	orgID := c.GetHeader("ORGANIZATION")
	if orgID == "" {
		orgID = "100001"
	}

	projID := c.GetHeader("PROJECT")
	if projID == "" {
		projID = "100001100001"
	}

	// 针对系统级操作的特殊标记
	if module == ModuleSystem || module == ModuleSettingUser {
		orgID = "SYSTEM"
		projID = "SYSTEM"
	}

	if sourceID == "" {
		sourceID = projID
	}

	method := ""
	path := ""
	if c.Request != nil {
		method = c.Request.Method
		if c.Request.URL != nil {
			path = c.Request.URL.Path
		}
	}

	go func(uid, oid, pid, sid, meth, pth, mod, ot, cnt string) {
		defer func() {
			if r := recover(); r != nil {
				// Prevent audit logging from crashing the app
			}
		}()

		// 3. 构建操作日志记录
		logRecord := model.OperationLog{
			ProjectID:      pid,
			OrganizationID: oid,
			CreatedAt:      time.Now().UnixMilli(),
			CreateUser:     uid,
			SourceID:       sid,
			Method:         meth,
			Type:           ot,
			Module:         mod,
			Content:        cnt,
			Path:           pth,
		}

		config.DB.Create(&logRecord)
	}(userID, orgID, projID, sourceID, method, path, module, opType, content)
}
