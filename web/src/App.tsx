import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ChatPage } from '@/pages/ChatPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { AboutPage } from '@/pages/AboutPage'
import { useTheme } from '@/hooks/useTheme'

/**
 * 根组件：全局主题初始化与 HashRouter 路由配置。
 * 桌面端使用 HashRouter 避免无 server 场景下的 404 问题。
 */
function App() {
  useTheme()

  return (
    <HashRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route path="/chat" element={<ChatPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/about" element={<AboutPage />} />
          <Route path="/" element={<Navigate to="/chat" replace />} />
          <Route path="*" element={<Navigate to="/chat" replace />} />
        </Route>
      </Routes>
    </HashRouter>
  )
}

export default App
