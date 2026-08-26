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

// syncInterval 相邻植物之间的请求间隔。Plantbook 免费账户有按天配额，
// 过快只会提前耗尽额度并触发长时间封禁。
const syncInterval = 300 * time.Millisecond

// SyncPopular 批量同步映射表内植物到本地资料库（离线可用）。
//
// 设计原则（避免滥用免费 API 配额）：
//   - 每轮最多向 Plantbook 发起 limit 个新条目的检索，分多次点击逐步完成
//   - 本地已有同学名条目直接跳过（不发任何请求）
//   - 请求级失败（网络/HTTP 错误）立即中止本轮；仅「库内无此植物」视为
//     单条缺失，跳过继续
//   - 触发 429 限流立即中止
//
// 返回成功/失败/跳过计数、首个失败原因、限流错误，以及尚未尝试的剩余条目数。
func (s *LibraryService) SyncPopular(limit int) (added, failed, skipped, remaining int, firstErr string, throttled error) {
	if !s.plantbook.Enabled() {
		return 0, 0, 0, 0, "", nil
	}
	existing, err := s.repo.ExistingMetrics()
	if err != nil {
		existing = map[string]bool{} // 查询失败时不跳过，按全量处理
	}
	seen := map[string]bool{} // 别名可能指向同一学名，去重避免重复请求
	processed := 0
	for _, seed := range plantAliases {
		if seen[seed.Latin] {
			continue
		}
		seen[seed.Latin] = true

		// 本地已同步且含结构化指标：不发请求直接跳过；
		// 缺指标的老条目会被重新拉取补齐
		if existing[seed.Latin] {
			skipped++
			continue
		}

		// 本轮配额用尽：不再发请求
		if limit > 0 && processed >= limit {
			continue
		}

		if processed > 0 {
			time.Sleep(syncInterval)
		}
		processed++

		cands, e := s.plantbook.Search(seed.Latin)
		if IsThrottled(e) {
			throttled = e
			break
		}
		if e != nil {
			// 请求级故障：立即停止本轮，避免连续无效请求
			failed++
			firstErr = fmt.Sprintf("%s(%s): %v", seed.Zh, seed.Latin, e)
			break
		}
		if len(cands) == 0 {
			failed++ // 库内确实没有该植物，单条缺失不中止
			continue
		}

		pid := cands[0].PID
		if existing[pid] || pid == "" {
			skipped++
			continue
		}
		lib, e := s.plantbook.Detail(pid)
		if IsThrottled(e) {
			throttled = e
			break
		}
		if e != nil {
			failed++
			firstErr = fmt.Sprintf("%s(%s): %v", seed.Zh, seed.Latin, e)
			break
		}
		if lib == nil {
			failed++
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
	remaining = len(seen) - added - failed - skipped
	return added, failed, skipped, remaining, firstErr, throttled
}
