package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config — конфигурация приложения. Значения берутся из env с дефолтами для
// локального запуска "из коробки" (SQLite, порт 8080).
type Config struct {
	Port string
	// DBDriver: "sqlite" (по умолчанию) или "postgres".
	DBDriver string
	// DBDSN: путь к файлу для sqlite или DSN для postgres.
	DBDSN string
	// LogisticsPilotCities — города пилота бесплатной доставки (через запятую в env).
	LogisticsPilotCities []string
	// LogisticsPilotStart — дата старта пилота YYYY-MM-DD (опционально, для UI).
	LogisticsPilotStart string
	// StaticDir — каталог собранного фронтенда (Vite dist). Если задан и
	// существует, backend отдаёт SPA с этого пути (single-binary деплой). В dev
	// пусто — фронт обслуживает Vite на :3000 с прокси на API.
	StaticDir string
	// Analytics — автоматическая загрузка обезличенного трафика из Яндекс
	// Метрики и AppMetrica. Токены используются только backend-процессом.
	AnalyticsSyncEnabled  bool
	AnalyticsSyncInterval time.Duration
	AnalyticsLookbackDays int
	AnalyticsBackfillDays int
	AnalyticsTimezone     string
	MetrikaCounterID      string
	MetrikaOAuthToken     string
	AppMetricaAppID       string
	AppMetricaOAuthToken  string
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
		Port:                  getenv("PORT", "8080"),
		DBDriver:              driver,
		DBDSN:                 dsn,
		LogisticsPilotCities:  splitEnvList(os.Getenv("LOGISTICS_PILOT_CITIES")),
		LogisticsPilotStart:   strings.TrimSpace(os.Getenv("LOGISTICS_PILOT_START")),
		StaticDir:             strings.TrimSpace(os.Getenv("STATIC_DIR")),
		AnalyticsSyncEnabled:  getenvBool("ANALYTICS_SYNC_ENABLED", false),
		AnalyticsSyncInterval: getenvDuration("ANALYTICS_SYNC_INTERVAL", 6*time.Hour),
		AnalyticsLookbackDays: getenvInt("ANALYTICS_SYNC_LOOKBACK_DAYS", 7),
		AnalyticsBackfillDays: getenvInt("ANALYTICS_BACKFILL_DAYS", 365),
		AnalyticsTimezone:     getenv("ANALYTICS_TIMEZONE", "Europe/Moscow"),
		MetrikaCounterID:      strings.TrimSpace(os.Getenv("METRIKA_COUNTER_ID")),
		MetrikaOAuthToken:     strings.TrimSpace(os.Getenv("METRIKA_OAUTH_TOKEN")),
		AppMetricaAppID:       strings.TrimSpace(os.Getenv("APPMETRICA_APPLICATION_ID")),
		AppMetricaOAuthToken:  strings.TrimSpace(os.Getenv("APPMETRICA_OAUTH_TOKEN")),
	}
}

func splitEnvList(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getenvInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func getenvDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return def
	}
	return d
}
