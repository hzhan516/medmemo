/**
 * AI 输入中指示器，显示三个跳动圆点。
 */
export function TypingIndicator() {
  return (
    <div className="flex gap-1.5 px-1 py-2">
      <span className="w-2 h-2 rounded-full bg-muted-foreground/40 animate-bounce [animation-delay:-0.3s]" />
      <span className="w-2 h-2 rounded-full bg-muted-foreground/40 animate-bounce [animation-delay:-0.15s]" />
      <span className="w-2 h-2 rounded-full bg-muted-foreground/40 animate-bounce" />
    </div>
  )
}
