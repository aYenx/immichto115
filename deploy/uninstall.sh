#!/usr/bin/env bash
set -euo pipefail

# ============================================================
#  ImmichTo115 卸载脚本
#  对应 install.sh 的反向操作
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

# 停止并移除 systemd 服务
remove_service() {
    if systemctl is-active --quiet "${APP_NAME}" 2>/dev/null; then
        info "正在停止 ${APP_NAME} 服务..."
        systemctl stop "${APP_NAME}"
    fi

    if systemctl is-enabled --quiet "${APP_NAME}" 2>/dev/null; then
        info "正在禁用 ${APP_NAME} 服务..."
        systemctl disable "${APP_NAME}"
    fi

    if [[ -f "${SERVICE_FILE}" ]]; then
        rm -f "${SERVICE_FILE}"
        systemctl daemon-reload
        info "已移除 systemd 服务"
    else
        info "未找到 systemd 服务文件，跳过"
    fi
}

# 删除二进制文件
remove_binary() {
    if [[ -f "${INSTALL_DIR}/${APP_NAME}" ]]; then
        rm -f "${INSTALL_DIR}/${APP_NAME}"
        info "已删除 ${INSTALL_DIR}/${APP_NAME}"
    else
        info "未找到二进制文件，跳过"
    fi
}

# 删除配置目录
remove_config() {
    if [[ -d "${CONFIG_DIR}" ]]; then
        read -rp "是否删除配置目录 ${CONFIG_DIR}？[y/N] " answer
        if [[ "${answer}" =~ ^[Yy]$ ]]; then
            rm -rf "${CONFIG_DIR}"
            info "已删除配置目录"
        else
            warn "保留配置目录 ${CONFIG_DIR}"
        fi
    fi
}

# 主流程
main() {
    echo ""
    echo "========================================"
    echo "   ImmichTo115 卸载脚本"
    echo "========================================"
    echo ""

    # 检查 root 权限
    if [[ $EUID -ne 0 ]]; then
        error "请使用 root 权限运行: sudo bash uninstall.sh"
    fi

    remove_service
    remove_binary
    remove_config

    echo ""
    info "✅ 卸载完成！"
    info "115 网盘上已备份的文件不受影响"
    echo ""
}

main "$@"
