.PHONY: run-server run-web seed migrate install-server install-web build-web dev stop help

# 默认端口：后端 8081（8080 常被其他应用占用），前端 5173
SERVER_PORT ?= 8081
WEB_PORT ?= 5173

help: ## 查看可用命令
	@echo "青野 QingYe 开发命令："
	@echo "  make install-server  安装后端依赖"
	@echo "  make install-web     安装前端依赖"
	@echo "  make run-server      启动后端 (http://localhost:8081)"
	@echo "  make run-web         启动前端 (http://localhost:5173)"
	@echo "  make dev             同时启动前后端（需终端多路复用工具）"
	@echo "  make stop            停止后端 / 前端进程"
	@echo "  make seed            重新执行种子数据（清空数据库后由 main 自动重建）"
	@echo "  make build-web       构建前端生产包"

install-server:
	cd server && go mod download

install-web:
	cd web && npm install

run-server:
	cd server && PORT=$(SERVER_PORT) go run .

run-web:
	cd web && npm run dev -- --port $(WEB_PORT)

build-web:
	cd web && npm run build

stop:
	@-pkill -f "exe/server" 2>/dev/null; pkill -f "qingye/server" 2>/dev/null; pkill -f "vite" 2>/dev/null; echo "已尝试停止相关进程"

seed:
	@echo "种子数据在后端启动时自动写入；如需重置，删除 server/data/qingye.db 后重启即可"
