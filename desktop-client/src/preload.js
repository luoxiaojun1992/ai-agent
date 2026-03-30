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
  showOpenDialog: (options) => ipcRenderer.invoke('show-open-dialog', options),
  readFileAsBase64: (filePath) => ipcRenderer.invoke('read-file-as-base64', filePath),
  writeFile: (filePath, content) => ipcRenderer.invoke('write-file', filePath, content),

  // Scheduled tasks
  getScheduledTasks: () => ipcRenderer.invoke('get-scheduled-tasks'),
  saveScheduledTasks: (tasksData) => ipcRenderer.invoke('save-scheduled-tasks', tasksData),
  addScheduledTask: (task) => ipcRenderer.invoke('add-scheduled-task', task),
  updateScheduledTask: (task) => ipcRenderer.invoke('update-scheduled-task', task),
  deleteScheduledTask: (taskId) => ipcRenderer.invoke('delete-scheduled-task', taskId),
  toggleScheduledTask: (taskId) => ipcRenderer.invoke('toggle-scheduled-task', taskId),
  executeScheduledTask: (taskId) => ipcRenderer.invoke('execute-scheduled-task', taskId),
  getTaskHistory: () => ipcRenderer.invoke('get-task-history'),
  deleteTaskHistoryEntry: (historyId) => ipcRenderer.invoke('delete-task-history-entry', historyId),
  resolveScheduledTaskPreparation: (payload) => ipcRenderer.invoke('resolve-scheduled-task-preparation', payload),

  // Scheduled task execution listener
  onScheduledTaskExecuted: (callback) => ipcRenderer.on('scheduled-task-executed', (event, data) => callback(data)),
  onScheduledTaskBeforeExecute: (callback) => ipcRenderer.on('scheduled-task-before-execute', (event, data) => callback(data)),
  onScheduledTaskRunState: (callback) => ipcRenderer.on('scheduled-task-run-state', (event, data) => callback(data)),
  removeScheduledTaskListener: () => ipcRenderer.removeAllListeners('scheduled-task-executed'),

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
