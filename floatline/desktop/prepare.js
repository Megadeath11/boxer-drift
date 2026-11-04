// Copies the game into app/ and points it at a bundled copy of three.js so the .exe works offline.
const fs = require('fs'), path = require('path');
const out = path.join(__dirname, 'app'); fs.mkdirSync(out, { recursive: true });
const html = fs.readFileSync(path.join(__dirname, '..', 'index.html'), 'utf8')
  .replace('https://cdn.jsdelivr.net/npm/three@0.160.0/build/three.module.js', './three.module.js');
fs.writeFileSync(path.join(out, 'index.html'), html);
fs.copyFileSync(path.join(__dirname, 'node_modules', 'three', 'build', 'three.module.js'), path.join(out, 'three.module.js'));
console.log('prepared app/');
