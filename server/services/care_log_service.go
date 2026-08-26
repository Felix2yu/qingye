package services

import (
	"qingye/server/models"
	"qingye/server/repositories"
)

// CareLogService 统一养护时间线查询
type CareLogService struct {
	careLogs *repositories.CareLogRepo
}

func NewCareLogService() *CareLogService {
	return &CareLogService{careLogs: repositories.NewCareLogRepo()}
}

// ListByPlant 某植物的养护时间线
func (s *CareLogService) ListByPlant(plantID uint) ([]models.CareLog, error) {
	return s.careLogs.ListByPlant(plantID)
}

// List 全局养护时间线（limit<=0 表示不限）
func (s *CareLogService) List(limit int) ([]models.CareLog, error) {
	return s.careLogs.List(limit)
}
