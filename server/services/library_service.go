package services

import (
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
		plantbook: NewPlantbookClient(cfg.PlantbookToken),
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

// popularSeeds 内置中文名清单（常见室内/露台植物），用于离线批量预置。
// Plantbook 的 common_names 含多语言，用中文名可直接 search 命中。
var popularSeeds = []string{
	// 常见室内观叶
	"绿萝", "龟背竹", "虎皮兰", "琴叶榕", "吊兰", "多肉植物", "薄荷", "富贵竹",
	"散尾葵", "鹅掌柴", "仙人掌", "芦荟", "文竹", "橡皮树", "发财树", "君子兰",
	"常春藤", "铜钱草", "金钻", "白掌", "红掌", "铁线蕨", "波士顿蕨", "沙漠玫瑰",
	"龙骨", "玉树", "金钱树", "平安树", "鸭脚木", "万年青", "一叶兰", "竹芋",
	"春羽", "天堂鸟", "鹤望兰", "袖珍椰子", "棕竹", "网纹草", "空气凤梨",
	"量天尺", "金琥", "虹之玉", "熊童子", "玉露", "生石花", "仙人球", "龙舌兰",
	// 室内开花
	"蝴蝶兰", "长寿花", "非洲菊", "仙客来", "蟹爪兰", "丽格海棠", "茶花", "兰花",
	"石斛兰", "文心兰", "大花蕙兰", "三角梅", "龙吐珠", "天竺葵",
	// 露台 / 阳台常见
	"月季", "茉莉", "栀子花", "绣球", "玫瑰", "薰衣草", "迷迭香",
	"罗勒", "百里香", "紫苏", "香菜", "小番茄", "辣椒", "草莓", "蓝莓",
	"葡萄", "无花果", "石榴", "金桔", "柠檬", "桂花", "紫藤", "凌霄",
	"牵牛花", "太阳花", "矮牵牛", "玛格丽特", "菊花",
	"铁线莲", "金银花", "睡莲", "荷花", "碗莲", "一叶莲",
}

// SyncPopular 批量同步热门植物到本地资料库（离线可用）。
// 对每个中文名 search 取首个候选的 pid，再 detail?lang=zh 写回。
// 已存在（同 pid）的条目会被覆盖刷新。返回同步成功/失败计数。
func (s *LibraryService) SyncPopular() (added int, failed int, err error) {
	if !s.plantbook.Enabled() {
		return 0, 0, nil
	}
	for _, name := range popularSeeds {
		cands, e := s.plantbook.Search(name)
		if e != nil {
			failed++
			continue
		}
		if len(cands) == 0 {
			failed++
			continue
		}
		lib, e := s.plantbook.Detail(cands[0].PID)
		if e != nil || lib == nil {
			failed++
			continue
		}
		lib.SyncedAt = time.Now().Unix()
		if e := s.repo.UpsertByPID(lib); e != nil {
			failed++
			continue
		}
		added++
	}
	return added, failed, nil
}
