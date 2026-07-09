import { useState, useEffect, useCallback } from 'react'
import { useWails } from '@/hooks/useWails'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'
import { Brain, CheckCircle, XCircle, Clock, Search, Trash2, Eye, ShieldCheck, ShieldX } from 'lucide-react'
import type { main } from '@wails/go/models'

type MemoryItem = main.MemoryItem
type MemoryStats = main.MemoryStats

type TabKey = 'all' | 'pending' | 'approved' | 'rejected'

const tabs: { key: TabKey; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'pending', label: '待审核' },
  { key: 'approved', label: '已审批' },
  { key: 'rejected', label: '已拒绝' },
]

export function MemoryPage() {
  const {
    getMemories,
    getPendingReviews,
    getMemoryStats,
    searchMemories,
    approveFact,
    rejectFact,
    deleteMemory,
  } = useWails()

  const [items, setItems] = useState<MemoryItem[]>([])
  const [stats, setStats] = useState<MemoryStats | null>(null)
  const [activeTab, setActiveTab] = useState<TabKey>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<MemoryItem | null>(null)
  const [page, setPage] = useState(0)
  const pageSize = 20

  const loadStats = useCallback(async () => {
    try {
      const s = await getMemoryStats()
      setStats(s)
    } catch (err) {
      console.error('Failed to load memory stats:', err)
    }
  }, [getMemoryStats])

  const loadItems = useCallback(async () => {
    setLoading(true)
    try {
      let data: MemoryItem[]
      if (searchQuery.trim()) {
        data = await searchMemories(searchQuery.trim())
      } else if (activeTab === 'pending') {
        data = await getPendingReviews(pageSize, page * pageSize)
      } else if (activeTab === 'all') {
        // 全部：分别获取已审批、待审核、已拒绝，然后合并
        const approved = await getMemories(pageSize, page * pageSize)
        data = approved
      } else {
        // approved / rejected：通过搜索全部后过滤
        const all = await getMemories(1000, 0)
        data = all.filter((m) => m.status === activeTab)
      }
      setItems(data)
    } catch (err) {
      console.error('Failed to load memories:', err)
    } finally {
      setLoading(false)
    }
  }, [activeTab, page, searchQuery, getMemories, getPendingReviews, searchMemories])

  useEffect(() => {
    loadStats()
  }, [loadStats])

  useEffect(() => {
    loadItems()
  }, [loadItems])

  const handleApprove = async (factID: string) => {
    try {
      await approveFact(factID)
      await Promise.all([loadItems(), loadStats()])
    } catch (err) {
      console.error('Failed to approve:', err)
    }
  }

  const handleReject = async (factID: string) => {
    try {
      await rejectFact(factID)
      await Promise.all([loadItems(), loadStats()])
    } catch (err) {
      console.error('Failed to reject:', err)
    }
  }

  const handleDelete = async (factID: string) => {
    if (!confirm('确定要删除这条记忆吗？此操作不可恢复。')) return
    // 乐观移除：立即更新本地状态，避免 UI 看起来没刷新
    setItems(prev => prev.filter(item => item.fact_id !== factID))
    setDetail(prev => prev?.fact_id === factID ? null : prev)
    try {
      await deleteMemory(factID)
      await Promise.all([loadItems(), loadStats()])
    } catch (err) {
      console.error('Failed to delete:', err)
      // 删除失败时回滚：重新拉取后端权威数据恢复列表
      await Promise.all([loadItems(), loadStats()])
    }
  }

  const statusBadge = (status: string) => {
    const map: Record<string, { cls: string; icon: React.ReactNode; label: string }> = {
      approved: { cls: 'bg-emerald-100 text-emerald-700', icon: <ShieldCheck size={12} />, label: '已审批' },
      pending: { cls: 'bg-amber-100 text-amber-700', icon: <Clock size={12} />, label: '待审核' },
      rejected: { cls: 'bg-red-100 text-red-700', icon: <ShieldX size={12} />, label: '已拒绝' },
    }
    const s = map[status] || { cls: 'bg-gray-100 text-gray-700', icon: null, label: status }
    return (
      <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${s.cls}`}>
        {s.icon}
        {s.label}
      </span>
    )
  }

  return (
    <div className="h-full flex flex-col p-4 gap-4 overflow-auto bg-background/30">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold flex items-center gap-2">
          <Brain size={22} className="text-primary" />
          记忆管理
        </h1>
      </div>

      {/* 统计卡片 — compact translucent */}
      {stats && (
        <div className="grid grid-cols-4 gap-3">
          {[
            { icon: Brain, color: 'text-muted-foreground', value: stats.total, label: '全部记忆' },
            { icon: CheckCircle, color: 'text-emerald-500', value: stats.approved, label: '已审批' },
            { icon: Clock, color: 'text-amber-500', value: stats.pending, label: '待审核' },
            { icon: XCircle, color: 'text-red-500', value: stats.rejected, label: '已拒绝' },
          ].map(({ icon: Icon, color, value, label }) => (
            <div key={label} className="flex items-center gap-3 p-3 rounded-xl bg-white/60 dark:bg-white/5 border border-border/60 backdrop-blur-sm shadow-sm">
              <Icon size={18} className={color} />
              <div>
                <div className="text-lg font-bold">{value}</div>
                <div className="text-xs text-muted-foreground">{label}</div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 搜索与筛选 */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search size={16} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索记忆..."
            value={searchQuery}
            onChange={(e) => { setSearchQuery(e.target.value); setPage(0) }}
            className="pl-8 mac-control"
          />
        </div>
        <div className="flex gap-1">
          {tabs.map((t) => (
            <button
              key={t.key}
              onClick={() => { setActiveTab(t.key); setPage(0) }}
              className={`px-3 py-1.5 rounded-md text-sm transition-colors ${
                activeTab === t.key
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-accent text-muted-foreground'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>

      {/* 列表 */}
      <Card className="flex-1 flex flex-col min-h-0">
        <CardContent className="p-0 flex-1 overflow-auto">
          {loading && items.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground text-sm">加载中…</div>
          ) : items.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground text-sm">暂无记忆</div>
          ) : (
            <table className="w-full text-sm">
              <thead className="sticky top-0 bg-background border-b border-border z-10">
                <tr>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">内容</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground w-24">可信度</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground w-24">状态</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground w-36">创建时间</th>
                  <th className="text-right px-4 py-2 font-medium text-muted-foreground w-32">操作</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.fact_id} className="border-b border-border/50 hover:bg-accent/50 transition-colors">
                    <td className="px-4 py-2.5">
                      <div className="font-medium">{item.subject} {item.predicate} {item.object}</div>
                    </td>
                    <td className="px-4 py-2.5">
                      <span className={`text-xs font-medium ${item.confidence >= 0.8 ? 'text-emerald-600' : item.confidence >= 0.5 ? 'text-amber-600' : 'text-red-600'}`}>
                        {Math.round(item.confidence * 100)}%
                      </span>
                    </td>
                    <td className="px-4 py-2.5">{statusBadge(item.status)}</td>
                    <td className="px-4 py-2.5 text-muted-foreground text-xs">
                      {new Date(item.created_at).toLocaleString('zh-CN')}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setDetail(item)}>
                          <Eye size={14} />
                        </Button>
                        {item.status === 'pending' && (
                          <>
                            <Button variant="ghost" size="icon" className="h-7 w-7 text-emerald-600" onClick={() => handleApprove(item.fact_id)}>
                              <CheckCircle size={14} />
                            </Button>
                            <Button variant="ghost" size="icon" className="h-7 w-7 text-red-600" onClick={() => handleReject(item.fact_id)}>
                              <XCircle size={14} />
                            </Button>
                          </>
                        )}
                        <Button variant="ghost" size="icon" className="h-7 w-7 text-red-500" onClick={() => handleDelete(item.fact_id)}>
                          <Trash2 size={14} />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      {/* 详情弹窗 */}
      {detail && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setDetail(null)}>
          <div className="bg-background rounded-lg shadow-lg border border-border w-full max-w-md p-5" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-lg font-semibold mb-3">记忆详情</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between"><span className="text-muted-foreground">ID</span><span className="font-mono text-xs">{detail.fact_id}</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">主语</span><span>{detail.subject}</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">谓语</span><span>{detail.predicate}</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">宾语</span><span>{detail.object}</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">可信度</span><span>{Math.round(detail.confidence * 100)}%</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">状态</span><span>{statusBadge(detail.status)}</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">创建时间</span><span>{new Date(detail.created_at).toLocaleString('zh-CN')}</span></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              {detail.status === 'pending' && (
                <>
                  <Button size="sm" variant="outline" onClick={() => { handleApprove(detail.fact_id); setDetail(null) }}>
                    <CheckCircle size={14} className="mr-1" /> 通过
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => { handleReject(detail.fact_id); setDetail(null) }}>
                    <XCircle size={14} className="mr-1" /> 拒绝
                  </Button>
                </>
              )}
              <Button size="sm" variant="destructive" onClick={async () => { await handleDelete(detail.fact_id) }}>
                <Trash2 size={14} className="mr-1" /> 删除
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
