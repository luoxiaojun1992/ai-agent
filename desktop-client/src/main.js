const { app, BrowserWindow, ipcMain, Menu, dialog, shell } = require('electron');
const path = require('path');
const fs = require('fs');

// Keep a global reference of the window object to prevent garbage collection
let mainWindow;

// Configuration file path
const configPath = path.join(app.getPath('userData'), 'config.json');

// Scheduled tasks file path
const scheduledTasksPath = path.join(app.getPath('userData'), 'scheduled-tasks.json');

// Default configuration
const defaultConfig = {
  apiBase: 'http://localhost:3001/api',
  windowWidth: 1400,
  windowHeight: 900,
  theme: 'light'
};

// Default scheduled task structure
const defaultScheduledTasks = {
  tasks: []
};

// Scheduled tasks storage
let scheduledTasks = { tasks: [] };
let taskSchedulers = new Map();

// Load configuration
function loadConfig() {
  try {
    if (fs.existsSync(configPath)) {
      const data = fs.readFileSync(configPath, 'utf8');
      return { ...defaultConfig, ...JSON.parse(data) };
    }
  } catch (error) {
    console.error('Error loading config:', error);
  }
  return defaultConfig;
}

// Save configuration
function saveConfig(config) {
  try {
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
  } catch (error) {
    console.error('Error saving config:', error);
  }
}

// Load scheduled tasks
function loadScheduledTasks() {
  try {
    if (fs.existsSync(scheduledTasksPath)) {
      const data = fs.readFileSync(scheduledTasksPath, 'utf8');
      scheduledTasks = JSON.parse(data);
      console.log('Scheduled tasks loaded:', scheduledTasks.tasks.length);
    }
  } catch (error) {
    console.error('Error loading scheduled tasks:', error);
    scheduledTasks = { tasks: [] };
  }
  return scheduledTasks;
}

// Save scheduled tasks
function saveScheduledTasks(tasksData) {
  try {
    fs.writeFileSync(scheduledTasksPath, JSON.stringify(tasksData, null, 2));
    scheduledTasks = tasksData;
  } catch (error) {
    console.error('Error saving scheduled tasks:', error);
  }
}

// Start a scheduled task
function startScheduledTask(task) {
  if (taskSchedulers.has(task.id)) {
    // Already running
    return;
  }

  if (!task.enabled) {
    console.log(`Task ${task.id} is disabled, not starting`);
    return;
  }

  const config = loadConfig();
  const apiBase = config.apiBase || 'http://localhost:3001/api';

  let intervalMs;

  // Parse schedule interval
  switch (task.scheduleType) {
    case 'interval':
      intervalMs = task.intervalMinutes * 60 * 1000;
      break;
    case 'hourly':
      intervalMs = 60 * 60 * 1000;
      break;
    case 'daily':
      intervalMs = 24 * 60 * 60 * 1000;
      break;
    case 'weekly':
      intervalMs = 7 * 24 * 60 * 60 * 1000;
      break;
    default:
      intervalMs = task.intervalMinutes * 60 * 1000;
  }

  const executeTask = async () => {
    console.log(`Executing scheduled task: ${task.name}`);
    try {
      const response = await fetch(`${apiBase}/agent/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          message: task.message,
          stream: false
        })
      });

      let result = '';
      if (response.ok) {
        const data = await response.json();
        result = data.response || 'Task executed successfully';
      } else {
        result = `Error: HTTP ${response.status}`;
      }

      // Notify renderer process
      if (mainWindow && !mainWindow.isDestroyed()) {
        mainWindow.webContents.send('scheduled-task-executed', {
          taskId: task.id,
          taskName: task.name,
          result: result,
          timestamp: new Date().toISOString()
        });
      }

      // Save task execution history
      saveTaskExecutionHistory(task.id, result);

    } catch (error) {
      console.error(`Error executing task ${task.id}:`, error);
      if (mainWindow && !mainWindow.isDestroyed()) {
        mainWindow.webContents.send('scheduled-task-executed', {
          taskId: task.id,
          taskName: task.name,
          result: 'Error: ' + error.message,
          timestamp: new Date().toISOString(),
          isError: true
        });
      }
    }
  };

  // Start the interval
  const intervalId = setInterval(executeTask, intervalMs);
  taskSchedulers.set(task.id, { intervalId, task });

  // Execute immediately for first time if enabled
  if (task.executeImmediately) {
    setTimeout(executeTask, 1000);
  }

  console.log(`Started scheduled task: ${task.name} with interval: ${intervalMs}ms`);
}

// Stop a scheduled task
function stopScheduledTask(taskId) {
  const scheduler = taskSchedulers.get(taskId);
  if (scheduler) {
    clearInterval(scheduler.intervalId);
    taskSchedulers.delete(taskId);
    console.log(`Stopped scheduled task: ${taskId}`);
  }
}

// Initialize all scheduled tasks
function initializeScheduledTasks() {
  const tasksData = loadScheduledTasks();
  tasksData.tasks.forEach(task => {
    if (task.enabled) {
      startScheduledTask(task);
    }
  });
  console.log(`Initialized ${tasksData.tasks.filter(t => t.enabled).length} scheduled tasks`);
}

// Task execution history
const taskHistoryPath = path.join(app.getPath('userData'), 'task-history.json');

function saveTaskExecutionHistory(taskId, result) {
  try {
    let history = [];
    if (fs.existsSync(taskHistoryPath)) {
      const data = fs.readFileSync(taskHistoryPath, 'utf8');
      history = JSON.parse(data);
    }

    history.unshift({
      taskId,
      result,
      timestamp: new Date().toISOString()
    });

    // Keep only last 100 entries
    if (history.length > 100) {
      history = history.slice(0, 100);
    }

    fs.writeFileSync(taskHistoryPath, JSON.stringify(history, null, 2));
  } catch (error) {
    console.error('Error saving task execution history:', error);
  }
}

function getTaskHistory() {
  try {
    if (fs.existsSync(taskHistoryPath)) {
      const data = fs.readFileSync(taskHistoryPath, 'utf8');
      return JSON.parse(data);
    }
  } catch (error) {
    console.error('Error loading task history:', error);
  }
  return [];
}

// Create main window
function createMainWindow() {
  const config = loadConfig();
  
  mainWindow = new BrowserWindow({
    width: config.windowWidth,
    height: config.windowHeight,
    minWidth: 1000,
    minHeight: 700,
    title: 'AI Agent Desktop',
    icon: path.join(__dirname, '../assets/icon.png'),
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    },
    show: false, // Don't show until loaded
    titleBarStyle: 'default'
  });

  // Load local HTML file
  mainWindow.loadFile(path.join(__dirname, 'index.html'));

  // DevTools (only in development environment)
  // mainWindow.webContents.openDevTools();

  // Show window after loaded
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  // Save size when window closed
  mainWindow.on('close', () => {
    const bounds = mainWindow.getBounds();
    const config = loadConfig();
    config.windowWidth = bounds.width;
    config.windowHeight = bounds.height;
    saveConfig(config);
  });

  // Cleanup when window closed
  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Create application menu
  createApplicationMenu();
}

// Create application menu
function createApplicationMenu() {
  const template = [
    {
      label: 'File',
      submenu: [
        {
          label: 'New Chat',
          accelerator: 'CmdOrCtrl+N',
          click: () => {
            mainWindow.webContents.send('menu-new-chat');
          }
        },
        {
          label: 'Clear Chat History',
          accelerator: 'CmdOrCtrl+Shift+C',
          click: () => {
            mainWindow.webContents.send('menu-clear-chat');
          }
        },
        { type: 'separator' },
        {
          label: 'Quit',
          accelerator: process.platform === 'darwin' ? 'Cmd+Q' : 'Ctrl+Q',
          click: () => {
            app.quit();
          }
        }
      ]
    },
    {
      label: 'Edit',
      submenu: [
        { role: 'undo', label: 'Undo' },
        { role: 'redo', label: 'Redo' },
        { type: 'separator' },
        { role: 'cut', label: 'Cut' },
        { role: 'copy', label: 'Copy' },
        { role: 'paste', label: 'Paste' },
        { role: 'selectall', label: 'Select All' }
      ]
    },
    {
      label: 'View',
      submenu: [
        {
          label: 'Reload',
          accelerator: 'CmdOrCtrl+R',
          click: () => {
            mainWindow.webContents.reload();
          }
        },
        { type: 'separator' },
        {
          label: 'Developer Tools',
          accelerator: 'F12',
          click: () => {
            mainWindow.webContents.toggleDevTools();
          }
        },
        { type: 'separator' },
        { role: 'resetzoom', label: 'Reset Zoom' },
        { role: 'zoomin', label: 'Zoom In' },
        { role: 'zoomout', label: 'Zoom Out' },
        { type: 'separator' },
        { role: 'togglefullscreen', label: 'Toggle Full Screen' }
      ]
    },
    {
      label: 'Configuration',
      submenu: [
        {
          label: 'API Settings',
          click: () => {
            mainWindow.webContents.send('menu-api-settings');
          }
        },
        {
          label: 'Refresh Config',
          accelerator: 'CmdOrCtrl+Shift+R',
          click: () => {
            mainWindow.webContents.send('menu-refresh-config');
          }
        }
      ]
    },
    {
      label: 'Help',
      submenu: [
        {
          label: 'About AI Agent Desktop',
          click: () => {
            dialog.showMessageBox(mainWindow, {
              type: 'info',
              title: 'About AI Agent Desktop',
              message: 'AI Agent Desktop',
              detail: 'Version: 1.0.0\nBuilt with Electron\n\nA powerful AI Agent desktop client.'
            });
          }
        },
        {
          label: 'GitHub Repository',
          click: () => {
            shell.openExternal('https://github.com/luoxiaojun1992/ai-agent');
          }
        }
      ]
    }
  ];

  // macOS specific handling
  if (process.platform === 'darwin') {
    template.unshift({
      label: app.getName(),
      submenu: [
        { role: 'about', label: 'About' },
        { type: 'separator' },
        { role: 'services', label: 'Services' },
        { type: 'separator' },
        { role: 'hide', label: 'Hide' },
        { role: 'hideothers', label: 'Hide Others' },
        { role: 'unhide', label: 'Unhide All' },
        { type: 'separator' },
        { role: 'quit', label: 'Quit' }
      ]
    });
  }

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// IPC handlers

// Get configuration
ipcMain.handle('get-config', () => {
  return loadConfig();
});

// Save configuration
ipcMain.handle('save-config', (event, config) => {
  saveConfig(config);
  return true;
});

// Open external link
ipcMain.handle('open-external', (event, url) => {
  shell.openExternal(url);
});

// Show save dialog
ipcMain.handle('show-save-dialog', async (event, options) => {
  const result = await dialog.showSaveDialog(mainWindow, options);
  return result;
});

// Write file
ipcMain.handle('write-file', async (event, filePath, content) => {
  try {
    fs.writeFileSync(filePath, content, 'utf8');
    return { success: true };
  } catch (error) {
    return { success: false, error: error.message };
  }
});

// Scheduled tasks handlers

// Get all scheduled tasks
ipcMain.handle('get-scheduled-tasks', () => {
  return loadScheduledTasks();
});

// Save scheduled tasks
ipcMain.handle('save-scheduled-tasks', (event, tasksData) => {
  saveScheduledTasks(tasksData);
  return { success: true };
});

// Add a new scheduled task
ipcMain.handle('add-scheduled-task', (event, task) => {
  const tasksData = loadScheduledTasks();
  task.id = Date.now().toString();
  task.createdAt = new Date().toISOString();
  tasksData.tasks.push(task);
  saveScheduledTasks(tasksData);

  if (task.enabled) {
    startScheduledTask(task);
  }

  return { success: true, task };
});

// Update a scheduled task
ipcMain.handle('update-scheduled-task', (event, task) => {
  const tasksData = loadScheduledTasks();
  const index = tasksData.tasks.findIndex(t => t.id === task.id);

  if (index !== -1) {
    // Stop existing scheduler if running
    stopScheduledTask(task.id);

    tasksData.tasks[index] = { ...tasksData.tasks[index], ...task };
    saveScheduledTasks(tasksData);

    // Start if enabled
    if (task.enabled) {
      startScheduledTask(tasksData.tasks[index]);
    }

    return { success: true };
  }

  return { success: false, error: 'Task not found' };
});

// Delete a scheduled task
ipcMain.handle('delete-scheduled-task', (event, taskId) => {
  stopScheduledTask(taskId);

  const tasksData = loadScheduledTasks();
  tasksData.tasks = tasksData.tasks.filter(t => t.id !== taskId);
  saveScheduledTasks(tasksData);

  return { success: true };
});

// Toggle scheduled task enabled/disabled
ipcMain.handle('toggle-scheduled-task', (event, taskId) => {
  const tasksData = loadScheduledTasks();
  const task = tasksData.tasks.find(t => t.id === taskId);

  if (task) {
    task.enabled = !task.enabled;
    saveScheduledTasks(tasksData);

    if (task.enabled) {
      startScheduledTask(task);
    } else {
      stopScheduledTask(taskId);
    }

    return { success: true, enabled: task.enabled };
  }

  return { success: false, error: 'Task not found' };
});

// Get task execution history
ipcMain.handle('get-task-history', () => {
  return getTaskHistory();
});

// Manually execute a scheduled task
ipcMain.handle('execute-scheduled-task', async (event, taskId) => {
  const tasksData = loadScheduledTasks();
  const task = tasksData.tasks.find(t => t.id === taskId);

  if (!task) {
    return { success: false, error: 'Task not found' };
  }

  const config = loadConfig();
  const apiBase = config.apiBase || 'http://localhost:3001/api';

  try {
    const response = await fetch(`${apiBase}/agent/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        message: task.message,
        stream: false
      })
    });

    if (response.ok) {
      const data = await response.json();
      const result = data.response || 'Task executed successfully';

      // Notify renderer
      if (mainWindow && !mainWindow.isDestroyed()) {
        mainWindow.webContents.send('scheduled-task-executed', {
          taskId: task.id,
          taskName: task.name,
          result: result,
          timestamp: new Date().toISOString()
        });
      }

      saveTaskExecutionHistory(task.id, result);
      return { success: true, result };
    } else {
      return { success: false, error: `HTTP ${response.status}` };
    }
  } catch (error) {
    return { success: false, error: error.message };
  }
});

// Application lifecycle

app.whenReady().then(() => {
  createMainWindow();

  // Initialize scheduled tasks after window is created
  initializeScheduledTasks();

  app.on('activate', () => {
    // macOS: recreate window when dock icon clicked
    if (mainWindow === null) {
      createMainWindow();
    }
  });
});

app.on('window-all-closed', () => {
  // macOS: keep app running unless user explicitly quits
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// Prevent multiple instances
const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
  app.quit();
} else {
  app.on('second-instance', (event, commandLine, workingDirectory) => {
    // Focus first instance when trying to run second instance
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();
    }
  });
}
