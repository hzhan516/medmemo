#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const docsDir = path.join(__dirname, '..', 'docs');
const i18nDir = path.join(docsDir, 'i18n', 'zh-Hans-CN');

function getMarkdownFiles(dir, base = dir) {
    const files = [];
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory() && entry.name !== 'i18n') {
            files.push(...getMarkdownFiles(fullPath, base));
        } else if (entry.isFile() && entry.name.endsWith('.md')) {
            files.push(path.relative(base, fullPath));
        }
    }
    return files;
}

const englishFiles = getMarkdownFiles(docsDir).filter(f => !f.startsWith('i18n' + path.sep));
let failed = false;

for (const enFile of englishFiles) {
    const zhFile = path.join(i18nDir, enFile);
    if (!fs.existsSync(zhFile)) {
        console.error(`Missing Chinese mirror: ${zhFile}`);
        failed = true;
    }
}

if (failed) {
    process.exit(1);
}

console.log(`All ${englishFiles.length} English docs have Chinese mirrors.`);
