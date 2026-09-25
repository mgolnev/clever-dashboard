package ordercleanup

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
)

func TestPurgeBeforeBacksUpAndDeletesOnlyOldOrders(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "orders.db")
	database, err := db.Open(config.Config{DBDriver: "sqlite", DBDSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Migrate(); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ number, date string }{
		{"old-1", "2021-07-06 10:00:00"},
		{"old-2", "2025-06-01 10:00:00"},
		{"old-3", "2025-06-02 10:00:00"},
		{"old-4", "2025-09-01 10:00:00"},
		{"current", "2026-01-01 10:00:00"},
	} {
		if _, err := database.Exec(`INSERT INTO orders(order_number, created_at) VALUES (?, ?)`, row.number, row.date); err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(`INSERT INTO order_items(order_number) VALUES (?)`, row.number); err != nil {
			t.Fatal(err)
		}
	}
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := PurgeBefore(database, dsn, cutoff, 3); err == nil {
		t.Fatal("ожидалось отклонение при неверном количестве строк")
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count); err != nil || count != 5 {
		t.Fatalf("после отклонения осталось %d заказов: %v", count, err)
	}

	result, err := PurgeBefore(database, dsn, cutoff, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.OrderNumbers) != 4 || result.ItemsDeleted != 4 || result.BackupPath == "" {
		t.Fatalf("неверный результат очистки: %+v", result)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("после очистки осталось %d заказов: %v", count, err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("после очистки осталось %d позиций: %v", count, err)
	}
	backup, err := sql.Open("sqlite", result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if err := backup.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count); err != nil || count != 5 {
		t.Fatalf("в резервной копии %d заказов: %v", count, err)
	}
	repeat, err := PurgeBefore(database, dsn, cutoff, 4)
	if err != nil || len(repeat.OrderNumbers) != 0 {
		t.Fatalf("повторная очистка: result=%+v err=%v", repeat, err)
	}
}
