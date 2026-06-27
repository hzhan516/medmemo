import { useState, useEffect, useCallback } from 'react'
import { useWails } from '@/hooks/useWails'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { FileText, Trash2, Upload, BookOpen, Loader2 } from 'lucide-react'
import type { main } from '@wails/go/models'

type KnowledgeDocumentDTO = main.KnowledgeDocumentDTO
type ImportKnowledgeResponse = main.ImportKnowledgeResponse

export function KnowledgePage() {
  const { selectKnowledgeFile, importKnowledgeFile, listKnowledgeDocuments, deleteKnowledgeDocument, getKnowledgeImportJob } = useWails()

  const [documents, setDocuments] = useState<KnowledgeDocumentDTO[]>([])
  const [loading, setLoading] = useState(false)
  const [importing, setImporting] = useState(false)
  const [job, setJob] = useState<ImportKnowledgeResponse | null>(null)

  const loadDocuments = useCallback(async () => {
    setLoading(true)
    try {
      const docs = await listKnowledgeDocuments()
      setDocuments(docs)
    } catch (err) {
      console.error('Failed to load knowledge documents:', err)
    } finally {
      setLoading(false)
    }
  }, [listKnowledgeDocuments])

  useEffect(() => {
    loadDocuments()
  }, [loadDocuments])

  const handleImport = useCallback(async () => {
    try {
      const filePath = await selectKnowledgeFile()
      setImporting(true)
      const result = await importKnowledgeFile(filePath)
      setJob(result)
      // 轮询任务状态直到完成或失败
      const poll = setInterval(async () => {
        try {
          const updated = await getKnowledgeImportJob(result.job_id)
          setJob(updated)
          if (updated.status !== 'pending' && updated.status !== 'indexed_vector_unavailable') {
            clearInterval(poll)
            setImporting(false)
            loadDocuments()
          }
        } catch (e) {
          clearInterval(poll)
          setImporting(false)
          console.error('Failed to poll import job:', e)
        }
      }, 1000)
      // 60 秒超时兜底
      setTimeout(() => {
        clearInterval(poll)
        setImporting(false)
      }, 60000)
    } catch (err) {
      console.error('Failed to import knowledge file:', err)
      setImporting(false)
    }
  }, [selectKnowledgeFile, importKnowledgeFile, getKnowledgeImportJob, loadDocuments])

  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteKnowledgeDocument(id)
      loadDocuments()
    } catch (err) {
      console.error('Failed to delete knowledge document:', err)
    }
  }, [deleteKnowledgeDocument, loadDocuments])

  return (
    <div className="flex flex-col h-full p-4 gap-4 overflow-auto">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold flex items-center gap-2">
          <BookOpen size={22} />
          知识库管理
        </h1>
        <Button onClick={handleImport} disabled={importing}>
          {importing ? <Loader2 size={16} className="animate-spin mr-2" /> : <Upload size={16} className="mr-2" />}
          导入文件
        </Button>
      </div>

      {job && (
        <Card>
          <CardContent className="p-4 text-sm">
            <div className="font-medium">导入任务</div>
            <div className="text-gray-600 dark:text-gray-400">
              状态: {job.status} · 进度: {job.processed}/{job.total}
              {job.error && <span className="text-red-500 ml-2">错误: {job.error}</span>}
            </div>
          </CardContent>
        </Card>
      )}

      {loading ? (
        <div className="flex items-center justify-center h-40">
          <Loader2 size={24} className="animate-spin text-gray-400" />
        </div>
      ) : documents.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-40 text-gray-500">
          <FileText size={40} className="mb-2 opacity-50" />
          暂无知识库文档，点击右上角导入 Markdown 或 JSONL 文件
        </div>
      ) : (
        <div className="space-y-3">
          {documents.map((doc) => (
            <Card key={doc.id}>
              <CardContent className="p-4 flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="font-medium truncate">{doc.title || doc.id}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    source: {doc.source} · language: {doc.language} · checksum: {doc.checksum.slice(0, 12)}...
                  </div>
                  {doc.citation && (
                    <div className="text-xs text-gray-500 dark:text-gray-400 mt-1 truncate">
                      引用: {doc.citation}
                    </div>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => handleDelete(doc.id)}
                  aria-label="删除文档"
                >
                  <Trash2 size={16} className="text-red-500" />
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
