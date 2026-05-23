#!/usr/bin/env node
/**
 * Provider 模板 JSON 验证脚本。
 * 校验 assets/provider-templates.json 的格式和必填字段完整性。
 * 在 CI 中执行，阻断格式错误的提交。
 */

const fs = require('fs')
const path = require('path')

const TEMPLATE_PATH = path.join(__dirname, '..', 'web', 'public', 'assets', 'provider-templates.json')

const REQUIRED_FIELDS = ['id', 'name', 'apiHost', 'defaultModel', 'models', 'description', 'docsUrl', 'type']
const VALID_TYPES = new Set(['cloud', 'local'])

function exitError(message) {
  console.error(`❌ ${message}`)
  process.exit(1)
}

function exitSuccess(message) {
  console.log(`✅ ${message}`)
  process.exit(0)
}

// 读取文件
let raw
try {
  raw = fs.readFileSync(TEMPLATE_PATH, 'utf-8')
} catch (err) {
  exitError(`无法读取模板文件: ${err.message}`)
}

// 解析 JSON
let templates
try {
  templates = JSON.parse(raw)
} catch (err) {
  exitError(`JSON 解析失败: ${err.message}`)
}

// 校验类型
if (!Array.isArray(templates)) {
  exitError('根节点必须是数组')
}

if (templates.length === 0) {
  exitError('模板列表不能为空')
}

// 校验每个模板
const seenIds = new Set()
let errorCount = 0

for (let i = 0; i < templates.length; i++) {
  const t = templates[i]
  const prefix = `模板[${i}]`

  // 校验必填字段
  for (const field of REQUIRED_FIELDS) {
    if (!(field in t)) {
      console.error(`❌ ${prefix} 缺少必填字段: ${field}`)
      errorCount++
    }
  }

  // 校验字段类型
  if (typeof t.id !== 'string' || t.id.trim() === '') {
    console.error(`❌ ${prefix} id 必须是非空字符串`)
    errorCount++
  }
  if (typeof t.name !== 'string' || t.name.trim() === '') {
    console.error(`❌ ${prefix} name 必须是非空字符串`)
    errorCount++
  }
  if (typeof t.apiHost !== 'string' || t.apiHost.trim() === '') {
    console.error(`❌ ${prefix} apiHost 必须是非空字符串`)
    errorCount++
  }
  if (typeof t.defaultModel !== 'string') {
    console.error(`❌ ${prefix} defaultModel 必须是字符串`)
    errorCount++
  }
  if (!Array.isArray(t.models)) {
    console.error(`❌ ${prefix} models 必须是数组`)
    errorCount++
  }
  if (typeof t.description !== 'string' || t.description.trim() === '') {
    console.error(`❌ ${prefix} description 必须是非空字符串`)
    errorCount++
  }
  if (typeof t.docsUrl !== 'string' || t.docsUrl.trim() === '') {
    console.error(`❌ ${prefix} docsUrl 必须是非空字符串`)
    errorCount++
  }
  if (!VALID_TYPES.has(t.type)) {
    console.error(`❌ ${prefix} type 必须是 "cloud" 或 "local"，当前值: ${t.type}`)
    errorCount++
  }

  // 校验 ID 唯一性
  if (t.id) {
    if (seenIds.has(t.id)) {
      console.error(`❌ ${prefix} id 重复: ${t.id}`)
      errorCount++
    }
    seenIds.add(t.id)
  }
}

if (errorCount > 0) {
  exitError(`校验失败，共 ${errorCount} 个错误`)
}

exitSuccess(`校验通过，共 ${templates.length} 个 Provider 模板`)
