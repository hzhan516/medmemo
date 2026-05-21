import { useState, isValidElement, cloneElement } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { Components } from 'react-markdown'
import { CodeBlock } from './CodeBlock'
import { MedicalText } from './MedicalText'
import { ChevronDown } from 'lucide-react'

interface MarkdownRendererProps {
  content: string
}

const MAX_LENGTH = 10000

/**
 * 递归处理 children 中的字符串节点，替换为医学术语高亮组件。
 */
function highlightTerms(node: React.ReactNode): React.ReactNode {
  if (typeof node === 'string') {
    return <MedicalText text={node} />
  }
  if (Array.isArray(node)) {
    return node.map((child, i) => <span key={i}>{highlightTerms(child)}</span>)
  }
  if (isValidElement(node)) {
    return cloneElement(node, {
      ...node.props,
      children: highlightTerms(node.props.children),
    })
  }
  return node
}

/**
 * Markdown 渲染引擎。
 * 集成 react-markdown + remark-gfm，支持：
 * - 标准 Markdown 语法（标题/列表/表格/链接/引用/分割线）
 * - GitHub Flavored Markdown（任务列表、删除线、表格等）
 * - 代码块 PrismJS 语法高亮 + 复制按钮
 * - 医学术语下划虚线 + Tooltip
 * - 超长消息折叠（>10000 字符）
 */
export function MarkdownRenderer({ content }: MarkdownRendererProps) {
  const [expanded, setExpanded] = useState(false)
  const shouldTruncate = content.length > MAX_LENGTH
  const displayContent = shouldTruncate && !expanded
    ? content.slice(0, MAX_LENGTH)
    : content

  return (
    <div className="markdown-body">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
        {displayContent}
      </ReactMarkdown>

      {shouldTruncate && !expanded && (
        <button
          onClick={() => setExpanded(true)}
          className="mt-3 flex items-center gap-1 text-xs text-primary hover:underline"
        >
          <ChevronDown size={14} />
          展开剩余内容（{content.length - MAX_LENGTH} 字符）
        </button>
      )}
    </div>
  )
}

/* ---------- 自定义 Markdown 组件映射 ---------- */

const markdownComponents: Components = {
  // 代码块 / 行内代码
  code({ className, children }) {
    // 行内代码（无 language- 前缀）
    if (!className) {
      return (
        <code className="px-1.5 py-0.5 rounded bg-muted text-sm font-mono text-foreground">
          {children}
        </code>
      )
    }
    // 代码块
    return <CodeBlock className={className}>{children}</CodeBlock>
  },

  // 表格
  table({ children }) {
    return (
      <div className="overflow-x-auto my-3 rounded-lg border border-border">
        <table className="w-full text-sm border-collapse">{children}</table>
      </div>
    )
  },
  thead({ children }) {
    return <thead className="bg-primary/10">{children}</thead>
  },
  th({ children }) {
    return (
      <th className="px-4 py-2 text-left font-semibold text-foreground border-b border-border">
        {highlightTerms(children)}
      </th>
    )
  },
  td({ children }) {
    return (
      <td className="px-4 py-2 text-foreground border-b border-border">
        {highlightTerms(children)}
      </td>
    )
  },
  tr({ children }) {
    return <tr className="even:bg-muted/30">{children}</tr>
  },

  // 标题
  h1({ children }) {
    return <h1 className="text-xl font-bold mt-6 mb-3 text-foreground">{highlightTerms(children)}</h1>
  },
  h2({ children }) {
    return <h2 className="text-lg font-bold mt-5 mb-2.5 text-foreground border-b border-border pb-1">{highlightTerms(children)}</h2>
  },
  h3({ children }) {
    return <h3 className="text-base font-semibold mt-4 mb-2 text-foreground">{highlightTerms(children)}</h3>
  },
  h4({ children }) {
    return <h4 className="text-sm font-semibold mt-3 mb-1.5 text-foreground">{highlightTerms(children)}</h4>
  },
  h5({ children }) {
    return <h5 className="text-sm font-medium mt-3 mb-1.5 text-muted-foreground">{highlightTerms(children)}</h5>
  },
  h6({ children }) {
    return <h6 className="text-xs font-medium mt-2 mb-1 text-muted-foreground">{highlightTerms(children)}</h6>
  },

  // 列表
  ul({ children }) {
    return <ul className="list-disc pl-5 my-2 space-y-1">{children}</ul>
  },
  ol({ children }) {
    return <ol className="list-decimal pl-5 my-2 space-y-1">{children}</ol>
  },
  li({ children }) {
    return <li className="text-sm text-foreground">{highlightTerms(children)}</li>
  },

  // 链接
  a({ href, children }) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className="text-primary underline underline-offset-2 hover:opacity-80 transition-opacity"
      >
        {children}
      </a>
    )
  },

  // 引用块
  blockquote({ children }) {
    return (
      <blockquote className="border-l-4 border-primary/30 pl-4 py-1 my-3 italic text-muted-foreground bg-muted/20 rounded-r-lg">
        {highlightTerms(children)}
      </blockquote>
    )
  },

  // 分割线
  hr() {
    return <hr className="my-4 border-border" />
  },

  // 段落
  p({ children }) {
    return <p className="text-sm leading-relaxed my-2 text-foreground">{highlightTerms(children)}</p>
  },
}
