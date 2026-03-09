const { contextBridge, ipcRenderer } = require('electron');

// Securely expose Electron APIs to the renderer process
contextBridge.exposeInMainWorld('electronAPI', {
  // Configuration related
  getConfig: () => ipcRenderer.invoke('get-config'),
  saveConfig: (config) => ipcRenderer.invoke('save-config', config),
  
  // External links
  openExternal: (url) => ipcRenderer.invoke('open-external', url),
  
  // File operations
  showSaveDialog: (options) => ipcRenderer.invoke('show-save-dialog', options),
  writeFile: (filePath, content) => ipcRenderer.invoke('write-file', filePath, content),
  
  // Menu event listeners
  onMenuNewChat: (callback) => ipcRenderer.on('menu-new-chat', callback),
  onMenuClearChat: (callback) => ipcRenderer.on('menu-clear-chat', callback),
  onMenuApiSettings: (callback) => ipcRenderer.on('menu-api-settings', callback),
  onMenuRefreshConfig: (callback) => ipcRenderer.on('menu-refresh-config', callback),
  
  // Remove listeners
  removeAllListeners: (channel) => ipcRenderer.removeAllListeners(channel)
});

// Expose platform information
contextBridge.exposeInMainWorld('platform', {
  isElectron: true,
  platform: process.platform,
  versions: process.versions
});