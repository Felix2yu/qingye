# ---- 阶段 1：构建前端（SvelteKit 静态 SPA） ----
FROM node:26-alpine AS web-build
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---- 阶段 2：构建后端（Go 二进制） ----
FROM golang:1.27-alpine AS server-build
WORKDIR /server
# 启用 CGO 以使用 mattn/go-sqlite3（SQLite 驱动）
RUN apk add --no-cache gcc musl-dev
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /out/qingye .

# ---- 阶段 3：运行镜像 ----
FROM alpine:3.24
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /app/data /app/uploads /app/web

# 后端二进制
COPY --from=server-build /out/qingye /app/qingye
# 前端静态产物
COPY --from=web-build /web/build /app/web

ENV PORT=8081 \
    DB_PATH=/app/data/qingye.db \
    UPLOAD_DIR=/app/uploads \
    WEB_DIR=/app/web \
    CORS_ORIGINS=* \
    GIN_MODE=release

EXPOSE 8081
VOLUME ["/app/data", "/app/uploads"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8081/healthz || exit 1

ENTRYPOINT ["/app/qingye"]
