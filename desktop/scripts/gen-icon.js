// 生成 electron-builder 打包图标（256x256 PNG，builder 自动转换为 ico/icns）
const fs = require('fs')
const path = require('path')
const { appIcon } = require('../electron/icons')

const dir = path.join(__dirname, '..', 'build')
fs.mkdirSync(dir, { recursive: true })
const out = path.join(dir, 'icon.png')
fs.writeFileSync(out, appIcon(256))
console.log('icon generated:', out)
