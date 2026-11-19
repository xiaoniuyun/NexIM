const {
  app, BrowserWindow, ipcMain, Tray, Menu, Notification, globalShortcut,
  desktopCapturer, dialog, shell, nativeImage, protocol, net
} = require('electron')
const path = require('path')
const fs = require('fs')
const os = require('os')
const { pathToFileURL } = require('node:url')
const { appIcon, dotIcon } = require('./icons')

const isDev = process.argv.includes('--dev') || process.env.NEXIM_DEV === '1'
const DEV_URL = process.env.NEXIM_DEV_URL || 'http://localhost:5174'

const DEFAULT_CONFIG = {
  apiBase: process.env.NEXIM_API_BASE || 'http://127.0.0.1:8080/api/v1',
  wsUrl: process.env.NEXIM_WS_URL || 'ws://127.0.0.1:8081/ws'
}

let win = null
let tray = null
let isQuitting = false
let unread = 0
let updater = null

// ---------- 配置 ----------
function configPath() { return path.join(app.getPath('userData'), 'config.json') }

function loadConfig() {
  try {
    const raw = JSON.parse(fs.readFileSync(configPath(), 'utf8'))
    return { ...DEFAULT_CONFIG, ...(raw || {}) }
  } catch (_) {
    return { ...DEFAULT_CONFIG }
  }
}
let config = null

function validUrl(v, protocols) {
  try {
    const u = new URL(v)
    return protocols.includes(u.protocol)
  } catch (_) { return false }
}

function saveConfig(patch = {}) {
  const next = { ...config }
  if (typeof patch.apiBase === 'string') {
    const v = patch.apiBase.trim().replace(/\/+$/, '')
    if (!validUrl(v, ['http:', 'https:'])) throw new Error('服务地址必须是以 http(s):// 开头的地址')
    next.apiBase = v
  }
  if (typeof patch.wsUrl === 'string') {
    const v = patch.wsUrl.trim()
    if (!validUrl(v, ['ws:', 'wss:'])) throw new Error('长连接地址必须是以 ws(s):// 开头的地址')
    next.wsUrl = v
  }
  fs.mkdirSync(path.dirname(configPath()), { recursive: true })
  fs.writeFileSync(configPath(), JSON.stringify(next, null, 2))
  config = next
  return next
}

// ---------- 图标 ----------
function iconPath(name, size, kind = 'app') {
  const dir = path.join(app.getPath('userData'), 'icons')
  fs.mkdirSync(dir, { recursive: true })
  const file = path.join(dir, `${name}-${size}.png`)
  if (!fs.existsSync(file)) {
    fs.writeFileSync(file, kind === 'dot' ? dotIcon(size) : appIcon(size))
  }
  return file
}

// ---------- 主窗口 ----------
function createWindow() {
  win = new BrowserWindow({
    width: 1240,
    height: 800,
    minWidth: 1000,
    minHeight: 640,
    show: false,
    backgroundColor: '#f5f7fb',
    autoHideMenuBar: true,
    icon: iconPath('app', 256),
    title: 'NexIM',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false
    }
  })

  if (isDev) {
    win.loadURL(DEV_URL)
    win.webContents.openDevTools({ mode: 'detach' })
  } else {
    win.loadURL('app://nexim/index.html')
  }

  win.once('ready-to-show', () => win.show())
  win.on('close', e => {
    // 关闭窗口最小化到托盘，托盘菜单可真正退出
    if (!isQuitting) {
      e.preventDefault()
      win.hide()
    }
  })
  win.on('closed', () => { win = null })
  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:/i.test(url)) shell.openExternal(url)
    return { action: 'deny' }
  })
}

function send(channel, payload) {
  if (win && !win.isDestroyed()) win.webContents.send(channel, payload)
}

function showWindow() {
  if (!win) return
  if (win.isMinimized()) win.restore()
  win.show()
  win.focus()
}

// ---------- 托盘 / 角标 ----------
function createTray() {
  const img = nativeImage.createFromPath(iconPath('tray', 32))
  tray = new Tray(img)
  tray.setToolTip('NexIM')
  tray.setContextMenu(Menu.buildFromTemplate([
    { label: '打开 NexIM', click: showWindow },
    { label: '全部标记已读', click: () => { showWindow(); send('ui:read-all') } },
    { type: 'separator' },
    {
      label: '退出',
      click: () => { isQuitting = true; app.quit() }
    }
  ]))
  tray.on('click', () => {
    if (!win) return createWindow()
    if (win.isVisible() && win.isFocused()) win.hide()
    else showWindow()
  })
}

function applyBadge(n) {
  unread = Math.max(0, Number(n) || 0)
  if (tray) tray.setToolTip(unread > 0 ? `NexIM · ${unread} 条未读` : 'NexIM')
  if (process.platform === 'darwin') app.dock?.setBadge(unread > 0 ? String(unread) : '')
  if (win && process.platform === 'win32') {
    try {
      win.setOverlayIcon(unread > 0 ? nativeImage.createFromPath(iconPath('dot', 16, 'dot')) : null, unread > 0 ? `${unread} 条未读` : '')
    } catch (_) {}
  }
}

// ---------- 截图 ----------
async function captureScreen() {
  const sources = await desktopCapturer.getSources({ types: ['screen'], thumbnailSize: { width: 1920, height: 1080 } })
  const src = sources[0]
  if (!src) throw new Error('未获取到屏幕画面')
  const png = src.thumbnail.toPNG()
  const file = path.join(os.tmpdir(), `nexim-shot-${Date.now()}.png`)
  fs.writeFileSync(file, png)
  return { path: file, name: path.basename(file) }
}

// ---------- 自动更新 ----------
function initUpdater() {
  // 通过 electron.exe 直接运行时 isPackaged 可能仍为 true，需同时排除 defaultApp
  if (!app.isPackaged || process.defaultApp) return
  try {
    const { autoUpdater } = require('electron-updater')
    updater = autoUpdater
    autoUpdater.autoDownload = true
    autoUpdater.on('update-available', info => send('update:status', { status: 'available', version: info.version }))
    autoUpdater.on('update-not-available', () => send('update:status', { status: 'latest' }))
    autoUpdater.on('download-progress', p => send('update:status', { status: 'downloading', percent: Math.round(p.percent) }))
    autoUpdater.on('update-downloaded', info => send('update:status', { status: 'downloaded', version: info.version }))
    autoUpdater.on('error', err => send('update:status', { status: 'error', message: err.message }))
    setTimeout(() => autoUpdater.checkForUpdates().catch(() => {}), 5000)
  } catch (_) {}
}

// ---------- 单实例 ----------
const gotLock = app.requestSingleInstanceLock()
if (!gotLock) {
  app.quit()
} else {
  app.on('second-instance', showWindow)

  protocol.registerSchemesAsPrivileged([{
    scheme: 'app',
    privileges: { standard: true, secure: true, supportFetchAPI: true, corsEnabled: true }
  }])

  app.whenReady().then(() => {
    if (process.platform === 'win32') app.setAppUserModelId('im.nex.desktop')
    config = loadConfig()

    // 生产环境用 app:// 提供渲染层，保证 localStorage / fetch / WebSocket 的稳定来源
    protocol.handle('app', request => {
      const u = new URL(request.url)
      let pathname = decodeURIComponent(u.pathname)
      if (!pathname || pathname === '/') pathname = '/index.html'
      const file = path.join(__dirname, '..', 'dist', path.normalize(pathname).replace(/^(\.\.[/\\])+/, ''))
      return net.fetch(pathToFileURL(file).toString())
    })

    registerIpc()
    createWindow()
    createTray()
    initUpdater()

    const bind = (accel, fn) => { try { globalShortcut.register(accel, fn) } catch (_) {} }
    bind('CommandOrControl+K', () => { showWindow(); send('ui:search') })
    bind('CommandOrControl+Shift+A', async () => {
      try {
        const shot = await captureScreen()
        showWindow()
        send('ui:screenshot', shot)
      } catch (e) {
        send('ui:screenshot-error', e.message)
      }
    })

    app.on('activate', () => { if (!win) createWindow(); else showWindow() })
  })

  app.on('before-quit', () => { isQuitting = true })
  app.on('will-quit', () => globalShortcut.unregisterAll())
  // 关闭窗口后常驻托盘，不退出应用
  app.on('window-all-closed', () => {})
}

// ---------- IPC ----------
function registerIpc() {
  ipcMain.on('config:get', e => {
    e.returnValue = {
      apiBase: config.apiBase,
      wsUrl: config.wsUrl,
      version: app.getVersion(),
      packaged: app.isPackaged
    }
  })

  ipcMain.handle('config:get-full', () => ({ ...config, version: app.getVersion(), packaged: app.isPackaged }))
  ipcMain.handle('config:set', (_e, patch) => {
    try { return { ok: true, config: saveConfig(patch || {}) } }
    catch (err) { return { ok: false, message: err.message } }
  })

  ipcMain.on('window:minimize', () => win?.minimize())
  ipcMain.on('window:maximize', () => {
    if (!win) return
    win.isMaximized() ? win.unmaximize() : win.maximize()
  })
  ipcMain.on('window:close', () => win?.close())

  ipcMain.on('app:badge', (_e, n) => applyBadge(n))
  ipcMain.on('app:notify', (_e, p) => {
    if (!Notification.isSupported()) return
    const n = new Notification({
      title: p?.title || 'NexIM',
      body: p?.body || '您有新消息',
      icon: iconPath('app', 128),
      silent: !!p?.silent
    })
    n.on('click', () => {
      showWindow()
      if (p?.convId) send('notify:click', { convId: p.convId })
    })
    n.show()
  })

  ipcMain.handle('screenshot:capture', () => captureScreen())
  ipcMain.handle('file:choose', async () => {
    const r = await dialog.showOpenDialog(win, { properties: ['openFile', 'multiSelections'] })
    if (r.canceled) return []
    return r.filePaths
  })
  ipcMain.handle('file:read', async (_e, p) => {
    if (!p) throw new Error('缺少文件路径')
    const buf = await fs.promises.readFile(p)
    return buf.toString('base64')
  })
  ipcMain.handle('file:save-temp', async (_e, { name, base64 }) => {
    const safe = String(name || 'file').replace(/[\\/:*?"<>|]/g, '_')
    const file = path.join(os.tmpdir(), `${Date.now()}-${safe}`)
    await fs.promises.writeFile(file, Buffer.from(base64 || '', 'base64'))
    return file
  })
  ipcMain.handle('shell:open', (_e, p) => shell.openPath(p))
  ipcMain.handle('shell:reveal', (_e, p) => { shell.showItemInFolder(p); return true })

  ipcMain.handle('app:auto-launch:get', () => app.getLoginItemSettings().openAtLogin)
  ipcMain.handle('app:auto-launch:set', (_e, v) => {
    app.setLoginItemSettings({ openAtLogin: !!v, path: process.execPath })
    return app.getLoginItemSettings().openAtLogin
  })

  ipcMain.handle('update:check', async () => {
    if (!app.isPackaged || !updater) return { status: 'dev', message: '开发模式不检查更新' }
    try {
      const r = await updater.checkForUpdates()
      return { status: 'checking', version: r?.updateInfo?.version }
    } catch (e) {
      return { status: 'error', message: e.message }
    }
  })
  ipcMain.handle('update:install', () => { if (updater) { isQuitting = true; updater.quitAndInstall() } })
}
