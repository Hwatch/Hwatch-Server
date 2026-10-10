#!/usr/bin/env bash
# HWatch one-line installer.
#
#   curl -fsSL https://raw.githubusercontent.com/Hwatch/Hwatch-Server/main/install.sh | sudo bash
#
# Options (env or flags):
#   HWATCH_VERSION=v1.0.0        install a specific release (default: latest)
#   HWATCH_DIR=/opt/hwatch       install directory
#   HWATCH_PORT=8787             listen port
#   HWATCH_MEDIA_ROOT=/mnt/media media root shared with Plex
#   --no-deps                    skip installing aria2/wget/ffmpeg
#   --no-service                 do not install/start the systemd service
#   --yes                        non-interactive (default when piped)
set -euo pipefail

REPO="Hwatch/Hwatch-Server"
INSTALL_DIR="${HWATCH_DIR:-/opt/hwatch}"
DATA_DIR="$INSTALL_DIR/data"
WEB_DIR="$INSTALL_DIR/web"
PORT="${HWATCH_PORT:-8787}"
MEDIA_ROOT="${HWATCH_MEDIA_ROOT:-/mnt/media}"
VERSION="${HWATCH_VERSION:-latest}"
SERVICE_USER="hwatch"
SKIP_DEPS=0
NO_SERVICE=0

while [ $# -gt 0 ]; do
  case "$1" in
    --no-deps) SKIP_DEPS=1 ;;
    --no-service) NO_SERVICE=1 ;;
    --yes|-y) : ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done

log()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[!]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[x]\033[0m %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "请以 root 运行（或使用 sudo）。"

# --- prerequisites: curl + tar --------------------------------------------
if ! command -v curl >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then apt-get update -qq && apt-get install -y -qq curl;
  elif command -v dnf >/dev/null 2>&1; then dnf install -y -q curl;
  elif command -v apk >/dev/null 2>&1; then apk add --no-cache curl;
  else die "缺少 curl，请先安装。"; fi
fi

# --- architecture ----------------------------------------------------------
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "暂不支持架构 $(uname -m)（仅 amd64/arm64）。" ;;
esac

# --- resolve version -------------------------------------------------------
if [ "$VERSION" = "latest" ]; then
  log "查询最新版本…"
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
  [ -n "$VERSION" ] || die "无法获取最新版本，请用 HWATCH_VERSION=vX.Y.Z 指定。"
fi
log "安装 HWatch $VERSION (linux-$ARCH)"

ASSET="hwatch-${VERSION}-linux-${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"

# --- dependencies ----------------------------------------------------------
install_deps() {
  log "安装运行依赖：aria2 wget ffmpeg …（可用 --no-deps 跳过）"
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq aria2 wget ffmpeg ca-certificates tzdata
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y -q aria2 wget ffmpeg ca-certificates tzdata || \
      warn "ffmpeg 可能在 RPM Fusion，请自行安装。"
  elif command -v yum >/dev/null 2>&1; then
    yum install -y -q aria2 wget ffmpeg ca-certificates tzdata || warn "请自行安装 ffmpeg。"
  elif command -v pacman >/dev/null 2>&1; then
    pacman -Sy --noconfirm --needed aria2 wget ffmpeg ca-certificates tzdata
  elif command -v zypper >/dev/null 2>&1; then
    zypper --non-interactive install aria2 wget ffmpeg ca-certificates tzdata
  elif command -v apk >/dev/null 2>&1; then
    apk add --no-cache aria2 wget ffmpeg ca-certificates tzdata
  else
    warn "未识别的包管理器，请手动安装 aria2 / wget / ffmpeg。"
  fi
}
[ "$SKIP_DEPS" -eq 1 ] || install_deps

# --- download + verify + extract ------------------------------------------
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
log "下载 $ASSET"
curl -fsSL -o "$TMP/$ASSET" "$BASE/$ASSET" || die "下载失败：$BASE/$ASSET"
if curl -fsSL -o "$TMP/$ASSET.sha256" "$BASE/$ASSET.sha256" 2>/dev/null; then
  ( cd "$TMP" && sha256sum -c "$ASSET.sha256" ) || die "SHA256 校验失败。"
  log "SHA256 校验通过"
fi
EXTRACT="$TMP/hwatch-${VERSION}-linux-${ARCH}"
tar xzf "$TMP/$ASSET" -C "$TMP"
[ -d "$EXTRACT" ] || die "解压后未找到目录 $EXTRACT"

# --- install ---------------------------------------------------------------
log "安装到 $INSTALL_DIR"
id "$SERVICE_USER" >/dev/null 2>&1 || useradd -r -s /usr/sbin/nologin "$SERVICE_USER" 2>/dev/null || true
mkdir -p "$INSTALL_DIR" "$WEB_DIR" "$DATA_DIR"
install -m 0755 "$EXTRACT/r34watch" "$INSTALL_DIR/r34watch"
rm -rf "$WEB_DIR/dist"
cp -r "$EXTRACT/web/dist" "$WEB_DIR/dist"
cp "$EXTRACT/openapi.yaml" "$INSTALL_DIR/openapi.yaml" 2>/dev/null || true
chown -R "$SERVICE_USER":"$SERVICE_USER" "$INSTALL_DIR" 2>/dev/null || true

# --- systemd service -------------------------------------------------------
if [ "$NO_SERVICE" -eq 0 ] && command -v systemctl >/dev/null 2>&1; then
  log "配置并启动 systemd 服务 hwatch"
  cat >/etc/systemd/system/hwatch.service <<UNIT
[Unit]
Description=HWatch media subscription and Plex library service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
Environment=R34WATCH_ADDR=:$PORT
Environment=R34WATCH_DB=$DATA_DIR/r34watch.db
Environment=R34WATCH_WEB=$WEB_DIR/dist
Environment=R34WATCH_MEDIA_ROOT=$MEDIA_ROOT
ExecStart=$INSTALL_DIR/r34watch
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT
  systemctl daemon-reload
  systemctl enable --now hwatch
  systemctl is-active --quiet hwatch && log "服务已启动" || warn "服务启动失败，请查看：journalctl -u hwatch"
else
  warn "未配置服务（无 systemd 或 --no-service）。手动启动："
  echo "  R34WATCH_ADDR=:$PORT R34WATCH_DB=$DATA_DIR/r34watch.db R34WATCH_WEB=$WEB_DIR/dist $INSTALL_DIR/r34watch"
fi

IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
cat <<DONE

$(printf '\033[1;32m✔ HWatch %s 安装完成\033[0m' "$VERSION")

  访问：http://${IP:-<本机IP>}:$PORT
  数据：$DATA_DIR
  前端：$WEB_DIR/dist
  媒体：$MEDIA_ROOT（请在「设置」中确认/修改，需与 Plex 指向同一目录）
  服务：systemctl status hwatch   （日志：journalctl -u hwatch -f）

  首次打开网页需创建管理员账号。
DONE
