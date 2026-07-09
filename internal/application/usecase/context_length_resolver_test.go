package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errProviderStore 的 Get 始终返回错误，用于覆盖解析失败回退默认值的分支。
type errProviderStore struct {
	fakeProviderStore
}

func (e *errProviderStore) Get(_ context.Context, _ string) (*models.ProviderConfig, error) {
	return nil, fmt.Errorf("provider lookup failed")
}

// TestContextLengthResolver_Resolve 覆盖各解析分支。
func TestContextLengthResolver_Resolve(t *testing.T) {
	ctx := context.Background()

	t.Run("empty ids return default", func(t *testing.T) {
		r := NewContextLengthResolver(&fakeProviderStore{})
		assert.Equal(t, models.DefaultMaxContextLen, r.Resolve(ctx, "", "m"))
		assert.Equal(t, models.DefaultMaxContextLen, r.Resolve(ctx, "p", ""))
	})

	t.Run("provider lookup error returns default", func(t *testing.T) {
		r := NewContextLengthResolver(&errProviderStore{})
		assert.Equal(t, models.DefaultMaxContextLen, r.Resolve(ctx, "p", "m"))
	})

	t.Run("nil provider returns default", func(t *testing.T) {
		r := NewContextLengthResolver(&fakeProviderStore{provider: nil})
		assert.Equal(t, models.DefaultMaxContextLen, r.Resolve(ctx, "p", "m"))
	})

	t.Run("configured model returns its max context length", func(t *testing.T) {
		r := NewContextLengthResolver(&fakeProviderStore{provider: &models.ProviderConfig{
			ID: "p",
			Models: []models.ProviderModel{
				{ID: "target", MaxContextLength: 32768},
			},
		}})
		assert.Equal(t, 32768, r.Resolve(ctx, "p", "target"))
	})

	t.Run("model below minimum falls back to default", func(t *testing.T) {
		r := NewContextLengthResolver(&fakeProviderStore{provider: &models.ProviderConfig{
			ID: "p",
			Models: []models.ProviderModel{
				{ID: "target", MaxContextLength: models.MinContextLength - 1},
			},
		}})
		assert.Equal(t, models.DefaultMaxContextLen, r.Resolve(ctx, "p", "target"))
	})

	t.Run("unknown model falls back to default", func(t *testing.T) {
		r := NewContextLengthResolver(&fakeProviderStore{provider: &models.ProviderConfig{
			ID: "p",
			Models: []models.ProviderModel{
				{ID: "other", MaxContextLength: 32768},
			},
		}})
		assert.Equal(t, models.DefaultMaxContextLen, r.Resolve(ctx, "p", "missing"))
	})
}

// TestContextLengthResolver_Validate 覆盖上下文长度的合法与非法边界。
func TestContextLengthResolver_Validate(t *testing.T) {
	r := NewContextLengthResolver(&fakeProviderStore{})

	require.NoError(t, r.Validate(models.MinContextLength))
	require.NoError(t, r.Validate(models.DefaultMaxContextLen))
	require.NoError(t, r.Validate(models.MaxContextLengthCap))

	require.Error(t, r.Validate(models.MinContextLength-1))
	require.Error(t, r.Validate(models.MaxContextLengthCap+1))
}
