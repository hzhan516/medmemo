#!/usr/bin/env node
/**
 * check-npm-audit-policy.js
 *
 * Fail-closed npm audit gate for MedMemo release hardening.
 *
 * Rules:
 *   - Any input file missing, empty or invalid JSON -> FAIL
 *   - npm audit top-level error / unsupported schema / tool failure -> FAIL
 *   - Any critical severity vulnerability -> FAIL
 *   - Any high severity vulnerability in production dependencies not covered by
 *     a reviewed production-scope allowlist -> FAIL
 *   - Any high severity vulnerability in the full audit not covered by the
 *     reviewed allowlist -> FAIL
 *   - Allowlist entries that are expired, package-mismatched, advisory-mismatched,
 *     scope-mismatched or no longer present in the audit -> FAIL
 *
 * Usage:
 *   node scripts/check-npm-audit-policy.js \
 *     --production /tmp/npm-audit-prod.json \
 *     --all /tmp/npm-audit-all.json \
 *     --allowlist scripts/npm-audit-allowlist.json
 */

const fs = require('fs');
const path = require('path');

const SUPPORTED_AUDIT_VERSION = 2;

function parseArgs(argv) {
  const args = {
    production: null,
    all: null,
    allowlist: null,
  };
  for (let i = 2; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--production') {
      args.production = argv[++i];
    } else if (arg === '--all') {
      args.all = argv[++i];
    } else if (arg === '--allowlist') {
      args.allowlist = argv[++i];
    }
  }
  return args;
}

function fail(message) {
  console.error(`ERROR: ${message}`);
  process.exit(1);
}

function readJSON(filePath, label) {
  if (!filePath) {
    fail(`${label} audit file path is required`);
  }
  let raw;
  try {
    raw = fs.readFileSync(filePath, 'utf8');
  } catch (err) {
    fail(`cannot read ${label} audit file ${filePath}: ${err.message}`);
  }
  if (!raw || raw.trim().length === 0) {
    fail(`${label} audit file ${filePath} is empty`);
  }
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    fail(`${label} audit file ${filePath} is not valid JSON: ${err.message}`);
  }
  return parsed;
}

function readAllowlist(filePath) {
  if (!filePath) {
    fail('allowlist file path is required');
  }
  let raw;
  try {
    raw = fs.readFileSync(filePath, 'utf8');
  } catch (err) {
    fail(`cannot read allowlist file ${filePath}: ${err.message}`);
  }
  if (!raw || raw.trim().length === 0) {
    fail(`allowlist file ${filePath} is empty`);
  }
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    fail(`allowlist file ${filePath} is not valid JSON: ${err.message}`);
  }
  if (!Array.isArray(parsed)) {
    fail(`allowlist file ${filePath} must contain a JSON array`);
  }
  return parsed;
}

function validateAuditSchema(data, label) {
  if (data.error) {
    const err = data.error;
    const summary = typeof err === 'string' ? err : JSON.stringify(err);
    fail(`${label} audit reported a top-level error: ${summary}`);
  }
  const version = data.auditReportVersion;
  if (typeof version !== 'number') {
    fail(`${label} audit is missing auditReportVersion`);
  }
  if (version !== SUPPORTED_AUDIT_VERSION) {
    fail(`${label} audit uses unsupported auditReportVersion ${version}; supported: ${SUPPORTED_AUDIT_VERSION}`);
  }
  if (!data.vulnerabilities || typeof data.vulnerabilities !== 'object') {
    fail(`${label} audit is missing vulnerabilities object`);
  }
  if (!data.metadata || typeof data.metadata !== 'object') {
    fail(`${label} audit is missing metadata object`);
  }
}

function isHigh(vuln) {
  return vuln.severity === 'high';
}

function isCritical(vuln) {
  return vuln.severity === 'critical';
}

function getAdvisories(vuln, allVulns) {
  const advisories = [];
  for (const item of vuln.via || []) {
    if (item && typeof item === 'object' && item.source) {
      advisories.push({
        source: String(item.source),
        severity: item.severity || vuln.severity,
        title: item.title || '',
        url: item.url || '',
        range: item.range || '',
      });
    } else if (typeof item === 'string' && allVulns && allVulns[item]) {
      // 继承被引用包的 advisory（例如 react-router-dom 通过 react-router 间接受影响）
      for (const adv of getAdvisories(allVulns[item], allVulns)) {
        advisories.push(adv);
      }
    }
  }
  return advisories;
}

function formatVuln(pkg, vuln) {
  const advisories = getAdvisories(vuln);
  const advStr = advisories
    .map(a => `#${a.source} (${a.severity}) ${a.title} range=${a.range}`)
    .join('; ');
  return `${pkg}@${vuln.range || 'unknown'} [${vuln.severity}]${advStr ? ' ' + advStr : ''}`;
}

function normalizeDate(input) {
  if (!input) return null;
  const d = new Date(input);
  if (isNaN(d.getTime())) {
    fail(`invalid expiry date "${input}" in allowlist`);
  }
  return d;
}

function validateAllowlistEntry(entry, index) {
  if (!entry || typeof entry !== 'object') {
    fail(`allowlist entry #${index} is not an object`);
  }
  if (typeof entry.package !== 'string' || entry.package.trim().length === 0) {
    fail(`allowlist entry #${index} is missing a non-empty "package" field`);
  }
  if (typeof entry.advisory !== 'string' || entry.advisory.trim().length === 0) {
    fail(`allowlist entry #${index} is missing a non-empty "advisory" field`);
  }
  if (typeof entry.justification !== 'string' || entry.justification.trim().length === 0) {
    fail(`allowlist entry #${index} is missing a non-empty "justification" field`);
  }
  if (typeof entry.mitigation !== 'string' || entry.mitigation.trim().length === 0) {
    fail(`allowlist entry #${index} is missing a non-empty "mitigation" field`);
  }
  if (typeof entry.reviewTargetVersion !== 'string' || entry.reviewTargetVersion.trim().length === 0) {
    fail(`allowlist entry #${index} is missing a non-empty "reviewTargetVersion" field`);
  }
  if (typeof entry.expires !== 'string' || entry.expires.trim().length === 0) {
    fail(`allowlist entry #${index} is missing a non-empty "expires" field`);
  }
  const scope = entry.scope || 'development';
  if (scope !== 'production' && scope !== 'development') {
    fail(`allowlist entry #${index} has invalid "scope" "${scope}"; must be "production" or "development"`);
  }
  const expiry = normalizeDate(entry.expires);
  return {
    package: entry.package.trim(),
    advisory: entry.advisory.trim(),
    justification: entry.justification.trim(),
    mitigation: entry.mitigation.trim(),
    reviewTargetVersion: entry.reviewTargetVersion.trim(),
    scope,
    expires: expiry,
  };
}

function buildAllowlistMap(entries, allVulns) {
  const map = new Map();
  for (let i = 0; i < entries.length; i++) {
    const normalized = validateAllowlistEntry(entries[i], i);
    const key = `${normalized.package}@${normalized.advisory}`;
    if (map.has(key)) {
      fail(`duplicate allowlist entry: ${key}`);
    }
    map.set(key, { ...normalized, index: i, used: false });
  }
  // 允许 getAdvisories 解析 via 中的字符串引用
  map.__allVulns = allVulns;
  return map;
}

function findAllowlistEntry(pkg, vuln, allowlistMap, now) {
  const advisories = getAdvisories(vuln, allowlistMap.__allVulns);
  if (advisories.length === 0) {
    return null;
  }
  for (const adv of advisories) {
    const key = `${pkg}@${adv.source}`;
    const entry = allowlistMap.get(key);
    if (entry) {
      entry.used = true;
      if (entry.expires && entry.expires <= now) {
        fail(`allowlist entry ${key} expired on ${entry.expires.toISOString().split('T')[0]}`);
      }
      return entry;
    }
  }
  return null;
}

function isAllowlisted(pkg, vuln, allowlistMap, now) {
  return findAllowlistEntry(pkg, vuln, allowlistMap, now) !== null;
}

function main() {
  const args = parseArgs(process.argv);
  const now = new Date();

  const prodData = readJSON(args.production, 'production');
  const allData = readJSON(args.all, 'all');
  const allowlistEntries = readAllowlist(args.allowlist);

  validateAuditSchema(prodData, 'production');
  validateAuditSchema(allData, 'all');

  const prodVulns = prodData.vulnerabilities;
  const allVulns = allData.vulnerabilities;

  const allowlistMap = buildAllowlistMap(allowlistEntries, allVulns);

  const criticals = [];
  const prodHighs = [];
  const unallowlistedHighs = [];

  for (const [pkg, vuln] of Object.entries(allVulns)) {
    if (isCritical(vuln)) {
      criticals.push(formatVuln(pkg, vuln));
    } else if (isHigh(vuln)) {
      const entry = findAllowlistEntry(pkg, vuln, allowlistMap, now);
      if (prodVulns[pkg]) {
        // 生产依赖的 high 必须命中明确标注 production scope 的 allowlist
        if (!entry || entry.scope !== 'production') {
          prodHighs.push(formatVuln(pkg, vuln));
        }
      } else if (!entry) {
        unallowlistedHighs.push(formatVuln(pkg, vuln));
      }
    }
  }

  for (const [pkg, vuln] of Object.entries(prodVulns)) {
    if (isHigh(vuln) && !allVulns[pkg]) {
      // Should not happen, but treat as unallowlisted high in production.
      prodHighs.push(formatVuln(pkg, vuln));
    }
  }

  // Detect stale allowlist entries (present in allowlist but not in audit).
  const stale = [];
  for (const [key, entry] of allowlistMap) {
    if (!entry.used) {
      stale.push(key);
    }
  }

  let failed = false;

  if (criticals.length > 0) {
    console.error('\nCritical severity vulnerabilities found:');
    for (const v of criticals) console.error(`  - ${v}`);
    failed = true;
  }

  if (prodHighs.length > 0) {
    console.error('\nHigh severity vulnerabilities in production dependencies found:');
    for (const v of prodHighs) console.error(`  - ${v}`);
    failed = true;
  }

  if (unallowlistedHighs.length > 0) {
    console.error('\nHigh severity vulnerabilities not covered by the reviewed allowlist:');
    for (const v of unallowlistedHighs) console.error(`  - ${v}`);
    failed = true;
  }

  if (stale.length > 0) {
    console.error('\nStale or mismatched allowlist entries (no matching vulnerability):');
    for (const s of stale) console.error(`  - ${s}`);
    failed = true;
  }

  const metaProd = prodData.metadata.vulnerabilities || {};
  const metaAll = allData.metadata.vulnerabilities || {};

  console.log('\nnpm audit policy check summary');
  console.log(`  production: ${metaProd.critical || 0} critical, ${metaProd.high || 0} high, ${metaProd.total || 0} total`);
  console.log(`  all:        ${metaAll.critical || 0} critical, ${metaAll.high || 0} high, ${metaAll.total || 0} total`);
  console.log(`  allowlist:  ${allowlistEntries.length} entries`);

  if (failed) {
    console.error('\nPolicy check FAILED.');
    process.exit(1);
  }

  console.log('\nPolicy check PASSED.');
}

main();
