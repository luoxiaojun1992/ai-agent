const { contextBridge, ipcRenderer } = require('electron');

// 安全地暴露Electron API给渲染进程
contextBridge.exposeInMainWorld('electronAPI', {
  // 配置相关
  getConfig: () => ipcRenderer.invoke('get-config'),
  saveConfig: (config) => ipcRenderer.invoke('save-config', config),
  
  // 外部链接
  openExternal: (url) => ipcRenderer.invoke('open-external', url),
  
  // 文件操作
  showSaveDialog: (options) => ipcRenderer.invoke('show-save-dialog', options),
  writeFile: (filePath, content) => ipcRenderer.invoke('write-file', filePath, content),
  
  // 菜单事件监听
  onMenuNewChat: (callback) => ipcRenderer.on('menu-new-chat', callback),
  onMenuClearChat: (callback) => ipcRenderer.on('menu-clear-chat', callback),
  onMenuApiSettings: (callback) => ipcRenderer.on('menu-api-settings', callback),
  onMenuRefreshConfig: (callback) => ipcRenderer.on('menu-refresh-config', callback),
  
  // 移除监听器
  removeAllListeners: (channel) => ipcRenderer.removeAllListeners(channel)
});

// 暴露平台信息
contextBridge.exposeInMainWorld('platform', {
  isElectron: true,
  platform: process.platform,
  versions: process.versions
});
