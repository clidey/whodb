const { contextBridge, ipcRenderer } = require('electron');

// Expose protected methods that allow the renderer process to use
// a limited set of Electron APIs. This maintains security while 
// allowing necessary desktop integrations.
contextBridge.exposeInMainWorld('electronAPI', {
  // Platform information
  platform: process.platform,
  
  // Version information
  versions: {
    node: process.versions.node,
    chrome: process.versions.chrome,
    electron: process.versions.electron
  },
  
  // IPC communication for future desktop features
  send: (channel, data) => {
    // Whitelist allowed channels
    const validChannels = ['minimize', 'maximize', 'close'];
    if (validChannels.includes(channel)) {
      ipcRenderer.send(channel, data);
    }
  },
  
  on: (channel, callback) => {
    const validChannels = ['app-update'];
    if (validChannels.includes(channel)) {
      ipcRenderer.on(channel, callback);
    }
  }
});