package dedupe

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultSQLiteTable = "seen_posts"
)

type SQLiteStore struct {
	db         *sql.DB
	table      string
	tableIdent string
	ttl        time.Duration
}

func NewSQLiteStore(dsn string, table string, ttl time.Duration) (*SQLiteStore, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, fmt.Errorf("sqlite dsn is required")
	}
	if ttl < 0 {
		return nil, fmt.Errorf("sqlite ttl must be >= 0")
	}
	if table == "" {
		table = defaultSQLiteTable
	}
	tableIdent, err := quoteSQLiteIdentifier(table)
	if err != nil {
		return nil, err
	}
	if err := ensureSQLiteDir(dsn); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &SQLiteStore{
		db:         db,
		table:      table,
		tableIdent: tableIdent,
		ttl:        ttl,
	}
	if err := store.ensureSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) HasSeen(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	var seenAt time.Time
	query := fmt.Sprintf("SELECT seen_at FROM %s WHERE id = ?", s.tableIdent)
	err := s.db.QueryRowContext(ctx, query, id).Scan(&seenAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if s.ttl <= 0 {
		return true, nil
	}
	cutoff := time.Now().UTC().Add(-s.ttl)
	if seenAt.Before(cutoff) {
		if err := s.deleteID(ctx, id); err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (s *SQLiteStore) MarkSeen(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	_, err := s.db.ExecContext(
		ctx,
		fmt.Sprintf("INSERT INTO %s (id, seen_at) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET seen_at = excluded.seen_at", s.tableIdent),
		id,
		time.Now().UTC(),
	)
	return err
}

func (s *SQLiteStore) MarkSeenBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(
		ctx,
		fmt.Sprintf("INSERT INTO %s (id, seen_at) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET seen_at = excluded.seen_at", s.tableIdent),
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	now := time.Now().UTC()
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, id, now); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) ensureSchema(ctx context.Context) error {
	if s.table == "" {
		return fmt.Errorf("sqlite table name is required")
	}
	ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id TEXT PRIMARY KEY,
		seen_at TIMESTAMP NOT NULL
	)`, s.tableIdent)
	if _, err := s.db.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("create sqlite table: %w", err)
	}
	index := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_seen_at_idx ON %s (seen_at)", s.table, s.tableIdent)
	if _, err := s.db.ExecContext(ctx, index); err != nil {
		return fmt.Errorf("create sqlite index: %w", err)
	}
	return nil
}

func (s *SQLiteStore) deleteID(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.tableIdent), id)
	return err
}

func ensureSQLiteDir(dsn string) error {
	if strings.HasPrefix(dsn, "file:") {
		dsn = strings.TrimPrefix(dsn, "file:")
		if idx := strings.IndexRune(dsn, '?'); idx >= 0 {
			dsn = dsn[:idx]
		}
	}
	if dsn == "" || dsn == ":memory:" {
		return nil
	}
	dir := filepath.Dir(dsn)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

var sqliteIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteSQLiteIdentifier(identifier string) (string, error) {
	if identifier == "" {
		return "", fmt.Errorf("sqlite table name is required")
	}
	if !sqliteIdentifierPattern.MatchString(identifier) {
		return "", fmt.Errorf("sqlite table name %q must match %s", identifier, sqliteIdentifierPattern.String())
	}
	return `"` + identifier + `"`, nil
}
