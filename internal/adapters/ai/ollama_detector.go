package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/medmemo/medmemo/pkg/models"
)

// defaultOllamaEndpoint Ollama 默认 HTTP 端点。
const defaultOllamaEndpoint = "http://localhost:11434"

// DefaultModelName SmolLM2 默认模型名，供外部引用。
const DefaultModelName = "smollm2:135m"

// defaultCheckTimeout 单次 Ollama API 检测超时。
const defaultCheckTimeout = 2 * time.Second

// defaultWaitInterval 等待服务就绪的轮询间隔。
const defaultWaitInterval = 500 * time.Millisecond

// OllamaDetector 负责检测本地 Ollama 运行时环境的状态，
// 并在缺失时引导启动或下载模型。
type OllamaDetector struct {
	endpoint string
	client   *http.Client

	// lookPath 用于查找可执行文件路径，默认为 exec.LookPath，
	// 测试时可注入 mock 以控制安装状态。
	lookPath func(string) (string, error)
}

// NewOllamaDetector 使用默认端点创建检测器。
func NewOllamaDetector() *OllamaDetector {
	return NewOllamaDetectorWithEndpoint(defaultOllamaEndpoint)
}

// NewOllamaDetectorWithEndpoint 使用指定端点创建检测器。
func NewOllamaDetectorWithEndpoint(endpoint string) *OllamaDetector {
	return &OllamaDetector{
		endpoint: endpoint,
		client:   &http.Client{Timeout: defaultCheckTimeout},
		lookPath: exec.LookPath,
	}
}

// NewOllamaDetectorWithClient 使用自定义 HTTP 客户端创建检测器，
// 主要用于测试注入 mock transport。
func NewOllamaDetectorWithClient(endpoint string, client *http.Client) *OllamaDetector {
	return &OllamaDetector{
		endpoint: endpoint,
		client:   client,
		lookPath: exec.LookPath,
	}
}

// IsInstalled 检测 ollama 命令是否在系统 PATH 中。
func (d *OllamaDetector) IsInstalled() bool {
	_, err := d.lookPath("ollama")
	return err == nil
}

// IsRunning 检测 Ollama HTTP 服务是否响应（GET /api/tags）。
func (d *OllamaDetector) IsRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// tagModel 表示 Ollama /api/tags 返回的单个模型信息。
type tagModel struct {
	Name string `json:"name"`
}

// tagsResponse 表示 Ollama /api/tags 的响应结构。
type tagsResponse struct {
	Models []tagModel `json:"models"`
}

// HasModel 检测指定模型是否已在本地存在。
func (d *OllamaDetector) HasModel(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var result tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	for _, m := range result.Models {
		if m.Name == name {
			return true
		}
	}
	return false
}

// StartServer 在后台启动 ollama serve 进程。
// 返回的 *exec.Cmd 已启动但未被等待；调用方应通过 WaitForServer 确认就绪。
// 注意：本方法不跟踪进程生命周期——ollama serve 作为系统守护进程应独立运行。
func (d *OllamaDetector) StartServer() (*exec.Cmd, error) {
	cmd := exec.Command("ollama", "serve")
	setDetachedAttr(cmd)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ollama serve: %w", err)
	}
	return cmd, nil
}

// WaitForServer 轮询等待 Ollama HTTP 服务就绪，直到超时。
func (d *OllamaDetector) WaitForServer(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if d.IsRunning() {
			return nil
		}
		time.Sleep(defaultWaitInterval)
	}
	return fmt.Errorf("ollama server did not become ready within %v", timeout)
}

// PullModel 执行 ollama pull 下载指定模型，通过 onProgress 回调实时推送 stderr 进度。
func (d *OllamaDetector) PullModel(ctx context.Context, name string, onProgress func(string)) error {
	cmd := exec.CommandContext(ctx, "ollama", "pull", name)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe for ollama pull: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama pull %s: %w", name, err)
	}

	// 实时读取 stderr，逐行推送进度
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && onProgress != nil {
			onProgress(line)
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ollama pull %s failed: %w", name, err)
	}

	return nil
}

// GetInstallGuide 返回适合当前平台的 Ollama 安装引导文本。
func (d *OllamaDetector) GetInstallGuide() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS 安装 Ollama：\n\n" +
			"方式 1（推荐）: 打开终端执行\n  curl -fsSL https://ollama.com/install.sh | sh\n\n" +
			"方式 2: 使用 Homebrew\n  brew install ollama"
	case "linux":
		return "Linux 安装 Ollama：\n\n" +
			"打开终端执行\n  curl -fsSL https://ollama.com/install.sh | sh"
	case "windows":
		return "Windows 安装 Ollama：\n\n" +
			"1. 访问 https://ollama.com/download/windows 下载安装程序\n" +
			"2. 运行下载的 .exe 文件完成安装"
	default:
		return "安装 Ollama：\n\n" +
			"请访问 https://ollama.com 获取适合您平台的安装指南"
	}
}

// BuildProviderConfig 根据检测状态构建 Ollama ProviderConfig。
// AuthMethod 使用 api_key（Ollama 本地服务无需认证，APIKey 留空）。
func (d *OllamaDetector) BuildProviderConfig() *models.ProviderConfig {
	nowMs := time.Now().UnixMilli()
	return &models.ProviderConfig{
		ID:          "ollama-local-" + strconv.FormatInt(nowMs, 10),
		Name:        "Ollama (本地)",
		APIHost:     d.endpoint,
		ModelID:     DefaultModelName,
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "本地",
		Enabled:     true,
		AuthMethod:  models.AuthMethodAPIToken,
		CreatedAt:   nowMs,
		UpdatedAt:   nowMs,
	}
}

// DetectResult 汇总 Ollama 环境检测的各项状态。
type DetectResult struct {
	Installed    bool   // ollama 命令是否存在于 PATH
	Running      bool   // 11434 端口是否响应
	HasSmolLM2   bool   // smollm2:135m 模型是否已下载
	InstallGuide string // 未安装时返回的安装引导文本
}

// Detect 执行完整的 Ollama 环境检测，返回汇总结果。
func (d *OllamaDetector) Detect() *DetectResult {
	result := &DetectResult{
		Installed: d.IsInstalled(),
	}

	if !result.Installed {
		result.InstallGuide = d.GetInstallGuide()
		return result
	}

	result.Running = d.IsRunning()
	if result.Running {
		result.HasSmolLM2 = d.HasModel(DefaultModelName)
	}

	return result
}
