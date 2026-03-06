<div align="center">

# 🔄 ImmichTo115

**将自托管 [Immich](https://immich.app/) 照片库 + 数据库备份一键同步到 115 网盘**

[![GitHub Release](https://img.shields.io/github/v/release/aYenx/immichto115?style=flat-square&logo=github&label=Release)](https://github.com/aYenx/immichto115/releases)
[![Docker Image](https://img.shields.io/badge/GHCR-ghcr.io/ayenx/immichto115-blue?style=flat-square&logo=docker)](https://ghcr.io/ayenx/immichto115)
[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?style=flat-square&logo=vuedotjs)](https://vuejs.org/)
[![License](https://img.shields.io/github/license/aYenx/immichto115?style=flat-square)](LICENSE)

Go 后端 + Vue 3 前端，编译为**单个二进制文件**，开箱即用。

---

[快速开始](#-快速开始) · [功能特性](#-功能特性) · [配置说明](#️-配置说明) · [API 文档](#-api-文档) · [项目结构](#️-项目结构)

</div>

---

## ✨ 功能特性

|     | 功能                 | 说明                                                   |
| :-: | -------------------- | ------------------------------------------------------ |
| 🧙  | **Setup Wizard**     | 4 步引导式配置 — WebDAV 连接、备份路径、加密、定时任务 |
| 📡  | **实时日志**         | WebSocket 推送 Rclone 备份输出，秒级可观测             |
| ⏰  | **定时备份**         | 可视化 Cron 调度器：每日 / 每周 / 间隔 / 自定义表达式  |
| 🔐  | **加密传输**         | 可选 Rclone Crypt，数据在云端始终加密存储              |
| 🛡️  | **访问保护**         | 可选管理员账号密码，保护 Web UI / API / WebSocket      |
| 📂  | **Restore Explorer** | 浏览云端备份文件，支持透明解密查看与批量选择           |
| 📦  | **单文件部署**       | 前端资源 `go:embed` 内嵌，零外部依赖                   |
| 🏗️  | **多架构**           | `linux/amd64` + `linux/arm64` 双架构构建               |

---

## 🚀 快速开始

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
      - /你的Immich照片库路径:/data/library:ro
      - /你的Immich数据库备份路径:/data/backups:ro
      - ./config:/app/config
    environment:
      - TZ=Asia/Shanghai
```

```bash
docker compose up -d
```

> 访问 `http://服务器IP:8096`，首次进入 Setup Wizard 完成配置。

### 方式二：一键安装（Linux）

```bash
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/install.sh | sudo bash
```

自动完成：安装 Rclone → 下载二进制 → 创建 systemd 服务 → 启动。

<details>
<summary>💡 自定义下载源</summary>

```bash
RELEASE_URL=https://your-mirror.com/releases/latest/download sudo bash install.sh
```

</details>

### 方式三：从源码构建

<details>
<summary>展开查看</summary>

```bash
# 克隆仓库
git clone https://github.com/aYenx/immichto115.git
cd immichto115

# 编译前端
cd web && npm ci && npm run build && cd ..

# 将前端产物复制到 Go 内嵌目录
rm -rf cmd/server/dist && cp -r web/dist cmd/server/dist

# 编译后端（内嵌前端资源）
CGO_ENABLED=0 go build -ldflags="-s -w" -o immichto115 ./cmd/server/

# 运行
./immichto115 --config /path/to/config.yaml --port 8096
```

</details>

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

> 配置文件路径优先级：`--config` 参数 > `IMMICHTO115_CONFIG` 环境变量 > `{可执行文件目录}/config/config.yaml`

首次访问 Web UI 会进入 **Setup Wizard**，配置完成后自动生成 `config.yaml`。

| 配置项               | 说明                                           | 必填 |
| -------------------- | ---------------------------------------------- | :--: |
| WebDAV URL           | 115 网盘 WebDAV 地址                           |  ✅  |
| WebDAV 用户名 / 密码 | 登录凭据                                       |  ✅  |
| 照片库路径           | Immich 照片存储目录                            |  ✅  |
| 数据库备份路径       | Immich DB dump 目录                            |  ✅  |
| Cron 表达式          | 定时备份周期（如 `0 3 * * *` = 每天凌晨 3 点） |  ✅  |
| 加密密码             | Rclone Crypt 加密口令                          |  ⬜  |
| 管理员账号 / 密码    | HTTP Basic Auth 保护界面与 API                 |  ⬜  |

> [!IMPORTANT]
> 建议限制 `config/` 目录访问权限（`chmod 700`），避免敏感配置被其他用户读取。

---

## 🔧 运维手册

<table>
<tr><th>操作</th><th>Docker</th><th>Systemd（一键安装）</th></tr>
<tr><td>查看日志</td><td><code>docker compose logs -f</code></td><td><code>journalctl -u immichto115 -f</code></td></tr>
<tr><td>重启服务</td><td><code>docker compose restart</code></td><td><code>systemctl restart immichto115</code></td></tr>
<tr><td>停止服务</td><td><code>docker compose down</code></td><td><code>systemctl stop immichto115</code></td></tr>
<tr><td>查看状态</td><td><code>docker compose ps</code></td><td><code>systemctl status immichto115</code></td></tr>
</table>

### 🗑️ 卸载

```bash
# Docker：停止并删除容器和镜像
docker compose down --rmi all

# 一键安装：运行卸载脚本
curl -fsSL https://raw.githubusercontent.com/aYenx/immichto115/master/deploy/uninstall.sh | sudo bash
```

> 卸载不会影响 115 网盘上已备份的文件。

---

## 📋 API 文档

<details>
<summary>📡 完整 API 列表</summary>

|  方法  | 路径                    | 说明                                    | 鉴权 |
| :----: | ----------------------- | --------------------------------------- | :--: |
| `GET`  | `/api/health`           | 健康检查                                |  ⬜  |
| `GET`  | `/api/v1/ping`          | 连通测试                                |  ✅  |
| `GET`  | `/api/v1/system/status` | 系统状态（Rclone 版本、备份状态、Cron） |  ✅  |
| `GET`  | `/api/v1/config`        | 获取配置（敏感信息已脱敏）              |  ✅  |
| `POST` | `/api/v1/config`        | 保存配置                                |  ✅  |
| `POST` | `/api/v1/webdav/test`   | 测试 WebDAV 连接                        |  ✅  |
| `GET`  | `/api/v1/webdav/ls`     | 浏览 WebDAV 目录                        |  ✅  |
| `POST` | `/api/v1/backup/start`  | 手动触发备份                            |  ✅  |
| `POST` | `/api/v1/backup/stop`   | 停止备份                                |  ✅  |
| `GET`  | `/api/v1/remote/ls`     | 浏览云端文件                            |  ✅  |
| `GET`  | `/api/v1/local/ls`      | 浏览本地目录                            |  ✅  |
|  `WS`  | `/ws/logs`              | 实时备份日志流                          |  ✅  |

> 开启访问保护后，除 `/api/health` 外均需管理员账号密码（HTTP Basic Auth）。

</details>

---

## 🏗️ 项目结构

```
immichto115/
├── cmd/server/              # Go 入口 + go:embed 内嵌前端
├── internal/
│   ├── api/                 # Gin 路由 + WebSocket Hub
│   ├── config/              # Viper 配置管理 + rclone.conf 生成
│   ├── cron/                # 定时任务调度 (robfig/cron)
│   └── rclone/              # Rclone CLI 封装 (os/exec)
├── web/                     # Vue 3 前端
│   └── src/
│       ├── views/           # Dashboard / Wizard / RestoreExplorer
│       ├── components/      # Layout · CronScheduler
│       ├── api.ts           # 类型化 API 客户端
│       └── style.css        # 全局样式 (CSS Variables + Dark Mode)
├── deploy/
│   ├── Dockerfile           # 多阶段构建 (amd64 / arm64)
│   ├── docker-compose.yml
│   ├── install.sh           # Linux 一键安装
│   └── uninstall.sh         # 卸载脚本
└── .github/workflows/       # CI/CD: 构建 + Docker + Release
```

## 📦 技术栈

<table>
<tr><td><b>后端</b></td><td>Go 1.22 · Gin · Viper · gorilla/websocket · robfig/cron</td></tr>
<tr><td><b>前端</b></td><td>Vue 3 · Vue Router · Lucide Icons · CSS Variables (Dark&nbsp;Mode)</td></tr>
<tr><td><b>备份引擎</b></td><td>Rclone CLI（通过 os/exec 调用）</td></tr>
<tr><td><b>构建</b></td><td>go:embed 内嵌前端 · 多阶段 Docker · GitHub Actions CI/CD</td></tr>
</table>

---

## 🏷️ 发布

```bash
git tag v0.4.0
git push origin v0.4.0
```

GitHub Actions 自动完成：

1. 🔨 构建 `linux/amd64` + `linux/arm64` 二进制
2. 🐳 构建多架构 Docker 镜像并推送到 [GHCR](https://ghcr.io/ayenx/immichto115)
3. 📦 创建 [GitHub Release](https://github.com/aYenx/immichto115/releases) 并上传产物

---

<div align="center">

## 📄 License

[MIT](LICENSE)

**如果这个项目对你有帮助，欢迎 ⭐️ Star 支持！**

</div>
