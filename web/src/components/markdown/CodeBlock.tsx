import { useEffect, useRef, useState, useCallback } from 'react'
import Prism from 'prismjs'
import { Copy, Check } from 'lucide-react'

// 加载 PrismJS 常用语言高亮
import 'prismjs/components/prism-typescript'
import 'prismjs/components/prism-javascript'
import 'prismjs/components/prism-jsx'
import 'prismjs/components/prism-tsx'
import 'prismjs/components/prism-python'
import 'prismjs/components/prism-go'
import 'prismjs/components/prism-bash'
import 'prismjs/components/prism-json'
import 'prismjs/components/prism-yaml'
import 'prismjs/components/prism-sql'
import 'prismjs/components/prism-markdown'
import 'prismjs/components/prism-css'
import 'prismjs/components/prism-rust'
import 'prismjs/components/prism-java'
import 'prismjs/components/prism-c'
import 'prismjs/components/prism-cpp'

interface CodeBlockProps {
  className?: string
  children?: React.ReactNode
}

/**
 * 代码块高亮组件。
 * 使用 PrismJS 进行语法高亮，顶部展示语言标签和复制按钮。
 * 深色背景 (#1E1E1E)。
 */
export function CodeBlock({ className, children }: CodeBlockProps) {
  const codeRef = useRef<HTMLElement>(null)
  const [copied, setCopied] = useState(false)

  const language = className?.replace('language-', '') || 'text'
  const displayLang = language === 'text' ? '' : language

  const codeText = typeof children === 'string' ? children : ''

  useEffect(() => {
    if (codeRef.current && codeText) {
      Prism.highlightElement(codeRef.current)
    }
  }, [codeText, language])

  const handleCopy = useCallback(async () => {
    if (!codeText) return
    try {
      await navigator.clipboard.writeText(codeText)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // 复制失败静默处理
    }
  }, [codeText])

  return (
    <div className="rounded-xl overflow-hidden my-3 border border-border">
      {/* 顶部栏：语言标签 + 复制按钮 */}
      <div className="flex items-center justify-between px-4 py-2 bg-[#1E1E1E] border-b border-white/10">
        <span className="text-xs text-gray-400 font-mono">
          {displayLang}
        </span>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 text-xs text-gray-400 hover:text-gray-200 transition-colors"
          aria-label="复制代码"
        >
          {copied ? (
            <>
              <Check size={14} className="text-emerald-400" />
              <span className="text-emerald-400">已复制</span>
            </>
          ) : (
            <>
              <Copy size={14} />
              <span>复制</span>
            </>
          )}
        </button>
      </div>

      {/* 代码区域 */}
      <div className="bg-[#1E1E1E] overflow-x-auto">
        <pre className="p-4 m-0 text-sm leading-relaxed">
          <code
            ref={codeRef}
            className={className || 'language-text'}
          >
            {children}
          </code>
        </pre>
      </div>
    </div>
  )
}
