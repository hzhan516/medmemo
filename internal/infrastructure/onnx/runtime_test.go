package onnx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEngine_NoModel_ReturnsUnavailable 验证无模型时引擎降级为不可用，不 panic。
func TestNewEngine_NoModel_ReturnsUnavailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err, "无模型时不应返回错误")
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "模型缺失时引擎应标记为不可用")

	err = engine.Close()
	assert.NoError(t, err)
}

// TestNewEngine_NoLibrary_ReturnsUnavailable 验证无动态库时引擎降级为不可用。
func TestNewEngine_NoLibrary_ReturnsUnavailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "/nonexistent/resources", ModelPath: "resources/models/distilbert-ner"})
	require.NoError(t, err, "无动态库时不应返回错误")
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "动态库缺失时引擎应标记为不可用")

	err = engine.Close()
	assert.NoError(t, err)
}

// TestEngine_Predict_WhenUnavailable_ReturnsError 验证不可用引擎调用 Predict 返回错误。
func TestEngine_Predict_WhenUnavailable_ReturnsError(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	spans, err := engine.Predict(ctx, "测试文本")
	assert.Error(t, err)
	assert.Nil(t, spans)
	assert.Contains(t, err.Error(), "not available")
}

// TestNormalizeLabel 验证 BIO 标签归一化逻辑。
func TestNormalizeLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"B-PER", "PER"},
		{"I-PER", "PER"},
		{"B-ORG", "ORG"},
		{"I-LOC", "LOC"},
		{"PER", "PER"},
		{"O", "O"},
		{"B-DISEASE", "DISEASE"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeLabel(tt.input))
		})
	}
}

// TestPlatformLibPath 验证各平台动态库路径解析。
func TestPlatformLibPath(t *testing.T) {
	path, err := PlatformLibPath("/opt/medmemo")
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "/opt/medmemo/lib/")
}

// TestEntitySpan_Struct 验证 EntitySpan 数据结构。
func TestEntitySpan_Struct(t *testing.T) {
	span := EntitySpan{
		Text:  "张三",
		Label: "PER",
		Start: 0,
		End:   6,
		Score: 0.9876,
	}
	assert.Equal(t, "张三", span.Text)
	assert.Equal(t, "PER", span.Label)
	assert.Equal(t, 0, span.Start)
	assert.Equal(t, 6, span.End)
	assert.InDelta(t, float32(0.9876), span.Score, 0.0001)
}

// TestVerifyModelSHA256_Missing 验证无 .sha256 文件时跳过校验。
func TestVerifyModelSHA256_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	require.NoError(t, os.WriteFile(modelFile, []byte("dummy"), 0644))

	err := verifyModelSHA256(modelFile)
	assert.NoError(t, err, "无 .sha256 文件时应跳过校验")
}

// TestVerifyModelSHA256_Mismatch 验证 SHA-256 不匹配时返回错误。
func TestVerifyModelSHA256_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"

	require.NoError(t, os.WriteFile(modelFile, []byte("dummy model data"), 0644))
	require.NoError(t, os.WriteFile(shaFile, []byte("0000000000000000000000000000000000000000000000000000000000000000"), 0644))

	err := verifyModelSHA256(modelFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

// TestVerifyModelSHA256_Valid 验证正确的 SHA-256 通过校验。
func TestVerifyModelSHA256_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"

	data := []byte("valid model content for sha256 test")
	require.NoError(t, os.WriteFile(modelFile, data, 0644))

	hash := sha256.Sum256(data)
	require.NoError(t, os.WriteFile(shaFile, []byte(hex.EncodeToString(hash[:])), 0644))

	err := verifyModelSHA256(modelFile)
	assert.NoError(t, err)
}

// TestDefaultModelPath 验证默认模型路径拼接。
func TestDefaultModelPath(t *testing.T) {
	path := DefaultModelPath("/opt/medmemo")
	assert.Equal(t, filepath.Join("/opt/medmemo", "models", "distilbert-ner"), path)
}

// TestNewEngine_ModelDirExistsButModelMissing 验证模型目录存在但 model.onnx 缺失时降级。
func TestNewEngine_ModelDirExistsButModelMissing(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建虚拟动态库文件
	libDir := filepath.Join(tmpDir, "lib", "linux")
	require.NoError(t, os.MkdirAll(libDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "libonnxruntime.so"), []byte("dummy"), 0644))

	// 创建模型目录（空）
	modelDir := filepath.Join(tmpDir, "models", "distilbert-ner")
	require.NoError(t, os.MkdirAll(modelDir, 0755))

	engine, err := NewEngine(EngineConfig{ResourceDir: tmpDir, ModelPath: modelDir})
	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "model.onnx 缺失时应降级")
	assert.NoError(t, engine.Close())
}

// TestNewEngine_SHA256Mismatch 验证 SHA-256 校验失败时降级。
func TestNewEngine_SHA256Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建虚拟动态库文件
	libDir := filepath.Join(tmpDir, "lib", "linux")
	require.NoError(t, os.MkdirAll(libDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "libonnxruntime.so"), []byte("dummy"), 0644))

	// 创建模型目录 + model.onnx + 错误的 .sha256
	modelDir := filepath.Join(tmpDir, "models", "distilbert-ner")
	require.NoError(t, os.MkdirAll(modelDir, 0755))
	modelFile := filepath.Join(modelDir, "model.onnx")
	require.NoError(t, os.WriteFile(modelFile, []byte("dummy model"), 0644))
	require.NoError(t, os.WriteFile(modelFile+".sha256", []byte("0000000000000000000000000000000000000000000000000000000000000000"), 0644))

	engine, err := NewEngine(EngineConfig{ResourceDir: tmpDir, ModelPath: modelDir})
	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "SHA-256 不匹配时应降级")
	assert.NoError(t, engine.Close())
}

// TestVerifyModelSHA256_ModelFileMissing 验证 .sha256 存在但 model.onnx 不存在时返回错误。
func TestVerifyModelSHA256_ModelFileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"
	// 创建 .sha256 文件但不创建 model.onnx
	require.NoError(t, os.WriteFile(shaFile, []byte("dummyhash"), 0644))

	err := verifyModelSHA256(modelFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read model file failed")
}

// TestVerifyModelSHA256_HashFileReadError 验证 .sha256 文件存在但不可读时返回错误。
func TestVerifyModelSHA256_HashFileReadError(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"
	require.NoError(t, os.WriteFile(modelFile, []byte("dummy"), 0644))
	// 创建不可读的 .sha256 目录（导致 ReadFile 失败）
	require.NoError(t, os.Mkdir(shaFile, 0755))

	err := verifyModelSHA256(modelFile)
	require.Error(t, err)
}

// TestNewNERWorker 验证 Worker 构造函数。
func TestNewNERWorker(t *testing.T) {
	w := NewNERWorker(1, nil)
	assert.NotNil(t, w)
	assert.Equal(t, 1, w.id)
}

// TestEngine_Close_NilSession 验证无 session 时 Close 安全退出。
func TestEngine_Close_NilSession(t *testing.T) {
	// 使用无模型的引擎，session 为 nil
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	assert.NoError(t, engine.Close())
}
