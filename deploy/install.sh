#!/usr/bin/env bash
set -euo pipefail

# ============================================================
#  ImmichTo115 一键安装 / 更新脚本
#  支持 Linux (amd64 / arm64)
#
#  用法:
#    sudo bash install.sh [选项]
#    curl -fsSL https://...install.sh | sudo bash
#    curl -fsSL https://...install.sh | sudo bash -s -- --no-rclone
#
#  选项:
#    --no-rclone    跳过 Rclone 检查与安装
#    --force        强制覆写 systemd 服务文件
#    --help         显示帮助信息
#
#  环境变量:
#    RELEASE_URL    自定义下载地址前缀（镜像加速）
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

# ---- 日志 ---------------------------------------------------
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
step()  { echo -e "${BLUE}[STEP]${NC}  ${BOLD}$*${NC}"; }

# ---- 工具函数 -----------------------------------------------
link() {
    local url="$1"
    local text="${2:-$url}"
    printf '\e]8;;%s\a%b%s%b\e]8;;\a' "$url" "${UNDERLINE}${CYAN}" "$text" "${NC}"
}

banner() {
    local title="$1"
    echo ""
    echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
    printf "${BOLD}║${NC}  %-40s${BOLD}║${NC}\n" "$title"
    echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}"
    echo ""
}

require_root() {
    if [[ $EUID -ne 0 ]]; then
        error "请使用 root 权限运行: sudo bash install.sh"
    fi
}

detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             error "不支持的架构: $arch (仅支持 amd64/arm64)" ;;
    esac
}

is_container() {
    if [[ -f /.dockerenv ]]; then return 0; fi
    if grep -qE '(docker|lxc|containerd)' /proc/1/cgroup 2>/dev/null; then return 0; fi
    if [[ "$(head -1 /proc/1/sched 2>/dev/null)" =~ "bash" ]]; then return 0; fi
    return 1
}

download() {
    local url="$1" output="$2"
    if command -v curl &>/dev/null; then
        curl -fsSL -o "$output" "$url"
    elif command -v wget &>/dev/null; then
        wget -q -O "$output" "$url"
    else
        error "需要 curl 或 wget 才能下载文件"
    fi
}

get_installed_version() {
    if [[ -x "${INSTALL_DIR}/${APP_NAME}" ]]; then
        "${INSTALL_DIR}/${APP_NAME}" --version 2>/dev/null || echo "未知"
    else
        echo "未安装"
    fi
}

# ---- 参数默认值 ---------------------------------------------
OPT_NO_RCLONE=false
OPT_FORCE=false

# ---- 参数解析 -----------------------------------------------
show_help() {
    cat <<'HELP'
ImmichTo115 一键安装 / 更新脚本

用法:
  sudo bash install.sh [选项]
  curl -fsSL https://...install.sh | sudo bash
  curl -fsSL https://...install.sh | sudo bash -s -- --no-rclone

选项:
  --no-rclone    跳过 Rclone 检查与安装（适用于已使用 Open115 的用户）
  --force        强制覆写 systemd 服务文件（默认更新时保留）
  --help         显示此帮助信息

环境变量:
  RELEASE_URL    自定义下载地址前缀
                 示例: RELEASE_URL=https://mirror.example.com/releases/latest/download bash install.sh
HELP
    exit 0
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --no-rclone) OPT_NO_RCLONE=true ;;
            --force)     OPT_FORCE=true ;;
            --help|-h)   show_help ;;
            *)           warn "未知参数: $1（已忽略）" ;;
        esac
        shift
    done
}

# ---- Docker 环境警告 ----------------------------------------
check_container_env() {
    if is_container; then
        warn "检测到当前处于容器环境"
        warn "推荐使用 Docker 镜像部署: ghcr.io/${GITHUB_REPO}:latest"
        warn "详见 deploy/docker-compose.yml"
        echo ""
        read -rp "仍然继续裸安装？[y/N] " answer
        if [[ ! "${answer}" =~ ^[Yy]$ ]]; then
            info "已取消安装"
            exit 0
        fi
    fi
}

# ---- Rclone 检查与安装 --------------------------------------
check_rclone() {
    if $OPT_NO_RCLONE; then
        info "跳过 Rclone 检查（--no-rclone）"
        return
    fi

    if command -v rclone &>/dev/null; then
        local rclone_ver
        rclone_ver=$(rclone version 2>/dev/null | head -1 || echo "未知版本")
        info "Rclone 已安装: ${rclone_ver}"
        return
    fi

    step "安装 Rclone ..."
    local tmp_script="/tmp/rclone-install-$$.sh"
    download "https://rclone.org/install.sh" "$tmp_script"
    bash "$tmp_script" || {
        rm -f "$tmp_script"
        error "Rclone 安装失败"
    }
    rm -f "$tmp_script"

    if ! command -v rclone &>/dev/null; then
        error "Rclone 安装后仍无法找到命令"
    fi
    info "Rclone 安装成功"
}

# ---- 检测升级 -----------------------------------------------
is_upgrade() {
    [[ -f "${SERVICE_FILE}" ]] && systemctl is-active --quiet "${APP_NAME}" 2>/dev/null
}

# ---- 停止已运行的服务 ---------------------------------------
stop_service() {
    if systemctl is-active --quiet "${APP_NAME}" 2>/dev/null; then
        info "正在停止 ${APP_NAME} 服务 ..."
        systemctl stop "${APP_NAME}"
        info "服务已停止"
    fi
}

# ---- 下载二进制 & 校验 --------------------------------------
download_binary() {
    local arch os="linux"
    arch=$(detect_arch)

    local base_url="${RELEASE_URL:-https://github.com/${GITHUB_REPO}/releases/latest/download}"
    local binary_name="${APP_NAME}-${os}-${arch}"
    local download_url="${base_url}/${binary_name}"
    local checksum_url="${base_url}/checksums.txt"

    step "下载 ${APP_NAME} (${os}/${arch}) ..."

    local tmp_bin="/tmp/${APP_NAME}-$$"
    download "$download_url" "$tmp_bin" || error "下载二进制文件失败"

    # 尝试校验 checksum（非强制，失败时仅警告）
    local tmp_checksum="/tmp/${APP_NAME}-checksums-$$.txt"
    if download "$checksum_url" "$tmp_checksum" 2>/dev/null; then
        if command -v sha256sum &>/dev/null; then
            local expected
            expected=$(grep "${binary_name}$" "$tmp_checksum" | awk '{print $1}')
            if [[ -n "$expected" ]]; then
                local actual
                actual=$(sha256sum "$tmp_bin" | awk '{print $1}')
                if [[ "$expected" == "$actual" ]]; then
                    info "SHA256 校验通过 ✓"
                else
                    rm -f "$tmp_bin" "$tmp_checksum"
                    error "SHA256 校验失败！\n  期望: ${expected}\n  实际: ${actual}"
                fi
            else
                warn "checksums.txt 中未找到 ${binary_name} 的记录，跳过校验"
            fi
        else
            warn "未安装 sha256sum，跳过校验"
        fi
    else
        warn "无法下载 checksums.txt，跳过校验"
    fi
    rm -f "$tmp_checksum"

    chmod +x "$tmp_bin"
    mv "$tmp_bin" "${INSTALL_DIR}/${APP_NAME}"
    info "二进制已安装到 ${INSTALL_DIR}/${APP_NAME}"
}

# ---- 服务用户 -----------------------------------------------
create_service_user() {
    local svc_user="${APP_NAME}"
    if id "${svc_user}" &>/dev/null; then
        info "服务用户 ${svc_user} 已存在"
    else
        step "创建系统用户 ${svc_user} ..."
        useradd --system --no-create-home --shell /usr/sbin/nologin "${svc_user}" \
            || error "创建用户 ${svc_user} 失败"
        info "已创建系统用户: ${svc_user}"
    fi
}

# ---- 配置目录 -----------------------------------------------
setup_config() {
    if [[ -d "${CONFIG_DIR}" ]]; then
        info "配置目录已存在: ${CONFIG_DIR}"
    else
        mkdir -p "${CONFIG_DIR}"
        info "已创建配置目录: ${CONFIG_DIR}"
    fi
    # 限制配置目录权限，防止敏感信息泄露
    chown -R "${APP_NAME}:${APP_NAME}" "${CONFIG_DIR}"
    chmod 700 "${CONFIG_DIR}"
    if [[ -f "${CONFIG_DIR}/config.yaml" ]]; then
        chmod 600 "${CONFIG_DIR}/config.yaml"
    fi
}

# ---- systemd 服务 -------------------------------------------
setup_systemd() {
    local need_create=true

    if [[ -f "${SERVICE_FILE}" ]] && ! $OPT_FORCE; then
        info "保留现有 systemd 服务配置（使用 --force 覆写）"
        need_create=false
    fi

    if $need_create; then
        step "创建 systemd 服务 ..."
        cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=ImmichTo115 Web Backup Service
After=network.target

[Service]
Type=simple
User=${APP_NAME}
Group=${APP_NAME}
ExecStart=${INSTALL_DIR}/${APP_NAME} --config ${CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=10
Environment=TZ=Asia/Shanghai

[Install]
WantedBy=multi-user.target
EOF
        info "服务文件已写入: ${SERVICE_FILE}"
    fi

    systemctl daemon-reload
    systemctl enable "${APP_NAME}" 2>/dev/null
    systemctl start "${APP_NAME}"

    info "服务已启动"
}

# ---- 安装后验证 ---------------------------------------------
post_install_check() {
    sleep 2

    if systemctl is-active --quiet "${APP_NAME}" 2>/dev/null; then
        info "服务健康检查: 运行中 ✓"
    else
        warn "服务似乎未正常启动，请检查日志: journalctl -u ${APP_NAME} -n 30"
    fi
}

# ---- 安装完成摘要 -------------------------------------------
print_summary() {
    local is_upgrade="$1"
    local new_ver
    new_ver=$(get_installed_version)

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
    if $is_upgrade; then
        echo -e "${GREEN}║${NC}  ✅ 更新完成！                           ${GREEN}║${NC}"
    else
        echo -e "${GREEN}║${NC}  ✅ 安装完成！                           ${GREEN}║${NC}"
    fi
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  版本:    ${BOLD}${new_ver}${NC}"
    echo -e "  二进制:  ${DIM}${INSTALL_DIR}/${APP_NAME}${NC}"
    echo -e "  配置:    ${DIM}${CONFIG_DIR}${NC}"
    echo -e "  服务:    ${DIM}${SERVICE_FILE}${NC}"
    echo ""

    local host_ip
    host_ip=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "localhost")
    local url="http://${host_ip}:${DEFAULT_PORT}"
    info "请打开浏览器访问以下地址开始配置："
    echo ""
    echo -e "   👉 $(link "${url}")"
    echo ""
}

# ---- 主流程 -------------------------------------------------
main() {
    parse_args "$@"

    banner "ImmichTo115 安装 / 更新"

    require_root
    check_container_env

    local upgrade=false
    if is_upgrade; then
        upgrade=true
        local current_ver
        current_ver=$(get_installed_version)
        info "🔄 检测到已安装的服务，将执行更新 ..."
        info "当前版本: ${current_ver}"
    fi

    check_rclone

    if $upgrade; then
        stop_service
    fi

    download_binary
    create_service_user
    setup_config
    setup_systemd
    post_install_check
    print_summary "$upgrade"
}

main "$@"
