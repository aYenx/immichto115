<div align="center">

# 🔄 ImmichTo115

**把自托管 [Immich](https://immich.app/) 照片库与数据库备份，稳定同步到 115 网盘。**

[![GitHub Release](https://img.shields.io/github/v/release/aYenx/immichto115?style=flat-square&logo=github&label=Release)](https://github.com/aYenx/immichto115/releases)
[![Docker Image](https://img.shields.io/badge/GHCR-ghcr.io/ayenx/immichto115-blue?style=flat-square&logo=docker)](https://ghcr.io/ayenx/immichto115)
[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?style=flat-square&logo=vuedotjs)](https://vuejs.org/)
[![License](https://img.shields.io/github/license/aYenx/immichto115?style=flat-square)](LICENSE)

Go 后端 + Vue 3 前端，最终编译为**单个二进制文件**，开箱即用，支持 Docker、systemd 与源码部署。

[适用场景](#适用场景) · [接入模式](#接入模式) · [3 分钟上手](#3-分钟上手) · [配置要点](#配置要点) · [运维](#运维) · [开发说明](#开发说明) · [附录](#附录)

</div>

## 适用场景

如果你希望把 Immich 的原始照片库、数据库备份，甚至额外的摄影素材，一并归档到 115 网盘，而且更看重“少折腾、可持续、能观察”，这个项目就是为这类场景准备的。

它尤其适合下面这些需求：

- 把 `library` 和数据库备份目录定期同步到 115 网盘
- 尽量少依赖外部组件，优先通过 Web UI 完成配置
- 需要增量备份、计划任务、实时日志和手机通知
- 想按拍摄日期把 RAW / JPG 文件自动整理上传到 115
- 希望部署方式足够简单，最好 Docker 跑起来就能用

### 核心能力

- **115 Open 直传**：直接使用 `access_token` / `refresh_token`，Open115 模式内置 `manifest.db` 增量索引
- **WebDAV 兼容模式**：适合已有 Rclone / WebDAV 环境的用户继续沿用现有链路
- **定时与增量**：支持 `copy` / `sync` 两种模式，并提供远端删除保护
- **可选加密**：支持 Open115 本地加密上传，或 WebDAV 模式下的 Rclone Crypt
- **可观测性**：支持 WebSocket 实时日志流和 Bark 推送通知
- **单文件部署**：前端通过 `go:embed` 内嵌进 Go 服务，适合 Docker、systemd 和裸机运行

> [!TIP]
> 如果你没有现成的 WebDAV 环境，大多数情况下直接选 `115 Open` 就够了，依赖更少，目录浏览和增量索引也更完整。

## 接入模式

| 对比项 | `115 Open` 模式 | `WebDAV + Rclone` 模式 |
| --- | --- | --- |
| 推荐程度 | ⭐ 推荐 | 适合已有环境 |
| 接入方式 | 115 Open API（`access_token` / `refresh_token`） | `rclone` + WebDAV 协议 |
| 增量索引 | 内置 `manifest.db` SQLite 索引 | 依赖 rclone 自身行为 |
| 加密方式 | 本地加密上传（`temp` / `stream`） | Rclone Crypt |
| 目录浏览 | 直接浏览 115 目录树 | 浏览 WebDAV 目录 |
| 额外依赖 | 无 | 需要 `rclone` |
| 典型用户 | 想快速上手、尽量少配置 | 已有稳定 WebDAV / Rclone 流程 |

## 3 分钟上手

### 推荐路径：Docker Compose

可以直接使用仓库里的 [deploy/docker-compose.yml](deploy/docker-compose.yml) 作为模板，最常见的最小配置如下：

```yaml
services:
  immichto115:
    image: ghcr.io/ayenx/immichto115:latest
    container_name: immichto115
    restart: unless-stopped
    ports:
      - "8096:8096"
    volumes:
      - /你的Immich照片库路径:/data/library:ro
      - /你的Immich数据库备份路径:/data/backups:ro
      - ./config:/app/config
    environment:
      - TZ=Asia/Shanghai
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:8096/api/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
```

启动服务：

```bash
docker compose up -d
```

首次访问：

```text
http://服务器IP:8096
```

首次进入会打开 **Setup Wizard**。按向导填写本地目录、远端目录，并完成 `115 Open` 或 `WebDAV` 接入配置即可。

> [!NOTE]
> `backup.library_dir` 和 `backup.backups_dir` 至少填写一个。上面的 Compose 示例同时挂载了两个目录，实际使用时可按你的 Immich 部署情况调整。

<details>
<summary>其他安装方式</summary>

**Linux / systemd 一键安装**

```bash
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/install.sh | sudo bash
```

安装脚本会自动完成架构检测、二进制下载、可选 Rclone 安装和 systemd 服务注册。

**源码构建**

开发或自定义构建请直接查看下方的[开发说明](#开发说明)。

</details>

## 配置要点

首次访问 Web UI 会进入 **Setup Wizard**，保存后自动生成 `config.yaml`。大多数用户直接用向导即可，只有需要版本管理或批量部署时才建议手写配置。

### 推荐配置：115 Open

下面是一份适合大多数用户的基础示例，重点覆盖最常见的 115 Open 备份场景：

```yaml
provider: open115

server:
  port: 8096

open115:
  enabled: true
  access_token: your_access_token
  refresh_token: your_refresh_token
  root_id: "0"

open115_encrypt:
  enabled: false
  mode: temp
  filename_mode: plain
  algorithm: aes256gcm-v1
  temp_dir: /tmp/immichto115-open115-encrypt
  min_free_space_mb: 1024

backup:
  library_dir: /data/library
  backups_dir: /data/backups
  remote_dir: /immich-backup
  mode: copy
  manifest_path: ./config/manifest.db

cron:
  enabled: false
  expression: "0 2 * * *"

photo_upload:
  enabled: false
  watch_dir: /data/photos
  remote_dir: /摄影
  extensions: cr2,cr3,nef,arw,dng,raf,rw2,orf,pef,srw,jpg,jpeg
  date_format: "2006/01/02"
  delete_after_upload: true
```

<details>
<summary>WebDAV 模式示例</summary>

```yaml
provider: webdav

webdav:
  url: https://dav.example.com
  user: your_user
  password: your_password
  vendor: other

backup:
  library_dir: /data/library
  backups_dir: /data/backups
  remote_dir: /immich-backup
  mode: copy
```

</details>

### 配置项速查

#### 必填项

| 配置项 | 说明 |
| --- | --- |
| `provider` | 接入模式，`open115` 或 `webdav` |
| `backup.remote_dir` | 远端备份根目录 |
| `backup.library_dir` / `backup.backups_dir` | 两者至少填写一个 |
| `open115.access_token` / `open115.refresh_token` | `open115` 模式必填 |
| `webdav.url` / `webdav.user` / `webdav.password` / `webdav.vendor` | `webdav` 模式必填 |

#### 常用项

| 配置项 | 默认值 / 说明 |
| --- | --- |
| `server.port` | 默认 `8096` |
| `open115.root_id` | 默认 `"0"` |
| `backup.mode` | 默认 `copy`，也支持 `sync` |
| `backup.manifest_path` | Open115 模式的本地增量索引库路径 |
| `cron.enabled` / `cron.expression` | 默认关闭，cron 默认值为 `0 2 * * *` |
| `notify.enabled` / `notify.bark_url` | Bark 推送通知 |
| `photo_upload.watch_dir` / `photo_upload.remote_dir` | 摄影文件自动上传目录 |
| `photo_upload.extensions` / `date_format` / `delete_after_upload` | RAW / JPG 扩展名、日期目录格式与上传后清理策略 |

#### 高级与安全项

| 配置项 | 说明 |
| --- | --- |
| `server.auth_enabled` / `server.auth_user` | 是否启用访问保护以及管理员用户名 |
| `server.auth_password_hash` | 管理员密码的 bcrypt 哈希；由 Web UI 自动生成 |
| `server.jwt_secret` | JWT 签名密钥；首次登录后自动生成 |
| `open115_encrypt.*` | Open115 本地加密上传配置 |
| `encrypt.*` | WebDAV 模式下的 Rclone Crypt 配置 |
| `backup.allow_remote_delete` | `sync` 模式下是否允许删除远端多余文件 |
| `backup.sync_delete_grace_period` | 远端删除保护宽限期，默认 `24h` |
| `backup.sync_delete_dry_run` | 仅演练远端删除，不真正执行 |
| `updated_at` | 配置版本号，前端保存时用于乐观锁保护，自动维护 |

### 路径与启动参数

- 配置文件路径优先级：`--config` 参数 > `IMMICHTO115_CONFIG` 环境变量 > `{可执行文件目录}/config/config.yaml`
- 可通过 `--port` 覆盖配置中的监听端口
- 运行 `immichto115 --version` 可查看当前版本号

> [!TIP]
> 在 Web UI 中填写的是管理员明文密码，但保存到 `config.yaml` 时会自动转换为 `server.auth_password_hash`。浏览器登录成功后会使用 JWT Cookie + `X-CSRF-Token` 访问写接口。

> [!WARNING]
> 当 `backup.mode: sync` 且 `backup.allow_remote_delete: true` 时，系统会尝试删除远端存在但本地已删除的文件。建议先开启 `backup.sync_delete_dry_run: true` 演练，再决定是否放开真实删除。

> [!IMPORTANT]
> 建议限制 `config/` 目录访问权限，例如 `chmod 700`，避免敏感配置被其他用户读取。

## 运维

### 日常命令

| 操作 | Docker | Systemd |
| --- | --- | --- |
| 查看日志 | `docker compose logs -f` | `journalctl -u immichto115 -f` |
| 重启服务 | `docker compose restart` | `systemctl restart immichto115` |
| 停止服务 | `docker compose down` | `systemctl stop immichto115` |
| 查看状态 | `docker compose ps` | `systemctl status immichto115` |
| 更新服务 | `docker compose pull && docker compose up -d` | 重新运行 `install.sh` |

<details>
<summary>Linux / systemd 一键脚本</summary>

**安装**

```bash
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/install.sh | sudo bash
```

**更新**

```bash
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/install.sh | sudo bash
```

**卸载**

```bash
# 交互式卸载（默认保留配置目录）
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash

# 卸载并删除配置
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash -s -- --purge

# 非交互式卸载（CI / 自动化）
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash -s -- --yes
```

**常用选项**

- `install.sh --no-rclone`：跳过 Rclone 检查与安装
- `install.sh --force`：强制覆写 systemd 服务文件
- `uninstall.sh --purge`：卸载时同时删除配置目录
- `uninstall.sh --yes`：跳过交互确认

</details>

## 开发说明

开发环境默认采用前后端分离：Go 后端提供 API，Vite 负责前端开发服务器与代理。

### 本地开发

依赖建议：

- Go `1.23.4+`
- Node.js `20+`

先准备一份可用的 `config.yaml`，然后分别启动后端与前端：

```bash
# 终端 1：启动后端
go run ./cmd/server --config ./config/config.yaml --port 8096
```

```bash
# 终端 2：启动前端
cd web
npm ci
npm run dev
```

> Vite 默认会把 `/api` 代理到 `http://localhost:8096`，把 `/ws` 代理到 `ws://localhost:8096`。

### 从源码构建单文件

```bash
git clone https://github.com/aYenx/immichto115.git
cd immichto115

cd web
npm ci
npm run build
cd ..

VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
CGO_ENABLED=0 go build -tags embedfront -ldflags="-s -w -X main.version=${VERSION}" -o immichto115 ./cmd/server

./immichto115 --config ./config/config.yaml --port 8096
```

`go build -tags embedfront` 会把 `web/dist` 打包进最终二进制；如果不带 `embedfront`，后端不会内嵌前端静态资源。

### 前端目录

前端位于 [web/](web/)，常用脚本如下：

```bash
cd web
npm ci
npm run dev
npm run typecheck
npm run build
```

更简短的前端目录说明见 [web/README.md](web/README.md)。

<details>
<summary>Docker 源码构建</summary>

```bash
git clone https://github.com/aYenx/immichto115.git
cd immichto115/deploy
docker compose up -d --build
```

</details>

## 附录

<details>
<summary>API 参考</summary>

| 方法 | 路径 | 说明 | 访问要求 |
| --- | --- | --- | --- |
| `GET` | `/api/health` | 健康检查，返回 `status` / `version` / `checks` | 公开 |
| `POST` | `/api/v1/auth/login` | 管理员登录，签发 JWT Cookie 并返回 `csrf_token` | 公开 |
| `POST` | `/api/v1/auth/logout` | 清理当前登录态 | 已登录 |
| `GET` | `/api/v1/auth/csrf` | 获取当前会话的 CSRF Token | 已登录 |
| `GET` | `/api/v1/ping` | 连通测试 | 已登录 |
| `GET` | `/api/v1/system/status` | 系统状态（provider / Rclone / 备份状态 / build 信息） | 已登录 |
| `GET` | `/api/v1/config` | 获取配置安全视图（敏感字段已脱敏） | 已登录 |
| `POST` | `/api/v1/config` | 保存配置，返回新的 `updated_at` | 已登录 |
| `POST` | `/api/v1/webdav/test` | 测试 WebDAV 连接 | 已登录 |
| `POST` | `/api/v1/webdav/ls` | 浏览 WebDAV 目录 | 已登录 |
| `POST` | `/api/v1/open115/auth/start` | 启动 115 Open 扫码授权 | 已登录 |
| `GET` | `/api/v1/open115/auth/status` | 查询 115 Open 扫码状态 | 已登录 |
| `POST` | `/api/v1/open115/auth/finish` | 完成扫码授权并保存 token | 已登录 |
| `POST` | `/api/v1/open115/test` | 测试 115 Open token 可用性 | 已登录 |
| `POST` | `/api/v1/open115/ls` | 浏览 115 Open 目录 | 已登录 |
| `POST` | `/api/v1/open115/debug/stream-measure` | 调试 `stream` 模式的本地加密测量 | 已登录 |
| `POST` | `/api/v1/open115/debug/stream-upload` | 调试 `stream` 模式的单文件上传 | 已登录 |
| `POST` | `/api/v1/backup/start` | 手动触发备份 | 已登录 |
| `POST` | `/api/v1/backup/stop` | 停止备份 | 已登录 |
| `POST` | `/api/v1/photo-upload/start` | 开始摄影文件上传 | 已登录 |
| `POST` | `/api/v1/photo-upload/stop` | 停止摄影文件上传 | 已登录 |
| `GET` | `/api/v1/photo-upload/status` | 查询摄影上传状态 | 已登录 |
| `GET` | `/api/v1/remote/ls` | 浏览云端文件（Restore Explorer） | 已登录 |
| `GET` | `/api/v1/local/ls` | 浏览本地目录 | 已登录 |
| `POST` | `/api/v1/notify/test` | 测试 Bark 推送通知 | 已登录 |
| `WS` | `/ws/logs` | 实时备份日志流 | 已登录 |

> 浏览器登录后采用 JWT Session Cookie；所有写操作在 JWT 模式下还需要携带 `X-CSRF-Token`。命令行或其他 API 客户端仍可用 HTTP Basic Auth 作为回退方式。

</details>

<details>
<summary>项目结构</summary>

```text
immichto115/
├── cmd/server/              # Go 服务入口
├── internal/
│   ├── api/                 # Gin 路由、认证、WebSocket、备份控制接口
│   ├── backup/              # 备份后端抽象（WebDAV / Open115）
│   ├── config/              # 配置结构、默认值、DTO、安全视图
│   ├── cron/                # 定时任务调度
│   ├── manifest/            # Open115 增量索引（SQLite）
│   ├── notify/              # Bark 推送
│   ├── open115/             # 115 Open 客户端与上传逻辑
│   ├── open115crypt/        # Open115 本地加密
│   ├── photoupload/         # 摄影文件扫描与上传
│   └── rclone/              # Rclone CLI 封装
├── web/                     # Vue 3 + Vite + TypeScript 前端
├── deploy/                  # Docker 与 systemd 部署脚本
├── web_embed.go             # 非 embedfront 构建入口
├── web_embed_prod.go        # embedfront 构建入口
└── .github/workflows/       # CI / Release 工作流
```

</details>

<details>
<summary>自检与发布</summary>

本地自检命令：

```bash
go vet ./...
go test ./... -race -count=1

cd web
npm ci
npm run typecheck
npm run build
cd ..
```

打标签发布：

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

</details>

## License

[MIT](LICENSE)

如果这个项目对你有帮助，欢迎 Star 支持。
