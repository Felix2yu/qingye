package services

import (
	"fmt"
	"time"

	"qingye/server/config"
	"qingye/server/models"
	"qingye/server/repositories"
)

// LibraryService 资料库业务逻辑
type LibraryService struct {
	repo      *repositories.LibraryRepo
	plantbook *PlantbookClient
}

func NewLibraryService(cfg *config.Config) *LibraryService {
	return &LibraryService{
		repo:      repositories.NewLibraryRepo(),
		plantbook: NewPlantbookClient(cfg.PlantbookClientID, cfg.PlantbookSecret, cfg.PlantbookToken),
	}
}

func (s *LibraryService) Search(keyword string) ([]models.PlantLibrary, error) {
	return s.repo.Search(keyword)
}

// OnlineEnabled 在线匹配是否可用（取决于是否配置了 token）
func (s *LibraryService) OnlineEnabled() bool {
	return s.plantbook.Enabled()
}

// SearchOnline 在线搜索候选（未配置 token 时返回空列表）
func (s *LibraryService) SearchOnline(keyword string) ([]OnlineCandidate, error) {
	return s.plantbook.Search(keyword)
}

// ImportOnline 按 pid 拉取详情并写回本地资料库（pid 作唯一键，重同步不冲突）。
// 返回沉淀后的本地条目。
func (s *LibraryService) ImportOnline(pid string) (*models.PlantLibrary, error) {
	lib, err := s.plantbook.Detail(pid)
	if err != nil {
		return nil, err
	}
	if lib == nil {
		return nil, nil
	}
	lib.SyncedAt = time.Now().Unix()
	if err := s.repo.UpsertByPID(lib); err != nil {
		return nil, err
	}
	return s.repo.GetByPID(pid)
}

// SyncPopular 批量同步映射表内全部植物到本地资料库（离线可用）。
// 对每个学名 search 取首个候选的 pid，再 detail?lang=zh 写回；
// 详情缺少中文 common name 时以表中中文名作为展示名。
// 已存在（同 pid）的条目会被覆盖刷新。返回成功/失败计数与首个失败原因。
// 注意：全表数百条 × (search+detail+间隔)，耗时数分钟，属长任务。
func (s *LibraryService) SyncPopular() (added int, failed int, firstErr string) {
	if !s.plantbook.Enabled() {
		return 0, 0, ""
	}
	seen := map[string]bool{} // 别名可能指向同一学名，去重避免重复请求
	for i, seed := range plantAliases {
		if seen[seed.Latin] {
			continue
		}
		seen[seed.Latin] = true
		// 轻微间隔，避免连续数百个请求触发服务端限流
		if i > 0 {
			time.Sleep(150 * time.Millisecond)
		}
		cands, e := s.plantbook.Search(seed.Latin)
		if e != nil {
			failed++
			if firstErr == "" {
				firstErr = fmt.Sprintf("%s(%s): %v", seed.Zh, seed.Latin, e)
			}
			continue
		}
		if len(cands) == 0 {
			failed++
			continue
		}
		lib, e := s.plantbook.Detail(cands[0].PID)
		if e != nil || lib == nil {
			failed++
			if firstErr == "" && e != nil {
				firstErr = fmt.Sprintf("%s(%s): %v", seed.Zh, seed.Latin, e)
			}
			continue
		}
		// 展示名兜底：详情未带中文 common name 时使用表中中文名
		if lib.Name != "" && !isChinese(lib.Name) && isChinese(seed.Zh) {
			lib.Name = seed.Zh
		}
		lib.SyncedAt = time.Now().Unix()
		if e := s.repo.UpsertByPID(lib); e != nil {
			failed++
			continue
		}
		added++
	}
	return added, failed, firstErr
}
