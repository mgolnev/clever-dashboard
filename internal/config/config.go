package config

import (
	"os"
	"strings"
)

// Config — конфигурация приложения. Значения берутся из env с дефолтами для
// локального запуска "из коробки" (SQLite, порт 8080).
type Config struct {
	Port string
	// DBDriver: "sqlite" (по умолчанию) или "postgres".
	DBDriver string
	// DBDSN: путь к файлу для sqlite или DSN для postgres.
	DBDSN string
}

func Load() Config {
	driver := strings.ToLower(getenv("DB_DRIVER", "sqlite"))
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		if driver == "postgres" {
			dsn = "postgres://localhost:5432/clever_dashboard?sslmode=disable"
		} else {
			dsn = "data/clever.db"
		}
	}
	return Config{
		Port:     getenv("PORT", "8080"),
		DBDriver: driver,
		DBDSN:    dsn,
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
