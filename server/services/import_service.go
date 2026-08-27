package services

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"qingye/server/models"
	"qingye/server/repositories"
)

// ImportService 批量导入业务（CSV 植物/任务 + 模板复制），支持先预览后确认
type ImportService struct {
	plants *repositories.PlantRepo
	rooms  *repositories.RoomRepo
	tasks  *repositories.TaskRepo
}

// NewImportService 构造
func NewImportService() *ImportService {
	return &ImportService{
		plants: repositories.NewPlantRepo(),
		rooms:  repositories.NewRoomRepo(),
		tasks:  repositories.NewTaskRepo(),
	}
}

// parseCSV 读取 CSV 文本为二维切片（按 UTF-8，逗号分隔，支持引号）
func parseCSV(content string) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(content))
	r.FieldsPerRecord = -1 // 允许每行字段数不一致，后续按列名对齐
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 解析失败: %w", err)
	}
	return records, nil
}

// columnIndex 根据表头行找到列下标，找不到返回 -1
func columnIndex(header []string, names ...string) int {
	for i, h := range header {
		hl := strings.ToLower(strings.TrimSpace(h))
		for _, n := range names {
			if hl == strings.ToLower(strings.TrimSpace(n)) {
				return i
			}
		}
	}
	return -1
}

// cell 安全取单元格
func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// ---------- 植物 CSV ----------

// PreviewPlants 解析植物 CSV，返回预览
// 表头示例：name,species,room,note
func (s *ImportService) PreviewPlants(content string) (*models.ImportPreview, error) {
	records, err := parseCSV(content)
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("文件至少包含表头与一行数据")
	}
	header := records[0]
	nameIdx := columnIndex(header, "name", "名称", "植物名")
	speciesIdx := columnIndex(header, "species", "品种", "学名")
	roomIdx := columnIndex(header, "room", "房间", "分组", "位置")
	noteIdx := columnIndex(header, "note", "备注", "notes")
	locationIdx := columnIndex(header, "location", "位置", "方位")
	acquiredIdx := columnIndex(header, "acquiredDate", "acquired", "获得日期", "入库日期")
	lightIdx := columnIndex(header, "lightReq", "light", "光照", "光照需求")
	attrIdx := columnIndex(header, "attributes", "attr", "属性", "扩展")

	preview := &models.ImportPreview{Kind: "plants"}
	seen := map[string]bool{}
	for i, row := range records[1:] {
		line := i + 1
		name := cell(row, nameIdx)
		if name == "" {
			preview.Rows = append(preview.Rows, models.ImportRow{
				Line: line, Status: models.ImportError, Reason: "缺少植物名称",
			})
			preview.Invalid++
			continue
		}
		room := cell(row, roomIdx)
		rowData := map[string]any{
			"name":        name,
			"species":     cell(row, speciesIdx),
			"room":        room,
			"note":        cell(row, noteIdx),
			"location":    cell(row, locationIdx),
			"acquiredDate": cell(row, acquiredIdx),
			"lightReq":    cell(row, lightIdx),
			"attributes":  cell(row, attrIdx),
		}
		status := models.ImportOK
		reason := ""
		if room != "" {
			status = models.ImportWarning
			reason = "房间「" + room + "」不存在时将自动创建"
		}
		if seen[name] {
			status = models.ImportWarning
			reason = joinReason(reason, "名称重复")
		}
		seen[name] = true
		preview.Rows = append(preview.Rows, models.ImportRow{Line: line, Status: status, Reason: reason, Data: rowData})
		if status != models.ImportError {
			preview.Valid++
		} else {
			preview.Invalid++
		}
	}
	preview.Summary = fmt.Sprintf("共解析 %d 株植物，其中 %d 行可导入、%d 行错误。", len(preview.Rows), preview.Valid, preview.Invalid)
	return preview, nil
}

// ConfirmPlants 确认落库植物（仅导入 accepted 中的行，默认全部有效行）
func (s *ImportService) ConfirmPlants(preview *models.ImportPreview, accepted []int) (*models.ImportResult, error) {
	acceptSet := toSet(accepted)
	res := &models.ImportResult{Kind: "plants"}
	for _, r := range preview.Rows {
		if r.Status == models.ImportError {
			res.Skipped++
			continue
		}
		if len(acceptSet) > 0 && !acceptSet[r.Line] {
			res.Skipped++
			continue
		}
		m := r.Data.(map[string]any)
		name := m["name"].(string)
		roomName, _ := m["room"].(string)
		var roomID uint
		if roomName != "" {
			room, err := s.ensureRoom(roomName)
			if err != nil {
				res.Skipped++
				continue
			}
			roomID = room.ID
		}
		plant := &models.Plant{
			Name:    name,
			Species: optStr(m["species"]),
			RoomID:  roomID,
			Note:    optStr(m["note"]),
			Location: optStr(m["location"]),
			LightReq: optStr(m["lightReq"]),
			Attributes: optStr(m["attributes"]),
		}
		if ad := optStr(m["acquiredDate"]); ad != "" {
			if t, perr := time.Parse("2006-01-02", ad); perr == nil {
				plant.AcquiredDate = &t
			}
		}
		if err := s.plants.Create(plant); err != nil {
			res.Skipped++
			continue
		}
		res.Created++
		res.PlantIDs = append(res.PlantIDs, plant.ID)
	}
	res.Message = fmt.Sprintf("成功导入 %d 株植物，跳过 %d 行。", res.Created, res.Skipped)
	return res, nil
}

// ---------- 任务 CSV ----------

// PreviewTasks 解析任务 CSV，返回预览（按植物名匹配）
// 表头示例：plant,type,intervalDays,title,startDate
// type 取值：water(浇水) / fertilize(施肥) / repot(换盆) / prune(修剪) / other(其他)
func (s *ImportService) PreviewTasks(content string) (*models.ImportPreview, error) {
	records, err := parseCSV(content)
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("文件至少包含表头与一行数据")
	}
	header := records[0]
	plantIdx := columnIndex(header, "plant", "植物", "植物名")
	typeIdx := columnIndex(header, "type", "类型", "任务类型")
	intervalIdx := columnIndex(header, "intervalDays", "interval", "周期", "间隔天数")
	titleIdx := columnIndex(header, "title", "标题", "名称")
	startIdx := columnIndex(header, "startDate", "start", "开始日期")

	plants, err := s.plants.List(0)
	if err != nil {
		return nil, err
	}
	plantByName := map[string]models.Plant{}
	for _, p := range plants {
		plantByName[strings.ToLower(p.Name)] = p
	}

	preview := &models.ImportPreview{Kind: "tasks"}
	for i, row := range records[1:] {
		line := i + 1
		plantName := cell(row, plantIdx)
		typeRaw := strings.ToLower(cell(row, typeIdx))
		intervalRaw := cell(row, intervalIdx)
		titleRaw := cell(row, titleIdx)
		startRaw := cell(row, startIdx)

		if plantName == "" || typeRaw == "" {
			preview.Rows = append(preview.Rows, models.ImportRow{
				Line: line, Status: models.ImportError, Reason: "缺少植物名或任务类型",
			})
			preview.Invalid++
			continue
		}
		taskType, ok := normalizeTaskType(typeRaw)
		if !ok {
			preview.Rows = append(preview.Rows, models.ImportRow{
				Line: line, Status: models.ImportError, Reason: "未知任务类型: " + typeRaw,
			})
			preview.Invalid++
			continue
		}
		interval, err := strconv.Atoi(intervalRaw)
		if err != nil || interval <= 0 {
			preview.Rows = append(preview.Rows, models.ImportRow{
				Line: line, Status: models.ImportError, Reason: "周期(intervalDays)需为正整数: " + intervalRaw,
			})
			preview.Invalid++
			continue
		}
		if startRaw != "" {
			if _, perr := time.Parse("2006-01-02", startRaw); perr != nil {
				preview.Rows = append(preview.Rows, models.ImportRow{
					Line: line, Status: models.ImportError, Reason: "开始日期格式应为 YYYY-MM-DD: " + startRaw,
				})
				preview.Invalid++
				continue
				}
				}

				p, found := plantByName[strings.ToLower(plantName)]
		status := models.ImportOK
		reason := ""
		if !found {
			status = models.ImportError
			reason = "未找到植物「" + plantName + "」"
			preview.Invalid++
		}
		rowData := map[string]any{
			"plantName":    plantName,
			"plantId":      p.ID,
			"type":         taskType,
			"title":        titleRaw,
			"intervalDays": interval,
			"startDate":    startRaw,
		}
		preview.Rows = append(preview.Rows, models.ImportRow{Line: line, Status: status, Reason: reason, Data: rowData})
		if status == models.ImportOK {
			preview.Valid++
		}
	}
	preview.Summary = fmt.Sprintf("共解析 %d 条任务，其中 %d 行可导入、%d 行错误。", len(preview.Rows), preview.Valid, preview.Invalid)
	return preview, nil
}

// ConfirmTasks 确认落库任务
func (s *ImportService) ConfirmTasks(preview *models.ImportPreview, accepted []int) (*models.ImportResult, error) {
	acceptSet := toSet(accepted)
	res := &models.ImportResult{Kind: "tasks"}
	for _, r := range preview.Rows {
		if r.Status != models.ImportOK {
			res.Skipped++
			continue
		}
		if len(acceptSet) > 0 && !acceptSet[r.Line] {
			res.Skipped++
			continue
		}
		m := r.Data.(map[string]any)
		plantID := m["plantId"].(uint)
		taskType := m["type"].(string)
		interval := m["intervalDays"].(int)
		title, _ := m["title"].(string)
		startRaw, _ := m["startDate"].(string)

		now := time.Now()
		var startDate time.Time
		if startRaw != "" {
			startDate, _ = time.Parse("2006-01-02", startRaw)
		} else {
			startDate = now
		}
		nextDue := startDate.AddDate(0, 0, interval)
		task := &models.Task{
			PlantID:          plantID,
			Type:             taskType,
			Title:            title,
			IntervalDays:     interval,
			BaseIntervalDays: interval,
			NextDue:          nextDue,
			Active:           true,
		}
		if err := s.tasks.Create(task); err != nil {
			res.Skipped++
			continue
		}
		res.Created++
		res.TaskIDs = append(res.TaskIDs, task.ID)
	}
	res.Message = fmt.Sprintf("成功创建 %d 条任务，跳过 %d 行。", res.Created, res.Skipped)
	return res, nil
}

// ---------- 模板复制 ----------

// PreviewTemplate 预览：把来源植物的任务配置复制到目标植物
func (s *ImportService) PreviewTemplate(sourceID uint, targetIDs []uint) (*models.ImportPreview, error) {
	if len(targetIDs) == 0 {
		return nil, fmt.Errorf("请选择至少一株目标植物")
	}
	sourcePlant, err := s.plants.Get(sourceID)
	if err != nil {
		return nil, fmt.Errorf("来源植物不存在: %w", err)
	}
	sourceTasks, err := s.tasks.List("", false, sourceID)
	if err != nil {
		return nil, err
	}
	preview := &models.ImportPreview{Kind: "template"}
	for _, tid := range targetIDs {
		target, err := s.plants.Get(tid)
		if err != nil {
			preview.Rows = append(preview.Rows, models.ImportRow{
				Line: int(tid), Status: models.ImportError, Reason: "目标植物不存在", Data: map[string]any{"targetId": tid},
			})
			preview.Invalid++
			continue
		}
		rowData := map[string]any{
			"targetId":   tid,
			"targetName": target.Name,
			"sourceName": sourcePlant.Name,
			"taskCount":  len(sourceTasks),
		}
		status := models.ImportOK
		reason := ""
		if len(sourceTasks) == 0 {
			status = models.ImportWarning
			reason = "来源暂无任务可复制"
		}
		preview.Rows = append(preview.Rows, models.ImportRow{Line: int(tid), Status: status, Reason: reason, Data: rowData})
		if status != models.ImportError {
			preview.Valid++
		}
	}
	preview.Summary = fmt.Sprintf("将为 %d 株目标植物复制 %d 条任务模板。", preview.Valid, len(sourceTasks))
	return preview, nil
}

// ConfirmTemplate 执行模板复制
func (s *ImportService) ConfirmTemplate(sourceID uint, targetIDs []uint) (*models.ImportResult, error) {
	sourceTasks, err := s.tasks.List("", false, sourceID)
	if err != nil {
		return nil, err
	}
	res := &models.ImportResult{Kind: "template"}
	for _, tid := range targetIDs {
		target, err := s.plants.Get(tid)
		if err != nil {
			res.Skipped++
			continue
		}
		for _, st := range sourceTasks {
			now := time.Now()
			nextDue := now.AddDate(0, 0, st.IntervalDays)
			task := &models.Task{
				PlantID:          target.ID,
				Type:             st.Type,
				Title:            st.Title,
				IntervalDays:     st.IntervalDays,
				BaseIntervalDays: st.IntervalDays,
				NextDue:          nextDue,
				Active:           true,
			}
			if cerr := s.tasks.Create(task); cerr != nil {
				res.Skipped++
				continue
			}
			res.Created++
			res.TaskIDs = append(res.TaskIDs, task.ID)
		}
	}
	res.Message = fmt.Sprintf("成功复制 %d 条任务到 %d 株植物，跳过 %d 项。", res.Created, len(targetIDs), res.Skipped)
	return res, nil
}

// ---------- 辅助 ----------

func joinReason(base, extra string) string {
	if base == "" {
		return extra
	}
	return base + "；" + extra
}

func toSet(accepted []int) map[int]bool {
	if len(accepted) == 0 {
		return nil
	}
	m := map[int]bool{}
	for _, a := range accepted {
		m[a] = true
	}
	return m
}

func optStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// ensureRoom 按名称查找房间，不存在则创建
func (s *ImportService) ensureRoom(name string) (*models.Room, error) {
	rooms, err := s.rooms.List()
	if err != nil {
		return nil, err
	}
	for i := range rooms {
		if rooms[i].Name == name {
			return &rooms[i], nil
		}
	}
	room := &models.Room{Name: name}
	if err := s.rooms.Create(room); err != nil {
		return nil, err
	}
	return room, nil
}

// normalizeTaskType 将用户输入的任务类型归一化为系统常量
// 系统支持：water(浇水) / fertilize(施肥) / mist(喷雾) / repot(换盆) / prune(修剪) / clean(清理) / pesticide(除虫) / other(其他)
func normalizeTaskType(raw string) (string, bool) {
	switch raw {
	case TaskTypeWater, "浇水":
		return TaskTypeWater, true
	case TaskTypeFertilize, "施肥":
		return TaskTypeFertilize, true
	case TaskTypeMist, "喷雾":
		return TaskTypeMist, true
	case TaskTypeRepot, "换盆":
		return TaskTypeRepot, true
	case TaskTypePrune, "修剪":
		return TaskTypePrune, true
	case TaskTypeClean, "清理":
		return TaskTypeClean, true
	case TaskTypePesticide, "除虫":
		return TaskTypePesticide, true
	case TaskTypeOther, "其他":
		return TaskTypeOther, true
	}
	return "", false
}
