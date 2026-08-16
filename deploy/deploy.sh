#!/usr/bin/env bash
#
# ============================================================
#  outlook-manager 一键部署脚本（在 Linux 服务器上运行）
#
#  功能：检测/安装 Go + Node → 构建单二进制 → 配置 systemd
#        开机自启 → 启动服务 → 输出管理员登录信息
#
#  用法：
#    sudo bash deploy.sh                    # 常规部署（源码已在 /opt/outlook-manager）
#    sudo bash deploy.sh --git <仓库URL>     # 先从 git 拉取源码再部署
#    sudo bash deploy.sh --dir /path        # 指定源码目录
#    sudo bash deploy.sh --no-proxy         # 关闭 config 中的全局代理（服务器无代理时用）
#    sudo bash deploy.sh --skip-deps        # 跳过 Go/Node 安装（已装好时加速）
#    sudo bash deploy.sh --no-systemd       # 前台运行，不装 systemd（调试用）
# ============================================================
set -euo pipefail

# ---------- 默认值 ----------
APP_NAME="outlook-manager"
APP_DIR="/opt/outlook-manager"
SERVICE_NAME="outlook-manager"
GO_MIN="1.25"
GO_FALLBACK="1.25.4"
NODE_MIN="20"
NODE_MAJOR="22"
NVM_VERSION="v0.40.1"

GIT_URL=""
NO_PROXY=0
SKIP_DEPS=0
NO_SYSTEMD=0

# ---------- 颜色输出 ----------
RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; CYAN=$'\033[36m'; NC=$'\033[0m'
log_info()  { echo -e "${CYAN}[部署]${NC} $*"; }
log_ok()    { echo -e "${GREEN}[成功]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[警告]${NC} $*"; }
log_err()   { echo -e "${RED}[错误]${NC} $*" >&2; }

# ---------- 帮助 ----------
usage() {
  sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
  exit 0
}

# ---------- 参数解析 ----------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --git)        GIT_URL="${2:-}"; [ -n "$GIT_URL" ] || { log_err "--git 缺少仓库地址"; exit 1; }; shift 2;;
    --dir)        APP_DIR="${2:-}"; [ -n "$APP_DIR" ] || { log_err "--dir 缺少路径"; exit 1; }; shift 2;;
    --no-proxy)   NO_PROXY=1; shift;;
    --skip-deps)  SKIP_DEPS=1; shift;;
    --no-systemd) NO_SYSTEMD=1; shift;;
    -h|--help)    usage;;
    *)            log_err "未知参数: $1（用 --help 查看用法）"; exit 1;;
  esac
done

# ---------- 工具函数 ----------
ver_ge() { # $1 >= $2 ?（版本比较）
  printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

check_root() {
  if [ "$(id -u)" -ne 0 ]; then
    log_err "需要 root 权限，请用: sudo bash $0 $*"
    exit 1
  fi
}

# ---------- 安装 Go（缺失或 < 1.25 时） ----------
install_go() {
  if command -v go >/dev/null 2>&1 && ver_ge "$(go version | awk '{print $3}' | sed 's/^go//')" "$GO_MIN"; then
    log_ok "Go 已满足要求: $(go version)"
    return
  fi
  log_info "安装 Go（官方二进制，需下载 ~100MB）..."
  local ver arch
  ver=$(curl -fsSL --max-time 20 "https://go.dev/VERSION?m=text" 2>/dev/null | head -1 || echo "go${GO_FALLBACK}")
  ver="${ver:-go${GO_FALLBACK}}"
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64;;
    aarch64|arm64) arch=arm64;;
    *) arch=amd64;;
  esac
  curl -fsSL "https://go.dev/dl/${ver}.linux-${arch}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz
  export PATH="/usr/local/go/bin:$PATH"
  echo 'export PATH=/usr/local/go/bin:$PATH' > /etc/profile.d/go.sh
  log_ok "Go 安装完成: $(go version)"
}

# ---------- 安装 Node（缺失或 < 20 时，经 nvm） ----------
install_node() {
  if command -v node >/dev/null 2>&1 && ver_ge "$(node -v | tr -d 'v')" "$NODE_MIN"; then
    log_ok "Node 已满足要求: $(node -v) / npm $(npm -v)"
    return
  fi
  log_info "通过 nvm 安装 Node ${NODE_MAJOR}（仅构建需要）..."
  export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
  if [ ! -s "$NVM_DIR/nvm.sh" ]; then
    curl -o- "https://raw.githubusercontent.com/nvm-sh/nvm/${NVM_VERSION}/install.sh" | bash
  fi
  # shellcheck source=/dev/null
  . "$NVM_DIR/nvm.sh"
  nvm install "$NODE_MAJOR" >/dev/null
  nvm alias default "$NODE_MAJOR" >/dev/null
  nvm use default >/dev/null
  log_ok "Node 安装完成: $(node -v) / npm $(npm -v)"
}

# ---------- 获取源码 ----------
fetch_source() {
  if [ -n "$GIT_URL" ]; then
    log_info "从 $GIT_URL 获取源码到 $APP_DIR ..."
    if [ -d "$APP_DIR/.git" ]; then
      (cd "$APP_DIR" && git pull --ff-only)
    else
      mkdir -p "$(dirname "$APP_DIR")"
      git clone "$GIT_URL" "$APP_DIR"
    fi
  fi
  if [ ! -f "$APP_DIR/go.mod" ]; then
    log_err "未在 $APP_DIR 找到 go.mod，请先上传源码（或使用 --git <仓库URL>）"
    exit 1
  fi
  [ -f "$APP_DIR/configs/config.yaml" ] || log_warn "缺少 configs/config.yaml，首次启动会自动生成默认配置"
}

# ---------- 构建 ----------
build_app() {
  log_info "构建 ${APP_NAME}（npm 前端 + go 后端，需几分钟）..."
  if ! (cd "$APP_DIR" && make build); then
    log_warn "构建失败。若是 npm 下载慢/超时，可重试：npm config set registry https://registry.npmmirror.com"
    exit 1
  fi
  [ -x "$APP_DIR/bin/$APP_NAME" ] || { log_err "构建产物缺失: $APP_DIR/bin/$APP_NAME"; exit 1; }
  log_ok "构建完成: $APP_DIR/bin/$APP_NAME"
}

# ---------- 关闭 config 中的全局代理（--no-proxy） ----------
disable_proxy() {
  local cfg="$APP_DIR/configs/config.yaml"
  [ -f "$cfg" ] || return 0
  # 仅修改 proxy: 段内的 enabled: true -> false（用段范围限定，不影响 scheduler.enabled）
  sed -i '/^proxy:/,/^[a-zA-Z][a-zA-Z0-9_]*:/{s/^\([[:space:]]*enabled:\)[[:space:]]*true/\1 false/}' "$cfg"
  if sed -n '/^proxy:/,/^[a-zA-Z][a-zA-Z0-9_]*:/p' "$cfg" | grep -q 'enabled: false'; then
    log_ok "已关闭 configs/config.yaml 中的全局代理（proxy.enabled: false）"
  fi
}

# ---------- 配置 systemd ----------
setup_systemd() {
  log_info "配置 systemd 服务 ${SERVICE_NAME} ..."
  cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Outlook Manager (Hotmail/Outlook account automation)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${APP_DIR}
ExecStart=${APP_DIR}/bin/${APP_NAME}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable --now "${SERVICE_NAME}"
  sleep 4
  if systemctl is-active --quiet "${SERVICE_NAME}"; then
    log_ok "服务已启动并设为开机自启"
  else
    log_warn "服务启动失败，最近日志如下："
    journalctl -u "${SERVICE_NAME}" -n 30 --no-pager || true
  fi
}

# ---------- 前台运行（调试） ----------
run_foreground() {
  log_info "前台启动（Ctrl+C 退出）..."
  cd "$APP_DIR" && exec ./bin/${APP_NAME}
}

# ---------- 展示管理员信息 ----------
show_admin_info() {
  local pwd_file="$APP_DIR/data/initial_admin_password.txt"
  if [ -f "$pwd_file" ]; then
    echo
    echo -e "${YELLOW}  管理员账号（首次启动随机生成，仅此一份，请妥善保存）：${NC}"
    sed 's/^/    /' "$pwd_file"
    echo -e "${YELLOW}  建议登录后尽快修改密码并删除该文件：rm $pwd_file${NC}"
  else
    log_info "未发现初始密码文件（管理员账号已存在或非首次启动），使用已有账号登录即可"
  fi
}

# ---------- 完成提示 ----------
summary() {
  local ip port
  ip=$(hostname -I 2>/dev/null | awk '{print $1}')
  port=$(sed -n '/^server:/,/^[a-zA-Z][a-zA-Z0-9_]*:/p' "$APP_DIR/configs/config.yaml" 2>/dev/null | grep -oP 'port:\s*\K[0-9]+' | head -1)
  port="${port:-18327}"
  echo
  echo -e "${GREEN}============================================================${NC}"
  echo -e "${GREEN}  部署完成！${NC}"
  echo -e "  访问地址 : ${CYAN}http://${ip:-服务器IP}:${port}${NC}"
  echo -e "  服务管理 : systemctl status ${SERVICE_NAME}"
  echo -e "  查看日志 : journalctl -u ${SERVICE_NAME} -f"
  echo -e "  停止/重启: systemctl stop|restart ${SERVICE_NAME}"
  echo -e "${GREEN}============================================================${NC}"
}

# ---------- 主流程 ----------
main() {
  check_root
  fetch_source
  [ "$SKIP_DEPS" = 1 ] || { install_go; install_node; }
  build_app
  [ "$NO_PROXY" = 1 ] && disable_proxy
  if [ "$NO_SYSTEMD" = 1 ]; then
    run_foreground
  else
    setup_systemd
    show_admin_info
    summary
  fi
}

main "$@"
