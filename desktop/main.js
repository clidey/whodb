const { app, BrowserWindow, Menu } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const fs = require('fs');
const os = require('os');

let mainWindow;
let backendProcess;
let isDev = false;
let tempExecutablePath = null;

// Initialize isDev using dynamic import
async function initializeIsDev() {
  try {
    const electronIsDev = await import('electron-is-dev');
    isDev = electronIsDev.default;
  } catch (error) {
    console.error('Failed to load electron-is-dev:', error);
    // Fallback to checking NODE_ENV
    isDev = process.env.NODE_ENV === 'development';
  }
}

// Backend executable name varies by platform
function getBackendExecutable() {
  const platform = process.platform;
  const arch = process.arch;
  
  if (platform === 'win32') {
    return 'whodb.exe';
  } else if (platform === 'darwin') {
    if (arch === 'arm64') {
      return 'whodb-darwin-arm64';
    } else {
      return 'whodb-darwin-amd64';
    }
  } else if (platform === 'linux') {
    if (arch === 'arm64') {
      return 'whodb-linux-arm64';
    } else {
      return 'whodb-linux-amd64';
    }
  }
  
  // Fallback
  return 'whodb';
}

// Start the Go backend
function startBackend() {
  let backendPath;
  
  if (isDev) {
    // Development mode - use executable from core/dist
    backendPath = path.join(__dirname, '..', 'core', 'dist', getBackendExecutable());
  } else {
    // Production mode - extract embedded executable to temp directory
    const tempDir = os.tmpdir();
    const tempExecutablePath = path.join(tempDir, getBackendExecutable());
    
    try {
      // Copy the embedded executable to temp directory
      const embeddedPath = path.join(process.resourcesPath, 'dist', getBackendExecutable());
      fs.copyFileSync(embeddedPath, tempExecutablePath);
      
      // Make executable on Unix systems
      if (process.platform !== 'win32') {
        fs.chmodSync(tempExecutablePath, '755');
      }
      
      backendPath = tempExecutablePath;
      tempExecutablePath = tempExecutablePath; // Store for cleanup
    } catch (error) {
      console.error('Failed to extract embedded executable:', error);
      app.quit();
      return;
    }
  }

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
app.whenReady().then(async () => {
  await initializeIsDev();
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
  
  // Clean up temporary executable
  if (tempExecutablePath && fs.existsSync(tempExecutablePath)) {
    try {
      fs.unlinkSync(tempExecutablePath);
    } catch (error) {
      console.error('Failed to clean up temporary executable:', error);
    }
  }
});