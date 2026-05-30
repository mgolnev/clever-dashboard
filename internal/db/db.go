package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/clever/clever-dashboard/internal/config"

	// Драйверы БД.
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// DB оборачивает *sql.DB и знает свой диалект для ветвления SQL.
type DB struct {
	*sql.DB
	driver string
}

func (d *DB) IsPostgres() bool { return d.driver == "postgres" }

// Placeholder возвращает плейсхолдер параметра для позиции n (1-based):
// "?" для SQLite, "$n" для Postgres.
func (d *DB) Placeholder(n int) string {
	if d.IsPostgres() {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// Rebind переводит запрос с плейсхолдерами "?" в "$1, $2, ..." для Postgres.
// Для SQLite возвращает запрос как есть.
func (d *DB) Rebind(query string) string {
	if !d.IsPostgres() {
		return query
	}
	var b []byte
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b = append(b, '$')
			b = append(b, []byte(fmt.Sprintf("%d", n))...)
			continue
		}
		b = append(b, query[i])
	}
	return string(b)
}

func Open(cfg config.Config) (*DB, error) {
	driverName := "sqlite"
	if cfg.DBDriver == "postgres" {
		driverName = "pgx"
	}

	if cfg.DBDriver == "sqlite" {
		if dir := filepath.Dir(cfg.DBDSN); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("mkdir db dir: %w", err)
			}
		}
	}

	sqlDB, err := sql.Open(driverName, cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &DB{DB: sqlDB, driver: cfg.DBDriver}, nil
}
