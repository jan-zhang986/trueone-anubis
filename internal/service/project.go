package service

import (
	"time"

	"github.com/google/uuid"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

type ProjectService struct{}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}

func (s *ProjectService) GetProjectsByOrg(orgId string) ([]model.ProjectSimple, error) {
	var projects []model.Project
	query := config.DB.Where("deleted = 0 AND enable = 1")
	if orgId != "" && orgId != "all" {
		query = query.Where("organization_id = ?", orgId)
	}
	err := query.Order("created_at DESC").Find(&projects).Error
	if err != nil {
		return nil, err
	}

	result := make([]model.ProjectSimple, 0, len(projects))
	for _, p := range projects {
		result = append(result, model.ProjectSimple{
			ID:   p.ID,
			Name: p.Name,
		})
	}
	return result, nil
}

func (s *ProjectService) GetPublicProjects() ([]model.ProjectSimple, error) {
	return s.GetProjectsByOrg("")
}

func (s *ProjectService) GetProjectDetail(id string) (*model.Project, error) {
	var p model.Project
	err := config.DB.Where("id = ? AND deleted = 0", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *ProjectService) CreateProject(name, desc, orgId, userId string) (*model.Project, error) {
	now := time.Now().UnixMilli()
	p := &model.Project{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    desc,
		OrganizationID: orgId,
		Enable:         model.BitBool(true),
		CreateUser:     userId,
		UpdateUser:     userId,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err := config.DB.Create(p).Error
	return p, err
}

func (s *ProjectService) UpdateProject(id, name, desc, userId string) error {
	now := time.Now().UnixMilli()
	updates := map[string]interface{}{
		"updated_at": now,
		"update_user": userId,
	}
	if name != "" {
		updates["name"] = name
	}
	if desc != "" {
		updates["description"] = desc
	}
	return config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ProjectService) DeleteProject(id, userId string) error {
	now := time.Now().UnixMilli()
	return config.DB.Model(&model.Project{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":     true,
		"delete_time": now,
		"delete_user": userId,
	}).Error
}
