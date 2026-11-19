// 白名单式暴露原生能力，渲染进程无 Node 权限
const { contextBridge, ipcRenderer } = require('electron')

const cfg = ipcRenderer.sendSync('config:get')

contextBridge.exposeInMainWorld('neximConfig', {
  apiBase: cfg.apiBase,
  wsUrl: cfg.wsUrl,
  version: cfg.version,
  packaged: cfg.packaged
})

function on(channel, cb) {
  const handler = (_e, data) => cb(data)
  ipcRenderer.on(channel, handler)
  return () => ipcRenderer.removeListener(channel, handler)
}

contextBridge.exposeInMainWorld('desktop', {
  isDesktop: true,
  platform: process.platform,
  version: cfg.version,
  packaged: cfg.packaged,

  minimize: () => ipcRenderer.send('window:minimize'),
  maximize: () => ipcRenderer.send('window:maximize'),
  close: () => ipcRenderer.send('window:close'),

  setBadge: n => ipcRenderer.send('app:badge', n),
  notify: p => ipcRenderer.send('app:notify', p),
  onNotifyClick: cb => on('notify:click', cb),

  onSearch: cb => on('ui:search', cb),
  onScreenshot: cb => on('ui:screenshot', cb),
  onScreenshotError: cb => on('ui:screenshot-error', cb),
  onReadAll: cb => on('ui:read-all', cb),

  captureScreen: () => ipcRenderer.invoke('screenshot:capture'),
  chooseFile: () => ipcRenderer.invoke('file:choose'),
  readFile: p => ipcRenderer.invoke('file:read', p),
  saveTemp: (name, base64) => ipcRenderer.invoke('file:save-temp', { name, base64 }),
  openPath: p => ipcRenderer.invoke('shell:open', p),
  showInFolder: p => ipcRenderer.invoke('shell:reveal', p),

  getConfig: () => ipcRenderer.invoke('config:get-full'),
  setConfig: c => ipcRenderer.invoke('config:set', c),

  getAutoLaunch: () => ipcRenderer.invoke('app:auto-launch:get'),
  setAutoLaunch: v => ipcRenderer.invoke('app:auto-launch:set', v),

  checkUpdate: () => ipcRenderer.invoke('update:check'),
  installUpdate: () => ipcRenderer.invoke('update:install'),
  onUpdate: cb => on('update:status', cb)
})
