#!/usr/bin/env node
const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const outPath = path.join(__dirname, 'go-licenses.md');
const rootDir = path.join(__dirname, '..', '..');

function runGoList() {
    return new Promise((resolve, reject) => {
        const proc = spawn('go', ['list', '-m', '-json', 'all'], {
            cwd: rootDir,
            env: { ...process.env, GOTOOLCHAIN: 'go1.26.4' },
        });

        const modules = [];
        let buffer = '';
        let depth = 0;
        let inString = false;
        let escape = false;

        proc.stdout.on('data', (chunk) => {
            const text = chunk.toString('utf-8');
            for (const ch of text) {
                buffer += ch;
                if (escape) {
                    escape = false;
                    continue;
                }
                if (ch === '\\') {
                    escape = true;
                    continue;
                }
                if (ch === '"') {
                    inString = !inString;
                    continue;
                }
                if (inString) continue;
                if (ch === '{') {
                    depth++;
                } else if (ch === '}') {
                    depth--;
                    if (depth === 0) {
                        try {
                            const m = JSON.parse(buffer.trim());
                            if (m.Path && m.Version && !m.Main) {
                                modules.push(m);
                            }
                        } catch (e) {
                            console.error('Failed to parse module JSON:', buffer.slice(0, 200));
                        }
                        buffer = '';
                    }
                }
            }
        });

        let stderr = '';
        proc.stderr.on('data', (chunk) => {
            stderr += chunk.toString('utf-8');
        });

        proc.on('close', (code) => {
            if (code !== 0) {
                console.error(stderr);
                reject(new Error(`go list exited with code ${code}`));
            } else {
                resolve(modules);
            }
        });
    });
}

function findLicenseFile(dir) {
    const names = ['LICENSE', 'LICENSE.md', 'LICENSE.txt', 'COPYING', 'NOTICE'];
    for (const name of names) {
        const p = path.join(dir, name);
        if (fs.existsSync(p)) return p;
    }
    return null;
}

function detectLicense(dir) {
    const file = findLicenseFile(dir);
    if (!file) return 'UNKNOWN';
    const text = fs.readFileSync(file, 'utf-8').toLowerCase();
    if (text.includes('mit license')) return 'MIT';
    if (text.includes('apache license, version 2.0') || text.includes('apache-2.0')) return 'Apache-2.0';
    if (text.includes('bsd 3-clause')) return 'BSD-3-Clause';
    if (text.includes('bsd 2-clause')) return 'BSD-2-Clause';
    if (text.includes('isc license')) return 'ISC';
    if (text.includes('mozilla public license, version 2.0') || text.includes('mpl-2.0')) return 'MPL-2.0';
    return 'UNKNOWN';
}

async function main() {
    const modules = await runGoList();
    const gopath = require('child_process').execSync('go env GOPATH', { encoding: 'utf-8', cwd: rootDir }).trim();
    const lines = [];

    for (const m of modules) {
        const modDir = m.Dir || path.join(gopath, 'pkg', 'mod', `${m.Path}@${m.Version}`);
        const license = fs.existsSync(modDir) ? detectLicense(modDir) : 'UNKNOWN';
        lines.push(`| ${m.Path} | ${m.Version} | ${license} |`);
    }

    fs.writeFileSync(outPath, lines.join('\n') + '\n');
    console.log(`Go licenses written to ${outPath} (${modules.length} modules).`);
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});
