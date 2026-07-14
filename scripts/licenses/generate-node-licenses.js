#!/usr/bin/env node
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const outPath = path.join(__dirname, 'node-licenses.md');
const webDir = path.join(__dirname, '..', '..', 'web');

const raw = execSync('npx --yes license-checker --production --json', {
    cwd: webDir,
    encoding: 'utf-8',
    stdio: ['pipe', 'pipe', 'inherit'],
});

const packages = JSON.parse(raw);

const lines = [
    '| Package | Version | License | Repository |',
    '|---------|---------|---------|------------|',
];

for (const [pkg, info] of Object.entries(packages)) {
    // license-checker keys are like "package@version"
    const atIndex = pkg.lastIndexOf('@');
    const name = pkg.slice(0, atIndex);
    const version = pkg.slice(atIndex + 1);
    const license = Array.isArray(info.licenses) ? info.licenses.join(', ') : (info.licenses || 'UNKNOWN');
    lines.push(`| ${name} | ${version} | ${license} | ${info.repository || ''} |`);
}

fs.writeFileSync(outPath, lines.join('\n') + '\n');
console.log(`Node licenses written to ${outPath} (${Object.keys(packages).length} packages).`);
