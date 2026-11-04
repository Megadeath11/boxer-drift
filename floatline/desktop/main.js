const { app, BrowserWindow, Menu } = require('electron');
const path = require('path');
app.whenReady().then(() => {
  Menu.setApplicationMenu(null);
  const win = new BrowserWindow({ width: 1400, height: 860, title: 'Floatline', backgroundColor: '#9fd8ff', autoHideMenuBar: true });
  win.maximize();
  win.loadFile(path.join(__dirname, 'app', 'index.html'));
  win.webContents.on('before-input-event', (e, input) => { if (input.type === 'keyDown' && input.key === 'F11') win.setFullScreen(!win.isFullScreen()); });
});
app.on('window-all-closed', () => app.quit());
