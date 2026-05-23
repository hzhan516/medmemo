import type { Conversation } from '@/stores/chatStore'

export interface GroupedConversations {
  pinned: Conversation[]
  today: Conversation[]
  yesterday: Conversation[]
  last7Days: Conversation[]
  earlier: Conversation[]
}

/**
 * 按更新时间将会话分组为：置顶、今天、昨天、近7天、更早。
 */
export function groupConversationsByTime(
  conversations: Conversation[]
): GroupedConversations {
  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const yesterdayStart = todayStart - 24 * 60 * 60 * 1000
  const sevenDaysStart = todayStart - 7 * 24 * 60 * 60 * 1000

  const result: GroupedConversations = {
    pinned: [],
    today: [],
    yesterday: [],
    last7Days: [],
    earlier: [],
  }

  for (const conv of conversations) {
    if (conv.isPinned) {
      result.pinned.push(conv)
      continue
    }

    const updated = conv.updatedAt
    if (updated >= todayStart) {
      result.today.push(conv)
    } else if (updated >= yesterdayStart) {
      result.yesterday.push(conv)
    } else if (updated >= sevenDaysStart) {
      result.last7Days.push(conv)
    } else {
      result.earlier.push(conv)
    }
  }

  // 每组内按 updatedAt 降序排列
  for (const key of Object.keys(result) as Array<keyof GroupedConversations>) {
    result[key].sort((a, b) => b.updatedAt - a.updatedAt)
  }

  return result
}

/**
 * 根据搜索关键词过滤会话（按标题和 preview 匹配）。
 */
export function filterConversations(
  conversations: Conversation[],
  query: string
): Conversation[] {
  if (!query.trim()) return conversations
  const lower = query.trim().toLowerCase()
  return conversations.filter(
    (c) =>
      c.title.toLowerCase().includes(lower) ||
      (c.preview && c.preview.toLowerCase().includes(lower))
  )
}

/**
 * 将文本中的匹配关键词用高亮标记包裹。
 * 返回数组，包含普通文本字符串和高亮文本对象。
 */
export function highlightText(
  text: string,
  query: string
): Array<{ text: string; highlight: boolean }> {
  if (!query.trim() || !text) {
    return [{ text, highlight: false }]
  }

  const lowerQuery = query.trim().toLowerCase()
  const lowerText = text.toLowerCase()
  const parts: Array<{ text: string; highlight: boolean }> = []

  let lastIndex = 0
  let index = lowerText.indexOf(lowerQuery)

  while (index !== -1) {
    if (index > lastIndex) {
      parts.push({ text: text.slice(lastIndex, index), highlight: false })
    }
    parts.push({
      text: text.slice(index, index + query.trim().length),
      highlight: true,
    })
    lastIndex = index + query.trim().length
    index = lowerText.indexOf(lowerQuery, lastIndex)
  }

  if (lastIndex < text.length) {
    parts.push({ text: text.slice(lastIndex), highlight: false })
  }

  return parts
}

/**
 * 校验会话标题合法性。
 * @returns valid 是否通过校验，error 错误信息（未通过时）
 */
export function validateConversationTitle(title: string): {
  valid: boolean
  error?: string
} {
  const trimmed = title.trim()
  if (!trimmed) {
    return { valid: false, error: '标题不能为空' }
  }

  // 按 Unicode 码点计数（中英文混排时公平计算）
  const charCount = [...trimmed].length
  if (charCount > 16) {
    return { valid: false, error: '标题不能超过 16 个字符' }
  }

  // 仅允许字母、数字、汉字、空格、-、_、·
  const validPattern = /^[\w\s\-\u00B7\u4e00-\u9fa5]+$/u
  if (!validPattern.test(trimmed)) {
    return { valid: false, error: '标题包含非法字符' }
  }

  return { valid: true }
}

/**
 * 格式化时间戳为相对时间描述。
 */
export function formatRelativeTime(timestamp: number): string {
  const now = Date.now()
  const diff = now - timestamp
  const minutes = Math.floor(diff / (60 * 1000))
  const hours = Math.floor(diff / (60 * 60 * 1000))
  const days = Math.floor(diff / (24 * 60 * 60 * 1000))

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (hours < 24) return `${hours} 小时前`
  if (days === 1) return '昨天'
  if (days < 7) return `${days} 天前`
  return new Date(timestamp).toLocaleDateString('zh-CN')
}
