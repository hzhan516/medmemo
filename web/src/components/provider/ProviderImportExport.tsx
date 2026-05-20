import { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import { X, Upload, Download, FileJson, AlertTriangle, CheckCircle, AlertCircle } from 'lucide-react'
import { useProviderStore, type ImportResult, type ImportMode } from '@/stores/providerStore'
import type { ProviderConfig } from '@/types/provider'

interface ProviderImportExportProps {
  open: boolean
  onClose: () => void
}

interface ValidationError {
  index: number
  message: string
}

/**
 * Provider 配置导入导出组件。
 * 支持导出为 JSON 文件（可选包含 API Key、按分组筛选），
 * 支持从 JSON 文件导入（合并/覆盖模式、格式验证、冲突检测）。
 */
export function ProviderImportExport({ open, onClose }: ProviderImportExportProps) {
  const providers = useProviderStore((s) => s.providers)
  const importProviders = useProviderStore((s) => s.importProviders)
  const replaceAllProviders = useProviderStore((s) => s.replaceAllProviders)

  // 导出状态
  const [includeApiKey, setIncludeApiKey] = useState(false)
  const existingGroups = useMemo(() => {
    const groups = new Set<string>()
    for (const p of providers) groups.add(p.group)
    return Array.from(groups).sort()
  }, [providers])
  const [selectedGroups, setSelectedGroups] = useState<Set<string>>(new Set())

  // 导入状态
  const [importMode, setImportMode] = useState<ImportMode>('merge')
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importPreview, setImportPreview] = useState<ProviderConfig[] | null>(null)
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([])
  const [importResult, setImportResult] = useState<ImportResult | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const prevOpenRef = useRef(false)

  // 弹窗刚打开时重置导入相关状态
  useEffect(() => {
    if (open && !prevOpenRef.current) {
      setImportFile(null)
      setImportPreview(null)
      setValidationErrors([])
      setImportResult(null)
      setImportMode('merge')
    }
    prevOpenRef.current = open
  }, [open])

  // 弹窗打开且分组变化时更新导出分组选择
  useEffect(() => {
    if (open && existingGroups.length > 0) {
      setSelectedGroups(new Set(existingGroups))
    }
  }, [open, existingGroups])

  // ---- 导出逻辑 ----
  const handleExport = useCallback(() => {
    const groupsToExport = selectedGroups.size > 0 ? selectedGroups : new Set(existingGroups)
    const filtered = providers.filter((p) => groupsToExport.has(p.group))
    const exportData = {
      version: '1.0',
      exportedAt: new Date().toISOString(),
      providers: filtered.map((p) => ({
        ...p,
        apiKey: includeApiKey ? p.apiKey : '',
        needsApiKey: !includeApiKey || !p.apiKey,
      })),
    }
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    const dateStr = new Date().toISOString().slice(0, 10)
    a.href = url
    a.download = `medmemo-providers-${dateStr}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }, [providers, selectedGroups, existingGroups, includeApiKey])

  const toggleGroup = (group: string) => {
    setSelectedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(group)) next.delete(group)
      else next.add(group)
      return next
    })
  }

  // ---- 导入逻辑 ----
  const validateAndPreview = useCallback(
    (content: string) => {
      const errors: ValidationError[] = []
      let parsed: unknown
      try {
        parsed = JSON.parse(content)
      } catch {
        errors.push({ index: -1, message: 'JSON 解析失败，文件格式不正确' })
        setValidationErrors(errors)
        setImportPreview(null)
        return
      }

      if (!parsed || typeof parsed !== 'object') {
        errors.push({ index: -1, message: 'JSON 根节点必须是对象' })
        setValidationErrors(errors)
        setImportPreview(null)
        return
      }

      const data = parsed as Record<string, unknown>
      const providersArray = data.providers
      if (!Array.isArray(providersArray)) {
        errors.push({ index: -1, message: '缺少 providers 数组字段' })
        setValidationErrors(errors)
        setImportPreview(null)
        return
      }

      const preview: ProviderConfig[] = []
      for (let i = 0; i < providersArray.length; i++) {
        const item = providersArray[i]
        if (!item || typeof item !== 'object') {
          errors.push({ index: i, message: `第 ${i + 1} 条记录不是对象` })
          continue
        }
        const record = item as Record<string, unknown>
        const missing: string[] = []
        if (!record.name || typeof record.name !== 'string') missing.push('name')
        if (!record.apiHost || typeof record.apiHost !== 'string') missing.push('apiHost')
        if (!record.modelId || typeof record.modelId !== 'string') missing.push('modelId')
        if (missing.length > 0) {
          errors.push({ index: i, message: `第 ${i + 1} 条记录缺少必填字段：${missing.join('、')}` })
          continue
        }
        preview.push(record as unknown as ProviderConfig)
      }

      setValidationErrors(errors)
      setImportPreview(preview)
    },
    []
  )

  const handleFileSelect = useCallback(
    (file: File) => {
      setImportFile(file)
      setImportResult(null)
      const reader = new FileReader()
      reader.onload = (e) => {
        const content = e.target?.result as string
        if (content) validateAndPreview(content)
      }
      reader.readAsText(file)
    },
    [validateAndPreview]
  )

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      setIsDragging(false)
      const file = e.dataTransfer.files[0]
      if (file && file.type === 'application/json') {
        handleFileSelect(file)
      } else if (file) {
        setValidationErrors([{ index: -1, message: '请上传 .json 格式的文件' }])
        setImportPreview(null)
        setImportFile(null)
      }
    },
    [handleFileSelect]
  )

  const handleImport = useCallback(() => {
    if (!importPreview || importPreview.length === 0) return
    const configs = importPreview.map((p) => ({
      templateId: (p.templateId as string) || 'custom',
      name: p.name,
      apiHost: p.apiHost,
      apiKey: p.apiKey || '',
      modelId: p.modelId,
      temperature: typeof p.temperature === 'number' ? p.temperature : 0.7,
      timeoutMs: typeof p.timeoutMs === 'number' ? p.timeoutMs : 30000,
      maxRetries: typeof p.maxRetries === 'number' ? p.maxRetries : 3,
      group: (p.group as string) || '默认',
      enabled: typeof p.enabled === 'boolean' ? p.enabled : true,
      sortOrder: typeof p.sortOrder === 'number' ? p.sortOrder : 0,
    }))

    let result: ImportResult
    if (importMode === 'overwrite') {
      result = replaceAllProviders(configs)
    } else {
      result = importProviders(configs, 'merge')
    }
    setImportResult(result)
  }, [importPreview, importMode, importProviders, replaceAllProviders])

  if (!open) return null

  const hasValidPreview = importPreview && importPreview.length > 0 && validationErrors.length === 0
  const importDisabled = !hasValidPreview

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div
        className="w-full max-w-lg mx-4 rounded-xl border border-border bg-card shadow-xl flex flex-col max-h-[85vh]"
        role="dialog"
        aria-modal="true"
        data-testid="provider-import-export-dialog"
      >
        {/* 头部 */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border shrink-0">
          <h2 className="text-base font-semibold text-foreground">配置导入 / 导出</h2>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md hover:bg-accent transition-colors"
            aria-label="关闭"
          >
            <X className="w-4 h-4 text-muted-foreground" />
          </button>
        </div>

        {/* 主体（可滚动） */}
        <div className="flex-1 overflow-y-auto px-5 py-4 space-y-6">
          {/* ===== 导出区域 ===== */}
          <section data-testid="export-section">
            <div className="flex items-center gap-2 mb-3">
              <Download className="w-4 h-4 text-primary" />
              <h3 className="text-sm font-medium text-foreground">导出配置</h3>
            </div>

            {/* 分组选择 */}
            {existingGroups.length > 0 && (
              <div className="mb-3">
                <p className="text-xs text-muted-foreground mb-1.5">选择要导出的分组</p>
                <div className="flex flex-wrap gap-2">
                  {existingGroups.map((group) => (
                    <label
                      key={group}
                      className="flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border bg-background cursor-pointer hover:bg-accent/50 transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={selectedGroups.has(group)}
                        onChange={() => toggleGroup(group)}
                        className="accent-primary"
                      />
                      <span className="text-xs text-foreground">{group}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            {/* 包含 API Key */}
            <label className="flex items-center gap-2 mb-3 cursor-pointer">
              <input
                type="checkbox"
                checked={includeApiKey}
                onChange={(e) => setIncludeApiKey(e.target.checked)}
                className="accent-primary"
                data-testid="include-apikey-checkbox"
              />
              <span className="text-xs text-foreground">包含 API Key（默认不包含，出于安全考虑）</span>
            </label>

            <button
              onClick={handleExport}
              disabled={providers.length === 0}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              data-testid="export-btn"
            >
              <Download className="w-3.5 h-3.5" />
              导出配置
            </button>
          </section>

          <div className="border-t border-border" />

          {/* ===== 导入区域 ===== */}
          <section data-testid="import-section">
            <div className="flex items-center gap-2 mb-3">
              <Upload className="w-4 h-4 text-primary" />
              <h3 className="text-sm font-medium text-foreground">导入配置</h3>
            </div>

            {/* 模式选择 */}
            <div className="flex gap-3 mb-3">
              <label className="flex items-center gap-1.5 cursor-pointer">
                <input
                  type="radio"
                  name="import-mode"
                  checked={importMode === 'merge'}
                  onChange={() => setImportMode('merge')}
                  className="accent-primary"
                  data-testid="import-mode-merge"
                />
                <span className="text-xs text-foreground">合并（保留现有）</span>
              </label>
              <label className="flex items-center gap-1.5 cursor-pointer">
                <input
                  type="radio"
                  name="import-mode"
                  checked={importMode === 'overwrite'}
                  onChange={() => setImportMode('overwrite')}
                  className="accent-primary"
                  data-testid="import-mode-overwrite"
                />
                <span className="text-xs text-foreground">覆盖（完全替换）</span>
              </label>
            </div>

            {/* 拖拽/选择区域 */}
            <div
              onDragOver={(e) => {
                e.preventDefault()
                setIsDragging(true)
              }}
              onDragLeave={() => setIsDragging(false)}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              className={`relative flex flex-col items-center justify-center gap-2 px-4 py-6 rounded-lg border-2 border-dashed cursor-pointer transition-colors ${
                isDragging
                  ? 'border-primary bg-primary/5'
                  : importFile
                    ? 'border-green-500/40 bg-green-500/5'
                    : 'border-border hover:border-primary/30 hover:bg-accent/30'
              }`}
              data-testid="import-drop-zone"
            >
              <input
                ref={fileInputRef}
                type="file"
                accept=".json,application/json"
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  if (file) handleFileSelect(file)
                }}
                className="hidden"
                data-testid="import-file-input"
              />
              {importFile ? (
                <>
                  <FileJson className="w-6 h-6 text-green-600" />
                  <span className="text-sm text-foreground font-medium">{importFile.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {(importFile.size / 1024).toFixed(1)} KB
                  </span>
                </>
              ) : (
                <>
                  <Upload className="w-6 h-6 text-muted-foreground" />
                  <span className="text-sm text-muted-foreground">点击或拖拽上传 JSON 文件</span>
                </>
              )}
            </div>

            {/* 验证结果 */}
            {validationErrors.length > 0 && (
              <div className="mt-3 space-y-1.5" data-testid="validation-errors">
                {validationErrors.map((err, idx) => (
                  <div key={idx} className="flex items-start gap-1.5 text-xs text-destructive">
                    <AlertCircle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                    <span>{err.message}</span>
                  </div>
                ))}
              </div>
            )}

            {/* 预览统计 */}
            {importPreview && validationErrors.length === 0 && (
              <div className="mt-3 flex items-center gap-2 text-xs" data-testid="import-preview">
                <CheckCircle className="w-3.5 h-3.5 text-green-600" />
                <span className="text-foreground">
                  有效记录 {importPreview.length} 条
                  {importMode === 'merge' && providers.length > 0 && (
                    <span className="text-muted-foreground">（现有 {providers.length} 条）</span>
                  )}
                </span>
              </div>
            )}

            {/* 冲突提示 */}
            {importMode === 'merge' &&
              importPreview &&
              importPreview.length > 0 &&
              providers.length > 0 && (
                <div className="mt-2" data-testid="import-conflicts">
                  {importPreview
                    .filter((p) => providers.some((existing) => existing.name === p.name))
                    .map((p) => (
                      <div
                        key={p.name}
                        className="flex items-start gap-1.5 text-xs text-amber-600 mt-1"
                      >
                        <AlertTriangle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                        <span>
                          已存在同名 Provider「{p.name}」，合并模式下将跳过此记录
                        </span>
                      </div>
                    ))}
                </div>
              )}

            {/* 导入结果 */}
            {importResult && (
              <div className="mt-3 p-3 rounded-lg bg-green-500/5 border border-green-500/20" data-testid="import-result">
                <div className="flex items-center gap-1.5 text-xs text-green-700 mb-1">
                  <CheckCircle className="w-3.5 h-3.5" />
                  <span className="font-medium">导入完成</span>
                </div>
                <div className="text-xs text-muted-foreground space-y-0.5">
                  <div>新增 {importResult.added} 条</div>
                  {importResult.skipped > 0 && <div>跳过 {importResult.skipped} 条（同名冲突）</div>}
                  {importResult.errors.length > 0 && (
                    <div className="text-destructive">
                      错误 {importResult.errors.length} 条
                    </div>
                  )}
                </div>
              </div>
            )}

            <button
              onClick={handleImport}
              disabled={importDisabled}
              className="mt-3 flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              data-testid="import-btn"
            >
              <Upload className="w-3.5 h-3.5" />
              导入配置
            </button>
          </section>
        </div>
      </div>
    </div>
  )
}
