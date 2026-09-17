package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"qingye/server/models"
)

var plantbookBaseURL = "https://open.plantbook.io/api/v1"

// PlantbookClient 封装 Plantbook 在线植物资料 API。
// 设计要点：
//   - 认证采用官方 OAuth2 Client Credentials：client_id/client_secret 换取临时
//     access_token（POST /token/），自动缓存并在过期前刷新；也兼容直接传入
//     预取的 access_token（对应官方 PLANTBOOK_ACCESS_TOKEN 用法）
//   - 以学名 pid（如 monstera_deliciosa）作为唯一键，避免中文名歧义
//   - 详情请求带 lang=zh，优先取 Plantbook 自带的中文 common_name 与字段，免去第三方翻译
//   - 无凭据时所有方法优雅返回空结果，由上层降级为"仅本地资料库"
type PlantbookClient struct {
	clientID string
	secret   string
	staticTk string // 直接使用的 access_token（可选，调试/已有 token 时用）

	mu        sync.Mutex
	token     string
	expiresAt time.Time

	quota *dailyQuota // 每日请求配额（客户端侧护栏，避免触发服务端 429）

	client *http.Client
}

// NewPlantbookClient 创建客户端；clientID+secret 或 staticToken 任一有效即启用。
func NewPlantbookClient(clientID, secret, staticToken string) *PlantbookClient {
	return &PlantbookClient{
		clientID: clientID,
		secret:   secret,
		staticTk: staticToken,
		client:   &http.Client{Timeout: 12 * time.Second},
	}
}

// dailyQuota 客户端侧每日请求配额护栏。
//
// Plantbook 免费账户按自然日限额（200 次/天，自 2025-11-01 起生效）。
// 一旦触及，服务端返回 429 并需等待到次日重置，期间整库不可同步。
// 为此在客户端统计当日已用请求数并持久化到本地文件，跨进程/重启不丢失，
// 在剩余额度不足以完成下一种植物（最坏 3 次：直接详情失败→搜索→详情）时
// 主动停止本轮，避免触发服务端限流。
type dailyQuota struct {
	mu    sync.Mutex
	date  string // 当前统计所属日期 YYYY-MM-DD
	used  int    // 当日已用请求数
	limit int    // 每日上限
	file  string // 持久化文件路径（空表示不持久化）
}

func newDailyQuota(limit int, file string) *dailyQuota {
	q := &dailyQuota{limit: limit, file: file}
	if q.file != "" {
		if data, err := os.ReadFile(q.file); err == nil {
			var s struct {
				Date string `json:"date"`
				Used int    `json:"used"`
			}
			if json.Unmarshal(data, &s) == nil {
				q.date, q.used = s.Date, s.Used
			}
		}
	}
	q.resetIfNewDay()
	return q
}

// resetIfNewDay 跨日则清零（调用方需持锁）
func (q *dailyQuota) resetIfNewDay() {
	today := time.Now().Format("2006-01-02")
	if q.date != today {
		q.date = today
		q.used = 0
	}
}

func (q *dailyQuota) save() {
	if q.file == "" {
		return
	}
	s := struct {
		Date string `json:"date"`
		Used int    `json:"used"`
	}{q.date, q.used}
	if b, err := json.Marshal(s); err == nil {
		_ = os.WriteFile(q.file, b, 0o644)
	}
}

// tick 记一次请求消耗（线程安全，每日自动重置并落盘）
func (q *dailyQuota) tick() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.resetIfNewDay()
	q.used++
	q.save()
}

// remaining 当日剩余可用请求数（线程安全）
func (q *dailyQuota) remaining() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.resetIfNewDay()
	r := q.limit - q.used
	if r < 0 {
		return 0
	}
	return r
}

// canAffordPlant 是否还足以完成下一种植物。
// 最坏路径：搜索(1) → 详情(1) = 2 次（同步已统一为 search→detail 两步）。
// 按 2 预留即可，按 3 预留会把每日可同步株数从约 100 压到约 66，属过度保守。
func (q *dailyQuota) canAffordPlant() bool {
	return q.remaining() >= 2
}

// quotaTick 每次真正发起 API 请求时调用（无 quota 时为空操作）
func (p *PlantbookClient) quotaTick() {
	if p.quota != nil {
		p.quota.tick()
	}
}

// canAffordPlant 暴露配额判断（无 quota 时恒为 true）
func (p *PlantbookClient) canAffordPlant() bool {
	if p.quota == nil {
		return true
	}
	return p.quota.canAffordPlant()
}

func (p *PlantbookClient) Enabled() bool {
	return p.staticTk != "" || (p.clientID != "" && p.secret != "")
}

// getToken 获取可用的 access_token：优先静态 token，否则走 OAuth2 并缓存到过期前。
func (p *PlantbookClient) getToken() (string, error) {
	if p.staticTk != "" {
		return p.staticTk, nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && time.Now().Before(p.expiresAt) {
		return p.token, nil
	}
	resp, err := p.client.PostForm(plantbookBaseURL+"/token/", url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.clientID},
		"client_secret": {p.secret},
	})
	if err != nil {
		return "", fmt.Errorf("请求 Plantbook 凭据失败: %w", err)
	}
	defer resp.Body.Close()
	p.quotaTick() // 凭据请求同样计入当日配额
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return "", fmt.Errorf("Plantbook 认证失败(%d)，请检查 client_id/client_secret：%s", resp.StatusCode, msg)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(data, &tok); err != nil || tok.AccessToken == "" {
		return "", fmt.Errorf("解析 Plantbook 凭据响应失败")
	}
	p.token = tok.AccessToken
	// 提前 60s 过期，避免临界点请求失败
	p.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second)
	return p.token, nil
}

// invalidateToken 认证失效时清空缓存，下次重新获取
func (p *PlantbookClient) invalidateToken() {
	p.mu.Lock()
	p.token = ""
	p.expiresAt = time.Time{}
	p.mu.Unlock()
}

// pbSearchItem search 接口返回的单条候选（精简字段）
type pbSearchItem struct {
	PID         string `json:"pid"`
	Alias       string `json:"alias"`
	CommonNames any    `json:"common_names"`
	ImageURL    string `json:"image_url"`
	Link        string `json:"link"`
	Vegetable   bool   `json:"vegetable"`
	Edible      bool   `json:"edible"`
}

// pbSearchResp search 接口整体响应
type pbSearchResp struct {
	Count   int            `json:"count"`
	Next    any            `json:"next"`
	Prev    any            `json:"prev"`
	Results []pbSearchItem `json:"results"`
}

// pbCommonName Plantbook common_names 数组元素
type pbCommonName struct {
	Name         string `json:"name"`
	LanguageCode string `json:"language_code"`
}

// pbDetail Plantbook 植物详情。
// 注意：环境阈值为基础字段；养护描述（光照/浇水/土壤/施肥/修剪）属可选
// care 类别，请求必须带 ?include=care 才会返回。
type pbDetail struct {
	PID           string         `json:"pid"`
	DisplayPID    string         `json:"display_pid"`
	Alias         string         `json:"alias"`
	Category      string         `json:"category"`
	Origin        string         `json:"origin"`
	Link          string         `json:"link"`
	ImageURL      string         `json:"image_url"`
	CommonNames   []pbCommonName `json:"common_names"`
	MaxTemp       float64        `json:"max_temp"`
	MinTemp       float64        `json:"min_temp"`
	MaxLightMMOL  float64        `json:"max_light_mmol"`
	MinLightMMOL  float64        `json:"min_light_mmol"`
	MaxLightLux   float64        `json:"max_light_lux"`
	MinLightLux   float64        `json:"min_light_lux"`
	MaxEnvHumid   float64        `json:"max_env_humid"`
	MinEnvHumid   float64        `json:"min_env_humid"`
	MaxSoilMoist  float64        `json:"max_soil_moist"`
	MinSoilMoist  float64        `json:"min_soil_moist"`
	MaxSoilEC     float64        `json:"max_soil_ec"`
	MinSoilEC     float64        `json:"min_soil_ec"`
	MaxLight      string         `json:"max_light_human"`
	MinLight      string         `json:"min_light_human"`

	// 养护信息：官方文档未给出明确嵌套层级，这里同时兼容
	// 「平铺在顶层」与「嵌套在 care 对象内」两种形态
	Light         string  `json:"light"`
	Sunlight      string  `json:"sunlight"`
	Watering      string  `json:"watering"`
	SoilText      string  `json:"soil"`
	Fertilization string  `json:"fertilization"`
	Pruning       string  `json:"pruning"`
	Care          *pbCare `json:"care"`
}

// pbCare care 类别的养护信息
type pbCare struct {
	Light         string `json:"light"`
	Sunlight      string `json:"sunlight"`
	Watering      string `json:"watering"`
	Soil          string `json:"soil"`
	Fertilization string `json:"fertilization"`
	Pruning       string `json:"pruning"`
}

// careInfo 归一化养护信息：平铺字段与 care 嵌套对象取并集（非空优先）
func (d *pbDetail) careInfo() pbCare {
	c := pbCare{
		Light:         firstNonEmpty(d.Light, d.Sunlight),
		Watering:      d.Watering,
		Soil:          d.SoilText,
		Fertilization: d.Fertilization,
		Pruning:       d.Pruning,
	}
	if d.Care != nil {
		c.Light = firstNonEmpty(d.Care.Light, d.Care.Sunlight, c.Light)
		c.Watering = firstNonEmpty(d.Care.Watering, c.Watering)
		c.Soil = firstNonEmpty(d.Care.Soil, c.Soil)
		c.Fertilization = firstNonEmpty(d.Care.Fertilization, c.Fertilization)
		c.Pruning = firstNonEmpty(d.Care.Pruning, c.Pruning)
	}
	return c
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// PlantMetrics 结构化环境阈值（存入 PlantLibrary.Metrics 的 JSON 形态），
// 单位：光照 mmol(µmol/m²·s) 与 lux、温度 ℃、湿度与土壤水分 %、EC µS/cm。
// 可供后续智能养护策略直接使用，无需再解析文本指南。
type PlantMetrics struct {
	MinLightMMOL float64 `json:"minLightMmol,omitempty"`
	MaxLightMMOL float64 `json:"maxLightMmol,omitempty"`
	MinLightLux  float64 `json:"minLightLux,omitempty"`
	MaxLightLux  float64 `json:"maxLightLux,omitempty"`
	MinTemp      float64 `json:"minTemp,omitempty"`
	MaxTemp      float64 `json:"maxTemp,omitempty"`
	MinEnvHumid  float64 `json:"minEnvHumid,omitempty"`
	MaxEnvHumid  float64 `json:"maxEnvHumid,omitempty"`
	MinSoilMoist float64 `json:"minSoilMoist,omitempty"`
	MaxSoilMoist float64 `json:"maxSoilMoist,omitempty"`
	MinSoilEC    float64 `json:"minSoilEc,omitempty"`
	MaxSoilEC    float64 `json:"maxSoilEc,omitempty"`
}

// OnlineCandidate 在线搜索返回的候选条目（供前端展示让用户选择）
type OnlineCandidate struct {
	PID         string   `json:"pid"`
	Name        string   `json:"name"`        // 优先中文 common_name，否则首个 common_name
	Alias       string   `json:"alias"`       // 学名（alias）
	Image       string   `json:"image"`       // 缩略图
	CommonNames []string `json:"commonNames"` // 所有常见名（含中文）
}

// Search 按关键词搜索植物。服务端要求关键词至少 3 个字符，且库内中文
// common name 覆盖极低；因此中文关键词先经本地映射表（plant_alias.go）
// 转换为拉丁学名再查询，未命中映射且长度不足时返回空列表。
func (p *PlantbookClient) Search(keyword string) ([]OnlineCandidate, error) {
	kw := strings.TrimSpace(keyword)
	if !p.Enabled() {
		return nil, nil
	}
	if latin, hit := lookupLatin(kw); hit {
		kw = latin
	} else if utf8.RuneCountInString(kw) < 3 {
		return nil, nil
	}
	u := fmt.Sprintf("%s/plant/search?alias=%s&limit=20",
		plantbookBaseURL, url.QueryEscape(kw))
	body, err := p.doGet(u)
	if err != nil {
		return nil, err
	}
	var resp pbSearchResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析搜索响应失败: %w", err)
	}
	cands := make([]OnlineCandidate, 0, len(resp.Results))
	for _, it := range resp.Results {
		cands = append(cands, toCandidate(it))
	}
	return cands, nil
}

// Detail 按 pid 拉取详情，并映射为中文 PlantLibrary 条目。
func (p *PlantbookClient) Detail(pid string) (*models.PlantLibrary, error) {
	if !p.Enabled() || pid == "" {
		return nil, nil
	}
	u := fmt.Sprintf("%s/plant/detail/%s/?lang=zh&include=care", plantbookBaseURL, url.PathEscape(pid))
	body, err := p.doGet(u)
	if err != nil {
		return nil, err
	}
	var d pbDetail
	if err := json.Unmarshal(body, &d); err != nil {
		return nil, fmt.Errorf("解析详情响应失败: %w", err)
	}
	return detailToLibrary(d), nil
}

// ThrottledError Plantbook 限流（HTTP 429）。免费账户按天配额，
// 触发后需等待服务端提示的时长（通常到次日重置）。
type ThrottledError struct {
	RetryAfter time.Duration // 服务端建议的等待时长；0 表示未知
}

func (e *ThrottledError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("触发 Plantbook 限流，约 %s 后恢复", formatDuration(e.RetryAfter))
	}
	return "触发 Plantbook 限流"
}

// IsThrottled 判断错误是否为限流类错误
func IsThrottled(e error) bool {
	_, ok := e.(*ThrottledError)
	return ok
}

// formatDuration 把秒数转成中文可读时长
func formatDuration(d time.Duration) string {
	mins := int(d.Minutes())
	if mins >= 60 {
		h := mins / 60
		m := mins % 60
		if m > 0 {
			return fmt.Sprintf("%d 小时 %d 分钟", h, m)
		}
		return fmt.Sprintf("%d 小时", h)
	}
	if mins > 0 {
		return fmt.Sprintf("%d 分钟", mins)
	}
	return fmt.Sprintf("%d 秒", int(d.Seconds()))
}

// parseThrottle 解析 429 响应：优先 Retry-After 头，否则从响应体
// 「Expected available in N seconds」提取等待秒数。
var throttleRe = regexp.MustCompile(`available in (\d+) seconds`)

func parseThrottle(retryAfterHeader, body string) *ThrottledError {
	var secs int
	if n, err := strconv.Atoi(strings.TrimSpace(retryAfterHeader)); err == nil && n > 0 {
		secs = n
	} else if m := throttleRe.FindStringSubmatch(body); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			secs = n
		}
	}
	return &ThrottledError{RetryAfter: time.Duration(secs) * time.Second}
}

// doGet 带 Bearer access_token 的 GET 请求；401 时刷新凭据重试一次
func (p *PlantbookClient) doGet(rawURL string) ([]byte, error) {
	body, err := p.tryGet(rawURL)
	if err == nil {
		return body, nil
	}
	// 仅认证类错误重试一次（access_token 可能已被吊销或提前失效）
	if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
		if p.staticTk == "" {
			p.invalidateToken()
			return p.tryGet(rawURL)
		}
	}
	return nil, err
}

func (p *PlantbookClient) tryGet(rawURL string) ([]byte, error) {
	tk, err := p.getToken()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tk)
	req.Header.Set("Accept", "application/json")
	p.quotaTick() // 每次 API GET 计入当日配额
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Plantbook 失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, parseThrottle(resp.Header.Get("Retry-After"), string(data))
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("Plantbook 返回 %d: %s", resp.StatusCode, msg)
	}
	return data, nil
}

// toCandidate 将 search 项转为前端候选（提取中文 common_name）
func toCandidate(it pbSearchItem) OnlineCandidate {
	names := extractCommonNames(it.CommonNames)
	zh := firstZhName(names)
	if zh == "" {
		zh = it.Alias
	}
	return OnlineCandidate{
		PID:         it.PID,
		Name:        zh,
		Alias:       it.Alias,
		Image:       it.ImageURL,
		CommonNames: names,
	}
}

// extractCommonNames 兼容 common_names 为数组或字符串两种形态
func extractCommonNames(raw any) []string {
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if m, ok := e.(map[string]any); ok {
				if n, ok := m["name"].(string); ok {
					out = append(out, n)
				}
			} else if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v != "" {
			return []string{v}
		}
	}
	return nil
}

func firstZhName(names []string) string {
	for _, n := range names {
		if isChinese(n) {
			return n
		}
	}
	return ""
}

func isChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// detailToLibrary 将 Plantbook 详情映射为中文 PlantLibrary（三段式 Guide + 结构化指标）
func detailToLibrary(d pbDetail) *models.PlantLibrary {
	zh := ""
	for _, c := range d.CommonNames {
		if isChinese(c.Name) {
			zh = c.Name
			break
		}
	}
	name := zh
	if name == "" {
		name = d.Alias
	}
	guide := buildGuide(d)
	lib := &models.PlantLibrary{
		PID:        d.PID,
		DisplayPID: d.DisplayPID,
		Name:       name,
		Alias:      d.Alias,
		Category:   d.Category,
		Origin:     d.Origin,
		Guide:      guide,
		Image:      d.ImageURL,
		Link:       d.Link,
	}
	// 全部常见名序列化存储（JSON 字符串数组）
	if len(d.CommonNames) > 0 {
		names := make([]string, 0, len(d.CommonNames))
		for _, c := range d.CommonNames {
			if c.Name != "" {
				names = append(names, c.Name)
			}
		}
		if b, err := json.Marshal(names); err == nil {
			lib.CommonNames = string(b)
		}
	}
	m := PlantMetrics{
		MinLightMMOL: d.MinLightMMOL,
		MaxLightMMOL: d.MaxLightMMOL,
		MinLightLux:  d.MinLightLux,
		MaxLightLux:  d.MaxLightLux,
		MinTemp:      d.MinTemp,
		MaxTemp:      d.MaxTemp,
		MinEnvHumid:  d.MinEnvHumid,
		MaxEnvHumid:  d.MaxEnvHumid,
		MinSoilMoist: d.MinSoilMoist,
		MaxSoilMoist: d.MaxSoilMoist,
		MinSoilEC:    d.MinSoilEC,
		MaxSoilEC:    d.MaxSoilEC,
	}
	if m != (PlantMetrics{}) {
		if b, err := json.Marshal(m); err == nil {
			lib.Metrics = string(b)
		}
	}
	return lib
}

// translateCareEnToZh 将英文养护描述翻译为中文（完整短语替换，避免中英混合）
func translateCareEnToZh(s string) string {
	if s == "" {
		return s
	}

	// ===== 浇水 =====
	s = replaceAll(s, "Prefers wet conditions; water when soil dries out", "喜湿润环境，土壤干透后浇水")
	s = replaceAll(s, "Prefers wet conditions", "喜湿润环境")
	s = replaceAll(s, "Prefers moist conditions", "喜湿润环境")
	s = replaceAll(s, "Keep soil consistently moist", "保持土壤持续湿润")
	s = replaceAll(s, "Keep soil moist", "保持土壤湿润")
	s = replaceAll(s, "keep soil evenly moist", "保持土壤均匀湿润")
	s = replaceAll(s, "keep soil slightly moist", "保持土壤微润")
	s = replaceAll(s, "Drought-resistant; water when soil is dry", "耐旱，土壤干透后浇水")
	s = replaceAll(s, "Drought-resistant", "耐旱")
	s = replaceAll(s, "water when soil dries out", "土壤干透后浇水")
	s = replaceAll(s, "water when soil is dry", "土壤干透后浇水")
	s = replaceAll(s, "water when the top inch of soil is dry", "表土干燥后浇水")
	s = replaceAll(s, "water when the topsoil feels dry", "表土干燥后浇水")
	s = replaceAll(s, "Allow soil to dry out between waterings", "浇水间让土壤干透")
	s = replaceAll(s, "Allow soil to dry between waterings", "浇水间让土壤干透")
	s = replaceAll(s, "avoid saturation", "避免积水")
	s = replaceAll(s, "avoid overwatering", "避免浇水过多")
	s = replaceAll(s, "reduce watering in winter", "冬季减少浇水")
	s = replaceAll(s, "reduce watering in summer", "夏季减少浇水")
	s = replaceAll(s, "mist leaves in summer", "夏季向叶片喷水")
	s = replaceAll(s, "water thoroughly", "浇透水")
	s = replaceAll(s, "water deeply", "浇透水")
	s = replaceAll(s, "soak thoroughly", "浇透水")
	s = replaceAll(s, "prefers regular watering", "需规律浇水")
	s = replaceAll(s, "water regularly", "规律浇水")
	s = replaceAll(s, "water sparingly", "少量浇水")
	s = replaceAll(s, "water moderately", "适量浇水")
	s = replaceAll(s, "Water deeply when soil is dry", "土壤干了就深浇")
	s = replaceAll(s, "Spray water once a day", "每天喷水一次")
	s = replaceAll(s, "Provide more water during flowering and fruiting", "开花结果期增加浇水")
	s = replaceAll(s, "Gradually reduce watering", "逐渐减少浇水")
	s = replaceAll(s, "shorten summer dormancy", "缩短夏季休眠期")
	s = replaceAll(s, "lower water and fertilizer", "减少浇水和施肥")
	s = replaceAll(s, "mist for cooling", "喷雾降温")
	s = replaceAll(s, "avoid saturating", "避免积水")
	s = replaceAll(s, "Slight drought tolerance", "稍耐旱")
	s = replaceAll(s, "slight drought tolerance", "稍耐旱")
	s = replaceAll(s, "2-3 times weekly", "每周 2-3 次")
	s = replaceAll(s, "Every 2-3 days", "每 2-3 天")
	s = replaceAll(s, "every 2-3 days", "每 2-3 天")
	s = replaceAll(s, "Every 2-4 days", "每 2-4 天")
	s = replaceAll(s, "every 2-4 days", "每 2-4 天")
	s = replaceAll(s, "every 8-10 days", "每 8-10 天")
	s = replaceAll(s, "Every 10-15 days", "每 10-15 天")
	s = replaceAll(s, "every 10-15 days", "每 10-15 天")
	s = replaceAll(s, "Weekly", "每周一次")
	s = replaceAll(s, "weekly", "每周一次")
	s = replaceAll(s, "daily", "每天")
	s = replaceAll(s, "once a day", "每天一次")
	s = replaceAll(s, "as flowers open", "随着花朵开放")
	s = replaceAll(s, "stop when leaves turn yellowish", "叶片变黄时停止")
	s = replaceAll(s, "for peak growth", "以促进旺盛生长")
	s = replaceAll(s, "for better care", "养护更佳")

	// ===== 光照 =====
	s = replaceAll(s, "Like partial shade, place in areas with scattered light", "喜半阴环境，放置在有散射光的地方")
	s = replaceAll(s, "Likes moderate sunshine, shade from strong summer sun", "喜适度阳光，夏季避免强光直射")
	s = replaceAll(s, "Likes full sun", "喜全日照")
	s = replaceAll(s, "Likes sunshine", "喜阳光")
	s = replaceAll(s, "Like partial shade", "喜半阴环境")
	s = replaceAll(s, "likes moderate sunshine", "喜适度阳光")
	s = replaceAll(s, "likes full sun", "喜全日照")
	s = replaceAll(s, "likes sunshine", "喜阳光")
	s = replaceAll(s, "place in areas with scattered light", "放置在有散射光的地方")
	s = replaceAll(s, "place in an area with scattered light", "放置在有散射光的地方")
	s = replaceAll(s, "shade from strong summer sun", "夏季避免强光直射")
	s = replaceAll(s, "shade from intense sun", "避免强光直射")
	s = replaceAll(s, "avoid strong direct light", "避免强光直射")
	s = replaceAll(s, "avoid direct sunlight", "避免阳光直射")
	s = replaceAll(s, "bright indirect light", "明亮散射光")
	s = replaceAll(s, "bright, indirect light", "明亮散射光")
	s = replaceAll(s, "bright, indirect sunlight", "明亮散射光")
	s = replaceAll(s, "bright indirect sunlight", "明亮散射光")
	s = replaceAll(s, "direct sunlight", "阳光直射")
	s = replaceAll(s, "direct sun", "阳光直射")
	s = replaceAll(s, "Tolerates shade", "耐阴")
	s = replaceAll(s, "tolerates shade", "耐阴")
	s = replaceAll(s, "Tolerates low light", "耐阴")
	s = replaceAll(s, "tolerates low light", "耐阴")
	s = replaceAll(s, "full sun", "全日照")
	s = replaceAll(s, "partial shade", "半阴")
	s = replaceAll(s, "partial sun", "半日照")
	s = replaceAll(s, "indirect light", "散射光")
	s = replaceAll(s, "scattered light", "散射光")
	s = replaceAll(s, "Grows in sunny spots", "在阳光充足处生长")
	s = replaceAll(s, "grows in sunny spots", "在阳光充足处生长")
	s = replaceAll(s, "Thrives in sunny spots", "在阳光充足处生长良好")
	s = replaceAll(s, "thrives in sunny spots", "在阳光充足处生长良好")
	s = replaceAll(s, "produces high-quality fruit in sunny spots", "在阳光充足处结出优质果实")
	s = replaceAll(s, "yield high-quality fruit in sunny spots", "在阳光充足处结出优质果实")
	s = replaceAll(s, "Resistant like sunshine is to half shade", "耐半阴")
	s = replaceAll(s, "Avoid summer midday sun", "避免夏季正午强光")
	s = replaceAll(s, "avoid summer midday sun", "避免夏季正午强光")
	s = replaceAll(s, "avoid strong sun in summer midday", "避免夏季正午强光")
	s = replaceAll(s, "Turns red or red-purple with enough sunlight", "充足阳光下叶片变红或紫红")
	s = replaceAll(s, "turns red or red-purple with enough sunlight", "充足阳光下叶片变红或紫红")
	s = replaceAll(s, "high mountain varieties require a larger day-night temperature difference", "高山品种需要更大的昼夜温差")
	s = replaceAll(s, "needs a larger day-night temperature difference", "需要更大的昼夜温差")
	s = replaceAll(s, "place in a shaded area", "放置于阴凉处")
	s = replaceAll(s, "Place in a shaded area", "放置于阴凉处")
	s = replaceAll(s, "during flowering", "花期")
	s = replaceAll(s, "in sunny spots", "在光照充足处")
	s = replaceAll(s, "in bright indirect light", "在明亮散射光处")
	s = replaceAll(s, "in bright 散射光", "在明亮散射光处")
	s = replaceAll(s, "under strong summer light", "夏季强光下")
	s = replaceAll(s, "under summer's direct strong light", "夏季强光直射下")
	s = replaceAll(s, "half-dormant in summer", "夏季半休眠")
	s = replaceAll(s, "needs shade in strong summer light", "夏季强光时需遮阴")
	s = replaceAll(s, "avoid direct strong light", "避免强光直射")
	s = replaceAll(s, "Avoids strong direct light", "避免强光直射")
	s = replaceAll(s, "use half-shade in summer", "夏季半阴养护")
	s = replaceAll(s, "Use half-shade in summer", "夏季半阴养护")
	s = replaceAll(s, "use half-shade 夏季 for better care", "夏季半阴养护更佳")
	s = replaceAll(s, "best in bright 散射光", "明亮散射光处最佳")
	s = replaceAll(s, "place under 散射光", "放置于散射光下")
	s = replaceAll(s, "耐光照", "耐强光")
	s = replaceAll(s, "produce high-quality fruit", "结出优质果实")
	s = replaceAll(s, "yields high-quality fruit", "结出优质优质果实")
	s = replaceAll(s, "and fruit in sunny spots", "并在光照充足处结果")
	s = replaceAll(s, "sunny places", "阳光充足处")
	s = replaceAll(s, "warm, sunny places", "温暖阳光充足处")
	s = replaceAll(s, "喜温暖, sunny places", "喜温暖阳光充足处")

	// ===== 土壤 =====
	s = replaceAll(s, "Soil enriched with specific nutrients", "富含特定营养的土壤")
	s = replaceAll(s, "soil enriched with specific nutrients", "富含特定营养的土壤")
	s = replaceAll(s, "Peat and akadama mixed in a 3:1 ratio", "泥炭和赤玉土按3:1比例混合")
	s = replaceAll(s, "peat and akadama mixed in a 3:1 ratio", "泥炭和赤玉土按3:1比例混合")
	s = replaceAll(s, "Loose-textured soil or soil with specific nutrients", "疏松透气的土壤")
	s = replaceAll(s, "loose-textured soil or soil with specific nutrients", "疏松透气的土壤")
	s = replaceAll(s, "Peat with coarse sand soil", "泥炭与粗砂土混合")
	s = replaceAll(s, "peat with coarse sand soil", "泥炭与粗砂土混合")
	s = replaceAll(s, "well-draining soil", "排水良好的土壤")
	s = replaceAll(s, "Well-draining soil", "排水良好的土壤")
	s = replaceAll(s, "well-drained soil", "排水良好的土壤")
	s = replaceAll(s, "Well-drained soil", "排水良好的土壤")
	s = replaceAll(s, "rich, well-draining soil", "肥沃且排水良好的土壤")
	s = replaceAll(s, "Rich, well-draining soil", "肥沃且排水良好的土壤")
	s = replaceAll(s, "moist, well-drained soil", "湿润且排水良好的土壤")
	s = replaceAll(s, "Moist, well-drained soil", "湿润且排水良好的土壤")
	s = replaceAll(s, "slightly acidic soil", "微酸性土壤")
	s = replaceAll(s, "Slightly acidic soil", "微酸性土壤")
	s = replaceAll(s, "acidic soil", "酸性土壤")
	s = replaceAll(s, "Acidic soil", "酸性土壤")
	s = replaceAll(s, "loamy soil", "壤土")
	s = replaceAll(s, "Loamy soil", "壤土")
	s = replaceAll(s, "sandy soil", "沙质土壤")
	s = replaceAll(s, "Sandy soil", "沙质土壤")
	s = replaceAll(s, "Potting mix with", "盆栽混合土含")
	s = replaceAll(s, "potting mix with", "盆栽混合土含")
	s = replaceAll(s, "Often planted in shallow pools", "常种于浅水池中")
	s = replaceAll(s, "often planted in shallow pools", "常种于浅水池中")
	s = replaceAll(s, "prefer loose, breathable soil with peat and perlite", "偏好疏松透气的泥炭珍珠岩混合土")
	s = replaceAll(s, "prefers loose, breathable soil with peat and perlite", "偏好疏松透气的泥炭珍珠岩混合土")
	s = replaceAll(s, "prefer loose, breathable soil", "偏好疏松透气的土壤")
	s = replaceAll(s, "1/3 fertile silt, 2/3 water", "1/3 肥沃粉砂土，2/3 水")
	s = replaceAll(s, "1/3 fertile silt or river sand and 2/3 water", "1/3 肥沃粉砂土或河沙，2/3 水")
	s = replaceAll(s, "壤土 with excellent drainage", "排水良好的壤土")
	s = replaceAll(s, "Loamy sand", "壤质沙土")
	s = replaceAll(s, "Fertile silt", "肥沃粉砂土")
	s = replaceAll(s, "fertile silt", "肥沃粉砂土")
	s = replaceAll(s, "Loose, breathable soil", "疏松透气的土壤")
	s = replaceAll(s, "loose, breathable soil", "疏松透气的土壤")
	s = replaceAll(s, "Sandy, slightly acidic", "沙质微酸性")
	s = replaceAll(s, "sandy, slightly acidic", "沙质微酸性")
	s = replaceAll(s, "acid soil", "酸性土壤")
	s = replaceAll(s, "Acid soil", "酸性土壤")
	s = replaceAll(s, "Soils that are like acid soil", "酸性土壤")
	s = replaceAll(s, "with peat and perlite", "含泥炭和珍珠岩")
	s = replaceAll(s, "with peat and akadama", "含泥炭和赤玉土")
	s = replaceAll(s, "with specific nutrients", "富含特定养分")
	s = replaceAll(s, "peat or soil with specific nutrients", "泥炭或含特定养分的土壤")
	s = replaceAll(s, "or have specific nutrients", "或含特定养分")
	s = replaceAll(s, "pH 6-7", "pH 6-7")
	s = replaceAll(s, "Excellent drainage", "排水良好")
	s = replaceAll(s, "excellent drainage", "排水良好")

	// ===== 施肥 =====
	s = replaceAll(s, "Dilute fertilizers as directed; apply 1-2 times monthly in spring and autumn", "按说明稀释肥料，春秋两季每月施1-2次")
	s = replaceAll(s, "Dilute fertilizers as directed; apply 1-2 times monthly", "按说明稀释肥料，每月施1-2次")
	s = replaceAll(s, "Dilute fertilizers as directed; apply monthly", "按说明稀释肥料，每月施一次")
	s = replaceAll(s, "Dilute fertilizers as directed; apply 1-2 times a month in spring and autumn", "按说明稀释肥料，春秋两季每月施1-2次")
	s = replaceAll(s, "Dilute fertilizers as directed; apply 1-2 times a month", "按说明稀释肥料，每月施1-2次")
	s = replaceAll(s, "Dilute fertilizers as directed; apply monthly in spring and autumn", "按说明稀释肥料，春秋两季每月施一次")
	s = replaceAll(s, "Dilute fertilizers as directed", "按说明稀释肥料")
	s = replaceAll(s, "dilute fertilizers as directed", "按说明稀释肥料")
	s = replaceAll(s, "Dilute fertilizers as instructed", "按说明稀释肥料")
	s = replaceAll(s, "Feed monthly during spring and summer", "春夏每月施肥一次")
	s = replaceAll(s, "feed monthly during spring and summer", "春夏每月施肥一次")
	s = replaceAll(s, "Fertilize during the growing season", "生长季施肥")
	s = replaceAll(s, "fertilize during the growing season", "生长季施肥")
	s = replaceAll(s, "apply 1-2 times monthly", "每月施1-2次")
	s = replaceAll(s, "apply 1-2 times a month", "每月施1-2次")
	s = replaceAll(s, "apply monthly", "每月施一次")
	s = replaceAll(s, "apply once monthly", "每月施一次")
	s = replaceAll(s, "spring and autumn", "春秋两季")
	s = replaceAll(s, "spring and fall", "春秋两季")
	s = replaceAll(s, "growing season", "生长季")
	s = replaceAll(s, "balanced fertilizer", "均衡肥料")
	s = replaceAll(s, "Balanced fertilizer", "均衡肥料")
	s = replaceAll(s, "liquid fertilizer", "液态肥料")
	s = replaceAll(s, "Liquid fertilizer", "液态肥料")
	s = replaceAll(s, "slow-release fertilizer", "缓释肥料")
	s = replaceAll(s, "Slow-release fertilizer", "缓释肥料")
	s = replaceAll(s, "Apply dilute fertilizers as instructed", "按说明施用稀释肥料")
	s = replaceAll(s, "apply dilute fertilizers as instructed", "按说明施用稀释肥料")
	s = replaceAll(s, "Omit during dormancy", "休眠期停施")
	s = replaceAll(s, "omit during dormancy", "休眠期停施")
	s = replaceAll(s, "pre-flowering", "花前")
	s = replaceAll(s, "post-flowering", "花后")
	s = replaceAll(s, "pre-flowering; reduce in summer", "花前；夏季减少")
	s = replaceAll(s, "pre- and post-flowering and defoliation", "花前花后及脱皮期")
	s = replaceAll(s, "twice a month", "每月两次")
	s = replaceAll(s, "twice monthly", "每月两次")
	s = replaceAll(s, "once after flowering", "花后一次")
	s = replaceAll(s, "before flower buds appear", "花芽出现前")
	s = replaceAll(s, "before flowers open", "花前")
	s = replaceAll(s, "once a month", "每月一次")
	s = replaceAll(s, "Apply compost when planting", "种植时添加堆肥")
	s = replaceAll(s, "add compost when planting", "种植时添加堆肥")
	s = replaceAll(s, "increase fertilizing during fruiting", "结果期增加施肥")
	s = replaceAll(s, "Low-concentration fertilizers", "低浓度肥料")
	s = replaceAll(s, "low-concentration fertilizers", "低浓度肥料")
	s = replaceAll(s, "foliage at one-quarter the recommended strength", "以推荐浓度的四分之一喷施叶片")
	s = replaceAll(s, "can be applied to leaves", "可喷施于叶片")
	s = replaceAll(s, "can also be self-sustaining by capturing insects", "也可通过捕虫自养")
	s = replaceAll(s, "during the growing season", "生长季期间")
	s = replaceAll(s, "during growth", "生长期间")
	s = replaceAll(s, "spring to summer", "春夏两季")
	s = replaceAll(s, "spring to autumn", "春秋两季")
	s = replaceAll(s, "Spring to autumn", "春秋两季")
	s = replaceAll(s, "avoid winter", "冬季停施")
	s = replaceAll(s, "not in summer", "夏季除外")
	s = replaceAll(s, "except in summer", "夏季除外")
	s = replaceAll(s, "except during flowering", "花期除外")
	s = replaceAll(s, "during flowering", "花期")
	s = replaceAll(s, "to every two months", "每 1-2 个月")

	// ===== 修剪 =====
	s = replaceAll(s, "Remove dead leaves promptly", "及时清除枯叶")
	s = replaceAll(s, "remove dead leaves promptly", "及时清除枯叶")
	s = replaceAll(s, "Timely remove old, diseased, and dead leaves", "及时清除老叶、病叶和枯叶")
	s = replaceAll(s, "timely remove old, diseased, and dead leaves", "及时清除老叶、病叶和枯叶")
	s = replaceAll(s, "Remove old, diseased, and dead leaves", "清除老叶、病叶和枯叶")
	s = replaceAll(s, "remove old, diseased, and dead leaves", "清除老叶、病叶和枯叶")
	s = replaceAll(s, "Timely remove diseased and dead leaves", "及时清除病叶和枯叶")
	s = replaceAll(s, "timely remove diseased and dead leaves", "及时清除病叶和枯叶")
	s = replaceAll(s, "Remove diseased and dead leaves", "清除病叶和枯叶")
	s = replaceAll(s, "remove diseased and dead leaves", "清除病叶和枯叶")
	s = replaceAll(s, "prune to maintain shape", "修剪以保持株型")
	s = replaceAll(s, "Prune to maintain shape", "修剪以保持株型")
	s = replaceAll(s, "prune regularly", "定期修剪")
	s = replaceAll(s, "Prune regularly", "定期修剪")
	s = replaceAll(s, "trim regularly", "定期修剪")
	s = replaceAll(s, "Trim regularly", "定期修剪")
	s = replaceAll(s, "pinch back", "摘心")
	s = replaceAll(s, "Pinch back", "摘心")
	s = replaceAll(s, "After flowering, remove the top part", "花后去除上部")
	s = replaceAll(s, "after flowering, remove the top part", "花后去除上部")
	s = replaceAll(s, "starting 10 cm above soil", "从土面上方 10cm 处")
	s = replaceAll(s, "starting 5 cm above soil surface", "从土面上方 5cm 处")
	s = replaceAll(s, "leaving 3-5 leaves", "保留 3-5 片叶")
	s = replaceAll(s, "Remove the top part", "去除上部")
	s = replaceAll(s, "anytime", "随时")
	s = replaceAll(s, "Harvest before flowers open", "花前采收")
	s = replaceAll(s, "harvest before flowers open", "花前采收")
	s = replaceAll(s, "Harvest", "采收")
	s = replaceAll(s, "harvest", "采收")
	s = replaceAll(s, "Harvest potatoes when most leaves turn yellow", "大部分叶片变黄时采收土豆")
	s = replaceAll(s, "remover stems, diseased leaves", "去除茎和病叶")
	s = replaceAll(s, "harvest tender melons 20 days post-flowering", "花后 20 天采收嫩瓜")
	s = replaceAll(s, "place in a shaded area", "放置于阴凉处")
	s = replaceAll(s, "when leaves turn yellow", "叶片变黄时")
	s = replaceAll(s, "keep tubers for next year", "保留块根供来年种植")
	s = replaceAll(s, "keep bulbs for next year", "保留球根供来年种植")
	s = replaceAll(s, "leave bulbs for next year's germination", "保留球根供来年萌发")
	s = replaceAll(s, "leave bulbs for next year", "保留球根供来年种植")
	s = replaceAll(s, "the plant sprouts again in spring", "植株春季重新萌发")
	s = replaceAll(s, "sprouts again in spring", "春季重新萌发")
	s = replaceAll(s, "as parts wither in winter", "部分枝条冬季枯萎")
	s = replaceAll(s, "with parts wither in winter", "部分枝条冬季枯萎")
	s = replaceAll(s, "the upper part", "上部")
	s = replaceAll(s, "the top part", "上部")
	s = replaceAll(s, "above soil surface", "土面上方")
	s = replaceAll(s, "above soil", "土面上方")
	s = replaceAll(s, "3-5 leaves", "3-5 片叶")
	s = replaceAll(s, "flower stems", "花茎")
	s = replaceAll(s, "diseased leaves", "病叶")
	s = replaceAll(s, "withered flowers", "残花")
	s = replaceAll(s, "dead and diseased", "枯死和病变")
	s = replaceAll(s, "dead andyellowish", "枯叶和发黄")

	// ===== 温度 =====
	s = replaceAll(s, "protect from frost", "避免霜冻")
	s = replaceAll(s, "Protect from frost", "避免霜冻")
	s = replaceAll(s, "avoid frost", "避免霜冻")
	s = replaceAll(s, "Avoid frost", "避免霜冻")
	s = replaceAll(s, "keep above freezing", "保持零度以上")
	s = replaceAll(s, "Keep above freezing", "保持零度以上")
	s = replaceAll(s, "tolerates temperatures down to", "可耐低至")
	s = replaceAll(s, "Tolerates temperatures down to", "可耐低至")

	// ===== 湿度 =====
	s = replaceAll(s, "prefers high humidity", "喜高湿度")
	s = replaceAll(s, "Prefers high humidity", "喜高湿度")
	s = replaceAll(s, "high humidity", "高湿度")
	s = replaceAll(s, "High humidity", "高湿度")
	s = replaceAll(s, "mist regularly", "定期喷雾")
	s = replaceAll(s, "Mist regularly", "定期喷雾")
	s = replaceAll(s, "increase humidity", "增加湿度")
	s = replaceAll(s, "Increase humidity", "增加湿度")
	s = replaceAll(s, "use a humidifier", "使用加湿器")
	s = replaceAll(s, "Use a humidifier", "使用加湿器")

	// ===== 通用残留英文词 =====
	s = replaceAll(s, "during dormancy", "休眠期")
	s = replaceAll(s, "During dormancy", "休眠期")
	s = replaceAll(s, "during decrustation", "脱皮期")
	s = replaceAll(s, "during ecdysis", "脱皮期")
	s = replaceAll(s, "Decrustation period", "脱皮期")
	s = replaceAll(s, "decustation period", "脱皮期")
	s = replaceAll(s, "in spring", "春季")
	s = replaceAll(s, "in summer", "夏季")
	s = replaceAll(s, "in autumn", "秋季")
	s = replaceAll(s, "in winter", "冬季")
	s = replaceAll(s, "in 春秋两季", "春秋两季")
	s = replaceAll(s, "in spring and summer", "春夏两季")
	s = replaceAll(s, "in spring and autumn", "春秋两季")
	s = replaceAll(s, "during flowering and fruiting", "开花结果期")
	s = replaceAll(s, "during the growing season", "生长季")
	s = replaceAll(s, "and 耐阴", "耐阴")
	s = replaceAll(s, "and avoid", "避免")
	s = replaceAll(s, "but avoids", "但避免")
	s = replaceAll(s, "but needs", "但需")
	s = replaceAll(s, "but tolerates", "但耐")
	s = replaceAll(s, "and", "和")
	s = replaceAll(s, "foliage", "叶片")
	s = replaceAll(s, "blooming", "开花")
	s = replaceAll(s, "bloom", "开花")

	return s
}

// replaceAll 多次替换以处理重叠匹配（如 "Keep soil moist" 先于 "Keep"）
func replaceAll(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// buildGuide 将结构化字段拼成中文养护指南（与内置条目风格一致）。
// 有数据的项才输出，避免出现「浇水：」这类空行。
func buildGuide(d pbDetail) string {
	care := d.careInfo()
	var b strings.Builder
	addLine := func(label, v string) {
		if v = strings.TrimSpace(v); v != "" {
			// 将 care 描述从英文翻译为中文
			v = translateCare(label, v)
			fmt.Fprintf(&b, "%s：%s\n", label, v)
		}
	}

	// 浇水：优先 care 文本描述，其次旧版枚举（frequent/average/…）
	water := care.Watering
	if water == "" {
		if code := strings.TrimSpace(d.Watering); code != "" {
			water = wateringEnumText(code)
		}
	}
	addLine("浇水", translateCareEnToZh(water))

	// 光照：优先 care 描述，其次 human 可读阈值
	light := care.Light
	if light == "" && (d.MinLight != "" || d.MaxLight != "") {
		light = fmt.Sprintf("%s ~ %s", d.MinLight, d.MaxLight)
	}
	addLine("光照", translateCareEnToZh(light))

	if d.MinTemp != 0 || d.MaxTemp != 0 {
		fmt.Fprintf(&b, "温度：%.0f℃ ~ %.0f℃\n", d.MinTemp, d.MaxTemp)
	}

	addLine("土壤", translateCareEnToZh(care.Soil))
	addLine("施肥", translateCareEnToZh(care.Fertilization))
	addLine("修剪", translateCareEnToZh(care.Pruning))

	return strings.TrimSpace(b.String())
}

// wateringEnumText 旧版 watering 枚举的中文翻译；未知原样返回
func wateringEnumText(code string) string {
	switch strings.ToLower(code) {
	case "frequent":
		return "频繁（保持土壤湿润）"
	case "average":
		return "适中（表土干透再浇）"
	case "minimum":
		return "极少（耐旱，少浇）"
	case "none":
		return "无需浇水"
	default:
		return code
	}
}
