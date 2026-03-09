const { app, BrowserWindow, ipcMain, Menu, dialog, shell } = require('electron');
const path = require('path');
const fs = require('fs');

// Keep a global reference of the window object to prevent garbage collection
let mainWindow;

// Configuration file path
const configPath = path.join(app.getPath('userData'), 'config.json');

// Default configuration
const defaultConfig = {
  apiBase: 'http://localhost:3001/api',
  windowWidth: 1400,
  windowHeight: 900,
  theme: 'light'
};

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

// Application lifecycle

app.whenReady().then(() => {
  createMainWindow();

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
