# ============================================================
# Stage 1: 编译前端 (Vue 3 + Vite)
#   产物输出到 /build/cmd/server/dist，供 Stage 2 嵌入
# ============================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build/web
COPY web/package.json web/package-lock.json* ./
RUN npm install --frozen-lockfile || npm install

COPY web/ ./
RUN npm run build

# ============================================================
# Stage 2: 编译后端 (Go + go:embed)
#   将 Stage 1 的前端产物嵌入 Go 二进制
# ============================================================
FROM golang:1.22-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .

# 将前端编译产物放到 Go embed 目录
COPY --from=frontend-builder /build/cmd/server/dist ./cmd/server/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /immichto115 ./cmd/server/

# ============================================================
# Stage 3: 最终运行镜像 (Alpine + Rclone)
#   锁定 Rclone 版本以保证可复现性
# ============================================================
FROM alpine:3.20

# 锁定 Rclone 版本 — 升级时只需修改此变量
ARG RCLONE_VERSION=1.68.2

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    unzip \
    fuse3 \
    && curl -fsSL -O "https://downloads.rclone.org/v${RCLONE_VERSION}/rclone-v${RCLONE_VERSION}-linux-amd64.zip" \
    && unzip "rclone-v${RCLONE_VERSION}-linux-amd64.zip" \
    && cp "rclone-v${RCLONE_VERSION}-linux-amd64/rclone" /usr/local/bin/ \
    && chmod +x /usr/local/bin/rclone \
    && rm -rf rclone-* \
    && rclone version

# 创建应用目录和数据挂载点
RUN mkdir -p /app/config /data/library /data/backups

WORKDIR /app

COPY --from=backend-builder /immichto115 /app/immichto115

ENV TZ=Asia/Shanghai
ENV IMMICHTO115_CONFIG=/app/config/config.yaml

EXPOSE 8096

ENTRYPOINT ["/app/immichto115"]
