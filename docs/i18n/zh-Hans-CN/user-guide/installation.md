# 安装指南

> 🌐 [English Version](./../../../user-guide/installation.md)

> 本指南指导你在 Windows、macOS 和 Linux 上安装 MedMemo。

---

## 系统要求

| 平台 | 最低版本 | 架构 | 磁盘空间 |
|------|---------|------|---------|
| Windows | Windows 10 | x64, ARM64 | 250 MB |
| macOS | macOS 12 (Monterey) | Intel, Apple Silicon | 200 MB |
| Linux | Ubuntu 22.04+ / Fedora 38+ | x64, ARM64 | 200 MB |

> **Windows 用户注意：** 首次启动时，若安装包未捆绑 AI 运行库，MedMemo 可能会自动下载约 55 MB 的本地 AI 运行库（ONNX Runtime + Tokenizers）。首次运行需要互联网连接。

**额外要求：**
- 互联网连接（用于下载 AI 模型和访问云端 API）
- 至少一个支持提供商的 API Key（Kimi、OpenAI、阿里云，或本地 Ollama 实例）

---

## 下载

从 [GitHub Releases](https://github.com/hzhan516/medmemo/releases) 下载最新版本。

| 平台 | 文件 | 说明 |
|------|------|------|
| Windows | `MedMemoSetup.exe` | NSIS 单用户安装程序 |
| macOS (Intel) | `MedMemo_x86_64.dmg` | Intel Mac 拖拽安装包 |
| macOS (Apple Silicon) | `MedMemo_arm64.dmg` | M1/M2/M3 Mac 拖拽安装包 |
| Linux (AppImage) | `MedMemo-x86_64.AppImage` | 便携版，无需安装 |
| Linux (DEB) | `MedMemo_*.deb` | Debian/Ubuntu 安装包 |
| Linux (RPM) | `MedMemo-*.rpm` | Fedora/openSUSE 安装包 |

---

## Windows

### 第 1 步：运行安装程序

1. 下载 `MedMemoSetup.exe`
2. 双击运行
3. 如果出现 Windows SmartScreen，点击**更多信息 → 仍要运行**
4. 按向导提示完成安装
5. 选择是否创建开始菜单快捷方式和桌面图标

### 第 2 步：启动 MedMemo

- 从开始菜单：**MedMemo**
- 从桌面快捷方式（安装时选择创建）

### 第 3 步：本地 AI 模型初始化（首次启动）

首次启动时，MedMemo 会检查本地 AI 运行库（用于语义记忆检索和敏感信息检测）：

- **如果库已捆绑** — 自动继续，无需额外操作。
- **如果需要下载** — 将自动开始下载（约 55 MB），根据网络状况通常需要 10–30 秒。

> 💡 下载的库保存到 `%LOCALAPPDATA%\medmemo\lib\`，后续启动直接复用。升级 MedMemo 时才可能需要重新下载。

### 第 4 步：完成引导

首次启动将显示引导向导。详见 [首次设置](./README.md#首次设置3-步)。

### Windows 数据存储位置

MedMemo 在 Windows 上将数据存放在以下位置之一：

| 场景 | 默认数据位置 |
|------|-------------|
| 从 v1.1.9 或更早版本升级（存在旧库） | `%USERPROFILE%\.medmemo\data` |
| 全新安装且安装目录可写 | `<installDir>\data`（例如 `%LOCALAPPDATA%\Programs\MedMemo\data`） |
| 安装目录缺失或不可写 | `%USERPROFILE%\.medmemo\data` |
| 已设置 `MEDMEMO_DATA_DIR` 或 `config.yaml` 中的 `data_dir` | 配置指定的路径 |

MedMemo 不会在不同位置之间自动移动或合并数据库。如果旧库文件夹和安装目录同时存在数据，默认优先使用旧库，直到你通过 `MEDMEMO_DATA_DIR` 显式切换。

### 卸载

1. 打开**设置 → 应用 → 已安装的应用**
2. 找到 **MedMemo**
3. 点击**卸载**
4. （可选）删除用户数据文件夹。根据上述场景，可能是 `<installDir>\data` 或 `%USERPROFILE%\.medmemo\data`。

---

## macOS

### 第 1 步：打开 DMG

1. 根据你的 Mac 下载对应的 DMG：
   - **Intel Mac：** `MedMemo_x86_64.dmg`
   - **Apple Silicon Mac（M1/M2/M3）：** `MedMemo_arm64.dmg`
2. 双击挂载磁盘镜像
3. 将 **MedMemo.app** 拖拽到**应用程序**文件夹

### 第 2 步：首次启动

macOS 可能在首次启动时拦截应用（开源应用未做 Apple 公证属于正常现象）。

**方案 A — 系统设置：**
1. 尝试打开 MedMemo（会被拦截）
2. 打开**系统设置 → 隐私与安全性**
3. 向下滚动，在拦截消息旁点击**仍要打开**

**方案 B — 终端：**
```bash
xattr -d com.apple.quarantine /Applications/MedMemo.app
```

### 第 3 步：完成引导

详见 [首次设置](./README.md#首次设置3-步)。

### 卸载

1. 将 **MedMemo.app** 从应用程序拖到废纸篓
2. （可选）删除用户数据 `~/.medmemo/data`

---

## Linux

### AppImage（推荐）

1. 下载 `MedMemo-x86_64.AppImage`
2. 赋予执行权限：
   ```bash
   chmod +x MedMemo-x86_64.AppImage
   ```
3. 运行：
   ```bash
   ./MedMemo-x86_64.AppImage
   ```

> **Fedora 43+ 用户**：如果遇到 WebKit 问题，请确保已安装 `webkit2gtk4.1`：
> ```bash
> sudo dnf install webkit2gtk4.1
> ```

### DEB 安装包（Debian/Ubuntu）

1. 下载 `MedMemo_*.deb`
2. 使用包管理器安装：
   ```bash
   sudo dpkg -i MedMemo_*.deb
   sudo apt-get install -f  # 如有依赖缺失
   ```
3. 从应用菜单启动，或在终端运行 `medmemo`。

### RPM 安装包（Fedora/openSUSE）

1. 下载 `MedMemo-*.rpm`
2. 使用包管理器安装：
   ```bash
   sudo dnf install MedMemo-*.rpm
   # 或在 openSUSE 上：
   sudo zypper install MedMemo-*.rpm
   ```
3. 从应用菜单启动，或在终端运行 `medmemo`。

### DEB/RPM 更新

MedMemo 检测到新版本时，会下载对应的 `.deb` 或 `.rpm` 安装包并提示文件路径。请使用与初次安装相同的命令手动安装更新。

### 从源码构建

如果你希望从源码构建：

```bash
# 1. 前置条件
go version  # 需要 Go 1.26.4+
node --version  # 需要 Node.js 18+

# 2. 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 3. 克隆并构建
git clone https://github.com/hzhan516/medmemo.git
cd medmemo
go mod download
cd web && npm install && cd ..
make build
```

构建产物位于 `build/bin/medmemo`。

> **Fedora 43+**：需在构建标志中添加 `-tags webkit2_41`。

### 卸载

- **AppImage：** 删除 AppImage 文件。
- **DEB：** `sudo dpkg -r medmemo`
- **RPM：** `sudo dnf remove medmemo`

（可选）卸载任意 Linux 安装包后，删除用户数据 `~/.medmemo/data`。

---

## 安装后检查清单

安装完成后，验证一切正常：

- [ ] MedMemo 无错误启动
- [ ] 首次启动出现引导向导
- [ ] 可以接受免责声明并进入主对话界面
- [ ] 侧边栏显示"新建对话"按钮
- [ ] 可以在输入框中输入消息
- [ ] 设置页面可以打开（标题栏齿轮图标）

### Windows 专属问题

#### "本地 AI 模型初始化失败" 或下载错误

如果首次启动时自动下载库失败：

1. **检查网络连接** — 下载需要稳定的互联网连接。
2. **检查 Windows Defender / 杀毒软件** — 可能拦截下载。尝试临时关闭或将 MedMemo 加入白名单。
3. **手动下载**（高级用户）：
   ```powershell
   # 在 MedMemo 安装目录下以管理员身份运行 PowerShell
   .\scripts\build\download-onnx.ps1 -Platform windows
   .\scripts\build\download-tokenizers.ps1
   ```
4. **防火墙 / 企业代理** — 如果处于受限代理环境，可在启动前设置代理：
   ```powershell
   $env:HTTP_PROXY = "http://proxy.company.com:8080"
   ```

#### Windows 上 Embedding 功能不可用

如果看到"语义搜索不可用"或"embedding 引擎不可用"：

- 说明 ONNX Runtime 或 Tokenizers 库加载失败。
- MedMemo 会自动降级为基于关键词的记忆检索，无需 embedding 也可正常工作。
- **核心对话功能不受影响**。

如有其他步骤失败，请查阅 [故障排查](./troubleshooting.md)。

---

## 更新 MedMemo

MedMemo 支持自动更新检测：

1. 前往 **设置 → 自动更新**
2. 开启**"自动检测更新"**
3. 选择通道：
   - **稳定版** — 仅正式版本
   - **测试版** — 包含预发布版本，优先体验新功能

有可用更新时，会弹出通知。

- **Windows / macOS / Linux AppImage：** 应用会在可能的情况下自动下载并安装更新。
- **Linux DEB / RPM：** 应用会探测当前安装的包类型（通过 `.install_kind` 标记文件或查询 `dpkg`/`rpm`），然后下载对应安装包并提示文件路径。请使用与初次安装相同的命令手动安装。
- **无法识别的 Linux 安装方式：** 如果 MedMemo 无法确定你安装的是 DEB、RPM 还是 AppImage，会打开 Release 页面，由你手动下载适合的安装包。

> ⚠️ 如果你之前安装的是候选版本（rc）的 DEB/RPM 包，在 v1.1.10 正式版发布后需要手动下载并覆盖安装一次正式版 DEB/RPM。更新器不会自动切换包格式。
>
> ⚠️ 安全更新可能被标记为**强制更新**，不安装将无法继续使用。

---

*最后更新：2026-08-05*
