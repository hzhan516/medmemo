import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { MessageSquare, Settings, Info } from 'lucide-react'

const navItems = [
  { to: '/chat', label: '对话', icon: MessageSquare },
  { to: '/settings', label: '设置', icon: Settings },
  { to: '/about', label: '关于', icon: Info },
]

/**
 * 应用全局布局：左侧 64px 图标导航栏 + 右侧主内容区 + 底部 24px 状态栏。
 * 使用 NavLink 实现当前路由高亮，页面切换带 200ms 淡入动画。
 */
export function AppLayout() {
  const location = useLocation()

  return (
    <div className="h-screen w-screen overflow-hidden bg-background text-foreground flex">
      {/* 左侧导航栏 */}
      <nav className="shrink-0 w-16 border-r border-border bg-background flex flex-col items-center py-4 gap-2 select-none">
        {/* Logo */}
        <div className="mb-4 w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
          <span className="text-primary-foreground text-xs font-bold">M</span>
        </div>

        {/* 导航项 */}
        {navItems.map((item) => {
          const Icon = item.icon
          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/chat'}
              className={({ isActive }) =>
                `group relative flex items-center justify-center w-11 h-11 rounded-xl transition-all duration-200 ${
                  isActive
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                }`
              }
              aria-label={item.label}
            >
              <Icon size={20} />
              {/* Tooltip */}
              <span className="absolute left-full ml-2 px-2 py-1 rounded-md bg-popover text-popover-foreground text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none shadow-sm border border-border z-50">
                {item.label}
              </span>
            </NavLink>
          )
        })}
      </nav>

      {/* 右侧区域：内容区 + 底部状态栏 */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* 主内容区 */}
        <main className="flex-1 min-w-0 overflow-hidden">
          <div
            key={location.pathname}
            className="h-full animate-in fade-in duration-200"
          >
            <Outlet />
          </div>
        </main>

        {/* 底部状态栏（24px） */}
        <footer className="shrink-0 h-6 flex items-center justify-between px-3 border-t border-border bg-muted/30 text-[11px] text-muted-foreground select-none">
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
            <span>就绪</span>
          </div>
          <div className="flex items-center gap-3">
            <span>MedMemo v0.1.0-alpha</span>
          </div>
        </footer>
      </div>
    </div>
  )
}
