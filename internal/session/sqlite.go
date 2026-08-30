package session

import (
	"context"
	"database/sql"
	"time"
)

type SQLiteStore struct{ db *sql.DB }

func NewSQLiteStore(db *sql.DB) *SQLiteStore { return &SQLiteStore{db: db} }

func (s *SQLiteStore) Delete(token string) error { return s.DeleteCtx(context.Background(), token) }
func (s *SQLiteStore) Find(token string) ([]byte, bool, error) {
	return s.FindCtx(context.Background(), token)
}
func (s *SQLiteStore) Commit(token string, b []byte, expiry time.Time) error {
	return s.CommitCtx(context.Background(), token, b, expiry)
}

func (s *SQLiteStore) DeleteCtx(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (s *SQLiteStore) FindCtx(ctx context.Context, token string) ([]byte, bool, error) {
	var data []byte
	var expiry string
	err := s.db.QueryRowContext(ctx, `SELECT data, expiry FROM sessions WHERE token = ?`, token).Scan(&data, &expiry)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	t, err := time.Parse(time.RFC3339Nano, expiry)
	if err != nil {
		return nil, false, err
	}
	if time.Now().After(t) {
		_ = s.DeleteCtx(ctx, token)
		return nil, false, nil
	}
	return data, true, nil
}

func (s *SQLiteStore) CommitCtx(ctx context.Context, token string, b []byte, expiry time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (token, data, expiry) VALUES (?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET data = excluded.data, expiry = excluded.expiry`, token, b, expiry.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLiteStore) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expiry < ?`, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
