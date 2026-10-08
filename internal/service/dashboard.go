package service

import (
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

type DashboardService struct{}

func NewDashboardService() *DashboardService {
	return &DashboardService{}
}

type EfficiencyOverview struct {
	TotalCases      int64 `json:"totalCases"`
	TotalWorkspaces int64 `json:"totalWorkspaces"`
	TotalBugs       int64 `json:"totalBugs"`
	TotalProjects   int64 `json:"totalProjects"`
}

func (s *DashboardService) GetOverview(projectId string) (*EfficiencyOverview, error) {
	var totalCases, totalWorkspaces, totalBugs, totalProjects int64

	caseQuery := config.DB.Model(&model.FunctionalCase{}).Where("deleted = 0")
	wsQuery := config.DB.Model(&model.QualityWorkspace{}).Where("archived = 0")
	bugQuery := config.DB.Model(&model.Bug{}).Where("deleted = 0")
	projQuery := config.DB.Model(&model.Project{}).Where("deleted = 0")

	if projectId != "" && projectId != "all" {
		caseQuery = caseQuery.Where("project_id = ?", projectId)
		wsQuery = wsQuery.Where("project_id = ?", projectId)
		bugQuery = bugQuery.Where("project_id = ?", projectId)
	}

	caseQuery.Count(&totalCases)
	wsQuery.Count(&totalWorkspaces)
	bugQuery.Count(&totalBugs)
	projQuery.Count(&totalProjects)

	return &EfficiencyOverview{
		TotalCases:      totalCases,
		TotalWorkspaces: totalWorkspaces,
		TotalBugs:       totalBugs,
		TotalProjects:   totalProjects,
	}, nil
}
