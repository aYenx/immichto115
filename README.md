# ImmichTo115

将自托管 [Immich](https://immich.app/) 照片库数据（照片库 + 数据库备份）通过 WebDAV 协议备份到 115 网盘。

Go 后端 + Vue 3 前端，编译为单个二进制文件，开箱即用。

## ✨ 功能

- **Setup Wizard** — 4 步引导配置 WebDAV 连接、备份路径、加密和定时任务
- **实时日志** — 通过 WebSocket 实时查看 Rclone 备份输出
- **定时备份** — 支持 Cron 表达式，自动执行定期备份
- **加密传输** — 可选 Rclone Crypt 加密，数据在云端加密存储
- **访问保护** — 可选启用管理员账号密码，保护 Web UI、API 与日志流
- **Restore Explorer** — 浏览云端已备份文件，支持透明解密查看
- **单文件部署** — 前端资源内嵌到 Go 二进制，无需额外依赖

## 📦 技术栈

| 层 | 技术 |
|---|------|
| 后端 | Go 1.22 · Gin · Viper · gorilla/websocket · robfig/cron |
| 前端 | Vue 3 · Naive UI · Tailwind CSS v4 · Vue Router |
| 备份引擎 | Rclone CLI（通过 os/exec 调用） |
| 构建 | go:embed 内嵌前端 · 多阶段 Docker 构建 |

## 🚀 部署

### Docker（推荐）

```bash
git clone https://github.com/aYenx/immichto115.git
cd immichto115
```

编辑 `deploy/docker-compose.yml`，修改 Immich 数据路径：

```yaml
volumes:
  - /你的Immich照片库路径:/data/library:ro
  - /你的Immich数据库备份路径:/data/backups:ro
```

```bash
cd deploy
docker compose up -d
```

访问 `http://服务器IP:8096`

### 一键安装（Linux）

需要先 [发布 Release](#发布版本)，然后：

```bash
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/install.sh | sudo bash
```

自动完成：安装 Rclone → 下载二进制 → 创建 systemd 服务 → 启动。

支持自定义下载源：

```bash
RELEASE_URL=https://your-mirror.com/releases/latest/download sudo bash install.sh
```

### 手动编译

```bash
# 前端
cd web && npm install && npm run build && cd ..

# 将前端产物复制到 Go 内嵌目录
rm -rf cmd/server/dist && cp -r web/dist cmd/server/dist

# 后端（内嵌前端资源）
CGO_ENABLED=0 go build -ldflags="-s -w" -o immichto115 ./cmd/server/

# 运行
./immichto115 --config /path/to/config.yaml --port 8096
```

## ⚙️ 配置

配置文件路径优先级：`--config` 参数 > `IMMICHTO115_CONFIG` 环境变量 > `{可执行文件目录}/config/config.yaml`

首次访问 Web UI 会进入 Setup Wizard，配置完成后自动生成 `config.yaml`。

| 配置项 | 说明 |
|--------|------|
| WebDAV URL | 115 网盘 WebDAV 地址 |
| WebDAV 用户名/密码 | 登录凭据（会写入本地配置文件，并在运行时生成临时 `rclone.conf`） |
| 照片库路径 | Immich 照片存储目录 |
| 数据库备份路径 | Immich DB dump 目录 |
| Cron 表达式 | 定时备份周期（5 段标准格式，如 `0 3 * * *`） |
| 加密 | 可选，启用后使用 Rclone Crypt 加密 |
| 管理员账号密码 | 可选，启用后通过 HTTP Basic Auth 保护界面与 API |

> 建议限制 `config/` 目录访问权限，避免敏感配置被其他用户读取。

## 🔧 运维

```bash
# Docker
docker compose -f deploy/docker-compose.yml logs -f    # 查看日志
docker compose -f deploy/docker-compose.yml restart     # 重启
docker compose -f deploy/docker-compose.yml down        # 停止

# Systemd（一键安装）
systemctl status immichto115     # 状态
systemctl restart immichto115    # 重启
journalctl -u immichto115 -f    # 日志
```

## 🗑️ 卸载

### Docker

```bash
cd immichto115/deploy
docker compose down              # 停止并删除容器
docker compose down --rmi all    # 连镜像一起删除
```

### 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash
```

> 卸载不会影响 115 网盘上已备份的文件。

## 📋 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/v1/ping` | 连通测试 |
| GET | `/api/v1/system/status` | 系统状态（Rclone 版本、备份状态、Cron） |
| GET | `/api/v1/config` | 获取配置（敏感信息已脱敏） |
| POST | `/api/v1/config` | 保存配置 |
| POST | `/api/v1/webdav/test` | 测试 WebDAV 连接 |
| POST | `/api/v1/backup/start` | 手动触发备份 |
| POST | `/api/v1/backup/stop` | 停止备份 |
| GET | `/api/v1/remote/ls` | 浏览云端文件 |
| WS | `/ws/logs` | 实时备份日志流 |

> 开启访问保护后，除 `/api/health` 外，其余 Web UI、API 和 WebSocket 都需要管理员账号密码。

## 🏗️ 项目结构

```
immichto115/
├── cmd/server/          # Go 入口 + go:embed
├── internal/
│   ├── api/             # Gin 路由 + WebSocket Hub
│   ├── config/          # Viper 配置 + rclone.conf 生成
│   ├── cron/            # 定时任务调度
│   └── rclone/          # Rclone CLI 封装（os/exec）
├── web/                 # Vue 3 前端
│   └── src/
│       ├── views/       # Dashboard / Setup / Restore
│       ├── components/  # AppLayout
│       ├── composables/ # useWebSocket
│       └── api/         # 类型化 API 客户端
├── deploy/
│   ├── Dockerfile       # 多阶段构建
│   ├── docker-compose.yml
│   ├── install.sh       # Linux 一键安装
│   └── uninstall.sh     # 卸载脚本
└── .github/workflows/   # CI 自动构建 Release
```

## 发布版本

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions 自动构建 `linux-amd64` / `linux-arm64` 二进制并发布到 [Releases](https://github.com/aYenx/immichto115/releases)。

## 📄 License

MIT
