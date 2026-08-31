# 项目长期笔记

## 项目性质
qingye（青野）是一个**植物养护记录网站**，不是演出记录。核心实体：植物(Plant)、养护任务(Task)、养护日志(CareLog/PhotoDiary)、天气策略。任务类型共 8 种：water/fertilize/mist/repot/prune/clean/pesticide/other（见 format.ts 与 task_service.go 常量）。

## 架构约定
- 后端 Go + gorm（sqlite）；前端 SvelteKit（web/）。
- 任务类型字符串存于 Task.Type / CareLog.Type；生命周期方法（Done/Postpone/History/Today）对类型无硬编码，新增类型免改。
- 天气策略只在 water/fertilize 上调整频率/推迟，新类型不参与。
- CSV 导入：import_service.go 的 normalizeTaskType 负责类型归一化（中英文别名）。
