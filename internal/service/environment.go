package service

import (
	"time"

	"github.com/google/uuid"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

type EnvironmentService struct{}

func NewEnvironmentService() *EnvironmentService {
	return &EnvironmentService{}
}

func (s *EnvironmentService) GetList(projectId, keyword string) ([]model.Environment, error) {
	var envs []model.Environment
	query := config.DB.Where("project_id = ?", projectId)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	err := query.Order("pos ASC, created_at DESC").Find(&envs).Error
	return envs, err
}

func (s *EnvironmentService) GetDetail(id string) (*model.Environment, error) {
	var env model.Environment
	err := config.DB.Where("id = ?", id).First(&env).Error
	if err != nil {
		return nil, err
	}
	return &env, nil
}

func (s *EnvironmentService) Add(env *model.Environment, userId string) error {
	now := time.Now().UnixMilli()
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	env.CreateUser = userId
	env.UpdateUser = userId
	env.CreatedAt = now
	env.UpdatedAt = now
	return config.DB.Create(env).Error
}

func (s *EnvironmentService) Update(env *model.Environment, userId string) error {
	now := time.Now().UnixMilli()
	env.UpdateUser = userId
	env.UpdatedAt = now
	return config.DB.Model(&model.Environment{}).Where("id = ?", env.ID).Updates(env).Error
}

func (s *EnvironmentService) Delete(id string) error {
	return config.DB.Where("id = ?", id).Delete(&model.Environment{}).Error
}
