# qingye —— 目标约定见 Felix2yu/.github/docs/makefile.md
#
# 七个必备目标：build / dev / test / lint / fmt / docker / clean
# 保留的自有目标：install-server / install-web / run-server / run-web / build-web / seed / stop
.PHONY: build dev test lint lint-strict fmt docker clean \
        run-server run-web seed install-server install-web build-web stop help

BIN  := bin/qingye
MAIN := ./cmd/qingye
# 默认 dev；发布时用 `make build VERSION=v1.2.3` 覆盖
VERSION ?= dev

# 默认端口：后端 8081（8080 常被其他应用占用），前端 5173
SERVER_PORT ?= 8081
WEB_PORT ?= 5173

help: ## 查看可用命令
	@echo "青野集 QingYe 开发命令："
	@echo "  make build           前端 → web/build → dist/，再编译后端到 bin/qingye"
	@echo "  make dev             起后端（前端另开终端 make run-web）"
	@echo "  make test            跑 Go 与前端单测"
	@echo "  make lint            gofmt 检查 + go vet + svelte-check"
	@echo "  make docker          构建镜像（装配式，先出二进制与 dist/）"
	@echo "  make install-server  安装后端依赖"
	@echo "  make install-web     安装前端依赖"
	@echo "  make run-server      启动后端 (http://localhost:$(SERVER_PORT))"
	@echo "  make run-web         启动前端 (http://localhost:$(WEB_PORT))"
	@echo "  make build-web       只构建前端"
	@echo "  make stop            停止后端 / 前端进程"

# 前端产物落 web/build，再由 Dockerfile 从根 dist/ COPY 进镜像
build: build-web
	rm -rf dist
	mkdir -p dist
	cp -R web/build/. dist/
	mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN) $(MAIN)

# 后端单独起；前端热更新另开终端 make run-web
dev: run-server

test:
	go test ./...
	cd web && pnpm run test

# gofmt -l 只列出未格式化文件、不写盘，属于检查而非格式化
lint:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "未格式化："; echo "$$out"; exit 1; fi
	go vet ./...
	cd web && pnpm run check

# 深度检查：在 lint 之上加 golangci-lint（存量告警未清零前会红，故不并入 lint）
lint-strict: lint
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "  （未安装 golangci-lint，跳过）"; fi

fmt:
	gofmt -w $$(git ls-files '*.go')

# 装配式镜像：先出 bin/qingye 与 dist/，镜像只负责 COPY
docker: build
	docker build -t qingye:dev .

clean:
	rm -rf bin dist web/build web/.svelte-kit coverage.out

install-server:
	go mod download

install-web:
	cd web && pnpm install --frozen-lockfile

run-server:
	PORT=$(SERVER_PORT) go run $(MAIN)

run-web:
	cd web && pnpm run dev -- --port $(WEB_PORT)

build-web:
	cd web && pnpm install --frozen-lockfile && pnpm run build

stop:
	@-pkill -f "exe/qingye" 2>/dev/null; pkill -f "cmd/qingye" 2>/dev/null; pkill -f "vite" 2>/dev/null; echo "已尝试停止相关进程"

seed:
	@echo "种子数据在后端启动时自动写入；如需重置，删除 data/qingye.db 后重启即可"
