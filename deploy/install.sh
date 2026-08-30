#!/bin/bash
#
# Sub3API Installation Script
# Sub3API 安装脚本
#
# Usage (global): curl -sSL https://raw.githubusercontent.com/yuchen0x1/sub3api-relay/main/deploy/install.sh | bash
# Usage (中国大陆): curl -sSL https://gitee.com/yuchen0x1/personal-ai-api-gateway-sub3api/raw/main/deploy/install.sh | bash
#
# 双通道说明：脚本会按服务器 IP 归属地自动选择更新通道（detect_update_region）：
#   国内 IP  -> Gitee（API + 资产直链 + raw 脚本自更新均走 gitee.com）
#   国外 IP  -> GitHub
#   Gitee 失败时自动回退 GitHub。
# 注意：Gitee 仓库名与 GitHub 仓库名不同（均为官方发布仓库），见下方 GITHUB_REPO/GITEE_REPO。
#

set -e

# Bash 4+ is required for associative arrays used by the localized message table.
# Keep this guard before any Bash 4-only syntax so older shells fail with a clear hint.
if [ -z "${BASH_VERSION:-}" ]; then
    echo "Error: This installer must be run with Bash 4.0 or later." >&2
    echo "Please install Bash 4+ and run it with that interpreter." >&2
    exit 1
fi

BASH_MAJOR_VERSION="${BASH_VERSION%%.*}"
if [ "$BASH_MAJOR_VERSION" -lt 4 ]; then
    echo "Error: Bash 4.0 or later is required. Current version: $BASH_VERSION" >&2
    echo "Please install Bash 4+ and retry with that interpreter." >&2
    exit 1
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
# 注意：GitHub 仓库名与 Gitee 仓库名不同（历史原因）：
#   GitHub: yuchen0x1/sub3api-relay
#   Gitee : yuchen0x1/personal-ai-api-gateway-sub3api（当前 Gitee 仓库以此为准）
# GITHUB_REPO 仅用于 github.com / api.github.com 的 URL，保持原名不变；
# GITEE_REPO 用于 gitee.com 的 API 与资产直链。
GITHUB_REPO="yuchen0x1/sub3api-relay"
GITEE_REPO="yuchen0x1/personal-ai-api-gateway-sub3api"
INSTALL_DIR="/opt/sub3api"
SERVICE_NAME="sub3api"
SERVICE_USER="sub3api"
CONFIG_DIR="/etc/sub3api"

# Update channel (resolved by detect_update_region; cn -> Gitee, global -> GitHub)
# 可通过环境变量 UPDATE_REGION_FORCE=cn|global 强制指定，留空自动探测。
UPDATE_REGION=""
UPDATE_REGION_RESOLVED="false"

# Server configuration (will be set by user)
SERVER_HOST="0.0.0.0"
SERVER_PORT="8080"
SETUP_SERVER_HOST="127.0.0.1"

# Language (default: zh = Chinese)
LANG_CHOICE="zh"

# ============================================================
# Language strings / 语言字符串
# ============================================================

# Chinese strings
declare -A MSG_ZH=(
    # General
    ["info"]="信息"
    ["success"]="成功"
    ["warning"]="警告"
    ["error"]="错误"

    # Language selection
    ["select_lang"]="请选择语言 / Select language"
    ["lang_zh"]="中文"
    ["lang_en"]="English"
    ["enter_choice"]="请输入选择 (默认: 1)"

    # Installation
    ["install_title"]="Sub3API 安装脚本"
    ["run_as_root"]="请使用 root 权限运行 (使用 sudo)"
    ["detected_platform"]="检测到平台"
    ["unsupported_arch"]="不支持的架构"
    ["unsupported_os"]="不支持的操作系统"
    ["missing_deps"]="缺少依赖"
    ["install_deps_first"]="请先安装以下依赖"
    ["fetching_version"]="正在获取最新版本..."
    ["latest_version"]="最新版本"
    ["failed_get_version"]="获取最新版本失败"
    ["downloading"]="正在下载"
    ["download_failed"]="下载失败"
    ["verifying_checksum"]="正在校验文件..."
    ["checksum_verified"]="校验通过"
    ["checksum_failed"]="校验失败"
    ["checksum_not_found"]="无法验证校验和（checksums.txt 未找到）"
    ["extracting"]="正在解压..."
    ["binary_installed"]="二进制文件已安装到"
    ["user_exists"]="用户已存在"
    ["creating_user"]="正在创建系统用户"
    ["user_created"]="用户已创建"
    ["setting_up_dirs"]="正在设置目录..."
    ["dirs_configured"]="目录配置完成"
    ["installing_service"]="正在安装 systemd 服务..."
    ["service_installed"]="systemd 服务已安装"
    ["ready_for_setup"]="准备就绪，可以启动设置向导"

    # Completion
    ["install_complete"]="Sub3API 安装完成！"
    ["install_dir"]="安装目录"
    ["next_steps"]="后续步骤"
    ["step1_check_services"]="确保 PostgreSQL 和 Redis 正在运行："
    ["step2_start_service"]="启动 Sub3API 服务："
    ["step3_enable_autostart"]="设置开机自启："
    ["step4_open_wizard"]="在浏览器中打开设置向导："
    ["wizard_guide"]="设置向导将引导您完成："
    ["wizard_db"]="数据库配置"
    ["wizard_redis"]="Redis 配置"
    ["wizard_admin"]="管理员账号创建"
    ["useful_commands"]="常用命令"
    ["cmd_status"]="查看状态"
    ["cmd_logs"]="查看日志"
    ["cmd_restart"]="重启服务"
    ["cmd_stop"]="停止服务"

    # Upgrade
    ["upgrading"]="正在升级 Sub3API..."
    ["current_version"]="当前版本"
    ["stopping_service"]="正在停止服务..."
    ["backup_created"]="备份已创建"
    ["starting_service"]="正在启动服务..."
    ["upgrade_complete"]="升级完成！"

    # Version install
    ["installing_version"]="正在安装指定版本"
    ["version_not_found"]="指定版本不存在"
    ["same_version"]="已经是该版本，无需操作"
    ["rollback_complete"]="版本回退完成！"
    ["install_version_complete"]="指定版本安装完成！"
    ["validating_version"]="正在验证版本..."
    ["available_versions"]="可用版本列表"
    ["fetching_versions"]="正在获取可用版本..."
    ["not_installed"]="Sub3API 尚未安装，请先执行全新安装"
    ["fresh_install_hint"]="用法"

    # Uninstall
    ["uninstall_confirm"]="这将从系统中移除 Sub3API。"
    ["are_you_sure"]="确定要继续吗？(y/N)"
    ["uninstall_cancelled"]="卸载已取消"
    ["removing_files"]="正在移除文件..."
    ["removing_install_dir"]="正在移除安装目录..."
    ["removing_user"]="正在移除用户..."
    ["config_not_removed"]="配置目录未被移除"
    ["remove_manually"]="如不再需要，请手动删除"
    ["removing_install_lock"]="正在移除安装锁文件..."
    ["install_lock_removed"]="安装锁文件已移除，重新安装时将进入设置向导"
    ["purge_prompt"]="是否同时删除配置目录？这将清除所有配置和数据 [y/N]: "
    ["removing_config_dir"]="正在移除配置目录..."
    ["uninstall_complete"]="Sub3API 已卸载"

    # Help
    ["usage"]="用法"
    ["cmd_none"]="(无参数)"
    ["cmd_install"]="安装 Sub3API"
    ["cmd_upgrade"]="升级到最新版本"
    ["cmd_uninstall"]="卸载 Sub3API"
    ["cmd_install_version"]="安装/回退到指定版本"
    ["cmd_list_versions"]="列出可用版本"
    ["opt_version"]="指定要安装的版本号 (例如: v1.0.0)"

    # Server configuration
    ["server_config_title"]="服务器配置"
    ["server_config_desc"]="配置 Sub3API 服务监听地址"
    ["server_host_prompt"]="服务器监听地址"
    ["server_host_hint"]="0.0.0.0 表示监听所有网卡，127.0.0.1 仅本地访问"
    ["server_port_prompt"]="服务器端口"
    ["server_port_hint"]="建议使用 1024-65535 之间的端口"
    ["server_config_summary"]="服务器配置"
    ["invalid_port"]="无效端口号，请输入 1-65535 之间的数字"

    # Service management
    ["starting_service"]="正在启动服务..."
    ["service_started"]="服务已启动"
    ["service_start_failed"]="服务启动失败，请检查日志"
    ["enabling_autostart"]="正在设置开机自启..."
    ["autostart_enabled"]="开机自启已启用"
    ["getting_public_ip"]="正在获取公网 IP..."
    ["public_ip_failed"]="无法获取公网 IP，使用本地 IP"
)

# English strings
declare -A MSG_EN=(
    # General
    ["info"]="INFO"
    ["success"]="SUCCESS"
    ["warning"]="WARNING"
    ["error"]="ERROR"

    # Language selection
    ["select_lang"]="请选择语言 / Select language"
    ["lang_zh"]="中文"
    ["lang_en"]="English"
    ["enter_choice"]="Enter your choice (default: 1)"

    # Installation
    ["install_title"]="Sub3API Installation Script"
    ["run_as_root"]="Please run as root (use sudo)"
    ["detected_platform"]="Detected platform"
    ["unsupported_arch"]="Unsupported architecture"
    ["unsupported_os"]="Unsupported OS"
    ["missing_deps"]="Missing dependencies"
    ["install_deps_first"]="Please install them first"
    ["fetching_version"]="Fetching latest version..."
    ["latest_version"]="Latest version"
    ["failed_get_version"]="Failed to get latest version"
    ["downloading"]="Downloading"
    ["download_failed"]="Download failed"
    ["verifying_checksum"]="Verifying checksum..."
    ["checksum_verified"]="Checksum verified"
    ["checksum_failed"]="Checksum verification failed"
    ["checksum_not_found"]="Could not verify checksum (checksums.txt not found)"
    ["extracting"]="Extracting..."
    ["binary_installed"]="Binary installed to"
    ["user_exists"]="User already exists"
    ["creating_user"]="Creating system user"
    ["user_created"]="User created"
    ["setting_up_dirs"]="Setting up directories..."
    ["dirs_configured"]="Directories configured"
    ["installing_service"]="Installing systemd service..."
    ["service_installed"]="Systemd service installed"
    ["ready_for_setup"]="Ready for Setup Wizard"

    # Completion
    ["install_complete"]="Sub3API installation completed!"
    ["install_dir"]="Installation directory"
    ["next_steps"]="NEXT STEPS"
    ["step1_check_services"]="Make sure PostgreSQL and Redis are running:"
    ["step2_start_service"]="Start Sub3API service:"
    ["step3_enable_autostart"]="Enable auto-start on boot:"
    ["step4_open_wizard"]="Open the Setup Wizard in your browser:"
    ["wizard_guide"]="The Setup Wizard will guide you through:"
    ["wizard_db"]="Database configuration"
    ["wizard_redis"]="Redis configuration"
    ["wizard_admin"]="Admin account creation"
    ["useful_commands"]="USEFUL COMMANDS"
    ["cmd_status"]="Check status"
    ["cmd_logs"]="View logs"
    ["cmd_restart"]="Restart"
    ["cmd_stop"]="Stop"

    # Upgrade
    ["upgrading"]="Upgrading Sub3API..."
    ["current_version"]="Current version"
    ["stopping_service"]="Stopping service..."
    ["backup_created"]="Backup created"
    ["starting_service"]="Starting service..."
    ["upgrade_complete"]="Upgrade completed!"

    # Version install
    ["installing_version"]="Installing specified version"
    ["version_not_found"]="Specified version not found"
    ["same_version"]="Already at this version, no action needed"
    ["rollback_complete"]="Version rollback completed!"
    ["install_version_complete"]="Specified version installed!"
    ["validating_version"]="Validating version..."
    ["available_versions"]="Available versions"
    ["fetching_versions"]="Fetching available versions..."
    ["not_installed"]="Sub3API is not installed. Please run a fresh install first"
    ["fresh_install_hint"]="Usage"

    # Uninstall
    ["uninstall_confirm"]="This will remove Sub3API from your system."
    ["are_you_sure"]="Are you sure? (y/N)"
    ["uninstall_cancelled"]="Uninstall cancelled"
    ["removing_files"]="Removing files..."
    ["removing_install_dir"]="Removing installation directory..."
    ["removing_user"]="Removing user..."
    ["config_not_removed"]="Config directory was NOT removed."
    ["remove_manually"]="Remove it manually if you no longer need it."
    ["removing_install_lock"]="Removing install lock file..."
    ["install_lock_removed"]="Install lock removed. Setup wizard will appear on next install."
    ["purge_prompt"]="Also remove config directory? This will delete all config and data [y/N]: "
    ["removing_config_dir"]="Removing config directory..."
    ["uninstall_complete"]="Sub3API has been uninstalled"

    # Help
    ["usage"]="Usage"
    ["cmd_none"]="(none)"
    ["cmd_install"]="Install Sub3API"
    ["cmd_upgrade"]="Upgrade to the latest version"
    ["cmd_uninstall"]="Remove Sub3API"
    ["cmd_install_version"]="Install/rollback to a specific version"
    ["cmd_list_versions"]="List available versions"
    ["opt_version"]="Specify version to install (e.g., v1.0.0)"

    # Server configuration
    ["server_config_title"]="Server Configuration"
    ["server_config_desc"]="Configure Sub3API server listen address"
    ["server_host_prompt"]="Server listen address"
    ["server_host_hint"]="0.0.0.0 listens on all interfaces, 127.0.0.1 for local only"
    ["server_port_prompt"]="Server port"
    ["server_port_hint"]="Recommended range: 1024-65535"
    ["server_config_summary"]="Server configuration"
    ["invalid_port"]="Invalid port number, please enter a number between 1-65535"

    # Service management
    ["starting_service"]="Starting service..."
    ["service_started"]="Service started"
    ["service_start_failed"]="Service failed to start, please check logs"
    ["enabling_autostart"]="Enabling auto-start on boot..."
    ["autostart_enabled"]="Auto-start enabled"
    ["getting_public_ip"]="Getting public IP..."
    ["public_ip_failed"]="Failed to get public IP, using local IP"
)

# Get message based on current language
msg() {
    local key="$1"
    if [ "$LANG_CHOICE" = "en" ]; then
        echo "${MSG_EN[$key]}"
    else
        echo "${MSG_ZH[$key]}"
    fi
}

# Print functions
print_info() {
    echo -e "${BLUE}[$(msg 'info')]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[$(msg 'success')]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[$(msg 'warning')]${NC} $1"
}

print_error() {
    echo -e "${RED}[$(msg 'error')]${NC} $1"
}

# Check if running interactively (can access terminal)
# When piped (curl | bash), stdin is not a terminal, but /dev/tty may still be available
is_interactive() {
    # Check if /dev/tty is available (works even when piped)
    [ -e /dev/tty ] && [ -r /dev/tty ] && [ -w /dev/tty ]
}

# Select language
select_language() {
    # If not interactive (piped), use default language
    if ! is_interactive; then
        LANG_CHOICE="zh"
        return
    fi

    echo ""
    echo -e "${CYAN}=============================================="
    echo "  $(msg 'select_lang')"
    echo "==============================================${NC}"
    echo ""
    echo "  1) $(msg 'lang_zh') (默认/default)"
    echo "  2) $(msg 'lang_en')"
    echo ""

    read -p "$(msg 'enter_choice'): " lang_input < /dev/tty

    case "$lang_input" in
        2|en|EN|english|English)
            LANG_CHOICE="en"
            ;;
        *)
            LANG_CHOICE="zh"
            ;;
    esac

    echo ""
}

# Validate port number
validate_port() {
    local port="$1"
    if [[ "$port" =~ ^[0-9]+$ ]] && [ "$port" -ge 1 ] && [ "$port" -le 65535 ]; then
        return 0
    fi
    return 1
}

# Configure server settings
configure_server() {
    # If not interactive (piped), use default settings
    if ! is_interactive; then
        print_info "$(msg 'server_config_summary'): ${SERVER_HOST}:${SERVER_PORT} (default)"
        return
    fi

    echo ""
    echo -e "${CYAN}=============================================="
    echo "  $(msg 'server_config_title')"
    echo "==============================================${NC}"
    echo ""
    echo -e "${BLUE}$(msg 'server_config_desc')${NC}"
    echo ""

    # Server host
    echo -e "${YELLOW}$(msg 'server_host_hint')${NC}"
    read -p "$(msg 'server_host_prompt') [${SERVER_HOST}]: " input_host < /dev/tty
    if [ -n "$input_host" ]; then
        SERVER_HOST="$input_host"
    fi

    echo ""

    # Server port
    echo -e "${YELLOW}$(msg 'server_port_hint')${NC}"
    while true; do
        read -p "$(msg 'server_port_prompt') [${SERVER_PORT}]: " input_port < /dev/tty
        if [ -z "$input_port" ]; then
            # Use default
            break
        elif validate_port "$input_port"; then
            SERVER_PORT="$input_port"
            break
        else
            print_error "$(msg 'invalid_port')"
        fi
    done

    echo ""
    print_info "$(msg 'server_config_summary'): ${SERVER_HOST}:${SERVER_PORT}"
    echo ""
}

# Check if running as root
check_root() {
    # Use 'id -u' instead of $EUID for better compatibility
    # $EUID may not work reliably when script is piped to bash
    if [ "$(id -u)" -ne 0 ]; then
        print_error "$(msg 'run_as_root')"
        exit 1
    fi
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$ARCH" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            print_error "$(msg 'unsupported_arch'): $ARCH"
            exit 1
            ;;
    esac

    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            print_error "$(msg 'unsupported_os'): $OS"
            exit 1
            ;;
    esac

    print_info "$(msg 'detected_platform'): ${OS}_${ARCH}"
}

# Check dependencies
check_dependencies() {
    local missing=()

    if ! command -v curl &> /dev/null; then
        missing+=("curl")
    fi

    if ! command -v tar &> /dev/null; then
        missing+=("tar")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        print_error "$(msg 'missing_deps'): ${missing[*]}"
        print_info "$(msg 'install_deps_first')"
        exit 1
    fi
}

# Authenticate only GitHub REST API requests. Release asset downloads must stay anonymous.
github_api_curl() {
    local arg
    local expect_value=false
    local url

    if [ "$#" -lt 1 ]; then
        echo "github_api_curl requires exactly one GitHub API URL" >&2
        return 2
    fi
    url="${!#}"

    # Keep authenticated invocations constrained to the options used below. In
    # particular, curl config, --url, and --next could add another destination.
    for arg in "${@:1:$#-1}"; do
        if [ "$expect_value" = true ]; then
            expect_value=false
            continue
        fi
        case "$arg" in
            -s|--silent)
                ;;
            --connect-timeout|--max-time|-o|--output|-w|--write-out)
                expect_value=true
                ;;
            *)
                echo "Unsafe github_api_curl argument: $arg" >&2
                return 2
                ;;
        esac
    done

    if [ "$expect_value" = true ] || [[ "$url" != https://api.github.com/* ]]; then
        echo "github_api_curl requires exactly one GitHub API URL" >&2
        return 2
    fi

    if [ -n "${UPDATE_GITHUB_TOKEN:-}" ]; then
        if [[ "$UPDATE_GITHUB_TOKEN" == *$'\n'* || "$UPDATE_GITHUB_TOKEN" == *$'\r'* || "$UPDATE_GITHUB_TOKEN" == *'"'* || "$UPDATE_GITHUB_TOKEN" == *'\'* ]]; then
            echo "UPDATE_GITHUB_TOKEN contains unsupported characters" >&2
            return 2
        fi
        printf 'header = "Authorization: Bearer %s"\n' "$UPDATE_GITHUB_TOKEN" | UPDATE_GITHUB_TOKEN= GITHUB_TOKEN= GH_TOKEN= curl -q --globoff --config - "$@"
    else
        UPDATE_GITHUB_TOKEN= GITHUB_TOKEN= GH_TOKEN= curl -q --globoff "$@"
    fi
}

# Authenticate only Gitee REST API requests (no token needed for public repos).
gitee_api_curl() {
    local arg
    local expect_value=false
    local url

    if [ "$#" -lt 1 ]; then
        echo "gitee_api_curl requires exactly one Gitee API URL" >&2
        return 2
    fi
    url="${!#}"

    for arg in "${@:1:$#-1}"; do
        if [ "$expect_value" = true ]; then
            expect_value=false
            continue
        fi
        case "$arg" in
            -s|--silent)
                ;;
            --connect-timeout|--max-time|-o|--output|-w|--write-out)
                expect_value=true
                ;;
            *)
                echo "Unsafe gitee_api_curl argument: $arg" >&2
                return 2
                ;;
        esac
    done

    if [ "$expect_value" = true ] || [[ "$url" != https://gitee.com/api/v5/* ]]; then
        echo "gitee_api_curl requires exactly one Gitee API URL" >&2
        return 2
    fi

    UPDATE_GITHUB_TOKEN= GITHUB_TOKEN= GH_TOKEN= curl -q --globoff "$@"
}

# release_api_curl 按 URL host 分发到对应的安全 API 封装（GitHub/Gitee）
release_api_curl() {
    local url="${!#}"
    case "$url" in
        https://api.github.com/*)
            github_api_curl "$@"
            ;;
        https://gitee.com/api/v5/*)
            gitee_api_curl "$@"
            ;;
        *)
            echo "release_api_curl requires a GitHub/Gitee API URL" >&2
            return 2
            ;;
    esac
}

# ============================================================
# Update channel detection / 更新通道检测
# 规则（与 backend/internal/service/update_channel.go 保持一致）：
#   1) qifu-api.baidubce.com geo（3s 超时）：响应含 "中国"/"China" -> cn
#   2) api.ip.sb/geoip（3s 超时）：country_code === "CN" -> cn
#   3) 两个 geo 源都失败：竞速 Gitee / GitHub releases/latest，先成功者胜
#   4) 全部失败：global 兜底
# 安全约束：仅 HTTPS + 域名白名单；解析出的 IP 必须为公网地址
# （拒绝 localhost/环回/私有 10.x/172.16-31.x/192.168.x/169.254.x 与保留地址）。
# ============================================================

# channel_url_host 从 https URL 提取主机名（失败返回非 0）
channel_url_host() {
    local url="$1"
    case "$url" in
        https://*) ;;
        *) return 1 ;;
    esac
    local rest="${url#https://}"
    rest="${rest%%/*}"
    rest="${rest%%:*}"   # strip port
    rest="${rest%%@*}"   # strip userinfo (defensive)
    [ -n "$rest" ] || return 1
    echo "$rest"
}

# is_channel_allowed_host 探测/更新通道使用的域名白名单
is_channel_allowed_host() {
    case "$1" in
        qifu-api.baidubce.com|api.ip.sb|gitee.com|api.github.com) return 0 ;;
        *) return 1 ;;
    esac
}

# is_forbidden_ip 拒绝环回/私有/链路本地/保留地址
is_forbidden_ip() {
    local ip="$1"
    local second
    case "$ip" in
        ::1|::) return 0 ;;
        fe[89ab]*|f[cd]*|ff*) return 0 ;; # fe80::/10 链路本地, fc00::/7 ULA, ff00::/8 组播
        ::ffff:*)
            is_forbidden_ip "${ip##*:}" && return 0
            return 1
            ;;
        0.*|10.*|127.*|169.254.*|192.168.*|224.*|240.*|255.*) return 0 ;;
        192.0.0.*|192.0.2.*|198.51.100.*|203.0.113.*) return 0 ;;
        198.1[89].*) return 0 ;; # 198.18/15 benchmark
        172.*)
            # 172.16.0.0/12
            second="${ip#172.}"
            second="${second%%.*}"
            if [ "$second" -ge 16 ] && [ "$second" -le 31 ] 2>/dev/null; then
                return 0
            fi
            return 1
            ;;
        100.*)
            # 100.64.0.0/10 CGNAT
            second="${ip#100.}"
            second="${second%%.*}"
            if [ "$second" -ge 64 ] && [ "$second" -le 127 ] 2>/dev/null; then
                return 0
            fi
            return 1
            ;;
        *) return 1 ;;
    esac
}

# resolve_host_ips 解析域名（尽力而为；缺工具时输出为空，仅依赖白名单+HTTPS）
resolve_host_ips() {
    if command -v getent >/dev/null 2>&1; then
        getent ahosts "$1" 2>/dev/null | awk '{print $1}' | sort -u
    fi
}

# channel_validate_host_ips 白名单 + 解析 IP 校验
channel_validate_host_ips() {
    local host="$1"
    is_channel_allowed_host "$host" || return 1

    local ips ip
    ips=$(resolve_host_ips "$host")
    if [ -n "$ips" ]; then
        while read -r ip; do
            [ -z "$ip" ] && continue
            if is_forbidden_ip "$ip"; then
                echo "Blocked update channel host ${host}: resolved to forbidden address ${ip}" >&2
                return 1
            fi
        done <<< "$ips"
    fi
    return 0
}

# channel_probe_curl 白名单 + IP 校验后的 HTTPS GET（用于 geo/竞速探测）
channel_probe_curl() {
    local url="$1"
    local max_time="${2:-5}"
    local host
    host=$(channel_url_host "$url") || return 1
    channel_validate_host_ips "$host" || return 1
    curl -fsSL --connect-timeout 3 --max-time "$max_time" "$url" 2>/dev/null
}

# detect_update_region 解析更新通道并写入 UPDATE_REGION（cn|global）
detect_update_region() {
    # 显式覆盖
    if [ -n "${UPDATE_REGION_FORCE:-}" ]; then
        case "$UPDATE_REGION_FORCE" in
            cn|global)
                UPDATE_REGION="$UPDATE_REGION_FORCE"
                print_info "Update region (forced): $UPDATE_REGION"
                return 0
                ;;
        esac
    fi

    print_info "Detecting update region..."

    local body

    # 1) qifu-api.baidubce.com（3s 超时）
    body=$(channel_probe_curl "https://qifu-api.baidubce.com/ip/local/geo/v1/district" 3)
    if [ -n "$body" ]; then
        if echo "$body" | grep -qE '中国|China|china'; then
            UPDATE_REGION="cn"
            print_success "Update region: cn (via qifu-api)"
            return 0
        fi
        UPDATE_REGION="global"
        print_success "Update region: global (via qifu-api)"
        return 0
    fi

    # 2) api.ip.sb/geoip（3s 超时）
    body=$(channel_probe_curl "https://api.ip.sb/geoip" 3)
    if [ -n "$body" ]; then
        if echo "$body" | grep -q '"country_code"[: ]*"CN"'; then
            UPDATE_REGION="cn"
            print_success "Update region: cn (via ip.sb)"
            return 0
        fi
        UPDATE_REGION="global"
        print_success "Update region: global (via ip.sb)"
        return 0
    fi

    # 3) 竞速 Gitee / GitHub releases/latest，先成功者胜
    local tmpdir winner p1 p2 waited
    tmpdir=$(mktemp -d)
    ( channel_probe_curl "https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/latest" 8 >/dev/null 2>&1 && echo cn > "$tmpdir/winner" ) &
    p1=$!
    ( github_api_curl -s --connect-timeout 3 --max-time 8 "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" >/dev/null 2>&1 && echo global > "$tmpdir/winner" ) &
    p2=$!
    waited=0
    while [ "$waited" -lt 16 ]; do
        if [ -s "$tmpdir/winner" ]; then
            break
        fi
        if ! kill -0 "$p1" 2>/dev/null && ! kill -0 "$p2" 2>/dev/null; then
            break
        fi
        sleep 1
        waited=$((waited + 1))
    done
    winner=$(cat "$tmpdir/winner" 2>/dev/null || true)
    kill "$p1" "$p2" 2>/dev/null || true
    wait "$p1" 2>/dev/null || true
    wait "$p2" 2>/dev/null || true
    rm -rf "$tmpdir"

    if [ "$winner" = "cn" ] || [ "$winner" = "global" ]; then
        UPDATE_REGION="$winner"
        print_success "Update region: $UPDATE_REGION (via reachability race)"
        return 0
    fi

    # 4) 全部失败：global 兜底
    UPDATE_REGION="global"
    print_warning "Region detection failed, falling back to: global"
}

# ensure_update_region 懒解析通道（每轮安装/升级只探测一次）。
# 输出统一重定向到 stderr：validate_version 会在 $(...) 中捕获 stdout 取版本号，
# 探测过程日志不能污染其结果。
ensure_update_region() {
    [ "$UPDATE_REGION_RESOLVED" = "true" ] && return 0
    detect_update_region >&2
    UPDATE_REGION_RESOLVED="true"
}

# Get latest release version (channel-aware: cn -> Gitee, fallback GitHub)
get_latest_version() {
    ensure_update_region

    print_info "$(msg 'fetching_version')"

    local api_url
    if [ "$UPDATE_REGION" = "cn" ]; then
        api_url="https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/latest"
    else
        api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    fi

    LATEST_VERSION=$(release_api_curl -s --connect-timeout 10 --max-time 30 "$api_url" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')

    # 当前通道获取失败时回退另一通道
    if [ -z "$LATEST_VERSION" ]; then
        if [ "$UPDATE_REGION" = "cn" ]; then
            print_warning "Gitee failed, falling back to GitHub..."
            UPDATE_REGION="global"
            api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
        else
            print_warning "GitHub failed, trying Gitee mirror..."
            UPDATE_REGION="cn"
            api_url="https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/latest"
        fi
        LATEST_VERSION=$(release_api_curl -s --connect-timeout 10 --max-time 30 "$api_url" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
    fi

    if [ -z "$LATEST_VERSION" ]; then
        print_error "$(msg 'failed_get_version')"
        print_info "Please check your network connection or try again later."
        exit 1
    fi

    print_info "$(msg 'latest_version'): $LATEST_VERSION (channel: ${UPDATE_REGION})"
}

# List available versions (channel-aware: cn -> Gitee, fallback GitHub)
list_versions() {
    ensure_update_region

    print_info "$(msg 'fetching_versions')"

    local versions
    if [ "$UPDATE_REGION" = "cn" ]; then
        versions=$(release_api_curl -s --connect-timeout 10 --max-time 30 "https://gitee.com/api/v5/repos/${GITEE_REPO}/releases?page=1&per_page=20&direction=desc" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' | head -20)
        if [ -z "$versions" ]; then
            print_warning "Gitee failed, falling back to GitHub..."
            UPDATE_REGION="global"
        fi
    fi
    if [ -z "${versions:-}" ]; then
        versions=$(release_api_curl -s --connect-timeout 10 --max-time 30 "https://api.github.com/repos/${GITHUB_REPO}/releases?per_page=20" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' | head -20)
    fi

    if [ -z "$versions" ]; then
        print_error "$(msg 'failed_get_version')"
        print_info "Please check your network connection or try again later."
        exit 1
    fi

    echo ""
    echo "$(msg 'available_versions'): (channel: ${UPDATE_REGION})"
    echo "----------------------------------------"
    echo "$versions" | while read -r version; do
        echo "  $version"
    done
    echo "----------------------------------------"
    echo ""
}

# Validate if a version exists (channel-aware: cn -> Gitee, fallback GitHub)
validate_version() {
    local version="$1"

    # Check for empty version
    if [ -z "$version" ]; then
        print_error "$(msg 'opt_version')" >&2
        exit 1
    fi

    # Ensure version starts with 'v'
    if [[ ! "$version" =~ ^v ]]; then
        version="v$version"
    fi

    ensure_update_region

    print_info "$(msg 'validating_version') $version" >&2

    # Check if the release exists
    local http_code
    local check_url
    if [ "$UPDATE_REGION" = "cn" ]; then
        check_url="https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/tags/${version}"
    else
        check_url="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${version}"
    fi
    http_code=$(release_api_curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 --max-time 30 "$check_url" 2>/dev/null)

    # Check for network errors (empty or non-numeric response)
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        print_error "Network error: Failed to connect to release API" >&2
        exit 1
    fi

    if [ "$http_code" != "200" ]; then
        # cn 通道未命中时回退 GitHub 复核一次（Gitee 发布可能滞后）
        if [ "$UPDATE_REGION" = "cn" ]; then
            print_warning "Version not found on Gitee, retrying via GitHub..." >&2
            UPDATE_REGION="global"
            check_url="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${version}"
            http_code=$(release_api_curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 --max-time 30 "$check_url" 2>/dev/null)
            if [ "$http_code" = "200" ]; then
                echo "$version"
                return 0
            fi
            UPDATE_REGION="cn"
        fi
        print_error "$(msg 'version_not_found'): $version" >&2
        echo "" >&2
        list_versions >&2
        exit 1
    fi

    # Return the normalized version (to stdout)
    echo "$version"
}

# Get current installed version
get_current_version() {
    if [ -f "$INSTALL_DIR/sub3api" ]; then
        # Use grep -E for better compatibility (works on macOS and Linux)
        "$INSTALL_DIR/sub3api" --version 2>/dev/null | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown"
    else
        echo "not_installed"
    fi
}

# validate_download_host 仅允许从官方 GitHub/Gitee 域下载
validate_download_host() {
    local host
    host=$(channel_url_host "$1") || return 1
    case "$host" in
        github.com|gitee.com|objects.githubusercontent.com) return 0 ;;
        *) return 1 ;;
    esac
}

# download_release_file 按通道下载 release 资产：
#   cn     -> 先 Gitee 资产直链，失败回退 GitHub
#   global -> 先 GitHub，失败尝试 Gitee 备用
download_release_file() {
    local name="$1"
    local dest="$2"

    local urls=()
    if [ "$UPDATE_REGION" = "cn" ]; then
        urls+=("https://gitee.com/${GITEE_REPO}/releases/download/${LATEST_VERSION}/${name}")
        urls+=("https://github.com/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/${name}")
    else
        urls+=("https://github.com/${GITHUB_REPO}/releases/download/${LATEST_VERSION}/${name}")
        urls+=("https://gitee.com/${GITEE_REPO}/releases/download/${LATEST_VERSION}/${name}")
    fi

    local i url
    local total=${#urls[@]}
    for ((i = 0; i < total; i++)); do
        url="${urls[$i]}"
        if validate_download_host "$url" && curl -fsSL --connect-timeout 10 --max-time 900 "$url" -o "$dest"; then
            return 0
        fi
        rm -f "$dest"
        if [ "$((i + 1))" -lt "$total" ]; then
            print_warning "Download failed from $url, trying mirror..."
        fi
    done
    return 1
}

# Download and extract
download_and_extract() {
    local version_num=${LATEST_VERSION#v}
    local archive_name="sub3api_${version_num}_${OS}_${ARCH}.tar.gz"

    print_info "$(msg 'downloading') ${archive_name}..."

    # Create temp directory
    TEMP_DIR=$(mktemp -d)
    trap "rm -rf $TEMP_DIR" EXIT

    # Download archive (channel-aware with mirror fallback)
    if ! download_release_file "$archive_name" "$TEMP_DIR/$archive_name"; then
        print_error "$(msg 'download_failed')"
        exit 1
    fi

    # Download and verify checksum (channel-aware with mirror fallback)
    print_info "$(msg 'verifying_checksum')"
    if ! download_release_file "checksums.txt" "$TEMP_DIR/checksums.txt"; then
        print_error "$(msg 'checksum_not_found')"
        exit 1
    fi

    local expected_checksum
    local actual_checksum
    expected_checksum=$(awk -v archive="$archive_name" '$2 == archive || $2 == "*" archive { print $1; exit }' "$TEMP_DIR/checksums.txt")
    actual_checksum=$(sha256sum "$TEMP_DIR/$archive_name" | awk '{print $1}')

    if ! printf '%s' "$expected_checksum" | grep -Eq '^[0-9a-fA-F]{64}$'; then
        print_error "$(msg 'checksum_not_found')"
        exit 1
    fi
    if [ "$expected_checksum" != "$actual_checksum" ]; then
        print_error "$(msg 'checksum_failed')"
        print_error "Expected: $expected_checksum"
        print_error "Actual: $actual_checksum"
        exit 1
    fi
    print_success "$(msg 'checksum_verified')"

    # Extract
    print_info "$(msg 'extracting')"
    tar -xzf "$TEMP_DIR/$archive_name" -C "$TEMP_DIR"

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Copy binary
    cp "$TEMP_DIR/sub3api" "$INSTALL_DIR/sub3api"
    chmod +x "$INSTALL_DIR/sub3api"

    # Copy deploy files if they exist in the archive
    if [ -d "$TEMP_DIR/deploy" ]; then
        cp -r "$TEMP_DIR/deploy/"* "$INSTALL_DIR/" 2>/dev/null || true
    fi

    print_success "$(msg 'binary_installed') $INSTALL_DIR/sub3api"
}

# Create system user
create_user() {
    if id "$SERVICE_USER" &>/dev/null; then
        print_info "$(msg 'user_exists'): $SERVICE_USER"
        # Fix: Ensure existing user has /bin/sh shell for sudo to work
        # Previous versions used /bin/false which prevents sudo execution
        local current_shell
        current_shell=$(getent passwd "$SERVICE_USER" 2>/dev/null | cut -d: -f7)
        if [ "$current_shell" = "/bin/false" ] || [ "$current_shell" = "/sbin/nologin" ]; then
            print_info "Fixing user shell for sudo compatibility..."
            if usermod -s /bin/sh "$SERVICE_USER" 2>/dev/null; then
                print_success "User shell updated to /bin/sh"
            else
                print_warning "Failed to update user shell. Service restart may not work automatically."
                print_warning "Manual fix: sudo usermod -s /bin/sh $SERVICE_USER"
            fi
        fi
    else
        print_info "$(msg 'creating_user') $SERVICE_USER..."
        # Use /bin/sh instead of /bin/false to allow sudo execution
        # The user still cannot login interactively (no password set)
        useradd -r -s /bin/sh -d "$INSTALL_DIR" "$SERVICE_USER"
        print_success "$(msg 'user_created')"
    fi
}

# Setup directories and permissions
setup_directories() {
    print_info "$(msg 'setting_up_dirs')"

    # Create directories
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR/data"
    mkdir -p "$CONFIG_DIR"

    # Set ownership
    chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
    chown -R "$SERVICE_USER:$SERVICE_USER" "$CONFIG_DIR"

    print_success "$(msg 'dirs_configured')"
}

# Install systemd service
install_service() {
    print_info "$(msg 'installing_service')"

    # Create service file with configured host and port
    cat > /etc/systemd/system/sub3api.service << EOF
[Unit]
Description=Sub3API - AI API Gateway Platform
Documentation=https://github.com/yuchen0x1/sub3api-relay
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=sub3api
Group=sub3api
WorkingDirectory=/opt/sub3api
ExecStart=/opt/sub3api/sub3api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=sub3api

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/sub3api

# Environment - Server configuration
Environment=GIN_MODE=release
Environment=SERVER_HOST=${SERVER_HOST}
Environment=SERVER_PORT=${SERVER_PORT}
Environment=SETUP_SERVER_HOST=${SETUP_SERVER_HOST}
Environment=SUB3API_DEPLOYMENT_TYPE=binary

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd
    systemctl daemon-reload

    print_success "$(msg 'service_installed')"
}

# Prepare for setup wizard (no config file needed - setup wizard will create it)
prepare_for_setup() {
    print_success "$(msg 'ready_for_setup')"
}

# Get public IP address
get_public_ip() {
    print_info "$(msg 'getting_public_ip')"

    # Try to get public IP from ipinfo.io
    local response
    response=$(curl -s --connect-timeout 5 --max-time 10 "https://ipinfo.io/json" 2>/dev/null)

    if [ -n "$response" ]; then
        # Extract IP from JSON response using grep and sed (no jq dependency)
        PUBLIC_IP=$(echo "$response" | grep -o '"ip": *"[^"]*"' | sed 's/"ip": *"\([^"]*\)"/\1/')
        if [ -n "$PUBLIC_IP" ]; then
            print_success "Public IP: $PUBLIC_IP"
            return 0
        fi
    fi

    # Fallback to local IP
    print_warning "$(msg 'public_ip_failed')"
    PUBLIC_IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "YOUR_SERVER_IP")
    return 1
}

# Start service
start_service() {
    print_info "$(msg 'starting_service')"

    if systemctl start sub3api; then
        print_success "$(msg 'service_started')"
        return 0
    else
        print_error "$(msg 'service_start_failed')"
        print_info "sudo journalctl -u sub3api -n 50"
        return 1
    fi
}

# Enable service auto-start
enable_autostart() {
    print_info "$(msg 'enabling_autostart')"

    if systemctl enable sub3api 2>/dev/null; then
        print_success "$(msg 'autostart_enabled')"
        return 0
    else
        print_warning "Failed to enable auto-start"
        return 1
    fi
}

# Print completion message
print_completion() {
    # Use PUBLIC_IP which was set by get_public_ip()
    # Determine display address
    local display_host="127.0.0.1"

    echo ""
    echo "=============================================="
    print_success "$(msg 'install_complete')"
    echo "=============================================="
    echo ""
    echo "$(msg 'install_dir'): $INSTALL_DIR"
    echo "$(msg 'server_config_summary'): ${SERVER_HOST}:${SERVER_PORT}"
    echo ""
    echo "=============================================="
    echo "  $(msg 'step4_open_wizard')"
    echo "=============================================="
    echo ""
    print_info "     http://${display_host}:${SERVER_PORT}"
    print_info "     SSH: ssh -L ${SERVER_PORT}:127.0.0.1:${SERVER_PORT} user@${PUBLIC_IP:-YOUR_SERVER_IP}"
    echo ""
    echo "     $(msg 'wizard_guide')"
    echo "     - $(msg 'wizard_db')"
    echo "     - $(msg 'wizard_redis')"
    echo "     - $(msg 'wizard_admin')"
    echo ""
    echo "=============================================="
    echo "  $(msg 'useful_commands')"
    echo "=============================================="
    echo ""
    echo "  $(msg 'cmd_status'):   sudo systemctl status sub3api"
    echo "  $(msg 'cmd_logs'):     sudo journalctl -u sub3api -f"
    echo "  $(msg 'cmd_restart'):  sudo systemctl restart sub3api"
    echo "  $(msg 'cmd_stop'):     sudo systemctl stop sub3api"
    echo ""
    echo "=============================================="
}

# Upgrade function
upgrade() {
    # Check if Sub3API is installed
    if [ ! -f "$INSTALL_DIR/sub3api" ]; then
        print_error "$(msg 'not_installed')"
        print_info "$(msg 'fresh_install_hint'): $0 install"
        exit 1
    fi

    print_info "$(msg 'upgrading')"

    # Get current version
    CURRENT_VERSION=$("$INSTALL_DIR/sub3api" --version 2>/dev/null | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
    print_info "$(msg 'current_version'): $CURRENT_VERSION"

    # Stop service
    if systemctl is-active --quiet sub3api; then
        print_info "$(msg 'stopping_service')"
        systemctl stop sub3api
    fi

    # Backup current binary
    cp "$INSTALL_DIR/sub3api" "$INSTALL_DIR/sub3api.backup"
    print_info "$(msg 'backup_created'): $INSTALL_DIR/sub3api.backup"

    # Download and install new version
    get_latest_version
    download_and_extract

    # Set permissions
    chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/sub3api"

    # Start service
    print_info "$(msg 'starting_service')"
    systemctl start sub3api

    print_success "$(msg 'upgrade_complete')"
}

# Install specific version (for upgrade or rollback)
# Requires: Sub3API must already be installed
install_version() {
    local target_version="$1"

    # Check if Sub3API is installed
    if [ ! -f "$INSTALL_DIR/sub3api" ]; then
        print_error "$(msg 'not_installed')"
        print_info "$(msg 'fresh_install_hint'): $0 install -v $target_version"
        exit 1
    fi

    # Validate and normalize version
    target_version=$(validate_version "$target_version")

    print_info "$(msg 'installing_version'): $target_version"

    # Get current version
    local current_version
    current_version=$(get_current_version)
    print_info "$(msg 'current_version'): $current_version"

    # Check if same version
    if [ "$current_version" = "$target_version" ] || [ "$current_version" = "${target_version#v}" ]; then
        print_warning "$(msg 'same_version')"
        exit 0
    fi

    # Stop service if running
    if systemctl is-active --quiet sub3api; then
        print_info "$(msg 'stopping_service')"
        systemctl stop sub3api
    fi

    # Backup current binary (for potential recovery)
    if [ -f "$INSTALL_DIR/sub3api" ]; then
        local backup_name
        if [ "$current_version" != "unknown" ] && [ "$current_version" != "not_installed" ]; then
            backup_name="sub3api.backup.${current_version}"
        else
            backup_name="sub3api.backup.$(date +%Y%m%d%H%M%S)"
        fi
        cp "$INSTALL_DIR/sub3api" "$INSTALL_DIR/$backup_name"
        print_info "$(msg 'backup_created'): $INSTALL_DIR/$backup_name"
    fi

    # Set LATEST_VERSION to the target version for download_and_extract
    LATEST_VERSION="$target_version"

    # Download and install
    download_and_extract

    # Set permissions
    chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/sub3api"

    # Start service
    print_info "$(msg 'starting_service')"
    if systemctl start sub3api; then
        print_success "$(msg 'service_started')"
    else
        print_error "$(msg 'service_start_failed')"
        print_info "sudo journalctl -u sub3api -n 50"
    fi

    # Print completion message
    local new_version
    new_version=$(get_current_version)
    echo ""
    echo "=============================================="
    print_success "$(msg 'install_version_complete')"
    echo "=============================================="
    echo ""
    echo "  $(msg 'current_version'): $new_version"
    echo ""
}

# Uninstall function
uninstall() {
    print_warning "$(msg 'uninstall_confirm')"

    # If not interactive (piped), require -y flag or skip confirmation
    if ! is_interactive; then
        if [ "${FORCE_YES:-}" != "true" ]; then
            print_error "Non-interactive mode detected. Use 'curl ... | bash -s -- uninstall -y' to confirm."
            exit 1
        fi
    else
        read -p "$(msg 'are_you_sure') " -n 1 -r < /dev/tty
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "$(msg 'uninstall_cancelled')"
            exit 0
        fi
    fi

    print_info "$(msg 'stopping_service')"
    systemctl stop sub3api 2>/dev/null || true
    systemctl disable sub3api 2>/dev/null || true

    print_info "$(msg 'removing_files')"
    rm -f /etc/systemd/system/sub3api.service
    systemctl daemon-reload

    print_info "$(msg 'removing_install_dir')"
    rm -rf "$INSTALL_DIR"

    print_info "$(msg 'removing_user')"
    userdel "$SERVICE_USER" 2>/dev/null || true

    # Remove install lock file (.installed) to allow fresh setup on reinstall
    print_info "$(msg 'removing_install_lock')"
    rm -f "$CONFIG_DIR/.installed" 2>/dev/null || true
    rm -f "$INSTALL_DIR/.installed" 2>/dev/null || true
    print_success "$(msg 'install_lock_removed')"

    # Ask about config directory removal (interactive mode only)
    local remove_config=false
    if [ "${PURGE:-}" = "true" ]; then
        remove_config=true
    elif is_interactive; then
        read -p "$(msg 'purge_prompt')" -n 1 -r < /dev/tty
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            remove_config=true
        fi
    fi

    if [ "$remove_config" = true ]; then
        print_info "$(msg 'removing_config_dir')"
        rm -rf "$CONFIG_DIR"
    else
        print_warning "$(msg 'config_not_removed'): $CONFIG_DIR"
        print_warning "$(msg 'remove_manually')"
    fi

    print_success "$(msg 'uninstall_complete')"
}

# Main
main() {
    # Parse flags first
    local target_version=""
    local positional_args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            -y|--yes)
                FORCE_YES="true"
                shift
                ;;
            --purge)
                PURGE="true"
                shift
                ;;
            -v|--version)
                if [ -n "${2:-}" ] && [[ ! "$2" =~ ^- ]]; then
                    target_version="$2"
                    shift 2
                else
                    echo "Error: --version requires a version argument"
                    exit 1
                fi
                ;;
            --version=*)
                target_version="${1#*=}"
                if [ -z "$target_version" ]; then
                    echo "Error: --version requires a version argument"
                    exit 1
                fi
                shift
                ;;
            *)
                positional_args+=("$1")
                shift
                ;;
        esac
    done

    # Restore positional arguments
    set -- "${positional_args[@]}"

    # Select language first
    select_language

    echo ""
    echo "=============================================="
    echo "       $(msg 'install_title')"
    echo "=============================================="
    echo ""

    # Parse commands
    case "${1:-}" in
        upgrade|update)
            check_root
            detect_platform
            check_dependencies
            if [ -n "$target_version" ]; then
                # Upgrade to specific version
                install_version "$target_version"
            else
                # Upgrade to latest
                upgrade
            fi
            exit 0
            ;;
        install)
            # Install with optional version
            check_root
            detect_platform
            check_dependencies
            if [ -n "$target_version" ]; then
                # Install specific version (fresh install or rollback)
                if [ -f "$INSTALL_DIR/sub3api" ]; then
                    # Already installed, treat as version change
                    install_version "$target_version"
                else
                    # Fresh install with specific version
                    configure_server
                    LATEST_VERSION=$(validate_version "$target_version")
                    download_and_extract
                    create_user
                    setup_directories
                    install_service
                    prepare_for_setup
                    get_public_ip
                    start_service
                    enable_autostart
                    print_completion
                fi
            else
                # Fresh install with latest version
                configure_server
                get_latest_version
                download_and_extract
                create_user
                setup_directories
                install_service
                prepare_for_setup
                get_public_ip
                start_service
                enable_autostart
                print_completion
            fi
            exit 0
            ;;
        rollback)
            # Rollback to a specific version (alias for install with version)
            if [ -z "$target_version" ] && [ -n "${2:-}" ]; then
                target_version="$2"
            fi
            if [ -z "$target_version" ]; then
                print_error "$(msg 'opt_version')"
                echo ""
                echo "Usage: $0 rollback -v <version>"
                echo "       $0 rollback <version>"
                echo ""
                list_versions
                exit 1
            fi
            check_root
            detect_platform
            check_dependencies
            install_version "$target_version"
            exit 0
            ;;
        list-versions|versions)
            list_versions
            exit 0
            ;;
        uninstall|remove)
            check_root
            uninstall
            exit 0
            ;;
        --help|-h)
            echo "$(msg 'usage'): $0 [command] [options]"
            echo ""
            echo "Commands:"
            echo "  $(msg 'cmd_none')            $(msg 'cmd_install')"
            echo "  install              $(msg 'cmd_install')"
            echo "  upgrade              $(msg 'cmd_upgrade')"
            echo "  rollback <version>   $(msg 'cmd_install_version')"
            echo "  list-versions        $(msg 'cmd_list_versions')"
            echo "  uninstall            $(msg 'cmd_uninstall')"
            echo ""
            echo "Options:"
            echo "  -v, --version <ver>  $(msg 'opt_version')"
            echo "  -y, --yes            Skip confirmation prompts (for uninstall)"
            echo ""
            echo "Examples:"
            echo "  $0                        # Install latest version"
            echo "  $0 install -v v0.1.0      # Install specific version"
            echo "  $0 upgrade                # Upgrade to latest"
            echo "  $0 upgrade -v v0.2.0      # Upgrade to specific version"
            echo "  $0 rollback v0.1.0        # Rollback to v0.1.0"
            echo "  $0 list-versions          # List available versions"
            echo ""
            exit 0
            ;;
    esac

    # Default: Fresh install with latest version
    check_root
    detect_platform
    check_dependencies

    if [ -n "$target_version" ]; then
        # Install specific version
        if [ -f "$INSTALL_DIR/sub3api" ]; then
            install_version "$target_version"
        else
            configure_server
            LATEST_VERSION=$(validate_version "$target_version")
            download_and_extract
            create_user
            setup_directories
            install_service
            prepare_for_setup
            get_public_ip
            start_service
            enable_autostart
            print_completion
        fi
    else
        # Install latest version
        configure_server
        get_latest_version
        download_and_extract
        create_user
        setup_directories
        install_service
        prepare_for_setup
        get_public_ip
        start_service
        enable_autostart
        print_completion
    fi
}

main "$@"
