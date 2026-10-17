package models

// ImportRowStatus 单行解析状态
type ImportRowStatus string

const (
	ImportOK      ImportRowStatus = "ok"      // 可正常导入
	ImportWarning ImportRowStatus = "warning" // 有警告（如房间将自动创建）
	ImportError   ImportRowStatus = "error"   // 该行无法导入
)

// ImportRow 通用导入行（预览用）
type ImportRow struct {
	Line   int             `json:"line"`   // CSV 行号（从 1 开始，不含表头）
	Status ImportRowStatus `json:"status"` // 解析状态
	Reason string          `json:"reason"` // 状态说明
	Data   any             `json:"data"`   // 该行解析出的业务数据
}

// ImportPreview 导入预览结果
type ImportPreview struct {
	Kind    string      `json:"kind"`    // plants | tasks | template
	Rows    []ImportRow `json:"rows"`    // 解析后的行
	Valid   int         `json:"valid"`   // 可导入行数
	Invalid int         `json:"invalid"` // 错误行数
	Summary string      `json:"summary"` // 人类可读摘要
}

// ImportConfirmRequest 确认导入请求
type ImportConfirmRequest struct {
	Kind     string   `json:"kind"`     // plants | tasks | template
	Content  string   `json:"content"`  // 原始 CSV 文本（plants/tasks 复用解析）
	Accepted []int    `json:"accepted"` // 确认导入的行号（仅这些行落库）；为空表示全部有效行
	// 模板复制专用
	SourceID  uint   `json:"sourceId"`  // 模板来源植物 ID
	TargetIDs []uint `json:"targetIds"` // 目标植物 ID 列表
}

// ImportPreviewRequest 预览请求（直接上传 CSV 文本，便于先解析）
type ImportPreviewRequest struct {
	Kind    string `json:"kind"`    // plants | tasks
	Content string `json:"content"` // CSV 文本
}

// ImportResult 导入执行结果
type ImportResult struct {
	Kind     string `json:"kind"`
	Created  int    `json:"created"`  // 成功创建数
	Skipped  int    `json:"skipped"`  // 跳过数（重复/无效）
	PlantIDs []uint `json:"plantIds"` // 新建植物 ID（仅 kind=plants）
	TaskIDs  []uint `json:"taskIds"`  // 新建任务 ID（仅 kind=tasks/template）
	Message  string `json:"message"`
}
