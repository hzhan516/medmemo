// Package database 封装 SQLite / SQLCipher 连接池、迁移与事务管理。
package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hzhan516/medmemo/internal/infrastructure/secret"
	sqlite3 "github.com/mutecomm/go-sqlcipher"
	"github.com/viant/sqlite-vec/engine"
)

func init() {
	// 必须在 sql.Open 之前注册，确保所有 SQLite 连接都能使用 vec_cosine / vec_l2
	_ = engine.RegisterVectorFunctions(nil)
}

const dbKeyName = "db_key"

// SQLCipherConnector SQLCipher 加密数据库连接管理器。
// AES-256-GCM page-level 透明加密，密钥通过 secret.Store 管理。
type SQLCipherConnector struct {
	db   *sql.DB
	path string
}

// resolveDataDir 解析数据目录路径。
// 空值时优先从 MEDMEMO_DATA_DIR 环境变量读取，其次回退到 ~/.medmemo/data。
// 相对路径会自动解析为绝对路径（以用户主目录为基准）。
func resolveDataDir(dataDir string) string {
	if dataDir == "" {
		if envDir := os.Getenv("MEDMEMO_DATA_DIR"); envDir != "" {
			dataDir = envDir
		} else {
			dataDir = ".medmemo/data"
		}
	}
	if !filepath.IsAbs(dataDir) {
		if home, err := os.UserHomeDir(); err == nil {
			dataDir = filepath.Join(home, dataDir)
		}
	}
	return dataDir
}

// NewSQLCipherConnector 创建 SQLCipher 加密数据库连接。
// 流程：获取主密钥 → 检测明文迁移 → 打开加密库 → 验证密钥 → 配置连接池。
func NewSQLCipherConnector(dataDir string, store secret.Store) (*SQLCipherConnector, error) {
	dataDir = resolveDataDir(dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "medmemo.db")

	// 获取或生成 32 字节主密钥
	key, err := getOrCreateKey(store)
	if err != nil {
		return nil, fmt.Errorf("failed to get database key: %w", err)
	}

	// 检测是否需要明文迁移
	needsMigrate, err := isPlaintextDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check database encryption status: %w", err)
	}
	if needsMigrate {
		if err := migrateFromPlaintext(dbPath, key); err != nil {
			return nil, fmt.Errorf("failed to migrate plaintext database: %w", err)
		}
	}

	// 打开加密数据库（DSN 中通过 _pragma_key 传递密钥）
	dsn := fmt.Sprintf("%s?_pragma_key=x'%x'", dbPath, key)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlcipher database: %w", err)
	}

	// 验证密钥有效性
	if _, err := db.Exec("SELECT count(*) FROM sqlite_master"); err != nil {
		_ = db.Close() // 密钥验证失败后的清理关闭，关闭错误非关键（上方已返回主错误）
		return nil, fmt.Errorf("database key verification failed: %w", err)
	}

	// 连接池配置：桌面端并发场景（流式对话 + 异步事实提取 + UI 查询）需要更大池子
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	// 设置 busy_timeout：锁冲突时自动重试最多 5 秒，避免立即返回 SQLITE_BUSY
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close() // 外键启用失败后的清理关闭，关闭错误非关键（上方已返回主错误）
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// DELETE journal 模式，避免 WAL 空主文件导致的密钥验证问题
	if _, err := db.Exec("PRAGMA journal_mode = DELETE"); err != nil {
		_ = db.Close() // journal_mode 设置失败后的清理关闭，关闭错误非关键（上方已返回主错误）
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}

	// 加固数据库文件权限，仅允许当前用户读写
	if err := os.Chmod(dbPath, 0600); err != nil {
		fmt.Printf("[SQLCipher] failed to chmod %s to 0600: %v\n", dbPath, err)
	}

	return &SQLCipherConnector{db: db, path: dbPath}, nil
}

// DB 返回底层 *sql.DB，供 repository 层使用。
func (c *SQLCipherConnector) DB() *sql.DB {
	return c.db
}

// Close 关闭连接。
func (c *SQLCipherConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Migrate 执行版本化数据库迁移，复用 SQLiteConnector 的 schema 逻辑。
func (c *SQLCipherConnector) Migrate(ctx context.Context) error {
	// SQLCipher 的 schema 和 SQLite 完全一致，直接复用现有迁移
	return migrateSQLiteSchema(ctx, c.db)
}

// getOrCreateKey 从 secret.Store 获取数据库密钥，不存在则生成并存储。
// 密钥长度非 32 字节视为损坏，直接返回错误。
func getOrCreateKey(store secret.Store) ([]byte, error) {
	key, err := store.Get(dbKeyName)
	if err == nil {
		if len(key) == 32 {
			return key, nil
		}
		return nil, fmt.Errorf("database key in store has invalid length %d, expected 32", len(key))
	}

	// 生成 256 位随机密钥
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate database key: %w", err)
	}

	if err := store.Set(dbKeyName, key); err != nil {
		return nil, fmt.Errorf("failed to store database key: %w", err)
	}

	return key, nil
}

// isPlaintextDB 检测数据库是否为明文 SQLite。
func isPlaintextDB(dbPath string) (bool, error) {
	// 文件不存在 = 新建数据库，无需迁移
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false, nil
	}

	// 获取主文件信息
	info, err := os.Stat(dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat database file: %w", err)
	}

	// 主文件为空但存在 WAL：活跃的 WAL 模式数据库，不是明文
	if info.Size() == 0 {
		if _, err := os.Stat(dbPath + "-wal"); err == nil {
			return false, nil
		}
		// 空文件且无 WAL，保守视为无需迁移
		return false, nil
	}

	encrypted, err := sqlite3.IsEncrypted(dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to check encryption status: %w", err)
	}

	// 未加密 = 明文，需要迁移
	return !encrypted, nil
}

// validateAttachPath 验证 ATTACH DATABASE 路径安全性：
// 1. 必须是绝对路径
// 2. 不得包含 .. 目录穿越
// 3. 必须位于数据目录下（防止 attach 到系统敏感文件）
// 4. 单引号已在外层转义，此处额外拒绝含单引号的路径作为纵深防御
// Audit: RR-001 SQLCipher attach path validation
func validateAttachPath(path, dataDir string) error {
	if path == "" {
		return fmt.Errorf("attach path is empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("attach path must be absolute: %s", path)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("attach path contains directory traversal: %s", path)
	}
	if strings.Contains(path, "'") {
		return fmt.Errorf("attach path contains single quote: %s", path)
	}
	dataDir = filepath.Clean(dataDir)
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, dataDir+string(filepath.Separator)) && cleanPath != dataDir {
		return fmt.Errorf("attach path must be under data directory: %s", path)
	}
	return nil
}

// migrateFromPlaintext 将明文 SQLite 迁移为 SQLCipher 加密数据库。
// 使用 sqlcipher_export() 保证 schema、数据、索引完整复制。
// 原始文件保留为 .backup。
// Audit: RR-001 路径白名单验证 + 单引号转义防止 SQL 注入
func migrateFromPlaintext(dbPath string, key []byte) error {
	// 用 SQLCipher 打开明文数据库（不设置密钥即可打开明文 db）
	plainDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open plaintext database: %w", err)
	}

	// 验证是有效的 SQLite 数据库
	var count int
	if err := plainDB.QueryRow("SELECT count(*) FROM sqlite_master").Scan(&count); err != nil {
		_ = plainDB.Close() // 明文数据库验证失败后的清理关闭，关闭错误非关键（上方已返回主错误）
		return fmt.Errorf("plaintext database verification failed: %w", err)
	}

	// ATTACH 新的加密数据库
	newPath := dbPath + ".new"
	_ = os.Remove(newPath) // 清理可能残留的临时文件

	// 验证 attach 路径安全性（白名单 + 目录穿越防护）
	dataDir := filepath.Dir(dbPath)
	if err := validateAttachPath(newPath, dataDir); err != nil {
		_ = plainDB.Close()
		return fmt.Errorf("attach path validation failed: %w", err)
	}

	// 对路径中的单引号做 SQL 转义，防止注入（纵深防御，validateAttachPath 已拒绝含单引号路径）
	escapedPath := strings.ReplaceAll(newPath, "'", "''")
	attachSQL := fmt.Sprintf("ATTACH DATABASE '%s' AS encrypted KEY \"x'%x'\"", escapedPath, key)
	if _, err := plainDB.Exec(attachSQL); err != nil {
		_ = plainDB.Close()    // attach 失败后的清理关闭，关闭错误非关键（上方已返回主错误）
		_ = os.Remove(newPath) // 清理临时加密文件，Remove 错误不影响主错误返回
		return fmt.Errorf("failed to attach encrypted database: %w", err)
	}

	// 执行 SQLCipher 原生导出
	if _, err := plainDB.Exec("SELECT sqlcipher_export('encrypted')"); err != nil {
		_ = plainDB.Close()    // 导出失败后的清理关闭，关闭错误非关键（上方已返回主错误）
		_ = os.Remove(newPath) // 清理临时加密文件，Remove 错误不影响主错误返回
		return fmt.Errorf("sqlcipher_export failed: %w", err)
	}

	// DETACH
	if _, err := plainDB.Exec("DETACH DATABASE encrypted"); err != nil {
		_ = plainDB.Close()    // detach 失败后的清理关闭，关闭错误非关键（上方已返回主错误）
		_ = os.Remove(newPath) // 清理临时加密文件，Remove 错误不影响主错误返回
		return fmt.Errorf("failed to detach encrypted database: %w", err)
	}

	// 关闭明文连接
	if err := plainDB.Close(); err != nil {
		_ = os.Remove(newPath) // 明文关闭失败时清理临时加密文件，Remove 错误不影响主错误返回
		return fmt.Errorf("failed to close plaintext database: %w", err)
	}

	// 验证加密文件是否生成
	encrypted, err := sqlite3.IsEncrypted(newPath)
	if err != nil || !encrypted {
		_ = os.Remove(newPath) // 加密验证失败时清理临时文件，Remove 错误不影响主错误返回
		return fmt.Errorf("encrypted database verification failed")
	}

	// 原子替换：明文 → backup，加密 → 主路径
	backupPath := dbPath + ".backup"
	if err := os.Rename(dbPath, backupPath); err != nil {
		_ = os.Remove(newPath) // 备份失败时清理临时加密文件，Remove 错误不影响主错误返回
		return fmt.Errorf("failed to backup plaintext database: %w", err)
	}
	if err := os.Rename(newPath, dbPath); err != nil {
		// 尝试恢复
		_ = os.Rename(backupPath, dbPath) // 恢复回退，Rename 失败无额外补救措施（已返回主错误）
		_ = os.Remove(newPath)            // 清理残留临时文件，Remove 错误不影响主错误返回
		return fmt.Errorf("failed to replace with encrypted database: %w", err)
	}

	return nil
}

// migrateSQLiteSchema 执行 SQLite/SQLCipher 共用的 schema 迁移。
func migrateSQLiteSchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
			CREATE TABLE IF NOT EXISTS conversations (
				id TEXT PRIMARY KEY,
				title TEXT NOT NULL DEFAULT '',
				model TEXT NOT NULL DEFAULT '',
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				archived_at INTEGER,
				deleted_at INTEGER
			);
			CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at);
			CREATE INDEX IF NOT EXISTS idx_conversations_deleted ON conversations(deleted_at);
			`,
		},
		{
			version: 2,
			sql: `
			CREATE TABLE IF NOT EXISTS messages (
				id TEXT PRIMARY KEY,
				conversation_id TEXT NOT NULL,
				role TEXT NOT NULL,
				content TEXT NOT NULL,
				tokens INTEGER DEFAULT 0,
				created_at INTEGER NOT NULL,
				deleted_at INTEGER,
				FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id, created_at);
			CREATE INDEX IF NOT EXISTS idx_messages_deleted ON messages(deleted_at);
			`,
		},
		{
			version: 3,
			sql: `
			CREATE TABLE IF NOT EXISTS memories (
				id TEXT PRIMARY KEY,
				tier INTEGER NOT NULL,
				content TEXT NOT NULL,
				tags TEXT,
				source_conv TEXT,
				confidence REAL DEFAULT 1.0,
				created_at INTEGER NOT NULL,
				accessed_at INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_memories_tier ON memories(tier);
			CREATE INDEX IF NOT EXISTS idx_memories_accessed ON memories(accessed_at);
			`,
		},
		{
			version: 4,
			sql: `
			CREATE TABLE IF NOT EXISTS disclaimer_acceptance (
				version TEXT PRIMARY KEY,
				accepted_at INTEGER NOT NULL,
				text_hash TEXT
			);
			`,
		},
		{
			version: 5,
			sql: `
			CREATE TABLE IF NOT EXISTS providers (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				api_host TEXT NOT NULL,
				api_key BLOB NOT NULL,
				model_id TEXT NOT NULL,
				temperature REAL DEFAULT 0.7,
				timeout_ms INTEGER DEFAULT 30000,
				max_retries INTEGER DEFAULT 3,
				group_name TEXT DEFAULT '',
				enabled INTEGER DEFAULT 1,
				sort_order INTEGER DEFAULT 0,
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_providers_enabled ON providers(enabled);
			CREATE INDEX IF NOT EXISTS idx_providers_group ON providers(group_name, sort_order);
			`,
		},
		{
			version: 6,
			sql: `
			ALTER TABLE providers ADD COLUMN auth_method TEXT DEFAULT 'api_key';
			ALTER TABLE providers ADD COLUMN auth_params TEXT DEFAULT '{}';
			`,
		},
		{
			version: 7,
			sql: `
			CREATE TABLE IF NOT EXISTS raw_dialogues (
				message_id TEXT PRIMARY KEY,
				session_id TEXT NOT NULL,
				role TEXT NOT NULL CHECK(role IN ('user','assistant','system')),
				content TEXT NOT NULL,
				model_name TEXT,
				timestamp INTEGER NOT NULL,
				extraction_status TEXT DEFAULT 'unprocessed' CHECK(extraction_status IN ('unprocessed','processing','processed','failed')),
				created_at INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_raw_session_time ON raw_dialogues(session_id, timestamp);
			CREATE INDEX IF NOT EXISTS idx_raw_extraction_status ON raw_dialogues(extraction_status);

			CREATE TABLE IF NOT EXISTS extracted_facts (
				fact_id TEXT PRIMARY KEY,
				subject TEXT NOT NULL,
				predicate TEXT NOT NULL,
				object TEXT NOT NULL,
				confidence REAL NOT NULL CHECK(confidence >= 0 AND confidence <= 1),
				source_msg_ids TEXT NOT NULL,
				status TEXT DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected')),
				scored_at INTEGER,
				reviewed_at INTEGER,
				created_at INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_fact_confidence ON extracted_facts(confidence);
			CREATE INDEX IF NOT EXISTS idx_fact_status ON extracted_facts(status);

			CREATE TABLE IF NOT EXISTS semantic_embeddings (
				embedding_id TEXT PRIMARY KEY,
				fact_id TEXT NOT NULL UNIQUE,
				vector BLOB NOT NULL,
				model_version TEXT NOT NULL DEFAULT 'all-MiniLM-L6-v2',
				created_at INTEGER NOT NULL,
				FOREIGN KEY (fact_id) REFERENCES extracted_facts(fact_id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_embedding_fact ON semantic_embeddings(fact_id);
			`,
		},
		{
			version: 8,
			sql: `
			-- v1.1 DoD A1: 为 extracted_facts 添加敏感信息标记列
			ALTER TABLE extracted_facts ADD COLUMN is_sensitive INTEGER DEFAULT 0;
			`,
		},
		{
			version: 9,
			sql: `
			-- v1.1 DoD A3: 审计日志表
			CREATE TABLE IF NOT EXISTS audit_logs (
				id TEXT PRIMARY KEY,
				action TEXT NOT NULL CHECK(action IN ('CREATE','APPROVE','REJECT','DELETE')),
				target_type TEXT NOT NULL DEFAULT 'fact',
				target_id TEXT NOT NULL,
				old_value TEXT,
				new_value TEXT,
				actor TEXT NOT NULL DEFAULT 'user',
				timestamp INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_audit_target ON audit_logs(target_type, target_id);
			CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
			`,
		},
		{
			version: 10,
			sql: `
			-- v1.1 回答置信度机制: 扩展 messages 表存储 token 拆分与置信度
			ALTER TABLE messages ADD COLUMN prompt_tokens INTEGER DEFAULT 0;
			ALTER TABLE messages ADD COLUMN completion_tokens INTEGER DEFAULT 0;
			ALTER TABLE messages ADD COLUMN confidence_score REAL;
			ALTER TABLE messages ADD COLUMN confidence_level TEXT;
			ALTER TABLE messages ADD COLUMN confidence_json TEXT;
			`,
		},
		{
			version: 11,
			sql: `
			-- v1.1.4: 为 embedding 版本迁移优化查询性能
			CREATE INDEX IF NOT EXISTS idx_embedding_model_version 
			    ON semantic_embeddings(model_version);
			`,
		},
		{
			version: 12,
			sql: `
			-- v1.1.9: 回答准确率反馈持久化
			CREATE TABLE IF NOT EXISTS answer_feedback (
				message_id TEXT NOT NULL,
				answer_type TEXT NOT NULL,
				feedback TEXT NOT NULL CHECK(feedback IN ('helpful','inaccurate')),
				created_at INTEGER NOT NULL,
				PRIMARY KEY (message_id, answer_type)
			);

			CREATE TABLE IF NOT EXISTS answer_accuracy_stats (
				answer_type TEXT PRIMARY KEY,
				correct_count INTEGER NOT NULL DEFAULT 0,
				total_count INTEGER NOT NULL DEFAULT 0,
				updated_at INTEGER NOT NULL
			);
			`,
		},
		{
			version: 13,
			sql: `
			-- v1.1.9: 知识库 RAG 表结构
			CREATE TABLE IF NOT EXISTS knowledge_documents (
				document_id TEXT PRIMARY KEY,
				title TEXT NOT NULL DEFAULT '',
				source_type TEXT NOT NULL,
				citation TEXT NOT NULL DEFAULT '',
				url TEXT NOT NULL DEFAULT '',
				language TEXT NOT NULL DEFAULT '',
				checksum TEXT NOT NULL UNIQUE,
				metadata_json TEXT NOT NULL DEFAULT '{}',
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL
			);

			CREATE TABLE IF NOT EXISTS knowledge_chunks (
				chunk_id TEXT PRIMARY KEY,
				document_id TEXT NOT NULL,
				chunk_index INTEGER NOT NULL,
				content TEXT NOT NULL,
				token_count INTEGER NOT NULL DEFAULT 0,
				metadata_json TEXT NOT NULL DEFAULT '{}',
				created_at INTEGER NOT NULL,
				FOREIGN KEY (document_id) REFERENCES knowledge_documents(document_id) ON DELETE CASCADE,
				UNIQUE(document_id, chunk_index)
			);
			CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_document_id ON knowledge_chunks(document_id);

			CREATE TABLE IF NOT EXISTS knowledge_terms (
				term TEXT NOT NULL,
				chunk_id TEXT NOT NULL,
				tf INTEGER NOT NULL DEFAULT 0,
				document_id TEXT NOT NULL,
				PRIMARY KEY (term, chunk_id),
				FOREIGN KEY (chunk_id) REFERENCES knowledge_chunks(chunk_id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_knowledge_terms_term ON knowledge_terms(term);
			CREATE INDEX IF NOT EXISTS idx_knowledge_terms_chunk_id ON knowledge_terms(chunk_id);
			CREATE INDEX IF NOT EXISTS idx_knowledge_terms_document_id ON knowledge_terms(document_id);

			CREATE TABLE IF NOT EXISTS knowledge_embeddings (
				chunk_id TEXT PRIMARY KEY,
				model_version TEXT NOT NULL,
				dimension INTEGER NOT NULL,
				embedding BLOB NOT NULL,
				created_at INTEGER NOT NULL,
				FOREIGN KEY (chunk_id) REFERENCES knowledge_chunks(chunk_id) ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS knowledge_import_jobs (
				job_id TEXT PRIMARY KEY,
				status TEXT NOT NULL,
				total INTEGER NOT NULL DEFAULT 0,
				processed INTEGER NOT NULL DEFAULT 0,
				error_message TEXT NOT NULL DEFAULT '',
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL
			);
			`,
		},
		{
			version: 14,
			sql: `
			-- v1.1.10 M07: 持久化 provider 类型，避免每次运行时靠 api_host 推断本地/云端。
			-- 旧行 provider_type 为空，读取时回退 InferProviderType(api_host) 保持向后兼容。
			ALTER TABLE providers ADD COLUMN provider_type TEXT DEFAULT '';
			`,
		},
	}

	for _, m := range migrations {
		if version >= m.version {
			continue
		}
		if _, err := db.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("failed to apply migration v%d: %w", m.version, err)
		}
		// PRAGMA 不支持参数化，使用 strconv 避免 fmt.Sprintf 拼接
		pragmaSQL := "PRAGMA user_version = " + strconv.Itoa(m.version)
		if _, err := db.ExecContext(ctx, pragmaSQL); err != nil {
			return fmt.Errorf("failed to update schema version to v%d: %w", m.version, err)
		}
		version = m.version
	}

	return nil
}
