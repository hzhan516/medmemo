# 安装指南

> 🌐 [English Version](./../../../user-guide/installation.md)

> 本指南指导你在 Windows、macOS 和 Linux 上安装 MedMemo。

---

## 系统要求

| 平台 | 最低版本 | 架构 | 磁盘空间 |
|------|---------|------|---------|
| Windows | Windows 10 | x64, ARM64 | 200 MB |
| macOS | macOS 12 (Monterey) | Intel, Apple Silicon | 200 MB |
| Linux | Ubuntu 20.04+ / Fedora 38+ | x64, ARM64 | 200 MB |

**额外要求：**
- 互联网连接（用于下载 AI 模型和访问云端 API）
- 至少一个支持提供商的 API Key（Kimi、OpenAI、阿里云，或本地 Ollama 实例）

---

## 下载

从 [GitHub Releases](https://github.com/hzhan516/medmemo/releases) 下载最新版本。

| 平台 | 文件 | 说明 |
|------|------|------|
| Windows | `MedMemoSetup.exe` | NSIS 安装程序 |
| macOS | `MedMemo.dmg` | 拖拽式安装包 |
| Linux | `MedMemo.AppImage` | 便携版，无需安装 |

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

### 第 3 步：完成引导

首次启动将显示引导向导。详见 [首次设置](./README.md#首次设置3-步)。

### 卸载

1. 打开**设置 → 应用 → 已安装的应用**
2. 找到 **MedMemo**
3. 点击**卸载**
4. （可选）删除用户数据文件夹 `%LOCALAPPDATA%\medmemo`

---

## macOS

### 第 1 步：打开 DMG

1. 下载 `MedMemo.dmg`
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
2. （可选）删除用户数据 `~/Library/Application Support/medmemo`

---

## Linux

### AppImage（推荐）

1. 下载 `MedMemo.AppImage`
2. 赋予执行权限：
   ```bash
   chmod +x MedMemo.AppImage
   ```
3. 运行：
   ```bash
   ./MedMemo.AppImage
   ```

> **Fedora 43+ 用户**：如果遇到 WebKit 问题，请确保已安装 `webkit2gtk4.1`：
> ```bash
> sudo dnf install webkit2gtk4.1
> ```

### 从源码构建

如果你希望从源码构建：

```bash
# 1. 前置条件
go version  # 需要 Go 1.22+
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

直接删除 AppImage 文件即可。用户数据存储在 `~/.local/share/medmemo`。

---

## 安装后检查清单

安装完成后，验证一切正常：

- [ ] MedMemo 无错误启动
- [ ] 首次启动出现引导向导
- [ ] 可以接受免责声明并进入主对话界面
- [ ] 侧边栏显示"新建对话"按钮
- [ ] 可以在输入框中输入消息
- [ ] 设置页面可以打开（标题栏齿轮图标）

如有任何步骤失败，请查阅 [故障排查](./troubleshooting.md)。

---

## 更新 MedMemo

MedMemo 支持自动更新检测：

1. 前往 **设置 → 自动更新**
2. 开启**"自动检测更新"**
3. 选择通道：
   - **稳定版** — 仅正式版本
   - **测试版** — 包含预发布版本，优先体验新功能

有可用更新时，会弹出通知。你可以直接在应用内下载并安装。

> ⚠️ 安全更新可能被标记为**强制更新**，不安装将无法继续使用。

---

*最后更新：2026-05-19*
