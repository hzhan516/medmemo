// Package repository 实现数据持久化适配器。
// 将 domain 实体映射到具体数据库（DuckDB / SQLite / Kùzǔ）。
package repository

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// MemoryRepoDuckDB 基于 DuckDB 的记忆仓库实现。
type MemoryRepoDuckDB struct {
	db *DuckDBConnection // TODO(作者): 替换为 infrastructure/database 暴露的具体连接类型
}

// NewMemoryRepoDuckDB 构造函数。
func NewMemoryRepoDuckDB() *MemoryRepoDuckDB {
	return &MemoryRepoDuckDB{}
}

// Save 实现 port.MemoryRepository。
func (r *MemoryRepoDuckDB) Save(ctx context.Context, mem *entity.HealthMemory) error {
	// TODO(作者): 实现 INSERT OR UPDATE，处理冲突检测 [Issue#014]
	return fmt.Errorf("MemoryRepoDuckDB.Save not implemented")
}

// GetByID 实现 port.MemoryRepository。
func (r *MemoryRepoDuckDB) GetByID(ctx context.Context, id models.MemoryID) (*entity.HealthMemory, error) {
	return nil, fmt.Errorf("MemoryRepoDuckDB.GetByID not implemented")
}

// Search 实现 port.MemoryRepository。
func (r *MemoryRepoDuckDB) Search(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error) {
	return nil, fmt.Errorf("MemoryRepoDuckDB.Search not implemented")
}

// SemanticSearch 实现 port.MemoryRepository。
func (r *MemoryRepoDuckDB) SemanticSearch(ctx context.Context, embedding []float32, topK int) ([]*entity.HealthMemory, error) {
	// TODO(作者): 接入 DuckDB vss 扩展 HNSW 向量检索 [Issue#015]
	return nil, fmt.Errorf("MemoryRepoDuckDB.SemanticSearch not implemented")
}

// ListByTier 实现 port.MemoryRepository。
func (r *MemoryRepoDuckDB) ListByTier(ctx context.Context, tier entity.MemoryTier, limit int) ([]*entity.HealthMemory, error) {
	return nil, fmt.Errorf("MemoryRepoDuckDB.ListByTier not implemented")
}

// Delete 实现 port.MemoryRepository。
func (r *MemoryRepoDuckDB) Delete(ctx context.Context, id models.MemoryID) error {
	return fmt.Errorf("MemoryRepoDuckDB.Delete not implemented")
}

// DuckDBConnection 占位类型，待 infrastructure/database 完成后替换。
type DuckDBConnection struct{}

// RepositorySet 供 Wire 使用的 ProviderSet。
var RepositorySet = wire.NewSet(
	NewMemoryRepoDuckDB,
)
