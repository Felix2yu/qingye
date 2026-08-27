package services

import (
	"errors"

	"qingye/server/models"
	"qingye/server/repositories"
)

// PlantService 植物增删改查
type PlantService struct{ repo *repositories.PlantRepo }

func NewPlantService() *PlantService {
	return &PlantService{repo: repositories.NewPlantRepo()}
}

func (s *PlantService) Create(p *models.Plant) (*models.Plant, error) {
	if p.Name == "" {
		return nil, errors.New("植物名称不能为空")
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return s.repo.Get(p.ID)
}

func (s *PlantService) Update(p *models.Plant) (*models.Plant, error) {
	if p.ID == 0 {
		return nil, errors.New("植物 ID 不能为空")
	}
	if p.Name == "" {
		return nil, errors.New("植物名称不能为空")
	}
	if _, err := s.repo.Get(p.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	return s.repo.Get(p.ID)
}

// Delete 删除植物及其关联任务、日记
func (s *PlantService) Delete(id uint) error {
	if _, err := s.repo.Get(id); err != nil {
		return err
	}
	if err := repositories.DB.Where("plant_id = ?", id).Delete(&models.Task{}).Error; err != nil {
		return err
	}
	if err := repositories.DB.Where("plant_id = ?", id).Delete(&models.PhotoDiary{}).Error; err != nil {
		return err
	}
	if err := repositories.DB.Where("plant_id = ?", id).Delete(&models.CareLog{}).Error; err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *PlantService) Get(id uint) (*models.Plant, error) { return s.repo.Get(id) }

func (s *PlantService) List(roomID uint) ([]models.Plant, error) { return s.repo.List(roomID) }

// RoomService 房间 / 分组
type RoomService struct{ repo *repositories.RoomRepo }

func NewRoomService() *RoomService {
	return &RoomService{repo: repositories.NewRoomRepo()}
}

func (s *RoomService) Create(r *models.Room) (*models.Room, error) {
	if r.Name == "" {
		return nil, errors.New("房间名称不能为空")
	}
	if err := s.repo.Create(r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *RoomService) Update(r *models.Room) (*models.Room, error) {
	if r.ID == 0 {
		return nil, errors.New("房间 ID 不能为空")
	}
	if r.Name == "" {
		return nil, errors.New("房间名称不能为空")
	}
	if err := s.repo.Update(r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *RoomService) Delete(id uint) error {
	var count int64
	if err := repositories.DB.Model(&models.Plant{}).Where("room_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该房间下仍有植物，请先移出或删除")
	}
	return s.repo.Delete(id)
}

// ListWithStats 房间列表 + 各房间植物数量
func (s *RoomService) ListWithStats() ([]map[string]any, error) {
	rooms, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	plants, err := repositories.NewPlantRepo().List(0)
	if err != nil {
		return nil, err
	}
	stats := map[uint]int{}
	for _, p := range plants {
		stats[p.RoomID]++
	}
	out := make([]map[string]any, 0, len(rooms))
	for _, r := range rooms {
		out = append(out, map[string]any{
			"id":    r.ID,
			"name":  r.Name,
			"sort":  r.Sort,
			"count": stats[r.ID],
			"icon":  r.Icon,
		})
	}
	return out, nil
}
