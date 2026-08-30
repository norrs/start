package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/norrs/start/db/migrations"
	"github.com/norrs/start/internal/catalog"
	"github.com/norrs/start/internal/seed"
	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		log.Printf("import links: %v", err)
		os.Exit(1)
	}
}

func run() error {
	databasePath := flag.String("database", ".start-dev/start.db", "path to the Start SQLite database")
	seedPath := flag.String("file", "links.yaml", "path to the links YAML document")
	flag.Parse()
	*databasePath = workingPath(*databasePath)
	*seedPath = workingPath(*seedPath)

	document, err := seed.Load(*seedPath)
	if err != nil {
		return err
	}
	if directory := filepath.Dir(*databasePath); directory != "." {
		if err := os.MkdirAll(directory, 0750); err != nil {
			return fmt.Errorf("create database directory: %w", err)
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", *databasePath)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("open SQLite: %w", err)
	}
	defer database.Close()
	if err := migrations.Up(database); err != nil {
		return err
	}

	requestID := fmt.Sprintf("yaml-import-%d", time.Now().UTC().UnixNano())
	result, err := catalog.New(database).ImportSeed(context.Background(), document, requestID)
	if err != nil {
		return err
	}
	fmt.Printf("Imported %s: %d categories, %d owner groups, %d links created, %d updated, %d restored, %d unchanged\n", *seedPath, result.Categories, result.Groups, result.Created, result.Updated, result.Restored, result.Unchanged)
	return nil
}

func workingPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if directory := os.Getenv("BUILD_WORKING_DIRECTORY"); directory != "" {
		return filepath.Join(directory, path)
	}
	return path
}
