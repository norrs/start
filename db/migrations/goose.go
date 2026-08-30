package migrations

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func Up(db *sql.DB) error {
	goose.SetBaseFS(FS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set Goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("apply Goose migrations: %w", err)
	}
	return nil
}
