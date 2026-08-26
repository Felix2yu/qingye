# 青野 QingYe 🌿

家庭园艺植物记录与养护应用。以 **植物档案 + 任务清单 + 照片日记 + 资料库 + 智能规划 + 天气策略** 让多盆植物的日常养护井井有条、可追溯、有成就感。

前后端一体：后端 Go 提供 API 与页面托管，前端 SvelteKit 构建为静态 SPA，可由后端同源托管，也可独立运行。

- **后端** Go 1.27 + Gin + GORM + SQLite（零部署单文件数据库）
- **前端** SvelteKit（Svelte 5 + Vite + TypeScript），`adapter-static` 输出静态 SPA
- **交互** RESTful JSON API；开发期 Vite 代理联调，生产期由后端同源托管

---

## 目录结构

```
qingye/
├── README.md              # 项目说明
├── Makefile               # 快捷启动命令
├── Dockerfile             # 多阶段镜像（node24 / golang1.27 / alpine3.24）
├── docker-compose.yml     # 一键部署（使用 GHCR 镜像）
├── .github/
│   ├── workflows/docker.yml  # 多架构镜像构建 + 推送 GHCR
│   └── dependabot.yml        # 每日依赖更新检测
├── server/                # Go 后端（Gin + GORM + SQLite）
│   ├── main.go            # 入口：配置 / AutoMigrate / 路由 / 启动 / 天气轮询
│   ├── config/            # 配置加载（端口 / 数据库 / 上传目录 / CORS / WebDir）
│   ├── models/            # GORM 实体（含天气策略 WeatherConfig / WeatherLog）
│   ├── repositories/      # 数据访问层
│   ├── services/          # 业务层（今日任务、完成/推迟、日记、天气、Plantbook）
│   │   ├── weather_*.go    # 天气获取与智能策略调整
│   │   └── plantbook.go    # 在线植物资料库客户端
│   ├── handlers/          # REST 路由处理 + 统一响应
│   ├── router/            # 路由注册 + CORS / 日志 / Recovery + 静态站点托管
│   ├── seed/              # 资料库示例 + 初始设置 + 演示数据
│   ├── uploads/           # 本地照片存储
│   └── .env.example
└── web/                   # SvelteKit 前端（静态 SPA）
    ├── src/lib/           # api 封装 / 类型 / stores / 组件
    ├── src/routes/        # 页面路由（含 +layout.ts 关闭 SSR）
    ├── vite.config.ts     # API 代理
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
- 植物资料库示例（绿萝、龟背竹、虎皮兰等 10 种本地指南）
- 初始设置（默认工作日周一至周五）
- 演示数据（客厅/阳台/卧室 + 4 株植物 + 若干任务 + 2 条日记）

健康检查：`curl http://localhost:8081/healthz` → `{"status":"ok"}`

### 3. 启动前端（开发模式）

```bash
make run-web
# 或手动：
cd web && npm run dev
```

前端默认 `http://localhost:5173`，通过 Vite 代理把 `/api` 与 `/uploads` 转发到后端 8081，无需额外 CORS 配置。打开浏览器访问 `http://localhost:5173` 即可使用。

> 生产环境直接由后端托管前端（见「Docker 部署」），无需单独跑前端。

### 4. 生产构建（可选，手动）

```bash
make build-web          # 输出到 web/build（adapter-static 静态 SPA）
cd server && PORT=8081 CGO_ENABLED=1 go build -o qingye .
```

构建后的 `web/build` 可由后端 `WEB_DIR` 指向并同源托管。

---

## 核心功能

| 模块 | 说明 |
| --- | --- |
| 植物档案 | 增删改查、房间/分组整理、照片与备注 |
| 任务清单 | 浇水/施肥/换盆等可重复任务；完成、推迟、类型筛选、历史记录 |
| 今日概览 | 今日任务卡片、临近提醒、快捷入口；**休息日不展示任务，任务顺延不丢失** |
| 照片日记 | 按植物记录照片与文字，时间线展示成长瞬间 |
| 智能规划 | 用户设置工作日/休息日，系统仅在工作日展示当日任务 |
| 植物资料库 | 内置 10 种本地中文养护指南；配置 Plantbook OAuth2 凭据后可在线匹配并批量同步近 100 种常见室内/露台植物 |
| 天气策略 | 集成和风天气，按实时温度/降雨自动调整浇水施肥频率并记录日志（需 `QWEATHER_KEY`） |
| 批量导入 | CSV 批量导入植物/任务，**先预览后确认**；支持把某株植物的任务模板复制给多株植物 |
| 设置中心 | 工作日选择器、天气策略配置、资料库同步 |

---

## 设计要点

- **重复任务按需计算**：任务以「规则 + 上次完成时间」存储，查询时计算下一次到期，避免海量任务行；仅完成/推迟动作写入 `task_logs`。
- **工作日过滤**：`user_settings.workdays` 以 `1-7`（周一至周日）记录，今日任务仅在「今天为工作日且 next_due ≤ 今天」时展示。
- **照片存储**：SQLite 仅存元数据（路径/说明/时间），图片文件存本地 `server/uploads/`。
- **事务保证**：「完成任务 + 写日志 + 更新 next_due」在同一事务内完成。
- **统一响应**：所有接口返回 `{ code, message, data }`，`code=0` 表示成功。
- **前端静态托管**：生产镜像内前端为静态 SPA，由后端 `WEB_DIR` 指向并同源托管；未知非 API 路由回退 `index.html`（SPA 前端路由）。
- **在线资料库唯一键**：Plantbook 以学名 `pid` 作唯一键，避免中文名歧义；详情带 `lang=zh` 直接取回中文，无需第三方翻译。

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
| GET | `/library?q=` | 本地资料库搜索 |
| GET | `/library/online?keyword=` | 在线搜索候选（Plantbook，未配置 token 返回 `enabled:false`） |
| POST | `/library/import` | 按 `pid` 拉详情并写回本地资料库 |
| POST | `/library/sync-popular` | 批量同步热门植物到本地资料库 |
| GET | `/weather/current` | 当前天气 + 策略状态 |
| GET | `/weather/config` | 读取天气策略配置 |
| PUT | `/weather/config` | 保存天气策略配置 |
| GET | `/weather/logs?limit=50` | 天气调整日志 |
| POST | `/weather/refresh` | 手动触发一次策略调整 |
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
WEB_DIR=               # 前端静态构建目录；留空则只提供 API，不托管页面
CORS_ORIGINS=http://localhost:5173
MAX_UPLOAD_MB=10

# 可选：Plantbook 在线植物库 OAuth2 凭据（open.plantbook.io 注册后在 API keys 生成）
# 两个变量同时配置后启用「在线匹配 / 批量同步」，留空则只用本地资料库
PLANTBOOK_CLIENT_ID=
PLANTBOOK_CLIENT_SECRET=
# 可选：直接使用的 access_token（调试用，一般无需填写）
PLANTBOOK_ACCESS_TOKEN=

# 可选：和风天气 API Key（dev.qweather.com 注册获取）
# 配置后启用天气智能养护策略，留空则天气模块关闭
QWEATHER_KEY=
# 可选：和风天气接口地址（默认 https://devapi.qweather.com/v7/weather/now）
QWEATHER_API=
```

**前端** `web/.env`（参考 `web/.env.example`）：

```
# 留空走 Vite 代理；独立域名部署时填后端地址
VITE_API_BASE=
```

---

## 天气智能养护策略

集成和风天气，基于实时天气数据自动调整养护频率，并记录每次调整日志便于追溯。

- **低温**（低于 `coldTemp`）：自动降低浇水与施肥频率
- **高温**（高于 `hotTemp`）：自动降低施肥频率，保持或适度增加浇水
- **降雨**：自动推迟室外植物的浇水任务 `rainDelayH` 小时
- 调整幅度（`waterAdj` / `fertAdj`，百分比）、温度阈值、降雨推迟时长均可在设置页自定义
- 后台每 `pollMinutes` 分钟轮询一次；未配置 `QWEATHER_KEY` 或策略未启用时自动跳过
- 每次触发写入 `weather_logs`（类型 `cold` / `hot` / `rain` / `refresh`），设置页「策略日志」可查看

> 未配置 `QWEATHER_KEY` 时天气模块关闭，不影响其他功能。

---

## 在线植物资料库（Plantbook）

配置 `PLANTBOOK_CLIENT_ID` 与 `PLANTBOOK_CLIENT_SECRET` 后，可在线扩展资料库，无需人工录入。凭据在 [open.plantbook.io](https://open.plantbook.io) 注册后在「API keys」页生成（OAuth2 client credentials，服务端自动换取并缓存 access_token）。

### 工作原理

- **唯一键用学名 `pid`**：Plantbook 以学名（如 `monstera_deliciosa`）作为稳定唯一键，避免中文别名歧义；同一 `pid` 重复同步自动覆盖刷新。
- **中文直接取回**：详情请求带 `lang=zh`，优先使用 Plantbook 自带的中文常见名，无需第三方翻译。
- **结构化字段映射**：浇水频率 / 光照 / 温度 / 土壤 / 施肥 / 修剪等字段自动拼成中文三段式指南，与内置条目风格一致。
- **本地缓存、离线可用**：首次在线匹配 / 同步的结果会写回本地 `PlantLibrary`，之后添加植物直接本地命中，零网络、零额度。

### 两种用法

1. **录入时在线匹配**（懒加载）：添加植物时填写名称 → 点「在线查找」→ 从候选列表选一条 → 自动带回中文指南并落库。
2. **批量同步热门植物**（离线预置）：设置页「植物资料库同步」→ 点「同步热门植物」，把内置近 100 种常见室内/露台植物的中文指南一次性拉取沉淀到本地（清单见 `server/services/library_service.go` 的 `popularSeeds`）。

> 未配置 Plantbook 凭据时，在线功能优雅降级为「仅本地资料库」，不影响其他功能。

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

## Docker 部署

镜像内已包含前后端，访问 `http://<host>:8081/` 即可，无需单独部署前端或配置 CORS（同源）。

### 方式一：docker compose（推荐）

```bash
docker compose up -d
```

默认拉取 `ghcr.io/felix2yu/qingye:latest`。数据持久化在命名卷 `qingye-data`（SQLite）、`qingye-uploads`（照片）。可通过 `.env` 或环境变量覆盖端口、`PLANTBOOK_CLIENT_ID` / `PLANTBOOK_CLIENT_SECRET`、`QWEATHER_KEY` 等。

### 方式二：手动运行

```bash
docker run -d --name qingye \
  -p 8081:8081 \
  -e PLANTBOOK_CLIENT_ID=你的client_id \
  -e PLANTBOOK_CLIENT_SECRET=你的client_secret \
  -e QWEATHER_KEY=你的key \
  -v qingye-data:/app/data \
  -v qingye-uploads:/app/uploads \
  ghcr.io/felix2yu/qingye:latest
```

> 数据库与上传目录务必挂卷持久化，否则容器重建会丢数据。

---

## CI / 自动化

- **镜像构建**：`.github/workflows/docker.yml` 在 push 到 `main` 或打 `v*` tag 时，用原生架构 runner 矩阵（`ubuntu-latest`=amd64、`ubuntu-24.04-arm`=arm64，不用 qemu 模拟）分别构建，再合并为 `ghcr.io/felix2yu/qingye:latest` 与 `:sha`。`pull_request`（含 Dependabot）只构建验证、不推送镜像。
- **依赖更新**：`.github/dependabot.yml` 每日检测 `gomod` / `npm` / `github-actions` 三类依赖，自动开 PR 并打 `dependencies` 标签。

---

## 重置演示数据

删除 `server/data/qingye.db` 并重启后端即可重新生成全部种子与演示数据（Docker 部署则删除 `qingye-data` 卷后重建容器）。
