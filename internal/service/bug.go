package service

import (
	"time"

	"github.com/google/uuid"
	"trueone-anubis/config"
	"trueone-anubis/internal/model"
)

type BugService struct{}

func NewBugService() *BugService {
	return &BugService{}
}

type BugPageParams struct {
	ProjectID string   `json:"projectId"`
	Statuses  []string `json:"statuses"`
	Keyword   string   `json:"keyword"`
	Current   int      `json:"current"`
	PageSize  int      `json:"pageSize"`
}

func (s *BugService) GetBugPage(params BugPageParams) ([]model.Bug, int64, error) {
	if params.Current <= 0 {
		params.Current = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	var list []model.Bug
	var total int64

	query := config.DB.Model(&model.Bug{}).Where("deleted = 0")
	if params.ProjectID != "" {
		query = query.Where("project_id = ?", params.ProjectID)
	}
	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	}
	if params.Keyword != "" {
		query = query.Where("title LIKE ?", "%"+params.Keyword+"%")
	}

	query.Count(&total)
	offset := (params.Current - 1) * params.PageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(params.PageSize).Find(&list).Error

	return list, total, err
}

func (s *BugService) GetBugDetail(id string) (*model.Bug, error) {
	var b model.Bug
	err := config.DB.Where("id = ? AND deleted = 0", id).First(&b).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *BugService) AddBug(b *model.Bug, userId string) error {
	now := time.Now().UnixMilli()
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	b.CreateUser = userId
	b.UpdateUser = userId
	b.CreatedAt = now
	b.UpdatedAt = now
	if b.Platform == "" {
		b.Platform = "LOCAL"
	}
	if b.Status == "" {
		b.Status = "NEW"
	}
	return config.DB.Create(b).Error
}

func (s *BugService) UpdateBug(b *model.Bug, userId string) error {
	now := time.Now().UnixMilli()
	b.UpdateUser = userId
	b.UpdatedAt = now
	return config.DB.Model(&model.Bug{}).Where("id = ?", b.ID).Updates(b).Error
}

func (s *BugService) DeleteBug(id, userId string) error {
	now := time.Now().UnixMilli()
	return config.DB.Model(&model.Bug{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":     1,
		"delete_time": now,
		"delete_user": userId,
	}).Error
}
