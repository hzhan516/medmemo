# ADR-004: 采用 Hugot + ONNX Runtime 进行本地 NER 推理

> 🌐 [English Version](../../../adr/004-onnx-integration.md)

- **状态**: 已采纳 (Accepted)
- **日期**: 2026-05
- **决策人**: 后端技术负责人、AI 工程师

## 背景

MedMemo 的**三级脱敏流水线**（L1 规则 → L2 NER → L3 关键词）需要本地命名实体识别（NER）模型，在数据离开设备前识别用户健康文本中的敏感实体。L2 NER 阶段是准确性关键层 —— 必须识别：

- 人名（患者、医生、家庭成员）
- 机构名（医院、诊所、保险公司）
- 疾病名称和医疗状况
- 药品名称和药物别名

NER 引擎必须满足四项硬约束：

1. **隐私零出域**: 所有推理在本地完成；用户文本不会发送到任何外部服务进行实体识别。
2. **离线可用**: 必须能在无任何网络连接的情况下工作。
3. **延迟预算**: 短文本（<500 字符）<200ms，长文本（<2000 字符）<500ms —— 按 p95 测量。
4. **体积预算**: 模型文件 + 运行时库必须放入约 100MB 的资源目录预算中。

在项目启动前，团队评估了三种本地推理方案：

| 方案 | 优点 | 缺点 | 适用性 |
|:---------|:-----|:-----|:------------|
| **Hugot + ONNX Runtime** | 成熟的 Go 绑定（`github.com/knights-analytics/hugot`）；广泛的模型生态（BERT、DistilBERT、RoBERTa）；int8 量化将模型压缩至 ~50MB；CPU 推理对 NER 足够快 | CGO 交叉编译复杂；ONNX Runtime Session **非线程安全**，需要串行化 | ✅ 准确性 + 隐私 + 体积的最佳平衡 |
| **远程 NER API**（如云 NLP 服务） | 最高准确性；零本地模型体积 | 违反隐私优先原则；需要网络；增加延迟（RTT + 服务端耗时） | ❌ 违反零出域约束 |
| **纯 Go NLP**（如 `prose` v3、基于规则） | 零 CGO；交叉编译简单；体积极小 | 在医疗领域的深度学习 NER 面前准确率远低于；无法识别罕见疾病名或药物别名 | ❌ 不满足准确性要求 |

## 决策

采用 **Hugot v0.7+ + ONNX Runtime 1.17+** 进行本地 NER 推理，使用 **int8 量化的 DistilBERT token-classification 模型**（~50MB）。推理通过 **2 Worker 串行执行模型** 编排，以应对 ONNX Runtime 的线程不安全 Session 约束。

### 架构

```
用户输入文本
    │
    ▼
┌─────────────────────────────────────────┐
│  L2 NER 阶段 (internal/application/)     │
│  ┌─────────────────────────────────┐   │
│  │  PipelineInput (文本 + 元数据)   │   │
│  └─────────────┬───────────────────┘   │
│                │                        │
│  ┌─────────────▼───────────────────┐   │
│  │  2-Worker ONNX 推理池            │   │
│  │  ┌─────────┐    ┌─────────┐    │   │
│  │  │ Worker 0│    │ Worker 1│    │   │
│  │  │ Session │    │ Session │    │   │
│  │  │ (Mutex) │    │ (Mutex) │    │   │
│  │  └────┬────┘    └────┬────┘    │   │
│  │       └──────────────┘         │   │
│  │          (缓冲 Channel)         │   │
│  └─────────────┬───────────────────┘   │
│                │                        │
│  ┌─────────────▼───────────────────┐   │
│  │  EntityList → 敏感分级           │   │
│  │  (P1 公开 / P2 内部 / P3 机密)   │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### 关键技术选择

1. **模型**: 针对 token classification 微调后的 DistilBERT（BiLSTM-CRF 头），导出为 ONNX，然后 int8 量化。量化将模型体积从 ~250MB（fp32）压缩到 ~50MB，在医疗实体基准上准确率损失 <2%。

2. **Worker 模型**: 2 个固定推理 Worker，每个持有独立的 ONNX Session（每个 Session ~80–100MB RAM，总计 ~200MB）。任务通过有缓冲 channel（容量 16）派发。每个 Worker 串行化 `Session.Run()` 调用，因为 ONNX Runtime Session **非线程安全**。

3. **Go 绑定**: `github.com/knights-analytics/hugot` 提供高级 Pipeline API（`TokenClassificationPipeline`），将 ONNX Runtime C API 调用抽象为地道 Go 代码：
   ```go
   pipeline, err := hugot.NewTokenClassificationPipeline(
       sessionOptions, hugot.TokenClassificationConfig{
           ModelPath: modelPath,
           MaxLength: 512,
           Truncation: true,
       })
   ```

4. **平台特定库分发**: ONNX Runtime 动态库按平台打包：
   - macOS: `libonnxruntime.dylib` (~15MB)
   - Windows: `onnxruntime.dll` (~15MB)
   - Linux: `libonnxruntime.so` (~15MB)
   `internal/infrastructure/onnx/` 包在运行时通过 `runtime.GOOS` 选择正确的库路径。

5. **初始化策略**: ONNX Session 初始化在应用启动时**后台异步**进行。目标：<400ms 初始化延迟，不阻塞首屏渲染（PER-03）。

### 与脱敏流水线的集成

L2 NER 阶段位于 L1（规则引擎，<1ms）和 L3（关键词字典，<5ms）之间：

```
L1 规则引擎 ──(未命中片段)──► L2 NER ONNX ──(低置信度片段)──► L3 关键词
         │                                    │
         └───── P3 硬替换 ────────────────────┘
```

- L1 以 `Confidence == 1.0` 识别的片段完全跳过 L2（流水线短路）。
- L2 NER 输出按 P1/P2/P3 敏感等级分级。P3 片段在调用任何云端 API 前执行**硬替换**（不可逆掩码）。
- 低置信度 L2 输出（<0.7）作为兜底转发到 L3 关键词字典匹配。

## 结果

### 积极影响

- **隐私保证**: 用户健康文本在实体识别阶段永远不会离开设备；NER 模型完全在进程内运行。
- **离线韧性**: 脱敏在无网络情况下也能工作，满足离线优先原则。
- **准确性**: DistilBERT NER 在医疗实体识别基准上显著优于纯规则或纯 Go 方案。
- **模型可升级性**: 用户可以在 `resources/models/` 中替换 ONNX 模型文件，无需重新编译应用。

### 消极影响

- **CGO 交叉编译负担**: 为三平台构建需要平台特定的 ONNX Runtime 库和 CGO 工具链配置，使 CI 复杂化。
- **内存压力**: 2 个 ONNX Session 占用约 200MB RAM。在 <8GB RAM 的系统上，这与 DuckDB、本地 LLM（Ollama）和操作系统争夺资源。
- **非线程安全 Session 约束**: 2 Worker 串行模型限制了 NER 吞吐。在突发输入场景（如批量健康记录导入）下，请求在 channel 中排队。
- **模型体积**: 50MB 是资源目录中最大的单一文件；在慢速连接上的首次下载可能降低首次启动体验。

## 替代方案

| 替代方案 | 拒绝原因 |
|:------------|:-----------------|
| 远程 NER API（云 NLP 服务） | 违反核心隐私支柱；健康文本将在脱敏前离开设备，使流水线目的落空 |
| 纯 Go token classification（`prose` v3 + 自定义规则） | 医疗领域准确率不足；无法以可接受的精度识别罕见疾病名、药物别名或医院名 |
| TensorFlow Lite Go 绑定 | Go 生态较弱；预训练医疗 NER 模型更少；跨平台库分发比 ONNX Runtime 更复杂 |
| 在独立进程中运行 NER（gRPC/IPC） | 增加进程管理复杂度；内存节省不显著（模型仍需加载）；IPC 延迟开销 |

## 相关文档

- [docs/ARCHITECTURE.md](../ARCHITECTURE.md) — 系统架构概览与数据流
- `internal/infrastructure/onnx/` — ONNX Runtime Go 绑定与 Session 管理
- `pkg/desensitizer/` — 规则引擎（L1）与流水线编排器
- `internal/application/pipeline/` — 三级脱敏流水线
