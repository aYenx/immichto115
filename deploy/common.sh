#!/usr/bin/env bash
# ============================================================
#  ImmichTo115 部署脚本公共库
#  被 install.sh 和 uninstall.sh source 引入
# ============================================================

# ---- 常量 ---------------------------------------------------
APP_NAME="immichto115"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/${APP_NAME}"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
DEFAULT_PORT=8096
GITHUB_REPO="aYenx/immichto115"

# ---- 颜色 ---------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
CYAN='\033[1;36m'
BOLD='\033[1m'
DIM='\033[2m'
UNDERLINE='\033[4m'
NC='\033[0m'

# ---- 日志函数 -----------------------------------------------
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
step()  { echo -e "${BLUE}[STEP]${NC}  ${BOLD}$*${NC}"; }

# ---- 终端超链接 (OSC 8) -------------------------------------
# 现代终端会渲染为可点击链接
link() {
    local url="$1"
    local text="${2:-$url}"
    printf '\e]8;;%s\a%b%s%b\e]8;;\a' "$url" "${UNDERLINE}${CYAN}" "$text" "${NC}"
}

# ---- 横幅 ---------------------------------------------------
banner() {
    local title="$1"
    echo ""
    echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
    printf "${BOLD}║${NC}  %-40s${BOLD}║${NC}\n" "$title"
    echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}"
    echo ""
}

# ---- 权限检查 -----------------------------------------------
require_root() {
    if [[ $EUID -ne 0 ]]; then
        error "请使用 root 权限运行: sudo bash $0"
    fi
}

# ---- 架构检测 -----------------------------------------------
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "不支持的架构: $arch (仅支持 amd64/arm64)" ;;
    esac
}

# ---- 容器环境检测 -------------------------------------------
is_container() {
    # 检测 Docker / LXC / systemd-nspawn 等容器环境
    if [[ -f /.dockerenv ]]; then
        return 0
    fi
    if grep -qE '(docker|lxc|containerd)' /proc/1/cgroup 2>/dev/null; then
        return 0
    fi
    if [[ "$(head -1 /proc/1/sched 2>/dev/null)" =~ "bash" ]]; then
        return 0
    fi
    return 1
}

# ---- 服务状态检测 -------------------------------------------
service_exists() {
    [[ -f "${SERVICE_FILE}" ]]
}

service_is_active() {
    systemctl is-active --quiet "${APP_NAME}" 2>/dev/null
}

service_is_enabled() {
    systemctl is-enabled --quiet "${APP_NAME}" 2>/dev/null
}

# ---- 版本读取 -----------------------------------------------
get_installed_version() {
    if [[ -x "${INSTALL_DIR}/${APP_NAME}" ]]; then
        "${INSTALL_DIR}/${APP_NAME}" --version 2>/dev/null || echo "未知"
    else
        echo "未安装"
    fi
}

# ---- 确认提示 -----------------------------------------------
# 用法: confirm "提示文字" && do_something
confirm() {
    local prompt="${1:-确认继续？}"
    read -rp "${prompt} [y/N] " answer
    [[ "${answer}" =~ ^[Yy]$ ]]
}

# ---- 检查命令是否存在 ---------------------------------------
require_cmd() {
    local cmd="$1"
    local hint="${2:-}"
    if ! command -v "$cmd" &>/dev/null; then
        if [[ -n "$hint" ]]; then
            error "缺少命令: $cmd ($hint)"
        else
            error "缺少命令: $cmd"
        fi
    fi
}

# ---- 下载文件（自动选择 curl / wget）------------------------
download() {
    local url="$1"
    local output="$2"

    if command -v curl &>/dev/null; then
        curl -fsSL -o "$output" "$url"
    elif command -v wget &>/dev/null; then
        wget -q -O "$output" "$url"
    else
        error "需要 curl 或 wget 才能下载文件"
    fi
}
