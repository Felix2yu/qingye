package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"qingye/server/config"
	"qingye/server/models"
	"qingye/server/repositories"
)

// LibraryService 资料库业务逻辑
type LibraryService struct {
	repo          *repositories.LibraryRepo
	plantbook     *PlantbookClient
	syncStatePath string // 持久化「在线库未收录」学名的文件路径，避免反复请求未收录植物
}

func NewLibraryService(cfg *config.Config) *LibraryService {
	pb := NewPlantbookClient(cfg.PlantbookClientID, cfg.PlantbookSecret, cfg.PlantbookToken)
	// 客户端侧每日配额护栏：持久化到数据库同目录，避免跨重启重复计数。
	// Plantbook 免费账户 200 次/天，提前拦截以免触发服务端 429 长时封禁。
	if quotaPath := filepath.Join(filepath.Dir(cfg.DBPath), "plantbook_quota.json"); quotaPath != "" {
		pb.quota = newDailyQuota(plantbookDailyLimit, quotaPath)
	}
	return &LibraryService{
		repo:          repositories.NewLibraryRepo(),
		plantbook:     pb,
		syncStatePath: filepath.Join(filepath.Dir(cfg.DBPath), "plantbook_sync_state.json"),
	}
}

// plantbookDailyLimit Plantbook 免费账户每日请求上限（2025-11-01 起生效）。
const plantbookDailyLimit = 200

// SyncProgress 单条植物的同步进度事件，由 SyncPopularStream 通过回调实时推送。
type SyncProgress struct {
	Type      string `json:"type"`      // "progress"
	Index     int    `json:"index"`     // 当前植物在去重总表中的位置（1-based）
	Total     int    `json:"total"`     // 去重后总条目数
	Name      string `json:"name"`      // 当前植物中文名
	Status    string `json:"status"`    // added | failed | skipped
	Added     int    `json:"added"`     // 本轮累计新增
	Failed    int    `json:"failed"`    // 本轮累计失败
	Skipped   int    `json:"skipped"`   // 本轮累计跳过
	Remaining int    `json:"remaining"` // 整个列表里尚未开始（含当前之后）的条目数
}

// SyncReport 整轮同步结果汇总（作为 SSE 的 done 事件 / JSON 降级返回）。
type SyncReport struct {
	Added       int      `json:"added"`
	Failed      int      `json:"failed"`
	Skipped     int      `json:"skipped"`
	Remaining   int      `json:"remaining"`   // 尚未尝试、留待后续轮次的条目数
	Total       int      `json:"total"`       // 去重后总条目数
	Throttled   bool     `json:"throttled"`   // 触发服务端 429 限流
	QuotaHit    bool     `json:"quotaHit"`    // 触及客户端每日配额上限提前停止
	Message     string   `json:"message"`     // 面向用户的中文汇总
	FailedItems []string `json:"failedItems"` // 失败条目及原因（可空）
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

// MatchGuide 按植物的学名/名称在本地资料库中寻找最匹配的养护指南。
// 匹配优先级：学名(species) 精确命中 alias/name → 名称(name) 精确命中 →
// 否则退回首个模糊(LIKE)命中；都无匹配返回 nil。
// 这样「植物详情页」无需冗余字段即可挂载资料库指南。
func (s *LibraryService) MatchGuide(name, species string) *models.PlantLibrary {
	norm := func(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
	tryMatch := func(kw string) *models.PlantLibrary {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			return nil
		}
		list, err := s.repo.Search(kw)
		if err != nil || len(list) == 0 {
			return nil
		}
		nk := norm(kw)
		for i := range list {
			if norm(list[i].Alias) == nk || norm(list[i].Name) == nk {
				return &list[i]
			}
		}
		return &list[0]
	}
	if m := tryMatch(species); m != nil {
		return m
	}
	return tryMatch(name)
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

// SyncPopularStream 批量同步映射表内植物到本地资料库（离线可用），并实时推送进度。
//
// 设计原则（在免费 API 配额内稳步推进）：
//   - onProgress 每处理完一种植物被调用一次，前端据此显示「正在同步第 X/N 个」
//   - 每轮最多向 Plantbook 发起 limit 个新条目的检索，分多次点击逐步完成
//   - 本地已同步且含结构化指标的条目直接跳过（不发任何请求）——以「学名→pid」
//     的下划线形态比对（修复原按空格学名比对永远命不中的缺陷）
//   - 请求优化：双名（种级）学名优先用 pid 直接拉详情，省去一次 search 请求；
//     仅当直接详情失败（多为 404）时回落 search→detail。单字（属级）学名直接 search
//   - 客户端每日配额护栏：剩余不足以完成下一种时主动停止本轮，避免触发 429
//   - 触发 429 限流立即中止；请求级故障（网络/5xx）亦中止本轮
//   - 单条缺失（库内无此植物 / 详情为空 / 写入失败）不中止，记入 FailedItems 供排查
func (s *LibraryService) SyncPopularStream(ctx context.Context, limit int, onProgress func(SyncProgress)) SyncReport {
	var rep SyncReport
	if !s.plantbook.Enabled() {
		rep.Message = "未配置 Plantbook 凭据（PLANTBOOK_CLIENT_ID / PLANTBOOK_CLIENT_SECRET），无法在线同步"
		return rep
	}

	// 去重，得到别名总表
	uniq := make([]PlantAlias, 0, len(plantAliases))
	seenLatin := make(map[string]bool)
	for _, a := range plantAliases {
		if seenLatin[a.Latin] {
			continue
		}
		seenLatin[a.Latin] = true
		uniq = append(uniq, a)
	}

	existing, err := s.repo.ExistingMetrics()
	if err != nil {
		existing = map[string]bool{} // 查询失败时不跳过，按全量处理
	}

	// 已确认在线库未收录的学名（持久化，避免反复请求、永不推进）
	notFound := s.loadNotFound()

	// 构建真正待同步队列：排除「本地已同步」与「已确认未收录」，二者均不发请求。
	// 这样每一轮处理的都是从未尝试过的新植物，进度数字才有意义。
	pending := make([]PlantAlias, 0, len(uniq))
	excluded := 0
	for _, seed := range uniq {
		guess := aliasToPID(seed.Latin)
		if existing[guess] || notFound[guess] {
			excluded++
			continue
		}
		pending = append(pending, seed)
	}
	total := len(pending)
	rep.Total = total

	added, failed := 0, 0
	var firstErr string
	var throttled, quotaHit bool
	var failedItems []string

	emit := func(idx, remaining int, seed PlantAlias, status string) {
		if onProgress == nil {
			return
		}
		onProgress(SyncProgress{
			Type:      "progress",
			Index:     idx,
			Total:     total,
			Name:      seed.Zh,
			Status:    status,
			Added:     added,
			Failed:    failed,
			Skipped:   excluded,
			Remaining: remaining,
		})
	}

	processed := 0
	for i, seed := range pending {
		if ctx != nil {
			select {
			case <-ctx.Done():
				rep.Message = "客户端已断开，同步中断"
				rep.Added, rep.Failed, rep.Skipped = added, failed, excluded
				rep.Remaining = total - added - failed
				rep.FailedItems = failedItems
				return rep
			default:
			}
		}
		idx := i + 1
		remaining := total - idx
		guess := aliasToPID(seed.Latin)

		// 客户端每日配额不足：停止本轮（断点可续）
		if !s.plantbook.canAffordPlant() {
			quotaHit = true
			break
		}

		// 本轮配额用尽：不再发请求，留待后续点击
		if limit > 0 && processed >= limit {
			continue
		}
		if processed > 0 {
			time.Sleep(syncInterval)
		}
		processed++

		var lib *models.PlantLibrary
		var detailErr error
		directOK := false

		// 双名（种级）优先直接拉详情，省一次 search 请求
		if strings.Contains(seed.Latin, " ") {
			lib, detailErr = s.plantbook.Detail(guess)
			if IsThrottled(detailErr) {
				throttled = true
				firstErr = detailErr.Error()
				break
			}
			if lib != nil {
				directOK = true
			}
			// 失败（多为 404）则回落 search
		}

		if !directOK {
			cands, e := s.plantbook.Search(seed.Latin)
			if IsThrottled(e) {
				throttled = true
				firstErr = e.Error()
				break
			}
			if e != nil {
				// 请求级故障：立即停止本轮，避免连续无效请求
				failed++
				failedItems = append(failedItems, fmt.Sprintf("%s(%s): %v", seed.Zh, seed.Latin, e))
				firstErr = fmt.Sprintf("%s(%s): %v", seed.Zh, seed.Latin, e)
				emit(idx, remaining, seed, "failed")
				break
			}
			if len(cands) == 0 {
				// 在线库确实未收录：永久记入黑名单，后续轮次直接跳过，
				// 不再占用「前 N 个」位置、不再空耗请求配额。
				failed++
				failedItems = append(failedItems, fmt.Sprintf("%s(%s): 在线库未收录该植物", seed.Zh, seed.Latin))
				notFound[guess] = true
				s.saveNotFound(notFound)
				emit(idx, remaining, seed, "failed")
				continue
			}
			pid := cands[0].PID
			if existing[pid] || pid == "" {
				// 详情对应 pid 已被本地收录（罕见）：按已同步处理，不重复请求
				emit(idx, remaining, seed, "skipped")
				continue
			}
			lib, detailErr = s.plantbook.Detail(pid)
			if IsThrottled(detailErr) {
				throttled = true
				firstErr = detailErr.Error()
				break
			}
			if detailErr != nil {
				failed++
				failedItems = append(failedItems, fmt.Sprintf("%s(%s): 详情请求失败 %v", seed.Zh, seed.Latin, detailErr))
				firstErr = fmt.Sprintf("%s(%s): %v", seed.Zh, seed.Latin, detailErr)
				emit(idx, remaining, seed, "failed")
				break
			}
		}

		if lib == nil {
			failed++
			failedItems = append(failedItems, fmt.Sprintf("%s(%s): 在线库未返回详情", seed.Zh, seed.Latin))
			emit(idx, remaining, seed, "failed")
			continue
		}

		// 展示名兜底：详情未带中文 common name 时使用表中中文名
		if lib.Name != "" && !isChinese(lib.Name) && isChinese(seed.Zh) {
			lib.Name = seed.Zh
		}
		lib.SyncedAt = time.Now().Unix()
		if e := s.repo.UpsertByPID(lib); e != nil {
			failed++
			failedItems = append(failedItems, fmt.Sprintf("%s(%s): 写入本地库失败 %v", seed.Zh, seed.Latin, e))
			emit(idx, remaining, seed, "failed")
			continue
		}
		added++
		emit(idx, remaining, seed, "added")
	}

	rep.Added, rep.Failed, rep.Skipped = added, failed, excluded
	rep.Remaining = total - added - failed
	rep.Throttled, rep.QuotaHit = throttled, quotaHit
	rep.FailedItems = failedItems
	rep.Message = buildSyncMessage(added, failed, excluded, rep.Remaining, throttled, quotaHit, firstErr)
	return rep
}

// aliasToPID 学名 → Plantbook pid 形态（小写、空格转下划线），用于比对与直接详情请求。
func aliasToPID(latin string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(latin), " ", "_"))
}

// loadNotFound 读取已确认「在线库未收录」的学名集合（持久化到 data/plantbook_sync_state.json）。
func (s *LibraryService) loadNotFound() map[string]bool {
	set := map[string]bool{}
	if s.syncStatePath == "" {
		return set
	}
	b, err := os.ReadFile(s.syncStatePath)
	if err != nil {
		return set
	}
	var st struct {
		NotFound []string `json:"not_found"`
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return set
	}
	for _, k := range st.NotFound {
		set[k] = true
	}
	return set
}

// saveNotFound 持久化「在线库未收录」学名集合，供后续轮次跳过（避免反复空耗配额）。
func (s *LibraryService) saveNotFound(set map[string]bool) {
	if s.syncStatePath == "" {
		return
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	st := struct {
		NotFound []string `json:"not_found"`
	}{NotFound: keys}
	if b, err := json.MarshalIndent(st, "", "  "); err == nil {
		_ = os.WriteFile(s.syncStatePath, b, 0o644)
	}
}

// buildSyncMessage 拼装面向用户的中文汇总文案。
func buildSyncMessage(added, failed, excluded, remaining int, throttled, quotaHit bool, firstErr string) string {
	msg := fmt.Sprintf("本轮新增 %d 种，失败 %d 种", added, failed)
	if excluded > 0 {
		msg += fmt.Sprintf("，已排除 %d 种（本地已同步或在线库未收录）", excluded)
	}
	switch {
	case throttled:
		msg += fmt.Sprintf("；%s。稍后再次同步可从断点继续", firstErr)
	case quotaHit:
		msg += "；今日 Plantbook 请求额度已接近上限，请次日再同步（断点可续）"
	case firstErr != "":
		msg += fmt.Sprintf("；已停止：%s", firstErr)
	case remaining > 0:
		msg += fmt.Sprintf("。还有 %d 种待同步，稍后再次点击继续", remaining)
	default:
		msg += "。全部条目已处理完毕 🌿"
	}
	return msg
}
