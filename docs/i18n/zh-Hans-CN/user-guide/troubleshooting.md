# 故障排查

> 🌐 [English Version](../../../user-guide/troubleshooting.md)

> 常见问题及解决方案。如果你的问题不在此处，请 [提交 GitHub Issue](https://github.com/hzhan516/medmemo/issues)。

---

## 安装问题

### 应用无法启动（Windows）

**症状：** 双击安装包或应用无反应，或弹出错误对话框。

**解决方案：**
1. 确认 Windows 版本为 10 或更高（`Win + R`，输入 `winver`）
2. 尝试以普通用户运行（非管理员）
3. 若 SmartScreen 拦截：点击 **更多信息 → 仍要运行**
4. 检查 Windows Defender 排除项 —— 若误报请将 MedMemo 加入排除
5. 从 GitHub Releases 重新下载安装包（文件可能损坏）

### 应用无法启动（macOS）

**症状：** 提示 "MedMemo 无法打开，因为无法验证开发者"。

**解决方案：**
1. 打开 **系统设置 → 隐私与安全性**
2. 向下滚动点击 **仍要打开**
3. 若仍无法打开，在终端执行：
   ```bash
   xattr -d com.apple.quarantine /Applications/MedMemo.app
   ```
4. 确保 macOS 版本为 12 (Monterey) 或更高

### 应用无法启动（Linux）

**症状：** AppImage 无法启动，或提示库错误。

**解决方案：**
1. 赋予执行权限：`chmod +x MedMemo.AppImage`
2. 安装缺失依赖：
   - Ubuntu/Debian：`sudo apt install libwebkit2gtk-4.1-0`
   - Fedora：`sudo dnf install webkit2gtk4.1`
3. 若使用 Wayland，尝试：`./MedMemo.AppImage --enable-features=UseOzonePlatform --ozone-platform=wayland`
4. 检查 AppImage 与当前发行版版本的兼容性

---

## 数据库问题

### "database key verification failed"

**症状：** MedMemo 启动失败，日志出现 `database key verification failed`。

**根因：** 系统密钥链中 SQLCipher 数据库密钥丢失、损坏，或由不同用户配置文件创建。

**解决方案：**
1. 不要手动删除或修改数据目录中的文件。
2. 若近期重装系统或清空了系统密钥链，数据库密钥不可恢复 —— 如有备份请恢复。
3. 作为最后手段，可将旧数据目录移走，让 MedMemo 重新创建：
   - Windows：`%LOCALAPPDATA%\medmemo\`
   - macOS：`~/Library/Application Support/medmemo/`
   - Linux：`~/.local/share/medmemo/`

### "failed to open sqlcipher database"

**症状：** 启动时报 `failed to open sqlcipher database`。

**根因：** 数据目录不可写、磁盘已满，或数据库文件被其他进程锁定。

**解决方案：**
1. 检查磁盘空间。
2. 确保运行 MedMemo 的用户对 `MEDMEMO_DATA_DIR` 有写权限。
3. 关闭可能持有数据库锁的其他 MedMemo 实例。
4. 若默认路径受限，可设置 `MEDMEMO_DATA_DIR` 到可写路径。

### "failed to migrate plaintext database"

**症状：** 从旧版本首次启动后出现迁移错误。

**根因：** MedMemo 正在将未加密的 SQLite 数据库迁移到 SQLCipher。

**解决方案：**
1. 升级前备份数据目录。
2. 确保系统密钥链可访问（Linux：Secret Service / GNOME Keyring 正在运行）。
3. 若迁移持续失败，请恢复备份并 [提交 Issue](https://github.com/hzhan516/medmemo/issues)。

---

## ONNX / Embedding 问题

### "ONNX Runtime library not found"

**症状：** 本地 NER 或 embedding 不可用，日志出现 `ONNX Runtime library not found`。

**根因：** ONNX Runtime 原生库未下载，或不在 `resources/lib/<platform>/` 中。

**解决方案：**
1. 运行 `make download-resources`（Linux/macOS 运行 `scripts/build/download-onnx.sh`，Windows 运行 `.ps1`）。
2. 确认以下文件之一存在：
   - `resources/lib/linux/libonnxruntime.so`
   - `resources/lib/darwin/libonnxruntime.dylib`
   - `resources/lib/windows/onnxruntime.dll`
3. 在 Linux 源码运行时，确保 `LD_LIBRARY_PATH` 包含 `resources/lib/linux`。

### "embedding model dir not found" / "NER model dir not found"

**症状：** 日志出现 `embedding model dir not found: ...` 或 `NER model dir not found: ...`。

**根因：** `resources/models/distilbert-ner/` 中缺少 bundled 的 DistilBERT NER 模型。

**解决方案：**
1. 运行 `make download-resources` 下载模型。
2. 确认 `resources/models/distilbert-ner/model.onnx` 存在。
3. 若移动了模型位置，设置 `MEDMEMO_MODEL_DIR` 指向新目录。

### "embedding pipeline creation failed"

**症状：** Embedding 功能失败，报错 `embedding pipeline creation failed`。

**根因：** ONNX 模型与 ONNX Runtime 库版本不兼容，或 tokenizer 静态库缺失。

**解决方案：**
1. 运行 `make download-resources` 确保 tokenizer 静态库存在。
2. 检查 ONNX Runtime 版本是否与 `go.mod` 中的 Hugot / ortgenai 版本匹配。
3. 删除 `resources/lib/` 和 `resources/models/` 后重新运行 `make download-resources`。

---

## 认证问题

### "需要配置 OAuth client_id"

**症状：** 启动 OAuth Device Flow 时报错，包含 `需要配置 OAuth client_id`。

**根因：** OAuth Device Flow 需要你自行注册每个厂商的 `client_id`。MedMemo 作为开源项目不预置 OAuth 客户端。

**解决方案：**
1. 在对应厂商处注册 OAuth 应用。
2. 设置对应环境变量：
   - Kimi：`MEDMEMO_KIMI_CLIENT_ID`
   - Gemini：`MEDMEMO_GEMINI_CLIENT_ID`
   - Microsoft：`MEDMEMO_MICROSOFT_CLIENT_ID`
   - GitHub：`MEDMEMO_GITHUB_CLIENT_ID`
3. 重启 MedMemo 后重试。

详见 [`docs/api/auth.md`](../../../api/auth.md)。

### CLI Token 未检测到

**症状：** `DetectAuthMethods` 报告 `cli_token` 不可用，或 `BuildCLIProvider` 失败。

**根因：** 厂商 CLI 凭证文件不存在、为空或无法解析。

**解决方案：**
1. 使用厂商 CLI 登录（如 `kimi auth login`、`gcloud auth login`）。
2. 检查报错中提到的凭证文件路径。
3. 常见错误提示：
   - `credential file is empty` —— 重新执行厂商登录命令。
   - `failed to parse credential` —— 凭证文件格式可能已变更，升级厂商 CLI。
   - `unsupported cli provider type` —— 该厂商不支持 CLI Token 自动检测。

### Service Account JSON 解析错误

**症状：** `ParseServiceAccountJSON` 返回 `failed to parse service account JSON` 或 `invalid service account type`。

**根因：** Google Service Account JSON 格式错误，或下载了错误的凭证类型。

**解决方案：**
1. 从 Google Cloud Console → IAM 和管理 → 服务账号下载 JSON 密钥。
2. 确认文件中包含 `"type": "service_account"`。
3. 确认包含 `project_id`、`client_email` 和 `private_key`。

---

## Ollama 问题

### "ollama not reachable"

**症状：** 本地模型检测报告 `ollama not reachable`。

**根因：** Ollama 服务未运行，或未监听预期 URL。

**解决方案：**
1. 启动 Ollama：`ollama serve`
2. 在设置中验证服务 URL（默认：`http://localhost:11434`）。
3. 检查防火墙是否拦截 `localhost:11434`。

### "ollama server did not become ready within ..."

**症状：** MedMemo 尝试自动启动 `ollama serve`，但超时。

**根因：** Ollama 已安装，但初始化时间超过检测超时。

**解决方案：**
1. 手动启动 Ollama：`ollama serve`
2. 等待 `ollama list` 可用后，再在 MedMemo 中重试。
3. 若未安装 Ollama，请从 [ollama.com](https://ollama.com) 下载。

### "ollama pull ... failed"

**症状：** 通过 MedMemo 下载模型时报 `ollama pull ... failed`。

**根因：** 网络问题、磁盘空间不足，或模型名称无效。

**解决方案：**
1. 手动拉取模型：`ollama pull llama3.1:8b`
2. 确保磁盘空间充足（8B 模型约需 6 GB，70B 模型约需 40 GB）。
3. 确认模型名称在 Ollama 官方库中有效。

### "ollama returned HTTP 404"

**症状：** 使用本地模型聊天时报 `ollama returned HTTP 404`。

**根因：** 请求的模型未在本地下载。

**解决方案：**
1. 运行 `ollama list` 查看已下载模型。
2. 拉取缺失模型：`ollama pull <模型名>`。
3. 在 MedMemo 设置中选择 `ollama list` 中存在的模型。

---

## 流式生成问题

### "stream execution failed"

**症状：** 流式响应突然中断，UI 出现红色错误条，或日志出现 `stream execution failed`。

**根因：** 网络中断、厂商报错，或流式处理过程中出现 panic。

**解决方案：**
1. 检查网络连接。
2. 在设置中切换到其他模型或厂商。
3. 若使用本地模型，确认 Ollama 仍在运行。
4. 重启 MedMemo 后重试该会话。

### 流式响应为空或截断

**症状：** AI 消息气泡为空，或只显示预期内容的一部分。

**根因：** 厂商返回了空流，或合规 / 紧急症状拦截替换了内容。

**解决方案：**
1. 检查聊天区域是否有系统提示（橙色警告条 / 蓝色免责声明条）。
2. 查看应用日志中的 `chat:stream:compliance` 事件。
3. 若输入触发了紧急症状，请阅读警告并决定是否继续咨询。

### 流式过程中反复 Panic

**症状：** MedMemo 在长流式响应期间崩溃或无响应。

**根因：** 流式回调路径出现运行时 panic（如 UI 竞态），虽已恢复，但应用状态可能不一致。

**解决方案：**
1. 保存重要上下文后重启 MedMemo。
2. 若可复现，请记录模型、消息长度、是否有合规警告，并 [提交 Issue](https://github.com/hzhan516/medmemo/issues)。

---

## 隐私与安全问题

### 脱敏似乎未生效

**症状：** 怀疑敏感信息未经替换即被发送。

**验证方法：**
1. 检查 **设置 → 隐私 → 脱敏级别**，确保未设置为 "关闭"。
2. 脱敏过程自动静默进行。
3. 主动验证：发送包含假手机号 `13800138000` 的消息 —— 若 AI 引用你的输入，响应中不应包含该号码。

### 忘记数据存储位置

**位置：**
- **Windows**：`%LOCALAPPDATA%\medmemo\`
- **macOS**：`~/Library/Application Support/medmemo/`
- **Linux**：`~/.local/share/medmemo/`

> ⚠️ 这些文件夹包含加密数据，请勿手动编辑。

---

## UI 与界面问题

### 侧边栏消失

**症状：** 看不到会话列表。

**解决方案：**
1. 若窗口较窄（< 768px），侧边栏会自动折叠为图标。点击 **面板图标** 展开。
2. 若完全折叠，点击折叠侧边栏左上角的 **展开箭头**。
3. 将窗口拉宽以恢复完整侧边栏。

### 深色模式显示异常

**症状：** 部分 UI 元素在深色模式下显示不正确。

**解决方案：**
1. 在 **设置 → 外观** 中切换主题后再切回。
2. 重启 MedMemo。
3. 若 Linux 使用自定义主题，MedMemo 可能意外继承系统颜色。

### 文字过小或过大

**症状：** 聊天文字难以阅读。

**解决方案：**
1. 使用系统级缩放（MedMemo 会遵循系统显示缩放）。
2. 调整窗口大小 —— 聊天区域会自适应。
3. 未来版本将提供应用内字体大小控制。

---

## 性能问题

### 内存占用过高

**症状：** MedMemo 占用 500MB+ 内存。

**预期占用：**
- 基础应用：约 100–200 MB
- 加载 ONNX 模型后：额外 100–200 MB
- 长会话：逐步增长（受清理机制限制）

**若占用过高：**
1. 重启 MedMemo。
2. 关闭不用的会话（仍保留在侧边栏，但释放渲染内存）。
3. 若使用本地 Ollama 模型，内存占用取决于模型大小（8B 约 6GB，70B 约 40GB）。

### 应用卡顿

**症状：** UI 响应慢，滚动不流畅。

**解决方案：**
1. 减少可见消息数量（滚动加载更多）。
2. 检查系统是否内存不足。
3. 禁用不必要的浏览器扩展（MedMemo 使用 WebView）。
4. 在 Linux 上确保 WebKit 可用 GPU 加速。

---

## 更新问题

### 更新检查失败

**症状：** 提示 "Failed to check for updates"。

**解决方案：**
1. 检查网络连接。
2. GitHub 可能触发速率限制 —— 稍后再试。
3. 若处于企业代理后，配置系统代理。
4. 可随时手动从 GitHub Releases 下载最新版本。

### 更新下载中断

**症状：** 下载到一半停止，无法继续。

**解决方案：**
1. 下次启动时更新会自动重试。
2. 或手动从 GitHub Releases 下载并重新安装。

---

## 问题反馈

如果以上方案均未解决：

1. **查看已有 Issue**：[github.com/hzhan516/medmemo/issues](https://github.com/hzhan516/medmemo/issues)
2. **新建 Issue** 时提供：
   - MedMemo 版本（来自 **设置** 或 `帮助 → 关于`）
   - 操作系统及版本
   - 复现步骤
   - 预期行为与实际行为
   - 相关截图（如为 UI 问题）

---

*最后更新：2026-07-14*
