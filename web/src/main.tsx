import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

// 在渲染前读取 localStorage 主题设置，避免闪烁
const savedTheme = localStorage.getItem('medmemo-settings')
if (savedTheme) {
  try {
    const parsed = JSON.parse(savedTheme)
    const theme = parsed?.state?.theme || 'system'
    if (theme === 'dark') {
      document.documentElement.classList.add('dark')
    } else if (theme === 'light') {
      document.documentElement.classList.remove('dark')
    } else {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      document.documentElement.classList.toggle('dark', prefersDark)
    }
  } catch {
    // ignore parse error
  }
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
