#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const root = path.join(__dirname, '..');

function readJSON(p) {
    return JSON.parse(fs.readFileSync(path.join(root, p), 'utf-8'));
}

const wails = readJSON('wails.json');
const webPkg = readJSON('web/package.json');
const changelog = readJSON('internal/domain/entity/changelog/zh-Hans.json');

const wailsVersion = wails.info.productVersion;
const webVersion = webPkg.version;
const changelogHasVersion = Array.isArray(changelog) && changelog.some(v => {
    const cv = String(v.version);
    return cv === wailsVersion || cv === `v${wailsVersion}`;
});

let failed = false;
if (wailsVersion !== webVersion) {
    console.error(`Version mismatch: wails.json=${wailsVersion}, web/package.json=${webVersion}`);
    failed = true;
}
if (!changelogHasVersion) {
    console.error(`Changelog missing current version ${wailsVersion}`);
    failed = true;
}

if (failed) {
    process.exit(1);
}
console.log(`Version consistency OK: ${wailsVersion}`);
