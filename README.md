<div align="center">

# 🔄 ImmichTo115

**将自托管 [Immich](https://immich.app/) 照片库 + 数据库备份一键同步到 115 网盘**

[![GitHub Release](https://img.shields.io/github/v/release/aYenx/immichto115?style=flat-square&logo=github&label=Release)](https://github.com/aYenx/immichto115/releases)
[![Docker Image](https://img.shields.io/badge/GHCR-ghcr.io/ayenx/immichto115-blue?style=flat-square&logo=docker)](https://ghcr.io/ayenx/immichto115)
[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?style=flat-square&logo=vuedotjs)](https://vuejs.org/)
[![License](https://img.shields.io/github/license/aYenx/immichto115?style=flat-square)](LICENSE)

Go 后端 + Vue 3 前端，编译为**单个二进制文件**，开箱即用。

---

[功能特性](#-功能特性) · [快速开始](#-快速开始) · [配置说明](#️-配置说明) · [运维手册](#-运维手册) · [API 文档](#-api-文档) · [项目结构](#️-项目结构)

</div>

---

## ✨ 功能特性

### 适合谁用？

如果你想把自托管 Immich 的照片库和数据库备份到 115，这个项目提供两条路径：

- **最省心**：`115 Open` — 填入 Token 即可直接上传，无额外依赖
- **更传统**：`WebDAV + Rclone` — 适合已有 WebDAV 环境的用户

> [!TIP]
> 大多数用户直接选 115 Open 就行。在界面中点击"获取 Token（OpenList）"，拿到 Token 后直接填写即可。

### 核心能力

| | 功能 | 说明 |
| :-: | --- | --- |
| ☁️ | **115 Open 直传** | Token 授权即用，支持大文件 multipart 上传 + manifest.db 增量索引 |
| 📷 | **摄影文件上传** | 扫描本地 RAW + JPG，按 EXIF 拍摄日期自动分类上传到 115 网盘 |
| 🔐 | **端到端加密** | Open115 本地加密（temp / stream）或 WebDAV Rclone Crypt |
| ⏰ | **定时 + 增量备份** | 可视化 Cron 调度 + `copy`/`sync` 两种模式，`sync` 支持远端删除保护 |
| 🧙 | **Setup Wizard** | 4 步引导式配置，WebDAV / 115 Open 双模式 |
| 📡 | **实时可观测** | WebSocket 日志推送 + Bark 通知到手机 |
| 🛡️ | **访问保护** | 管理员登录后签发 JWT Session + CSRF 保护，API 客户端兼容 Basic Auth 回退 |
| 📦 | **单文件部署** | `go:embed` 内嵌前端，支持 `amd64` / `arm64`，Docker / systemd 一键启动 |

---

## 🚀 快速开始

### 接入方式对比

| 对比项   | WebDAV 模式                | 115 Open 模式 ⭐ 推荐                          |
| -------- | -------------------------- | ---------------------------------------------- |
| 接入方式 | `rclone` + WebDAV 协议     | 115 Open API（`access_token / refresh_token`） |
| 增量索引 | 依赖 rclone 本身           | 内置 `manifest.db` SQLite 索引                 |
| 加密     | Rclone Crypt               | 本地加密上传（`temp` / `stream`）              |
| 目录浏览 | WebDAV 目录                | 直接浏览 115 目录树                            |
| 依赖     | 需安装 rclone              | 无额外依赖                                     |



### 方式一：Docker Compose（推荐）

```yaml
# docker-compose.yml
services:
  immichto115:
    image: ghcr.io/ayenx/immichto115:latest
    container_name: immichto115
    restart: unless-stopped
    ports:
      - "8096:8096"
    volumes:
      # ⬇️ 【必须修改】替换为你的 Immich 实际数据目录
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

```bash
docker compose up -d
```

> 访问 `http://服务器IP:8096`，首次进入 Setup Wizard 完成配置。

---

### 方式二：一键安装（Linux / systemd）

```bash
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/install.sh | sudo bash
```

自动完成：检测架构 → 安装 Rclone → 下载二进制（SHA256 校验）→ 创建 systemd 服务 → 启动。

<details>
<summary>💡 安装脚本命令行选项</summary>

```bash
sudo bash install.sh [选项]

选项:
  --no-rclone    跳过 Rclone 检查与安装（适用于已使用 Open115 的用户）
  --force        强制覆写 systemd 服务文件（默认更新时保留）
  --help         显示帮助信息

环境变量:
  RELEASE_URL    自定义下载地址前缀
                 示例: RELEASE_URL=https://mirror.example.com/releases/latest/download bash install.sh
```

</details>

<details>
<summary>💡 更新 / 卸载</summary>

```bash
# 更新（重新运行安装脚本，自动保留 config.yaml 和 systemd 服务配置）
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/install.sh | sudo bash

# 卸载（交互式，默认保留配置目录）
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash

# 卸载并删除配置
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash -s -- --purge

# 非交互式卸载（CI / 自动化）
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash -s -- --yes
```

</details>

---

### 方式三：从源码构建

<details>
<summary>展开查看</summary>

```bash
# 依赖建议：Go 1.23.4+、Node.js 20+

# 克隆仓库
git clone https://github.com/aYenx/immichto115.git
cd immichto115

# 编译前端
cd web && npm ci --include=dev && npm run build && cd ..

# 编译后端（go:embed 内嵌前端资源，注入版本号）
VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
CGO_ENABLED=0 go build -tags embedfront -ldflags="-s -w -X main.version=${VERSION}" -o immichto115 ./cmd/server/

# 运行
./immichto115 --config /path/to/config.yaml --port 8096
```

</details>

### 本地开发（前后端分离）

```bash
# 终端 1：启动后端
go run ./cmd/server --config ./config/config.yaml --port 8096

# 终端 2：启动前端开发服务器
cd web
npm ci
npm run dev
```

> Vite 默认将 `/api` 和 `/ws` 代理到 `http://localhost:8096`。

### 方式四：Docker 源码构建

<details>
<summary>展开查看</summary>

```bash
git clone https://github.com/aYenx/immichto115.git
cd immichto115/deploy
# 编辑 docker-compose.yml 修改 volumes 路径
docker compose up -d --build
```

</details>

---

## ⚙️ 配置说明

首次访问 Web UI 会进入 **Setup Wizard**，配置完成后自动生成 `config.yaml`。

### 配置示例

<details>
<summary>📋 方案 A：WebDAV</summary>

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

<details open>
<summary>📋 方案 B：115 Open（推荐）</summary>

```yaml
provider: open115

open115:
  enabled: true
  client_id: ""              # 仅在项目内扫码授权时需要
  access_token: your_access_token
  refresh_token: your_refresh_token
  root_id: "0"
  token_expires_at: 0
  user_id: ""

open115_encrypt:
  enabled: false
  password: ""
  salt: ""
  mode: temp # temp | stream
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
  allow_remote_delete: false
  sync_delete_grace_period: 24h
  sync_delete_dry_run: false

# 摄影文件上传（可选）
photo_upload:
  enabled: true
  watch_dir: /data/photos       # 本地摄影文件目录
  remote_dir: /摄影              # 115 网盘目标目录
  extensions: cr2,cr3,nef,arw,dng,raf,rw2,orf,pef,srw,jpg,jpeg
  date_format: "2006/01/02"     # 远端子目录结构: 年/月/日
  delete_after_upload: true     # 上传成功后删除本地文件
```

</details>

> [!TIP]
> 访问保护在界面中填写的是管理员明文密码，但落盘到 `config.yaml` 时会自动转换为 `server.auth_password_hash`。浏览器登录成功后会使用 JWT Cookie + `X-CSRF-Token` 访问写接口，`server.jwt_secret` 会在首次登录时自动生成。

### 配置项速查

| 配置项 | 说明 | 必填 |
| ------ | ---- | :--: |
| `provider` | `webdav` 或 `open115` | ✅ |
| `server.port` | Web 服务监听端口，默认 `8096` | ⬜ |
| `server.auth_enabled` | 是否启用访问保护 | ⬜ |
| `server.auth_user` | 管理员用户名 | 启用访问保护时必填 |
| `server.auth_password_hash` | 管理员密码的 bcrypt 哈希；由 Web UI 自动生成 | 启用访问保护时必填 |
| `server.jwt_secret` | JWT 签名密钥；首次登录后自动生成 | ⬜ |
| `webdav.*` | WebDAV URL / 用户名 / 密码 / vendor | WebDAV 模式必填 |
| `open115.client_id` | 项目内扫码授权时使用的 Client ID | ⬜ |
| `open115.access_token` / `refresh_token` | 115 Open 模式 Token | Open115 模式必填 |
| `open115.root_id` | 115 Open 根目录 ID，默认 `"0"` | ⬜ |
| `open115.token_expires_at` / `user_id` | 当前授权状态信息，扫码授权后自动写入 | ⬜ |
| `open115_encrypt.enabled` | 是否启用 Open115 本地加密上传 | ⬜ |
| `open115_encrypt.mode` | 加密模式：`temp` 或 `stream` | 启用时建议设置 |
| `open115_encrypt.filename_mode` | 文件名处理模式，当前默认 `plain` | ⬜ |
| `open115_encrypt.password` / `salt` | Open115 本地加密参数 | 启用时必填 |
| `open115_encrypt.temp_dir` | `temp` 模式的临时目录 | ⬜ |
| `open115_encrypt.min_free_space_mb` | 临时目录要求的最小剩余空间 | ⬜ |
| `backup.library_dir` | Immich 照片库存储目录 | 与 `backup.backups_dir` 至少填写一个 |
| `backup.backups_dir` | Immich 数据库备份目录 | 与 `backup.library_dir` 至少填写一个 |
| `backup.remote_dir` | 远端备份根目录 | ✅ |
| `backup.mode` | `copy`（增量，默认）或 `sync`（镜像同步） | ⬜ |
| `backup.manifest_path` | Open115 模式下本地增量索引库路径 | ⬜ |
| `backup.allow_remote_delete` | `sync` 模式下是否允许删除远端多余文件 | ⬜ |
| `backup.sync_delete_grace_period` | `sync` 删除保护宽限期，默认 `24h` | ⬜ |
| `backup.sync_delete_dry_run` | 仅演练远端删除，不真正执行 | ⬜ |
| `cron.enabled` / `cron.expression` | 是否开启定时备份与 cron 表达式 | ⬜ |
| `encrypt.enabled` / `encrypt.password` / `encrypt.salt` | WebDAV 模式下 Rclone Crypt 配置 | 启用时必填 |
| `notify.enabled` / `notify.bark_url` | Bark 推送配置 | 启用时必填 |
| `photo_upload.watch_dir` | 本地摄影文件目录 | 启用摄影上传时必填 |
| `photo_upload.remote_dir` | 115 网盘目标目录 | 启用摄影上传时必填 |
| `photo_upload.extensions` | 监控的文件扩展名（逗号分隔） | ⬜ |
| `photo_upload.date_format` | 远端日期子目录格式（Go time 格式） | ⬜ |
| `photo_upload.delete_after_upload` | 上传成功后是否删除本地文件 | ⬜ |
| `updated_at` | 配置版本号，用于前端保存时的乐观锁 | 自动维护 |

> [!WARNING]
> `sync` 模式下如果开启 `allow_remote_delete: true`，系统会尝试删除远端存在但本地已删除的文件。建议先配合 `sync_delete_dry_run: true` 演练，再根据需要调短或关闭 `sync_delete_grace_period`。

> [!IMPORTANT]
> 建议限制 `config/` 目录访问权限（`chmod 700`），避免敏感配置被其他用户读取。

> 配置文件路径优先级：`--config` 参数 > `IMMICHTO115_CONFIG` 环境变量 > `{可执行文件目录}/config/config.yaml`
>
> 可通过 `--port` 参数覆盖配置中的监听端口。运行 `immichto115 --version` 可查看当前版本号。

---

## 🔧 运维手册

### 日常操作

| 操作     | Docker                                             | Systemd（一键安装）                    |
| -------- | -------------------------------------------------- | -------------------------------------- |
| 查看日志 | `docker compose logs -f`                           | `journalctl -u immichto115 -f`        |
| 重启服务 | `docker compose restart`                           | `systemctl restart immichto115`        |
| 停止服务 | `docker compose down`                              | `systemctl stop immichto115`           |
| 查看状态 | `docker compose ps`                                | `systemctl status immichto115`         |
| 更新     | `docker compose pull && docker compose up -d`      | 重新运行 `install.sh`                 |

### 卸载

**Docker**

```bash
docker compose down --rmi all
```

**Systemd（一键安装）**

```bash
# 交互式卸载（默认保留配置目录）
sudo bash deploy/uninstall.sh

# 卸载并清除配置
sudo bash deploy/uninstall.sh --purge

# 非交互式（自动化 / CI）
sudo bash deploy/uninstall.sh --yes --purge
```

> 卸载不会影响 115 网盘上已备份的文件。

---

## 📋 API 文档

<details>
<summary>📡 完整 API 列表</summary>

| 方法 | 路径 | 说明 | 访问要求 |
| :--: | ---- | ---- | :------: |
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

> 开启访问保护后，浏览器先通过 `/api/v1/auth/login` 获取 JWT Cookie；所有写操作在 JWT 模式下还需要携带 `X-CSRF-Token`。命令行或外部 API 客户端仍可使用 HTTP Basic Auth 作为回退方式。

</details>

---

## 🏗️ 项目结构

```
immichto115/
├── cmd/server/              # Go 入口（main.go）
├── internal/
│   ├── api/                 # Gin 路由 + WebSocket Hub + JWT/CSRF + 登录限流
│   ├── backup/              # 备份后端抽象 (WebDAV / Open115)
│   ├── config/              # 配置管理 + DTO 安全视图 + 乐观锁 + rclone.conf 生成
│   ├── cron/                # 定时任务调度 (robfig/cron)
│   ├── manifest/            # Open115 增量索引 (SQLite)
│   ├── notify/              # Bark 推送通知
│   ├── open115/             # 115 Open Client: 授权 / 上传 / 目录 / 删除
│   ├── open115crypt/        # Open115 本地加密 (AES-256-GCM)
│   ├── photoupload/         # 摄影文件扫描 + EXIF 日期提取 + 上传
│   └── rclone/              # Rclone CLI 封装 (os/exec)
├── web/                     # Vue 3 + Vite + TypeScript 前端
│   └── src/
│       ├── views/           # Dashboard · Wizard · Settings · PhotoUpload · RestoreExplorer
│       ├── components/      # Layout · CronScheduler · GlobalToast
│       ├── composables/     # 目录选择器 / Open115 授权 / Toast 等逻辑复用
│       ├── api.ts           # 类型化 API 客户端
│       └── style.css        # 全局样式 (CSS Variables + Dark Mode)
├── web_embed.go             # go:embed 前端资源入口
├── deploy/
│   ├── Dockerfile           # 多阶段构建 (amd64 / arm64)
│   ├── docker-compose.yml   # Docker Compose 参考配置
│   ├── common.sh            # 部署脚本公共工具库
│   ├── install.sh           # Linux 一键安装 / 更新
│   └── uninstall.sh         # 卸载脚本
└── .github/workflows/
    ├── ci.yml               # vet + typecheck + race tests + build smoke test
    └── release.yml          # tag 构建 + Docker 多架构镜像 + GitHub Release
```

---

## 📦 技术栈

| 层         | 技术                                                                                                 |
| ---------- | ---------------------------------------------------------------------------------------------------- |
| **后端**   | Go 1.23 · Gin · Viper · gorilla/websocket · robfig/cron · modernc.org/sqlite                        |
| **前端**   | Vue 3 · Vite · TypeScript · Vue Router · Lucide Icons · CSS Variables (Dark Mode)                    |
| **认证**   | bcrypt 密码哈希 · JWT Session Cookie · CSRF 保护 · 登录限流                                          |
| **备份**   | WebDAV 模式：Rclone CLI / Open115 模式：115 Open API + manifest 增量索引 + AES-256-GCM 本地加密     |
| **构建**   | go:embed 内嵌前端 · 多阶段 Docker · GitHub Actions CI/CD                                            |

---

## 🏷️ 发布

```bash
# 本地自检（与 CI 基本对齐）
go vet ./...
go test ./... -race -count=1

cd web
npm ci
npx vue-tsc --noEmit
npm run build
cd ..

# 打 tag 触发 CI
git tag vX.Y.Z
git push origin vX.Y.Z
```


## 📄 License

[MIT](LICENSE)

**如果这个项目对你有帮助，欢迎 ⭐️ Star 支持！**

</div>
