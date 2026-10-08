package handler

import (
	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/response"
	"trueone-anubis/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{
		dashboardService: service.NewDashboardService(),
	}
}

func (h *DashboardHandler) EfficiencyOverview(c *gin.Context) {
	projectId := c.GetHeader("PROJECT")
	if projectId == "" {
		projectId = c.Query("projectId")
	}

	overview, err := h.dashboardService.GetOverview(projectId)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, overview)
}

func (h *DashboardHandler) EfficiencyActivity(c *gin.Context) {
	response.Success(c, gin.H{
		"series":    []interface{}{},
		"breakdown": gin.H{},
	})
}

func (h *DashboardHandler) RequirementQualityMetrics(c *gin.Context) {
	response.Success(c, gin.H{
		"workspaceCount": 4,
		"caseCount":      14,
		"passRate":       100.0,
		"bugCount":       0,
	})
}

func (h *DashboardHandler) NotificationUnRead(c *gin.Context) {
	response.Success(c, 0)
}

func (h *DashboardHandler) PrecisionTestCov(c *gin.Context) {
	// Precision test coverage mock/proxy endpoint
	response.Success(c, gin.H{
		"branch":         "master",
		"lineCoverage":   88.5,
		"branchCoverage": 82.0,
		"status":         "SUCCESS",
	})
}

func (h *DashboardHandler) RequirementQualityList(c *gin.Context) {
	response.SuccessPage(c, []interface{}{}, 0, 1, 10)
}

func (h *DashboardHandler) RequirementQualityOverview(c *gin.Context) {
	response.Success(c, gin.H{
		"currentDemandCount": 4,
		"caseCount":          14,
		"bugCount":           0,
		"passRate":           100.0,
	})
}

func (h *DashboardHandler) RequirementQualityFilterOptions(c *gin.Context) {
	response.Success(c, gin.H{
		"projectList": []interface{}{},
		"statusList":  []interface{}{},
	})
}

func (h *DashboardHandler) RequirementQualityDetail(c *gin.Context) {
	response.Success(c, gin.H{})
}

