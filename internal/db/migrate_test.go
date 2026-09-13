package db

import (
	"testing"

	"github.com/clever/clever-dashboard/internal/config"
)

func TestMigrateAddsCumulativeImportCountersToExistingDatabase(t *testing.T) {
	database, err := Open(config.Config{
		DBDriver: "sqlite",
		DBDSN:    t.TempDir() + "/legacy.db",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	_, err = database.Exec(`CREATE TABLE raw_import (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL,
		source TEXT NOT NULL DEFAULT 'bitrix_file',
		rows_total INTEGER NOT NULL DEFAULT 0,
		orders_imported INTEGER NOT NULL DEFAULT 0,
		items_imported INTEGER NOT NULL DEFAULT 0,
		period_start TEXT,
		period_end TEXT,
		imported_at TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}

	if err := database.Migrate(); err != nil {
		t.Fatal(err)
	}
	// Повторный запуск проверяет идемпотентность ALTER на уже обновлённой БД.
	if err := database.Migrate(); err != nil {
		t.Fatal(err)
	}

	_, err = database.Exec(`INSERT INTO raw_import (
		filename, orders_added, orders_updated, orders_skipped, imported_at
	) VALUES ('history.csv', 10, 3, 2, '2026-09-13 12:00:00')`)
	if err != nil {
		t.Fatalf("new cumulative import columns are unavailable: %v", err)
	}

	var added, updated, skipped int
	if err := database.QueryRow(`SELECT orders_added, orders_updated, orders_skipped
		FROM raw_import WHERE filename = 'history.csv'`).Scan(&added, &updated, &skipped); err != nil {
		t.Fatal(err)
	}
	if added != 10 || updated != 3 || skipped != 2 {
		t.Fatalf("unexpected counters: added=%d updated=%d skipped=%d", added, updated, skipped)
	}
}
