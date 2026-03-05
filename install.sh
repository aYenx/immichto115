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
NC='\033[0m'

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

# 下载 ImmichTo115 二进制
download_binary() {
    local arch
    arch=$(detect_arch)
    local os="linux"

    # TODO: 替换为实际的 GitHub Release URL
    local download_url="https://github.com/immichto115/immichto115-web/releases/latest/download/${APP_NAME}-${os}-${arch}"

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
    echo "   ImmichTo115 一键安装脚本"
    echo "========================================"
    echo ""

    # 检查 root 权限
    if [[ $EUID -ne 0 ]]; then
        error "请使用 root 权限运行: sudo bash install.sh"
    fi

    check_rclone
    download_binary
    setup_config
    setup_systemd

    echo ""
    info "✅ 安装完成！"
    info "访问 http://$(hostname -I | awk '{print $1}'):8096 开始配置"
    echo ""
}

main "$@"
