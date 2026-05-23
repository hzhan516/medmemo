/**
 * 测试辅助：封装带 Providers 的 render 函数。
 * 自动包裹 HashRouter，适配 MedMemo 的路由结构。
 */

import React from 'react'
import { render as tlRender, type RenderOptions } from '@testing-library/react'
import { HashRouter } from 'react-router-dom'

interface AllTheProvidersProps {
  children: React.ReactNode
}

function AllTheProviders({ children }: AllTheProvidersProps) {
  return <HashRouter>{children}</HashRouter>
}

/**
 * 带 Providers 的 render 封装。
 * 所有测试应使用此函数替代 @testing-library/react 的 render。
 */
export function render(ui: React.ReactElement, options?: Omit<RenderOptions, 'wrapper'>) {
  return tlRender(ui, { wrapper: AllTheProviders, ...options })
}

// 重新导出 testing-library 的其他工具
export { screen, waitFor, within } from '@testing-library/react'
export { userEvent } from '@testing-library/user-event'
export { act } from 'react'
