package services

import (
	"errors"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"
)

// DiaryService 照片日记：分页时间线、上传元数据写入
type DiaryService struct {
	repo   *repositories.DiaryRepo
	plants *repositories.PlantRepo
}

func NewDiaryService() *DiaryService {
	return &DiaryService{
		repo:   repositories.NewDiaryRepo(),
		plants: repositories.NewPlantRepo(),
	}
}

// Create 新增日记（image 为已保存文件的 URL 路径）
func (s *DiaryService) Create(d *models.PhotoDiary) (*models.PhotoDiary, error) {
	if d.PlantID == 0 {
		return nil, errors.New("植物不能为空")
	}
	if d.Image == "" {
		return nil, errors.New("图片不能为空")
	}
	if _, err := s.plants.Get(d.PlantID); err != nil {
		return nil, errors.New("植物不存在")
	}
	if d.TakenAt.IsZero() {
		d.TakenAt = time.Now()
	}
	if err := s.repo.Create(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DiaryService) Delete(id uint) error { return s.repo.Delete(id) }

// Page 分页时间线（plantID=0 表示全部）
func (s *DiaryService) Page(plantID uint, page, pageSize int) ([]models.PhotoDiary, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	list, err := s.repo.Page(plantID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	q := repositories.DB.Model(&models.PhotoDiary{})
	if plantID > 0 {
		q = q.Where("plant_id = ?", plantID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
