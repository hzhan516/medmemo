import { useState, useCallback } from 'react'
import { Cloud, CheckCircle2, AlertCircle, XCircle, Loader2, FileJson, Eye, EyeOff } from 'lucide-react'
import { ParseServiceAccountJSON } from '@wails/go/main/WailsApp'
import type { AuthMethodDetectStatus, ProviderConfig } from '@/types/provider'

interface ServiceAccountPanelProps {
  status: AuthMethodDetectStatus | undefined
  onProviderCreated: (provider: ProviderConfig) => void
}

// Vertex AI 支持的地域列表
const vertexRegions = [
  { value: 'us-central1', label: 'us-central1 (爱荷华)' },
  { value: 'us-east1', label: 'us-east1 (南卡罗来纳)' },
  { value: 'us-west1', label: 'us-west1 (俄勒冈)' },
  { value: 'europe-west1', label: 'europe-west1 (比利时)' },
  { value: 'europe-west4', label: 'europe-west4 (荷兰)' },
  { value: 'asia-east1', label: 'asia-east1 (台湾)' },
  { value: 'asia-northeast1', label: 'asia-northeast1 (东京)' },
  { value: 'asia-southeast1', label: 'asia-southeast1 (新加坡)' },
]

/**
 * Service Account 配置面板（Vertex AI 专用）。
 * 粘贴 JSON → 解析提取 → 选择地区 → 输入模型 → 创建 Provider。
 */
export function ServiceAccountPanel({ status, onProviderCreated }: ServiceAccountPanelProps) {
  const [jsonInput, setJsonInput] = useState('')
  const [showJson, setShowJson] = useState(false)
  const [parsing, setParsing] = useState(false)
  const [parsed, setParsed] = useState<{ projectId: string; clientEmail: string; privateKey: string } | null>(null)

  const [region, setRegion] = useState('us-central1')
  const [modelId, setModelId] = useState('gemini-1.5-pro')
  const [error, setError] = useState<string | null>(null)

  const handleParse = useCallback(async () => {
    if (!jsonInput.trim()) {
      setError('请粘贴 Service Account JSON 密钥内容')
      return
    }
    setError(null)
    setParsing(true)
    try {
      const result = await ParseServiceAccountJSON(jsonInput.trim())
      if (result.project_id && result.private_key) {
        setParsed({
          projectId: result.project_id,
          clientEmail: result.client_email || '',
          privateKey: result.private_key,
        })
        // 解析成功后立即清空原始 JSON 输入（安全要求）
        setJsonInput('')
      } else {
        setError('解析结果缺少必要字段（project_id 或 private_key）')
      }
    } catch (err) {
      setError(typeof err === 'string' ? err : (err instanceof Error ? err.message : '解析 Service Account JSON 失败'))
    } finally {
      setParsing(false)
    }
  }, [jsonInput])

  const handleCreate = useCallback(() => {
    if (!parsed) {
      setError('请先解析 Service Account JSON')
      return
    }
    if (!region) {
      setError('请选择地域')
      return
    }
    if (!modelId.trim()) {
      setError('请输入模型 ID')
      return
    }

    const now = Date.now()
    const apiHost = `https://${region}-aiplatform.googleapis.com`

    const newProvider: ProviderConfig = {
      id: `vertex_sa_${now}`,
      templateId: 'vertex',
      name: `Vertex AI (${region})`,
      apiHost,
      apiKey: '',
      modelId: modelId.trim(),
      temperature: 0.7,
      timeoutMs: 30000,
      maxRetries: 3,
      group: 'Google Cloud',
      enabled: true,
      sortOrder: 0,
      createdAt: now,
      updatedAt: now,
      authMethod: 'service_account',
      authParams: {
        gcpProjectId: parsed.projectId,
        gcpRegion: region,
        saJson: parsed.privateKey,
      },
      models: [{ id: modelId.trim(), name: modelId.trim(), enabled: true }],
    }
    onProviderCreated(newProvider)
    // 重置状态
    setParsed(null)
    setJsonInput('')
    setModelId('gemini-1.5-pro')
    setRegion('us-central1')
    setError(null)
  }, [parsed, region, modelId, onProviderCreated])

  if (!status) {
    return (
      <div className="p-4 rounded-lg bg-muted/50 text-sm text-muted-foreground">
        尚未检测 Service Account 状态。
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 p-3 rounded-lg border">
        <Cloud className="w-5 h-5 text-muted-foreground shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">Vertex AI (Service Account)</span>
            {status.connected ? (
              <CheckCircle2 className="w-4 h-4 text-green-500" />
            ) : status.available ? (
              <AlertCircle className="w-4 h-4 text-amber-500" />
            ) : (
              <XCircle className="w-4 h-4 text-muted-foreground" />
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-1">{status.detail}</p>
        </div>
      </div>

      {!parsed ? (
        <>
          {/* JSON 粘贴区 */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Service Account JSON 密钥</label>
              <button
                type="button"
                onClick={() => setShowJson(!showJson)}
                className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                {showJson ? <EyeOff className="w-3 h-3 inline" /> : <Eye className="w-3 h-3 inline" />}
                {showJson ? ' 隐藏' : ' 显示'}
              </button>
            </div>
            <textarea
              value={jsonInput}
              onChange={(e) => {
                setJsonInput(e.target.value)
                setError(null)
              }}
              placeholder='{"type":"service_account","project_id":"...","private_key":"...","client_email":"..."}'
              rows={5}
              className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30 resize-y font-mono"
            />
            <p className="text-xs text-muted-foreground">
              粘贴从 Google Cloud Console 下载的 Service Account JSON 密钥文件内容。
              解析后原始 JSON 将被清空，仅保存 project_id、client_email 和 private_key。
            </p>
          </div>

          <button
            onClick={handleParse}
            disabled={parsing || !jsonInput.trim()}
            className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
          >
            {parsing ? <Loader2 className="w-4 h-4 animate-spin" /> : <FileJson className="w-4 h-4" />}
            {parsing ? '解析中…' : '解析 JSON'}
          </button>
        </>
      ) : (
        <>
          {/* 解析成功后的配置表单 */}
          <div className="p-3 rounded-lg bg-green-500/5 border border-green-500/20 space-y-2">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-green-500" />
              <span className="text-sm text-green-700 font-medium">解析成功</span>
            </div>
            <div className="text-xs text-muted-foreground space-y-1">
              <p><span className="font-medium">Project ID:</span> {parsed.projectId}</p>
              <p><span className="font-medium">Client Email:</span> {parsed.clientEmail}</p>
              <p><span className="font-medium">Private Key:</span> {'*'.repeat(20)}（已安全保存）</p>
            </div>
            <button
              onClick={() => {
                setParsed(null)
                setError(null)
              }}
              className="text-xs text-primary hover:underline"
            >
              重新粘贴 JSON
            </button>
          </div>

          {/* 地区选择 */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium">地域 <span className="text-destructive">*</span></label>
            <select
              value={region}
              onChange={(e) => setRegion(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
            >
              {vertexRegions.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </select>
            <p className="text-xs text-muted-foreground">选择与您的 Google Cloud 项目相同的地域。</p>
          </div>

          {/* 模型输入 */}
          <div className="space-y-1.5">
            <label className="text-sm font-medium">模型 ID <span className="text-destructive">*</span></label>
            <input
              type="text"
              value={modelId}
              onChange={(e) => {
                setModelId(e.target.value)
                setError(null)
              }}
              placeholder="例如：gemini-1.5-pro"
              className="w-full px-3 py-2 rounded-lg border border-border bg-background text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
            />
            <p className="text-xs text-muted-foreground">输入 Vertex AI 上的模型名称，如 gemini-1.5-pro、gemini-1.5-flash 等。</p>
          </div>

          <button
            onClick={handleCreate}
            className="w-full py-2.5 px-4 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors flex items-center justify-center gap-2"
          >
            <Cloud className="w-4 h-4" />
            创建 Vertex AI Provider
          </button>
        </>
      )}

      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}
