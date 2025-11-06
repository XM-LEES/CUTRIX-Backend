package db

import (
    "database/sql"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    _ "github.com/jackc/pgx/v5/stdlib"
)

// Open initializes a PostgreSQL connection using pgx driver.
func Open(dsn string) (*sql.DB, error) {
    if strings.TrimSpace(dsn) == "" {
        return nil, errors.New("empty DATABASE_URL")
    }
    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return nil, err
    }
    if err := db.Ping(); err != nil {
        return nil, err
    }
    return db, nil
}

// RunMigrations executes the initial schema SQL script.
// It is idempotent due to IF NOT EXISTS and OR REPLACE usage in the script.
func RunMigrations(db *sql.DB, scriptRelPath string) error {
    // Resolve path relative to project root or current working directory.
    // Attempt both current working dir and one level above if needed.
    tryPaths := []string{
        scriptRelPath,
        filepath.Join("migrations", scriptRelPath),
        filepath.Join("..", "migrations", scriptRelPath),
        filepath.Join("migrations", "000001_initial_schema.up.sql"),
    }
    var content []byte
    var lastErr error
    for _, p := range tryPaths {
        b, err := os.ReadFile(p)
        if err == nil {
            content = b
            lastErr = nil
            break
        }
        lastErr = err
    }
    if content == nil {
        return fmt.Errorf("migration script not found: %v", lastErr)
    }
    sqlText := string(content)
    // Execute as a single batch; pgx supports multi-statements.
    if _, err := db.Exec(sqlText); err != nil {
        return fmt.Errorf("executing migrations failed: %w", err)
    }
    return nil
}