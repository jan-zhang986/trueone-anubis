package service

import (
	"time"

	"github.com/google/uuid"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

type TestCaseService struct{}

func NewTestCaseService() *TestCaseService {
	return &TestCaseService{}
}

type CasePageParams struct {
	ProjectID string   `json:"projectId"`
	ModuleIDs []string `json:"moduleIds"`
	Keyword   string   `json:"keyword"`
	Current   int      `json:"current"`
	PageSize  int      `json:"pageSize"`
}

func (s *TestCaseService) GetCasePage(params CasePageParams) ([]model.FunctionalCase, int64, error) {
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var list []model.FunctionalCase
	var total int64

	query := config.DB.Model(&model.FunctionalCase{}).Where("deleted = 0")
	if params.ProjectID != "" {
		query = query.Where("project_id = ?", params.ProjectID)
	}
	if len(params.ModuleIDs) > 0 {
		query = query.Where("module_id IN ?", params.ModuleIDs)
	}
	if params.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+params.Keyword+"%")
	}

	query.Count(&total)
	offset := (params.Current - 1) * params.PageSize
	err := query.Order("pos ASC, created_at DESC").Offset(offset).Limit(params.PageSize).Find(&list).Error

	return list, total, err
}

func (s *TestCaseService) GetCaseDetail(id string) (*model.FunctionalCase, error) {
	var c model.FunctionalCase
	err := config.DB.Where("id = ? AND deleted = 0", id).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *TestCaseService) AddCase(c *model.FunctionalCase, userId string) error {
	now := time.Now().UnixMilli()
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	c.CreateUser = userId
	c.UpdateUser = userId
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.ReviewStatus == "" {
		c.ReviewStatus = "UN_REVIEWED"
	}
	if c.LastExecuteResult == "" {
		c.LastExecuteResult = "UN_EXECUTED"
	}
	return config.DB.Create(c).Error
}

func (s *TestCaseService) UpdateCase(c *model.FunctionalCase, userId string) error {
	now := time.Now().UnixMilli()
	c.UpdateUser = userId
	c.UpdatedAt = now
	return config.DB.Model(&model.FunctionalCase{}).Where("id = ?", c.ID).Updates(c).Error
}

func (s *TestCaseService) DeleteCase(id, userId string) error {
	now := time.Now().UnixMilli()
	return config.DB.Model(&model.FunctionalCase{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":     1,
		"delete_time": now,
		"delete_user": userId,
	}).Error
}

func (s *TestCaseService) GetModuleTree(projectId string) ([]*model.FunctionalCaseModule, error) {
	var modules []model.FunctionalCaseModule
	err := config.DB.Where("project_id = ?", projectId).Order("pos ASC, created_at ASC").Find(&modules).Error
	if err != nil {
		return nil, err
	}

	// Calculate case count per module
	type countResult struct {
		ModuleID string
		Count    int64
	}
	var counts []countResult
	config.DB.Model(&model.FunctionalCase{}).
		Select("module_id, count(*) as count").
		Where("project_id = ? AND deleted = 0", projectId).
		Group("module_id").
		Scan(&counts)

	countMap := make(map[string]int64)
	for _, c := range counts {
		countMap[c.ModuleID] = c.Count
	}

	moduleMap := make(map[string]*model.FunctionalCaseModule)
	var rootNodes []*model.FunctionalCaseModule

	for i := range modules {
		mod := &modules[i]
		mod.CaseCount = countMap[mod.ID]
		mod.Children = make([]*model.FunctionalCaseModule, 0)
		moduleMap[mod.ID] = mod
	}

	for _, mod := range moduleMap {
		if mod.ParentID == "" || mod.ParentID == "NONE" || mod.ParentID == "root" {
			rootNodes = append(rootNodes, mod)
		} else if parent, ok := moduleMap[mod.ParentID]; ok {
			parent.Children = append(parent.Children, mod)
		} else {
			rootNodes = append(rootNodes, mod)
		}
	}

	return rootNodes, nil
}

func (s *TestCaseService) AddModule(m *model.FunctionalCaseModule, userId string) error {
	now := time.Now().UnixMilli()
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.ParentID == "" {
		m.ParentID = "NONE"
	}
	m.CreateUser = userId
	m.UpdateUser = userId
	m.CreatedAt = now
	m.UpdatedAt = now
	return config.DB.Create(m).Error
}

func (s *TestCaseService) UpdateModule(m *model.FunctionalCaseModule, userId string) error {
	now := time.Now().UnixMilli()
	m.UpdateUser = userId
	m.UpdatedAt = now
	return config.DB.Model(&model.FunctionalCaseModule{}).Where("id = ?", m.ID).Updates(m).Error
}

func (s *TestCaseService) DeleteModule(id string) error {
	return config.DB.Where("id = ? OR parent_id = ?", id, id).Delete(&model.FunctionalCaseModule{}).Error
}
