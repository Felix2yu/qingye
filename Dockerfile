# 「清野」运行时镜像：二进制与前端产物均由 CI 预编译后拼装。
#
# 编译期依赖（node_modules、Go 工具链、gcc/musl-dev）全部留在 CI，不进镜像。
# 前端产物不打进二进制，运行时由 WEB_DIR 指向 /app/web。

FROM alpine:3.24

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    mkdir -p /app/data /app/uploads /app/web

COPY --chmod=755 bin/qingye /app/qingye
COPY dist /app/web

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
