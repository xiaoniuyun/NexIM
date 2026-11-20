# 将 web-admin 的构建产物同步到 server/webassets/dist，供 go:embed 打包进二进制
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot

function Sync-Dist($src, $dst) {
    if (-not (Test-Path $src)) { throw "缺少构建产物：$src，请先 npm run build" }
    if (Test-Path $dst) { Remove-Item $dst -Recurse -Force }
    New-Item -ItemType Directory -Path $dst -Force | Out-Null
    Copy-Item -Path (Join-Path $src '*') -Destination $dst -Recurse -Force
    Write-Host "synced $src -> $dst"
}

Sync-Dist (Join-Path $root '..\web-admin\dist') (Join-Path $root 'webassets\dist\admin')
Write-Host 'Web assets synced. Now run: go build -o nexim-server ./cmd/server'
