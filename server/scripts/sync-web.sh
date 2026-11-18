#!/usr/bin/env bash
# 将 web-h5 / web-admin 的构建产物同步到 server/assets，供 go:embed 打包进二进制
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

sync_dist() {
  local src="$1" dst="$2"
  [ -d "$src" ] || { echo "缺少构建产物：$src，请先 npm run build" >&2; exit 1; }
  rm -rf "$dst"; mkdir -p "$dst"
  cp -r "$src"/* "$dst"/
  echo "synced $src -> $dst"
}

sync_dist "$ROOT/../web-h5/dist" "$ROOT/webassets/dist/h5"
sync_dist "$ROOT/../web-admin/dist" "$ROOT/webassets/dist/admin"
echo "Web assets synced. Now run: go build -o nexim-server ./cmd/server"
