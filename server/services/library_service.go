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
	Type       string `json:"type"`       // "progress"
	Index      int    `json:"index"`      // 当前植物在待同步队列中的位置（1-based）
	Total      int    `json:"total"`      // 本轮待同步队列长度
	Name       string `json:"name"`       // 当前植物中文名
	Status     string `json:"status"`     // added | failed | duplicate
	Added      int    `json:"added"`      // 本轮累计新增
	Failed     int    `json:"failed"`     // 本轮累计失败
	Duplicated int    `json:"duplicated"` // 本轮累计「同物异名」（解析到的 pid 本地已有，仅耗 1 次搜索）
	Skipped    int    `json:"skipped"`    // 建队列时即排除的条目（本地已同步 / 已确认未收录）
	Remaining  int    `json:"remaining"`  // 队列中尚未开始（含当前之后）的条目数
}

// SyncReport 整轮同步结果汇总（作为 SSE 的 done 事件 / JSON 降级返回）。
type SyncReport struct {
	Added       int      `json:"added"`
	Failed      int      `json:"failed"`
	Duplicated  int      `json:"duplicated"`  // 同物异名：不同学名指向库内同一株，未重复入库
	Skipped     int      `json:"skipped"`     // 建队列时排除（本地已同步 / 已确认未收录）
	Remaining   int      `json:"remaining"`   // 尚未尝试、留待后续轮次的条目数
	Total       int      `json:"total"`       // 本轮待同步队列长度
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
//   - 本地已同步的条目直接跳过（不发任何请求）——两侧 key 统一用 aliasToPID
//     归一化（Plantbook 返回的 pid 是带空格的原始学名，修复此前永不命中的回归）
//   - 同物异名消除：持久化「别名 pid → Plantbook 规范 pid」解析表
//     (data/plantbook_sync_state.json 的 resolved 字段)。已解析且目标已在库的条目
//     建队列时即排除（0 请求）；轮内同一规范 pid 也只取一次详情。
//     这是请求消耗的主要来源——实测约 60% 的条目是同物异名重复。
//   - 请求路径固定为 search → detail（新增 2 次 / 同物异名 1 次 / 未收录 1 次）。
//     曾用「学名直拼 pid 拉详情」省一次 search，但实测命中率仅约 33%，低于 50%
//     盈亏平衡点（命中省 1 次、未命中要多付 404 那次），反而更耗配额，已移除。
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

	rawExisting, err := s.repo.ExistingMetrics()
	if err != nil {
		rawExisting = map[string]bool{} // 查询失败时不跳过，按全量处理
	}
	// 归一化两侧 key：Plantbook 返回的 pid 可能是「空格学名」(Monstera deliciosa)，
	// 而比对侧 guess 是下划线小写(monstera_deliciosa)。统一归一化后，已同步条目
	// 才能稳定命中（修复此前按学名比对永远不命中的回归，否则每轮都重请求已同步项）。
	existing := make(map[string]bool, len(rawExisting))
	for k, v := range rawExisting {
		existing[aliasToPID(k)] = v
	}

	// 已确认在线库未收录的学名（持久化，避免反复请求、永不推进）
	notFound, resolved := s.loadSyncState()

	// 构建真正待同步队列。以下三类在发请求前即排除（0 请求），保证每一轮
	// 处理的都是「从未尝试过、且库内确实没有」的新植物，进度数字才有意义：
	//   1. 本地已同步
	//   2. 已确认在线库未收录
	//   3. 同物异名：该学名此前已解析到某个规范 pid，且该 pid 已在库
	pending := make([]PlantAlias, 0, len(uniq))
	excluded := 0
	for _, seed := range uniq {
		guess := aliasToPID(seed.Latin)
		if existing[guess] || notFound[guess] {
			excluded++
			continue
		}
		if rp, ok := resolved[guess]; ok && rp != "" && existing[aliasToPID(rp)] {
			excluded++
			continue
		}
		pending = append(pending, seed)
	}
	total := len(pending)
	rep.Total = total

	added, failed, duplicated := 0, 0, 0
	var firstErr string
	var throttled, quotaHit bool
	var failedItems []string

	emit := func(idx, remaining int, seed PlantAlias, status string) {
		if onProgress == nil {
			return
		}
		onProgress(SyncProgress{
			Type:       "progress",
			Index:      idx,
			Total:      total,
			Name:       seed.Zh,
			Status:     status,
			Added:      added,
			Failed:     failed,
			Duplicated: duplicated,
			Skipped:    excluded,
			Remaining:  remaining,
		})
	}

	processed := 0
	// 本轮已取过详情的规范 pid：同一轮内若另有别名指向同一株，不再重复取详情
	roundPid := map[string]string{}
	for i, seed := range pending {
		if ctx != nil {
			select {
			case <-ctx.Done():
				rep.Message = "客户端已断开，同步中断"
				rep.Added, rep.Failed, rep.Skipped = added, failed, excluded
				rep.Duplicated = duplicated
				rep.Remaining = total - added - failed - duplicated
				rep.FailedItems = failedItems
				return rep
			default:
			}
		}
		// 本轮配额用尽：不再发请求，留待后续点击
		if limit > 0 && processed >= limit {
			break
		}
		idx := i + 1
		remaining := total - idx
		guess := aliasToPID(seed.Latin)

		// 客户端每日配额不足：停止本轮（断点可续）
		if !s.plantbook.canAffordPlant() {
			quotaHit = true
			break
		}
		if processed > 0 {
			time.Sleep(syncInterval)
		}
		processed++

		// 固定两步：search 拿规范 pid（1 次）→ 库内没有才 detail（1 次）。
		// 新增 2 次 / 同物异名 1 次 / 未收录 1 次，不再有「直接详情 404」的额外开销。
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
		pid := pickCandidate(cands, guess)
		if pid == "" {
			// 在线库确实未收录：永久记入黑名单，后续轮次直接跳过，
			// 不再占用「前 N 个」位置、不再空耗请求配额。
			failed++
			failedItems = append(failedItems, fmt.Sprintf("%s(%s): 在线库未收录该植物", seed.Zh, seed.Latin))
			if !notFound[guess] {
				notFound[guess] = true
				s.saveSyncState(notFound, resolved)
			}
			emit(idx, remaining, seed, "failed")
			continue
		}

		// 记录解析结果（即使本条因重复而跳过同样记录）：下轮起该学名在
		// 建队列阶段即可 0 请求排除——这是消除同物异名重复请求的关键。
		if resolved[guess] != pid {
			resolved[guess] = pid
			s.saveSyncState(notFound, resolved)
		}

		if existing[aliasToPID(pid)] || roundPid[pid] != "" {
			// 同物异名：不同学名指向库内（或本轮已取的）同一株，只消耗 1 次搜索
			duplicated++
			emit(idx, remaining, seed, "duplicate")
			continue
		}
		roundPid[pid] = seed.Latin

		lib, detailErr := s.plantbook.Detail(pid)
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
	rep.Duplicated = duplicated
	rep.Remaining = total - added - failed - duplicated
	rep.Throttled, rep.QuotaHit = throttled, quotaHit
	rep.FailedItems = failedItems
	rep.Message = buildSyncMessage(added, failed, excluded, duplicated, rep.Remaining, throttled, quotaHit, firstErr)
	return rep
}

// aliasToPID 学名 → Plantbook pid 形态（小写、空格转下划线），用于比对与直接详情请求。
func aliasToPID(latin string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(latin), " ", "_"))
}

// knownNotFound 已确认 Plantbook 在线库未收录的学名（pid 形态）。
// 硬编码以保证在任意部署环境首次同步即跳过，不依赖运行时文件（文件由服务器自身在
// 其运行目录写入，本地开发沙箱无法预置到用户运行时）。运行时新发现的未收录项会追加
// 进 data/plantbook_sync_state.json 持久化。
var knownNotFound = []string{
	"dracaena_reflexa",      // 百合竹
	"spathiphyllum_wallisii", // 白掌
}

// pickCandidate 从搜索结果中挑选最可信的候选 pid。
// 优先级：学名/pid 精确命中 > 同属命中 > 首个候选（沿用 Plantbook 自身的相关度排序）。
// 只做重排、不做过滤，因此不会比「直接取首个候选」更容易丢条目。
func pickCandidate(cands []OnlineCandidate, guess string) string {
	if len(cands) == 0 {
		return ""
	}
	genus := strings.SplitN(guess, "_", 2)[0]
	genusOf := func(v string) string { return strings.SplitN(aliasToPID(v), "_", 2)[0] }
	for _, c := range cands {
		if c.PID == "" {
			continue
		}
		if aliasToPID(c.Alias) == guess || aliasToPID(c.PID) == guess {
			return c.PID
		}
	}
	if genus != "" {
		for _, c := range cands {
			if c.PID == "" {
				continue
			}
			if genusOf(c.Alias) == genus || genusOf(c.PID) == genus {
				return c.PID
			}
		}
	}
	return cands[0].PID
}

// syncState 同步状态（持久化到 data/plantbook_sync_state.json）。
type syncState struct {
	NotFound []string          `json:"not_found"` // 已确认在线库未收录的学名（pid 形态）
	Resolved map[string]string `json:"resolved"`  // 别名 pid → Plantbook 规范 pid（消除同物异名重复请求）
}

// loadSyncState 读取持久化状态：已确认「在线库未收录」的学名集合 + 已解析的
// 「学名 → 规范 pid」映射；并合并硬编码的 knownNotFound，确保跨环境一致生效。
func (s *LibraryService) loadSyncState() (map[string]bool, map[string]string) {
	notFound := map[string]bool{}
	for _, k := range knownNotFound {
		notFound[k] = true
	}
	resolved := map[string]string{}
	if s.syncStatePath == "" {
		return notFound, resolved
	}
	b, err := os.ReadFile(s.syncStatePath)
	if err != nil {
		return notFound, resolved
	}
	var st syncState
	if err := json.Unmarshal(b, &st); err != nil {
		return notFound, resolved
	}
	for _, k := range st.NotFound {
		notFound[k] = true
	}
	for k, v := range st.Resolved {
		if k != "" && v != "" {
			resolved[k] = v
		}
	}
	return notFound, resolved
}

// saveSyncState 持久化状态，供后续轮次跳过（避免反复空耗配额）。
func (s *LibraryService) saveSyncState(notFound map[string]bool, resolved map[string]string) {
	if s.syncStatePath == "" {
		return
	}
	keys := make([]string, 0, len(notFound))
	for k := range notFound {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	st := syncState{NotFound: keys, Resolved: map[string]string{}}
	for k, v := range resolved {
		if k != "" && v != "" {
			st.Resolved[k] = v
		}
	}
	if b, err := json.MarshalIndent(st, "", "  "); err == nil {
		_ = os.WriteFile(s.syncStatePath, b, 0o644)
	}
}

// buildSyncMessage 拼装面向用户的中文汇总文案。
func buildSyncMessage(added, failed, excluded, duplicated, remaining int, throttled, quotaHit bool, firstErr string) string {
	msg := fmt.Sprintf("本轮新增 %d 种，失败 %d 种", added, failed)
	if duplicated > 0 {
		msg += fmt.Sprintf("，同物异名 %d 种（指向库内已有植物，各耗 1 次检索）", duplicated)
	}
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

// RefreshLocalGuides 本地批量翻译：将所有含英文的养护指南翻译为中文（不调用外部API）
func (s *LibraryService) RefreshLocalGuides() (int, error) {
	libs, err := s.repo.ListAll()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, lib := range libs {
		guide := lib.Guide
		if guide == "" {
			continue
		}
		translated := translateCareEnToZh(guide)
		if translated != guide {
			if err := s.repo.UpdateGuide(lib.ID, translated); err != nil {
				continue
			}
			count++
		}
	}
	return count, nil
}

// ClearLibrary 清空资料库所有条目（从Plantbook同步的数据），保留同步状态
func (s *LibraryService) ClearLibrary() error {
	return s.repo.DeleteAll()
}

// ResyncAndTranslateProgress 重新拉取并翻译的进度事件
type ResyncAndTranslateProgress struct {
	Type   string `json:"type"`   // "progress"
	Index  int    `json:"index"`  // 当前位置（1-based）
	Total  int    `json:"total"`  // 总数
	Name   string `json:"name"`   // 植物名称
	Status string `json:"status"` // success | failed | skipped
	Count  int    `json:"count"`  // 本轮成功数
}

// ResyncAndTranslateReport 重新拉取并翻译的结果报告
type ResyncAndTranslateReport struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// ResyncAndTranslate 重新拉取所有植物的英文Guide并翻译为中文
// limit: 本轮最多处理多少条（0=全部）
func (s *LibraryService) ResyncAndTranslate(ctx context.Context, limit int, onProgress func(ResyncAndTranslateProgress)) ResyncAndTranslateReport {
	var rep ResyncAndTranslateReport
	if !s.plantbook.Enabled() {
		return rep
	}

	libs, err := s.repo.ListAll()
	if err != nil {
		return rep
	}

	// 只处理有pid的条目（从Plantbook同步的）
	var toResync []models.PlantLibrary
	for _, lib := range libs {
		if lib.PID != "" {
			toResync = append(toResync, lib)
		}
	}
	rep.Total = len(toResync)

	processed := 0
	for i, lib := range toResync {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return rep
			default:
			}
		}
		if limit > 0 && processed >= limit {
			break
		}

		// 从Plantbook重新拉取详情
		newLib, err := s.plantbook.Detail(lib.PID)
		if err != nil {
			if onProgress != nil {
				onProgress(ResyncAndTranslateProgress{
					Type:   "progress",
					Index:  i + 1,
					Total:  rep.Total,
					Name:   lib.Name,
					Status: "failed",
					Count:  rep.Success,
				})
			}
			rep.Failed++
			continue
		}
		if newLib == nil {
			if onProgress != nil {
				onProgress(ResyncAndTranslateProgress{
					Type:   "progress",
					Index:  i + 1,
					Total:  rep.Total,
					Name:   lib.Name,
					Status: "skipped",
					Count:  rep.Success,
				})
			}
			continue
		}

		// 翻译英文Guide为中文
		translatedGuide := translateCareEnToZh(newLib.Guide)
		newLib.Guide = translatedGuide

		// 更新数据库
		if err := s.repo.UpsertByPID(newLib); err != nil {
			if onProgress != nil {
				onProgress(ResyncAndTranslateProgress{
					Type:   "progress",
					Index:  i + 1,
					Total:  rep.Total,
					Name:   lib.Name,
					Status: "failed",
					Count:  rep.Success,
				})
			}
			rep.Failed++
			continue
		}

		rep.Success++
		processed++
		if onProgress != nil {
			onProgress(ResyncAndTranslateProgress{
				Type:   "progress",
				Index:  i + 1,
				Total:  rep.Total,
				Name:   lib.Name,
				Status: "success",
				Count:  rep.Success,
			})
		}

		// 控制请求频率
		time.Sleep(syncInterval)
	}

	return rep
}
