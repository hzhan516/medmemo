import { useMemo } from 'react'
import { getMedicalTermsRegex, getMedicalDefinition } from '@/lib/medicalTerms'

interface MedicalTextProps {
  text: string
}

/**
 * 对纯文本中的医学术语进行下划虚线高亮，hover 展示 Tooltip 解释。
 */
export function MedicalText({ text }: MedicalTextProps) {
  const segments = useMemo(() => {
    const regex = getMedicalTermsRegex()
    const result: Array<{ type: 'text' | 'term'; value: string }> = []
    let lastIndex = 0

    // 每次重新创建正则，避免全局标志状态污染
    const localRegex = new RegExp(regex.source, 'g')
    let match: RegExpExecArray | null

    while ((match = localRegex.exec(text)) !== null) {
      if (match.index > lastIndex) {
        result.push({ type: 'text', value: text.slice(lastIndex, match.index) })
      }
      result.push({ type: 'term', value: match[0] })
      lastIndex = localRegex.lastIndex
    }

    if (lastIndex < text.length) {
      result.push({ type: 'text', value: text.slice(lastIndex) })
    }

    return result
  }, [text])

  return (
    <>
      {segments.map((seg, i) => {
        if (seg.type === 'term') {
          const definition = getMedicalDefinition(seg.value)
          return (
            <span
              key={i}
              className="relative group border-b border-dashed border-primary cursor-help"
              title={definition}
            >
              {seg.value}
              {/* Tooltip */}
              <span className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2.5 py-1.5 bg-popover text-popover-foreground text-xs rounded-md shadow-lg border border-border opacity-0 group-hover:opacity-100 transition-opacity duration-200 whitespace-nowrap pointer-events-none z-50 max-w-xs truncate">
                {definition}
              </span>
            </span>
          )
        }
        return <span key={i}>{seg.value}</span>
      })}
    </>
  )
}
