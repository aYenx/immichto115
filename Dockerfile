# ============================================================
# Stage 1: Build Frontend (Vue 3)
# ============================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build/web
COPY web/package.json web/package-lock.json* ./
RUN npm install --frozen-lockfile || npm install

COPY web/ ./
RUN npm run build

# ============================================================
# Stage 2: Build Backend (Go)
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
# Stage 3: Final Runtime Image
# ============================================================
FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    fuse3 \
    && curl -O https://downloads.rclone.org/current/rclone-current-linux-amd64.zip \
    && unzip rclone-current-linux-amd64.zip \
    && cp rclone-*-linux-amd64/rclone /usr/local/bin/ \
    && chmod +x /usr/local/bin/rclone \
    && rm -rf rclone-* \
    && rclone version

# 创建应用目录
RUN mkdir -p /app/config /data/library /data/backups

WORKDIR /app

COPY --from=backend-builder /immichto115 /app/immichto115

ENV TZ=Asia/Shanghai
ENV IMMICHTO115_CONFIG=/app/config/config.yaml

EXPOSE 8096

ENTRYPOINT ["/app/immichto115"]
