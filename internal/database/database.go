// Package database opens and configures the shared connection pool.
package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Open() error {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	// Reasonable defaults for an MVP.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return err
	}

	DB = db
	return nil
}
