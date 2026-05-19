import { Info, Github, Heart, Shield, ExternalLink } from 'lucide-react'

/**
 * 关于页面：展示产品信息、开源协议与免责声明。
 */
export function AboutPage() {
  return (
    <div className="h-full flex flex-col bg-background animate-fadeIn">
      <div className="h-14 flex items-center px-4 border-b border-border">
        <h1 className="text-lg font-semibold">关于</h1>
      </div>

      <div className="flex-1 overflow-y-auto p-6 max-w-2xl mx-auto w-full space-y-8">
        {/* 产品标识 */}
        <section className="text-center space-y-2">
          <div className="w-16 h-16 mx-auto rounded-2xl bg-primary/10 flex items-center justify-center">
            <Info size={32} className="text-primary" />
          </div>
          <h2 className="text-xl font-bold">MedMemo</h2>
          <p className="text-sm text-muted-foreground">v0.1.0-alpha</p>
          <p className="text-sm text-muted-foreground">
            开源桌面端健康咨询信息工具
          </p>
        </section>

        {/* 产品定位 */}
        <section>
          <h3 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider">
            产品定位
          </h3>
          <div className="rounded-xl border border-border p-4 space-y-2 text-sm text-muted-foreground">
            <p>
              MedMemo 定位为「医院导诊与健康咨询信息工具」，
              <strong className="text-foreground">明确不属于医疗器械</strong>。
            </p>
            <p>
              核心使命是打破就医信息壁垒，在用户从「感到不适」到「决定就医」的关键决策链路上，提供结构化、个性化、可信赖的健康信息引导。
            </p>
          </div>
        </section>

        {/* 免责声明 */}
        <section>
          <h3 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider flex items-center gap-2">
            <Shield size={14} />
            免责声明
          </h3>
          <div className="rounded-xl border border-border p-4 space-y-2 text-sm text-muted-foreground">
            <p>
              本工具提供的所有信息仅供参考，不构成医疗诊断、治疗建议或处方开具。
            </p>
            <p>
              本产品不从事AI疾病诊断、不开具处方、不推荐具体药品剂量，不涉及医疗器械认证。
            </p>
            <p className="text-destructive font-medium">
              如有紧急症状，请立即拨打120或前往就近医院急诊。
            </p>
          </div>
        </section>

        {/* 开源信息 */}
        <section>
          <h3 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider flex items-center gap-2">
            <Github size={14} />
            开源协议
          </h3>
          <div className="rounded-xl border border-border p-4 space-y-3 text-sm text-muted-foreground">
            <p>
              MedMemo 采用 MIT License 开源协议发布。任何人都可以自由使用、修改和分发本软件。
            </p>
            <a
              href="https://github.com/medmemo/health-assistant"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-primary hover:underline"
            >
              访问 GitHub 仓库 <ExternalLink size={12} />
            </a>
          </div>
        </section>

        {/* 致谢 */}
        <section>
          <h3 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wider flex items-center gap-2">
            <Heart size={14} />
            致谢
          </h3>
          <div className="rounded-xl border border-border p-4 text-sm text-muted-foreground">
            <p>感谢所有贡献者与开源社区的支持。</p>
            <p className="mt-1">Built with Wails v2, React, Tailwind CSS, and Go.</p>
          </div>
        </section>
      </div>
    </div>
  )
}
