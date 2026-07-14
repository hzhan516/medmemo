#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

// 统一术语表：key 为应避免的词，value 为推荐用词。
// 术语源：docs/glossary.md 与项目约定。
const terminology = {
    '鉴权': '认证',
    '已接受': '已采纳',
};

const docsDir = path.join(__dirname, '..', 'docs');
const files = [];

function collect(dir) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory() && entry.name !== 'node_modules' && entry.name !== 'i18n') {
            collect(full);
        } else if (entry.isFile() && entry.name.endsWith('.md')) {
            files.push(full);
        }
    }
}
collect(docsDir);

let failed = false;
for (const file of files) {
    const content = fs.readFileSync(file, 'utf-8');
    for (const [bad, good] of Object.entries(terminology)) {
        const re = new RegExp(bad, 'g');
        const matches = content.match(re);
        if (matches) {
            console.error(`${path.relative(process.cwd(), file)}: found "${bad}" (use "${good}") x${matches.length}`);
            failed = true;
        }
    }
}

if (failed) {
    process.exit(1);
}
console.log('Terminology check passed.');
