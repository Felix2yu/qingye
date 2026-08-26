# 青野 QingYe 🌿

家庭园艺植物记录与养护 Web 应用。复刻 iOS「植物宝」类工具的核心体验：以 **任务清单 + 植物收藏 + 照片日记 + 资料库 + 智能规划** 让多盆植物的日常养护井井有条、可追溯、有成就感。

前后端分离，各自可独立运行、联调顺畅。

- **后端** Go 1.22 + Gin + GORM + SQLite（零部署单文件数据库）
- **前端** SvelteKit（Svelte 5 + Vite + TypeScript）
- **交互** RESTful JSON API，开发期 Vite 代理联调

---

## 目录结构

```
qingye/
├── README.md            # 项目说明
├── Makefile             # 快捷启动命令
├── server/              # Go 后端（Gin + GORM + SQLite）
│   ├── main.go          # 入口：配置 / AutoMigrate / 路由 / 启动
│   ├── config/          # 配置加载（端口 / 数据库 / 上传目录 / CORS）
│   ├── models/          # GORM 实体
│   ├── repositories/    # 数据访问层
│   ├── services/        # 业务层（今日任务计算、完成/推迟、日记等）
│   ├── handlers/        # REST 路由处理 + 统一响应
│   ├── router/          # 路由注册 + CORS / 日志 / Recovery
│   ├── seed/            # 资料库示例 + 初始设置 + 演示数据
│   ├── uploads/         # 本地照片存储
│   └── .env.example
└── web/                 # SvelteKit 前端
    ├── src/lib/         # api 封装 / 类型 / stores / 组件
    ├── src/routes/      # 页面路由
    ├── vite.config.ts   # API 代理
    └── .env.example
```

---

## 快速开始

### 1. 安装依赖

```bash
make install-server   # go mod download
make install-web      # npm install
```

### 2. 启动后端

```bash
make run-server
# 或手动：
cd server && PORT=8081 go run .
```

后端默认监听 `http://localhost:8081`（8080 常被本机其他应用占用，已避开）。
首次启动会自动 `AutoMigrate` 建表，并写入：
- 植物资料库示例（绿萝、龟背竹、虎皮兰等 10 种）
- 初始设置（默认工作日周一至周五）
- 演示数据（客厅/阳台/卧室 + 4 株植物 + 若干任务 + 2 条日记）

健康检查：`curl http://localhost:8081/healthz` → `{"status":"ok"}`

### 3. 启动前端

```bash
make run-web
# 或手动：
cd web && npm run dev
```

前端默认 `http://localhost:5173`，通过 Vite 代理把 `/api` 与 `/uploads` 转发到后端 8081，无需额外 CORS 配置。

打开浏览器访问 `http://localhost:5173` 即可使用。

> 如需同时跑前后端，推荐使用 `make run-server` 与 `make run-web` 分别在两个终端，或终端多路复用（如 `tmux` / VS Code 多终端）。

### 4. 生产构建（可选）

```bash
make build-web          # 输出到 web/build（adapter-auto）
cd server && PORT=8081 go build -o qingye-server .   # 编译后端二进制
```

---

## 核心功能

| 模块 | 说明 |
| --- | --- |
| 植物档案 | 增删改查、房间/分组整理、照片与备注 |
| 任务清单 | 浇水/施肥/换盆等可重复任务；完成、推迟、类型筛选、历史记录 |
| 今日概览 | 今日任务卡片、临近提醒、快捷入口；**休息日不展示任务，任务顺延不丢失** |
| 照片日记 | 按植物记录照片与文字，时间线展示成长瞬间 |
| 智能规划 | 用户设置工作日/休息日，系统仅在工作日展示当日任务 |
| 植物资料库 | 内置常见室内植物养护指南，支持关键词搜索 |
| 批量导入 | CSV 批量导入植物/任务，**先预览后确认**；支持把某株植物的任务模板复制给多株植物 |
| 设置中心 | 工作日选择器、偏好存储 |

---

## 设计要点

- **重复任务按需计算**：任务以「规则 + 上次完成时间」存储，查询时计算下一次到期，避免海量任务行；仅完成/推迟动作写入 `task_logs`。
- **工作日过滤**：`user_settings.workdays` 以 `1-7`（周一至周日）记录，今日任务仅在「今天为工作日且 next_due ≤ 今天」时展示。
- **照片存储**：SQLite 仅存元数据（路径/说明/时间），图片文件存本地 `server/uploads/`。
- **事务保证**：「完成任务 + 写日志 + 更新 next_due」在同一事务内完成。
- **统一响应**：所有接口返回 `{ code, message, data }`，`code=0` 表示成功。

---

## API 索引

基础前缀 `/api`。完整示例（`localhost:8081`）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/rooms` | 房间列表（含植物统计） |
| POST | `/rooms` | 新建房间 |
| PUT | `/rooms/:id` | 更新房间 |
| DELETE | `/rooms/:id` | 删除房间 |
| GET | `/plants?roomId=` | 植物列表（按房间筛选） |
| POST | `/plants` | 新建植物 |
| GET | `/plants/:id` | 植物详情 |
| PUT | `/plants/:id` | 更新植物 |
| DELETE | `/plants/:id` | 删除植物（及关联任务/日记） |
| GET | `/tasks?type=&includeDone=&plantId=` | 任务列表 |
| GET | `/tasks/today` | 今日任务（工作日过滤） |
| GET | `/tasks/upcoming?days=3` | 临近任务 |
| POST | `/tasks` | 新建任务 |
| POST | `/tasks/:id/done` | 完成任务 |
| POST | `/tasks/:id/postpone` | 推迟任务 |
| GET | `/tasks/:id/logs` | 任务历史 |
| DELETE | `/tasks/:id` | 删除任务 |
| GET | `/diaries?plantId=&page=&pageSize=` | 日记分页 |
| POST | `/diaries` (multipart) | 新增日记（图片上传） |
| DELETE | `/diaries/:id` | 删除日记 |
| GET | `/library?q=` | 资料库搜索 |
| GET | `/settings` | 读取设置 |
| PUT | `/settings` | 更新工作日/偏好 |
| POST | `/import/preview` (multipart) | 上传 CSV 解析预览（`kind=plants\|tasks`） |
| POST | `/import/confirm` (JSON) | 确认落库（传入原始 CSV 与勾选行号） |
| POST | `/import/template-preview` (JSON) | 模板复制预览（`{sourceId, targetIds}`） |

---

## 环境变量

**后端** `server/.env`（参考 `server/.env.example`）：

```
PORT=8081
DB_PATH=./data/qingye.db
UPLOAD_DIR=./uploads
CORS_ORIGINS=http://localhost:5173
MAX_UPLOAD_MB=10
# 可选：Plantbook 在线植物库 token（open.plantbook.io 注册获取）
# 配置后启用「在线匹配 / 批量同步」，留空则只用本地资料库
PLANTBOOK_TOKEN=
```

**前端** `web/.env`（参考 `web/.env.example`）：

```
# 留空走 Vite 代理；部署到独立域名时填后端地址
VITE_API_BASE=
```

---

## 批量导入（CSV）

入口：前端导航「导入」。支持三种方式，**均为先预览、再确认落库**，错误行不会写入。

### 1. 导入植物

CSV 表头（列名不区分大小写，可为中文）：

```
name,species,room,note
龟背竹,Monstera deliciosa,客厅,喜散射光
绿萝,,卧室,耐阴
```

- `name` 必填；`room` 不存在时自动创建该房间；`species` 对应品种/学名；`note` 为备注。
- 同名、缺名会标记为提醒/错误。

### 2. 导入任务

CSV 表头：

```
plant,type,intervalDays,title,startDate
龟背竹,water,7,浇水,2026-09-01
绿萝,fertilize,30,施肥,
```

- `plant` 必须已存在（按名称匹配）；`type` 取值 `water`(浇水) / `fertilize`(施肥) / `repot`(换盆)；`intervalDays` 为正整数（周期天数）；`title`、`startDate`(YYYY-MM-DD) 可选，缺省则用今天起算。

### 3. 模板复制

选择「来源植物」（复制其任务配置）+ 勾选若干「目标植物」，预览后确认，即把来源植物的任务规则批量复制到每株目标植物。

---

## 在线植物资料库（Plantbook）

青野内置 10 种常见植物的中文养护指南。配置 `PLANTBOOK_TOKEN` 后，可在线扩展资料库，无需人工录入。

### 工作原理

- **唯一键用学名 `pid`**：Plantbook 以学名（如 `monstera_deliciosa`）作为稳定唯一键，避免中文别名歧义；同一 `pid` 重复同步自动覆盖刷新。
- **中文直接取回**：详情请求带 `lang=zh`，优先使用 Plantbook 自带的中文常见名，无需第三方翻译。
- **结构化字段映射**：浇水频率 / 光照 / 温度 / 土壤 / 施肥 / 修剪等字段自动拼成中文三段式指南，与内置条目风格一致。
- **本地缓存、离线可用**：首次在线匹配 / 同步的结果会写回本地 `PlantLibrary`，之后添加植物直接本地命中，零网络、零额度。

### 两种用法

1. **录入时在线匹配**（懒加载）：添加植物时填写名称 → 点「在线查找」→ 从候选列表选一条 → 自动带回中文指南并落库。
2. **批量同步热门植物**（离线预置）：设置页「植物资料库同步」→ 点「同步热门植物」，把内置 30 种常见植物的中文指南一次性拉取沉淀到本地（详见 `server/services/library_service.go` 的 `popularSeeds`）。

### 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/api/library/online?keyword=` | 在线搜索候选（未配置 token 返回 `enabled:false`） |
| POST | `/api/library/import` | 按 `pid` 拉详情并写回本地资料库 |
| POST | `/api/library/sync-popular` | 批量同步热门植物到本地资料库 |

> 未配置 `PLANTBOOK_TOKEN` 时，上述在线功能全部优雅降级为「仅本地资料库」，不影响其他功能。

---

## 重置演示数据

删除 `server/data/qingye.db` 并重启后端即可重新生成全部种子与演示数据。
