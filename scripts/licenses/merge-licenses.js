#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const scriptsDir = __dirname;
const root = path.join(scriptsDir, '..', '..');

const introPath = path.join(scriptsDir, 'intro.md');
if (!fs.existsSync(introPath)) {
    console.error(`Missing ${introPath}`);
    process.exit(1);
}
const intro = fs.readFileSync(introPath, 'utf-8');

const goTablePath = path.join(scriptsDir, 'go-licenses.md');
const nodeTablePath = path.join(scriptsDir, 'node-licenses.md');

if (!fs.existsSync(goTablePath)) {
    console.error(`Missing ${goTablePath}; run ./scripts/licenses/generate-go-licenses.sh first`);
    process.exit(1);
}
if (!fs.existsSync(nodeTablePath)) {
    console.error(`Missing ${nodeTablePath}; run node scripts/licenses/generate-node-licenses.js first`);
    process.exit(1);
}

const goTable = fs.readFileSync(goTablePath, 'utf-8');
const nodeTable = fs.readFileSync(nodeTablePath, 'utf-8');

const output = `${intro}\n## Go Dependencies\n\n| Package | Version | License |\n|---------|---------|---------|\n${goTable}\n## Node.js Dependencies\n\n${nodeTable}\n`;

fs.writeFileSync(path.join(root, 'THIRD_PARTY_LICENSES.md'), output);
fs.writeFileSync(path.join(root, 'docs', 'i18n', 'zh-Hans-CN', 'THIRD_PARTY_LICENSES.md'), output);
console.log('THIRD_PARTY_LICENSES.md generated (English + Chinese).');
