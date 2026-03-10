#!/usr/bin/env bash
set -euo pipefail

# ============================================================
#  ImmichTo115 卸载脚本
#  对应 install.sh 的反向操作
#
#  用法:
#    sudo bash uninstall.sh [选项]
#
#  选项:
#    --purge    同时删除配置目录（默认保留）
#    --yes      跳过所有确认提示
#    --help     显示帮助信息
# ============================================================

# 定位并加载公共库
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

# ---- 参数默认值 ---------------------------------------------
OPT_PURGE=false
OPT_YES=false

# ---- 参数解析 -----------------------------------------------
show_help() {
    cat <<'HELP'
ImmichTo115 卸载脚本

用法:
  sudo bash uninstall.sh [选项]

选项:
  --purge    同时删除配置目录（默认保留以便重新安装时恢复）
  --yes      跳过所有确认提示（适用于自动化 / CI）
  --help     显示此帮助信息

示例:
  sudo bash uninstall.sh            # 交互式卸载，保留配置
  sudo bash uninstall.sh --purge    # 交互式卸载，删除配置
  sudo bash uninstall.sh --yes      # 非交互卸载，保留配置
HELP
    exit 0
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --purge) OPT_PURGE=true ;;
            --yes)   OPT_YES=true ;;
            --help|-h) show_help ;;
            *)       warn "未知参数: $1（已忽略）" ;;
        esac
        shift
    done
}

# ---- 确认（可被 --yes 跳过）---------------------------------
auto_confirm() {
    local prompt="$1"
    if $OPT_YES; then
        return 0
    fi
    confirm "$prompt"
}

# ---- 停止并移除 systemd 服务 --------------------------------
remove_service() {
    step "移除 systemd 服务 ..."

    if service_is_active; then
        info "正在停止 ${APP_NAME} 服务 ..."
        systemctl stop "${APP_NAME}"
        info "服务已停止"
    fi

    if service_is_enabled; then
        info "正在禁用 ${APP_NAME} 服务 ..."
        systemctl disable "${APP_NAME}" 2>/dev/null
    fi

    if service_exists; then
        rm -f "${SERVICE_FILE}"
        systemctl daemon-reload
        info "已移除 systemd 服务文件"
    else
        info "未找到 systemd 服务文件，跳过"
    fi
}

# ---- 删除二进制文件 -----------------------------------------
remove_binary() {
    step "移除二进制文件 ..."

    if [[ -f "${INSTALL_DIR}/${APP_NAME}" ]]; then
        rm -f "${INSTALL_DIR}/${APP_NAME}"
        info "已删除 ${INSTALL_DIR}/${APP_NAME}"
    else
        info "未找到二进制文件，跳过"
    fi
}

# ---- 删除配置目录 -------------------------------------------
remove_config() {
    if [[ ! -d "${CONFIG_DIR}" ]]; then
        return
    fi

    if $OPT_PURGE; then
        step "删除配置目录 ..."
        if auto_confirm "确认删除配置目录 ${CONFIG_DIR}？（此操作不可恢复）"; then
            rm -rf "${CONFIG_DIR}"
            info "已删除配置目录"
        else
            warn "保留配置目录: ${CONFIG_DIR}"
        fi
    else
        info "保留配置目录: ${CONFIG_DIR}（使用 --purge 删除）"
    fi
}

# ---- Rclone 可选清理 ----------------------------------------
offer_rclone_removal() {
    if ! command -v rclone &>/dev/null; then
        return
    fi

    echo ""
    if auto_confirm "是否同时卸载 Rclone？"; then
        step "卸载 Rclone ..."
        if [[ -f /usr/bin/rclone ]]; then
            rm -f /usr/bin/rclone
            rm -f /usr/local/share/man/man1/rclone.1
            info "已卸载 Rclone"
        elif [[ -f /usr/local/bin/rclone ]]; then
            rm -f /usr/local/bin/rclone
            rm -f /usr/local/share/man/man1/rclone.1
            info "已卸载 Rclone"
        else
            warn "未找到 Rclone 二进制文件，可能需要手动卸载"
        fi
    else
        info "保留 Rclone"
    fi
}

# ---- 卸载摘要 -----------------------------------------------
print_summary() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}  ✅ 卸载完成                             ${GREEN}║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  已删除:"
    echo -e "    • systemd 服务 (${DIM}${SERVICE_FILE}${NC})"
    echo -e "    • 二进制文件   (${DIM}${INSTALL_DIR}/${APP_NAME}${NC})"

    if $OPT_PURGE && [[ ! -d "${CONFIG_DIR}" ]]; then
        echo -e "    • 配置目录     (${DIM}${CONFIG_DIR}${NC})"
    fi

    if [[ -d "${CONFIG_DIR}" ]]; then
        echo ""
        echo -e "  已保留:"
        echo -e "    • 配置目录     (${DIM}${CONFIG_DIR}${NC})"
    fi

    echo ""
    info "115 网盘上已备份的文件不受影响"
    echo ""
}

# ---- 主流程 -------------------------------------------------
main() {
    parse_args "$@"

    banner "ImmichTo115 卸载"

    require_root

    if ! service_exists && [[ ! -f "${INSTALL_DIR}/${APP_NAME}" ]]; then
        info "${APP_NAME} 似乎未安装，无需卸载"
        exit 0
    fi

    local current_ver
    current_ver=$(get_installed_version)
    info "当前安装: ${current_ver}"

    if ! $OPT_YES; then
        echo ""
        if ! confirm "确认卸载 ${APP_NAME}？"; then
            info "已取消卸载"
            exit 0
        fi
        echo ""
    fi

    remove_service
    remove_binary
    remove_config
    offer_rclone_removal
    print_summary
}

main "$@"
