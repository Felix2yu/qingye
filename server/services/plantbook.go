package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"qingye/server/models"
)

const plantbookBaseURL = "https://open.plantbook.io/api/v1"

// PlantbookClient 封装 Plantbook 在线植物资料 API。
// 设计要点：
//   - 以学名 pid（如 monstera_deliciosa）作为唯一键，避免中文名歧义
//   - 详情请求带 lang=zh，优先取 Plantbook 自带的中文 common_name 与字段，免去第三方翻译
//   - 无 token 时所有方法优雅返回空结果，由上层降级为"仅本地资料库"
type PlantbookClient struct {
	token  string
	client *http.Client
}

// NewPlantbookClient 创建客户端；token 为空表示禁用在线匹配。
func NewPlantbookClient(token string) *PlantbookClient {
	return &PlantbookClient{
		token:  token,
		client: &http.Client{Timeout: 12 * time.Second},
	}
}

func (p *PlantbookClient) Enabled() bool { return p.token != "" }

// pbSearchItem search 接口返回的单条候选（精简字段）
type pbSearchItem struct {
	PID            string `json:"pid"`
	Alias          string `json:"alias"`
	CommonNames    any    `json:"common_names"`
	ImageURL       string `json:"image_url"`
	Link           string `json:"link"`
	Vegetable       bool  `json:"vegetable"`
	Edible         bool  `json:"edible"`
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
	PID              string         `json:"pid"`
	Alias            string         `json:"alias"`
	CommonNames      []pbCommonName `json:"common_names"`
	ImageURL         string         `json:"image_url"`
	Link             string         `json:"link"`
	Watering         string         `json:"watering"`
	MaxTemp          float64        `json:"max_temp"`
	MinTemp          float64        `json:"min_temp"`
	MaxLight        string         `json:"max_light_human"`
	MinLight        string         `json:"min_light_human"`
	Fertilization    string         `json:"fertilization"`
	Pruning          string         `json:"pruning"`
	Soil             []string       `json:"soil"`
}

// OnlineCandidate 在线搜索返回的候选条目（供前端展示让用户选择）
type OnlineCandidate struct {
	PID        string `json:"pid"`
	Name       string `json:"name"`       // 优先中文 common_name，否则首个 common_name
	Alias      string `json:"alias"`      // 学名（alias）
	Image      string `json:"image"`      // 缩略图
	CommonNames []string `json:"commonNames"` // 所有常见名（含中文）
}

// Search 按关键词搜索植物（中文名也能搜，Plantbook common_names 含多语言）。
// keyword 长度需 >= 2；token 为空时返回空列表。
func (p *PlantbookClient) Search(keyword string) ([]OnlineCandidate, error) {
	if !p.Enabled() || len(strings.TrimSpace(keyword)) < 2 {
		return nil, nil
	}
	u := fmt.Sprintf("%s/plant/search?alias=%s&limit=20",
		plantbookBaseURL, url.QueryEscape(keyword))
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

// doGet 带 token 的 GET 请求
func (p *PlantbookClient) doGet(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+p.token)
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
		return nil, fmt.Errorf("Plantbook 返回 %d: %s", resp.StatusCode, string(data))
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
