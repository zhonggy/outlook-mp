#!/usr/bin/env bash
#
# outlook-manager 一键部署脚本（Linux 服务器）
#
# 功能：拉取源码 → 安装 Go/Node → 构建 → 配置 systemd → 启动
#
# 用法：
#   sudo bash deploy.sh                           # 源码已在 /opt/outlook-manager
#   sudo bash deploy.sh --git <仓库URL>            # 从 git 仓库拉取源码
#   sudo bash deploy.sh --dir /path/to/src         # 指定源码目录
#   sudo bash deploy.sh --no-proxy                 # 关闭 config 全局代理
#   sudo bash deploy.sh --skip-deps                # 跳过 Go/Node 安装
# ============================================================
set -euo pipefail

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

RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; CYAN=$'\033[36m'; NC=$'\033[0m'
log_info()  { echo -e "${CYAN}[部署]${NC} $*"; }
log_ok()    { echo -e "${GREEN}[成功]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[警告]${NC} $*"; }
log_err()   { echo -e "${RED}[错误]${NC} $*" >&2; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --git)        GIT_URL="${2:-}"; [ -n "$GIT_URL" ] || { log_err "--git 缺少仓库地址"; exit 1; }; shift 2;;
    --dir)        APP_DIR="${2:-}"; [ -n "$APP_DIR" ] || { log_err "--dir 缺少路径"; exit 1; }; shift 2;;
    --no-proxy)   NO_PROXY=1; shift;;
    --skip-deps)  SKIP_DEPS=1; shift;;
    -h|--help)    sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0;;
    *)            log_err "未知参数: $1"; exit 1;;
  esac
done

ver_ge() { printf '%s\n%s\n' "$2" "$1" | sort -V -C; }

check_root() {
  if [ "$(id -u)" -ne 0 ]; then log_err "请用 sudo bash $0"; exit 1; fi
}

install_go() {
  if command -v go >/dev/null 2>&1 && ver_ge "$(go version | awk '{print $3}' | sed 's/^go//')" "$GO_MIN"; then
    log_ok "Go 已满足: $(go version)"; return
  fi
  log_info "安装 Go ..."
  local ver arch
  ver=$(curl -fsSL --max-time 20 "https://go.dev/VERSION?m=text" 2>/dev/null | head -1 || echo "go${GO_FALLBACK}")
  ver="${ver:-go${GO_FALLBACK}}"
  case "$(uname -m)" in x86_64|amd64) arch=amd64;; aarch64|arm64) arch=arm64;; *) arch=amd64;; esac
  curl -fsSL "https://go.dev/dl/${ver}.linux-${arch}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz && rm -f /tmp/go.tar.gz
  export PATH="/usr/local/go/bin:$PATH"
  echo 'export PATH=/usr/local/go/bin:$PATH' > /etc/profile.d/go.sh
  log_ok "Go 安装完成: $(go version)"
}

install_node() {
  if command -v node >/dev/null 2>&1 && ver_ge "$(node -v | tr -d 'v')" "$NODE_MIN"; then
    log_ok "Node 已满足: $(node -v)"; return
  fi
  log_info "安装 Node ${NODE_MAJOR} ..."
  export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
  if [ ! -s "$NVM_DIR/nvm.sh" ]; then
    curl -o- "https://raw.githubusercontent.com/nvm-sh/nvm/${NVM_VERSION}/install.sh" | bash
  fi
  . "$NVM_DIR/nvm.sh" && nvm install "$NODE_MAJOR" >/dev/null && nvm alias default "$NODE_MAJOR" >/dev/null && nvm use default >/dev/null
  log_ok "Node 安装完成: $(node -v)"
}

fetch_source() {
  if [ -n "$GIT_URL" ]; then
    log_info "从 $GIT_URL 拉取源码 ..."
    if [ -d "$APP_DIR/.git" ]; then (cd "$APP_DIR" && git pull --ff-only); else git clone "$GIT_URL" "$APP_DIR"; fi
  fi
  [ -f "$APP_DIR/go.mod" ] || { log_err "未找到 go.mod，请先上传源码或使用 --git"; exit 1; }
}

build_app() {
  log_info "构建 ${APP_NAME} ..."
  (cd "$APP_DIR" && make build) || { log_warn "构建失败，尝试 npm 镜像: npm config set registry https://registry.npmmirror.com"; exit 1; }
  [ -x "$APP_DIR/bin/$APP_NAME" ] || { log_err "构建产物缺失"; exit 1; }
  log_ok "构建完成"
}

disable_proxy() {
  local cfg="$APP_DIR/configs/config.yaml"
  [ -f "$cfg" ] || return 0
  sed -i '/^proxy:/,/^[a-zA-Z][a-zA-Z0-9_]*:/{s/^\([[:space:]]*enabled:\)[[:space:]]*true/\1 false/}' "$cfg"
  log_ok "已关闭 config 全局代理"
}

setup_systemd() {
  log_info "配置 systemd 服务 ..."
  cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Outlook Manager
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
  systemctl daemon-reload && systemctl enable --now "${SERVICE_NAME}" && sleep 4
  if systemctl is-active --quiet "${SERVICE_NAME}"; then
    log_ok "服务已启动并设为开机自启"
  else
    log_warn "服务启动失败，日志如下："; journalctl -u "${SERVICE_NAME}" -n 20 --no-pager || true
  fi
}

show_admin() {
  local pwd_file="$APP_DIR/data/initial_admin_password.txt"
  if [ -f "$pwd_file" ]; then
    echo; echo -e "${YELLOW}  管理员初始密码（登录后请删除此文件）：${NC}"
    sed 's/^/    /' "$pwd_file"
    echo -e "${YELLOW}  rm $pwd_file${NC}"
  fi
}

summary() {
  local ip port
  ip=$(hostname -I 2>/dev/null | awk '{print $1}')
  port=$(sed -n '/^server:/,/^[a-zA-Z][a-zA-Z0-9_]*:/p' "$APP_DIR/configs/config.yaml" 2>/dev/null | grep -oP 'port:\s*\K[0-9]+' | head -1)
  port="${port:-18327}"
  echo; echo -e "${GREEN}============================================================${NC}"
  echo -e "${GREEN}  部署完成！${NC}"
  echo -e "  访问地址 : ${CYAN}http://${ip:-服务器IP}:${port}${NC}"
  echo -e "  管理服务 : systemctl status ${SERVICE_NAME}"
  echo -e "  查看日志 : journalctl -u ${SERVICE_NAME} -f"
  echo -e "${GREEN}============================================================${NC}"
}

main() {
  check_root
  fetch_source
  [ "$SKIP_DEPS" = 1 ] || { install_go; install_node; }
  build_app
  [ "$NO_PROXY" = 1 ] && disable_proxy
  setup_systemd
  show_admin
  summary
}

main "$@"