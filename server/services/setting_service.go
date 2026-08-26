package services

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"
)

// WorkdaySet 工作日集合，key 为 1-7（1=周一 … 7=周日）
type WorkdaySet map[int]bool

// ParseWorkdays 将 "1,2,3,4,5" 解析为集合；空串返回空集
func ParseWorkdays(s string) WorkdaySet {
	set := WorkdaySet{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil && n >= 1 && n <= 7 {
			set[n] = true
		}
	}
	return set
}

// FormatWorkdays 将集合格式化为升序 "1,2,3,4,5"
func FormatWorkdays(set WorkdaySet) string {
	keys := make([]int, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, strconv.Itoa(k))
	}
	return strings.Join(parts, ",")
}

// WeekdayToInt Go time.Weekday（Sun=0…Sat=6）→ 业务 1=周一…7=周日
func WeekdayToInt(d time.Weekday) int {
	if d == time.Sunday {
		return 7
	}
	return int(d)
}

// SettingService 工作日 / 偏好设置
type SettingService struct{ repo *repositories.SettingRepo }

func NewSettingService() *SettingService {
	return &SettingService{repo: repositories.NewSettingRepo()}
}

// Get 读取设置；首次使用返回默认工作日（周一至周五）
func (s *SettingService) Get() (*models.UserSetting, error) {
	st, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	if st.Workdays == "" {
		st.Workdays = "1,2,3,4,5"
	}
	if st.Prefs == "" {
		st.Prefs = "{}"
	}
	return st, nil
}

// Update 更新工作日与偏好
func (s *SettingService) Update(workdays []int, prefs map[string]any) (*models.UserSetting, error) {
	if len(workdays) == 0 {
		return nil, errors.New("至少选择一个工作日")
	}
	set := WorkdaySet{}
	for _, d := range workdays {
		if d < 1 || d > 7 {
			return nil, errors.New("工作日取值须在 1（周一）到 7（周日）之间")
		}
		set[d] = true
	}
	st, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	st.Workdays = FormatWorkdays(set)
	if prefs != nil {
		b, err := json.Marshal(prefs)
		if err != nil {
			return nil, err
		}
		st.Prefs = string(b)
	}
	if err := s.repo.Save(st); err != nil {
		return nil, err
	}
	return st, nil
}

// IsWorkday 判断某天是否为设置中的工作日
func (s *SettingService) IsWorkday(day time.Time) (bool, error) {
	st, err := s.Get()
	if err != nil {
		return false, err
	}
	return ParseWorkdays(st.Workdays)[WeekdayToInt(day.Weekday())], nil
}
