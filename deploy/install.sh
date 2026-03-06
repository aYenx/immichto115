#!/usr/bin/env bash
set -euo pipefail

# ============================================================
#  ImmichTo115 一键安装脚本
#  支持 Linux (amd64/arm64)
# ============================================================

APP_NAME="immichto115"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/${APP_NAME}"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
CYAN='\033[1;36m'
UNDERLINE='\033[4m'
NC='\033[0m'

# 生成可点击的终端超链接 (OSC 8)
# 用法: link <url> [显示文本]
# 现代终端 (iTerm2, GNOME Terminal ≥3.26, Windows Terminal, Alacritty 等) 会渲染为可点击链接
link() {
    local url="$1"
    local text="${2:-$url}"
    # OSC 8 超链接: \e]8;;URL\aTEXT\e]8;;\a
    # 使用 \a (BEL) 替代 \e\\ 作为终止符，兼容性更好
    printf '\e]8;;%s\a%b%s%b\e]8;;\a' "$url" "${UNDERLINE}${CYAN}" "$text" "${NC}"
}

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# 检测架构
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "不支持的架构: $arch" ;;
    esac
}

# 检查并安装 Rclone
check_rclone() {
    if command -v rclone &> /dev/null; then
        info "Rclone 已安装: $(rclone version --check 2>/dev/null || rclone version | head -1)"
    else
        info "正在安装 Rclone..."
        curl -fsSL https://rclone.org/install.sh | bash
        if ! command -v rclone &> /dev/null; then
            error "Rclone 安装失败"
        fi
        info "Rclone 安装成功"
    fi
}

# 检测是否为更新（服务已存在且正在运行）
is_upgrade() {
    [[ -f "${SERVICE_FILE}" ]] && systemctl is-active --quiet "${APP_NAME}" 2>/dev/null
}

# 停止正在运行的服务（更新前调用）
stop_service_if_running() {
    if systemctl is-active --quiet "${APP_NAME}" 2>/dev/null; then
        info "检测到正在运行的服务，正在停止..."
        systemctl stop "${APP_NAME}"
        info "服务已停止"
    fi
}

# 下载 ImmichTo115 二进制
download_binary() {
    local arch
    arch=$(detect_arch)
    local os="linux"

    # GitHub Release 下载地址 — 可通过环境变量 RELEASE_URL 覆盖
    # 示例: RELEASE_URL=https://my-mirror.com/releases/latest/download bash install.sh
    local base_url="${RELEASE_URL:-https://github.com/aYenx/immichto115/releases/latest/download}"
    local download_url="${base_url}/${APP_NAME}-${os}-${arch}"

    info "正在下载 ${APP_NAME} (${os}/${arch})..."

    if command -v curl &> /dev/null; then
        curl -fsSL -o "/tmp/${APP_NAME}" "${download_url}" || error "下载失败"
    elif command -v wget &> /dev/null; then
        wget -q -O "/tmp/${APP_NAME}" "${download_url}" || error "下载失败"
    else
        error "需要 curl 或 wget"
    fi

    chmod +x "/tmp/${APP_NAME}"
    mv "/tmp/${APP_NAME}" "${INSTALL_DIR}/${APP_NAME}"
    info "已安装到 ${INSTALL_DIR}/${APP_NAME}"
}

# 创建配置目录
setup_config() {
    mkdir -p "${CONFIG_DIR}"
    info "配置目录: ${CONFIG_DIR}"
}

# 创建 systemd 服务
setup_systemd() {
    cat > "${SERVICE_FILE}" << EOF
[Unit]
Description=ImmichTo115 Web Backup Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=${INSTALL_DIR}/${APP_NAME} --config ${CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=10
Environment=TZ=Asia/Shanghai

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "${APP_NAME}"
    systemctl start "${APP_NAME}"

    info "Systemd 服务已创建并启动"
}

# 主流程
main() {
    echo ""
    echo "========================================"
    echo "   ImmichTo115 一键安装 / 更新脚本"
    echo "========================================"
    echo ""

    # 检查 root 权限
    if [[ $EUID -ne 0 ]]; then
        error "请使用 root 权限运行: sudo bash install.sh"
    fi

    local upgrade=false
    if is_upgrade; then
        upgrade=true
        info "🔄 检测到已安装的服务，将执行更新..."
        # 显示当前版本
        if [[ -x "${INSTALL_DIR}/${APP_NAME}" ]]; then
            local current_ver
            current_ver=$("${INSTALL_DIR}/${APP_NAME}" --version 2>/dev/null || echo "未知")
            info "当前版本: ${current_ver}"
        fi
    fi

    check_rclone

    # 更新时先停止服务，再替换二进制
    if $upgrade; then
        stop_service_if_running
    fi

    download_binary
    setup_config
    setup_systemd

    echo ""
    if $upgrade; then
        info "✅ 更新完成！服务已重启。"
    else
        info "✅ 安装完成！"
    fi
    echo ""
    local host_ip
    host_ip=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "localhost")
    local url="http://${host_ip}:8096"
    info "请打开浏览器访问以下地址开始配置："
    echo ""
    echo -e "   👉 $(link "${url}")"
    echo ""
}

main "$@"
