const { app, BrowserWindow, Menu } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const isDev = require('electron-is-dev');

let mainWindow;
let backendProcess;

// Backend executable name varies by platform
function getBackendExecutable() {
  const platform = process.platform;
  if (platform === 'win32') {
    return 'whodb.exe';
  }
  return 'whodb';
}

// Start the Go backend
function startBackend() {
  const backendPath = isDev 
    ? path.join(__dirname, '..', 'core', getBackendExecutable())
    : path.join(process.resourcesPath, getBackendExecutable());

  console.log('Starting backend from:', backendPath);

  try {
    backendProcess = spawn(backendPath, [], {
      env: { ...process.env },
      stdio: 'inherit'
    });

    backendProcess.on('error', (error) => {
      console.error('Failed to start backend:', error);
      app.quit();
    });

    backendProcess.on('exit', (code) => {
      console.log(`Backend process exited with code ${code}`);
      if (code !== 0 && code !== null) {
        app.quit();
      }
    });
  } catch (error) {
    console.error('Error spawning backend:', error);
    app.quit();
  }
}

// Create the main application window
function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 800,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false
    },
    icon: path.join(__dirname, 'icon.png'),
    title: 'WhoDB'
  });

  // Wait a moment for the backend to start, then load the app
  setTimeout(() => {
    mainWindow.loadURL('http://localhost:8080');
  }, 2000);

  // Handle window closed
  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Handle failed load
  mainWindow.webContents.on('did-fail-load', (event, errorCode, errorDescription) => {
    console.error('Failed to load:', errorDescription);
    // Retry after a delay
    setTimeout(() => {
      mainWindow.loadURL('http://localhost:8080');
    }, 3000);
  });

  // Create application menu
  const template = [
    {
      label: 'File',
      submenu: [
        { role: 'quit' }
      ]
    },
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' }
      ]
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' },
        { role: 'forceReload' },
        { role: 'toggleDevTools' },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' }
      ]
    },
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'close' }
      ]
    },
    {
      label: 'Help',
      submenu: [
        {
          label: 'About WhoDB',
          click: () => {
            require('electron').shell.openExternal('https://github.com/clidey/whodb');
          }
        }
      ]
    }
  ];

  if (process.platform === 'darwin') {
    template.unshift({
      label: app.getName(),
      submenu: [
        { role: 'about' },
        { type: 'separator' },
        { role: 'services', submenu: [] },
        { type: 'separator' },
        { role: 'hide' },
        { role: 'hideOthers' },
        { role: 'unhide' },
        { type: 'separator' },
        { role: 'quit' }
      ]
    });
  }

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// App event handlers
app.whenReady().then(() => {
  startBackend();
  createWindow();
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  }
});

// Clean up backend process on quit
app.on('before-quit', () => {
  if (backendProcess) {
    backendProcess.kill();
  }
});