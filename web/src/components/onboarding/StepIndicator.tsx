interface StepIndicatorProps {
  currentStep: number
  totalSteps?: number
}

/**
 * 顶部步骤指示器：圆点 + 连接线。
 */
export function StepIndicator({ currentStep, totalSteps = 3 }: StepIndicatorProps) {
  return (
    <div className="flex items-center justify-center gap-2 mb-6">
      {Array.from({ length: totalSteps }).map((_, idx) => {
        const step = idx + 1
        const isActive = step === currentStep
        const isCompleted = step < currentStep

        return (
          <div key={step} className="flex items-center gap-2">
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors duration-200 ${
                isCompleted
                  ? 'bg-primary text-primary-foreground'
                  : isActive
                    ? 'bg-primary/20 text-primary ring-2 ring-primary'
                    : 'bg-muted text-muted-foreground'
              }`}
            >
              {isCompleted ? (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              ) : (
                step
              )}
            </div>
            {step < totalSteps && (
              <div
                className={`w-12 h-0.5 transition-colors duration-200 ${
                  isCompleted ? 'bg-primary' : 'bg-muted'
                }`}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}
