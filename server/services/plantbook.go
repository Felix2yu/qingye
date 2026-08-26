package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"qingye/server/models"
)

const plantbookBaseURL = "https://open.plantbook.io/api/v1"

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

// pbDetail Plantbook 植物详情（仅取本应用需要的字段）
type pbDetail struct {
	PID           string         `json:"pid"`
	Alias         string         `json:"alias"`
	CommonNames   []pbCommonName `json:"common_names"`
	ImageURL      string         `json:"image_url"`
	Link          string         `json:"link"`
	Watering      string         `json:"watering"`
	MaxTemp       float64        `json:"max_temp"`
	MinTemp       float64        `json:"min_temp"`
	MaxLight      string         `json:"max_light_human"`
	MinLight      string         `json:"min_light_human"`
	Fertilization string         `json:"fertilization"`
	Pruning       string         `json:"pruning"`
	Soil          []string       `json:"soil"`
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
	u := fmt.Sprintf("%s/plant/detail/%s/?lang=zh", plantbookBaseURL, url.PathEscape(pid))
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
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Plantbook 失败: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

// detailToLibrary 将 Plantbook 详情映射为中文 PlantLibrary（三段式 Guide）
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
	return &models.PlantLibrary{
		PID:   d.PID,
		Name:  name,
		Alias: d.Alias,
		Guide: guide,
		Image: d.ImageURL,
	}
}

// buildGuide 将结构化字段拼成中文三段式指南（与内置 10 条风格一致）
func buildGuide(d pbDetail) string {
	var b strings.Builder
	fmt.Fprintf(&b, "浇水：%s\n", wateringText(d.Watering))
	if d.MinLight != "" || d.MaxLight != "" {
		fmt.Fprintf(&b, "光照：%s ~ %s\n", d.MinLight, d.MaxLight)
	}
	if d.MinTemp != 0 || d.MaxTemp != 0 {
		fmt.Fprintf(&b, "温度：%.0f℃ ~ %.0f℃\n", d.MinTemp, d.MaxTemp)
	}
	if len(d.Soil) > 0 {
		fmt.Fprintf(&b, "土壤：%s\n", strings.Join(d.Soil, "、"))
	}
	fmt.Fprintf(&b, "施肥：%s\n", d.Fertilization)
	fmt.Fprintf(&b, "修剪：%s\n", d.Pruning)
	return strings.TrimSpace(b.String())
}

func wateringText(code string) string {
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
		if code == "" {
			return "适中"
		}
		return code
	}
}
