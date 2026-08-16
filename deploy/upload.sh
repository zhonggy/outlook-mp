#!/usr/bin/env bash
#
# ============================================================
#  outlook-manager 源码上传脚本（在本机运行）
#
#  用法:
#    ./upload.sh user@服务器IP            # 默认 ssh 端口 22
#    ./upload.sh user@服务器IP 2222       # 指定 ssh 端口
#
#  上传后登录服务器执行:
#    sudo bash /opt/outlook-manager/deploy/deploy.sh
# ============================================================
set -euo pipefail

TARGET="${1:?用法: ./upload.sh user@服务器IP [ssh端口]}"
SSH_PORT="${2:-22}"

# 脚本位于项目 deploy/ 下，项目根在其上一级
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REMOTE_DIR="/opt/outlook-manager"

[ -f "$ROOT_DIR/go.mod" ] || { echo "错误: 未找到项目根目录（$ROOT_DIR/go.mod 不存在）"; exit 1; }

echo "==> 上传源码到 ${TARGET}:${REMOTE_DIR} ..."

if command -v rsync >/dev/null 2>&1; then
  # rsync 增量传输（首选）
  rsync -az --delete \
    --exclude=.git --exclude=.gitignore \
    --exclude=node_modules --exclude=web/node_modules \
    --exclude=bin --exclude=data \
    --exclude=web/dist --exclude=*.tsbuildinfo \
    --exclude=.DS_Store \
    -e "ssh -p $SSH_PORT" \
    "$ROOT_DIR/" "${TARGET}:${REMOTE_DIR}/"
else
  # 无 rsync 时用 ssh + tar 管道（git bash 自带 tar/ssh，同样可用）
  ssh -p "$SSH_PORT" "$TARGET" "mkdir -p $REMOTE_DIR"
  tar --exclude=.git --exclude=node_modules --exclude=web/node_modules \
      --exclude=bin --exclude=data --exclude=web/dist \
      --exclude='*.tsbuildinfo' --exclude=.DS_Store \
      -C "$ROOT_DIR" -cf - . \
    | ssh -p "$SSH_PORT" "$TARGET" "tar -C $REMOTE_DIR -xf -"
fi

echo "==> 上传完成 ✅"
echo "==> 下一步，登录服务器执行："
echo "    sudo bash ${REMOTE_DIR}/deploy/deploy.sh"
echo "    # 服务器无本地代理时建议加 --no-proxy："
echo "    sudo bash ${REMOTE_DIR}/deploy/deploy.sh --no-proxy"
