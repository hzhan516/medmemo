import { useState } from 'react'

function App() {
  const [count, setCount] = useState(0)

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col items-center justify-center p-4">
      <h1 className="text-3xl font-bold mb-4">MedMemo</h1>
      <p className="text-muted-foreground mb-6">你的私人健康记忆助手</p>
      <div className="flex flex-col items-center gap-4">
        <button
          className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:opacity-90 transition-opacity"
          onClick={() => setCount((c) => c + 1)}
        >
          count is {count}
        </button>
        <p className="text-sm text-muted-foreground">
          Edit <code>src/App.tsx</code> and save to test HMR
        </p>
      </div>
    </div>
  )
}

export default App
