package metrics

import (
	"path/filepath"
	"testing"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(config.Config{DBDriver: "sqlite", DBDSN: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func TestKPIRedeemedGrossAndNet(t *testing.T) {
	database := testDB(t)
	insert := `INSERT INTO orders (order_number, created_at, total_amount, refund_amount, status_raw, status_stage, is_paid, is_canceled)
		VALUES (?, '2026-08-01 12:00:00', ?, ?, ?, ?, 1, 0)`
	for _, row := range []struct {
		number, raw, stage string
		total, refund      int
	}{
		{"completed", "Выполнен", "completed", 1000, 0},
		{"partial", "Совершён частичный возврат средств", "returned", 1000, 200},
		{"full", "Совершён возврат средств", "returned", 1000, 1000},
		{"closed", "Закрыт", "closed", 1000, 0},
	} {
		if _, err := database.Exec(insert, row.number, row.total, row.refund, row.raw, row.stage); err != nil {
			t.Fatalf("insert %s: %v", row.number, err)
		}
	}

	kpi, err := NewRepository(database).kpi("2026-08-01 00:00:00", "2026-08-01 23:59:59", Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if kpi.RedeemedGross != 3 || kpi.ReturnedOrders != 2 || kpi.FullyReturned != 1 || kpi.RedeemedNet != 2 {
		t.Fatalf("количество выкупа: %+v", kpi)
	}
	if kpi.RefundAmount != 1200 {
		t.Fatalf("refundAmount=%d, want 1200", kpi.RefundAmount)
	}
	if kpi.Stages.RedeemedGross.Revenue != 3000 || kpi.Stages.Returns.Revenue != 1200 || kpi.Stages.RedeemedNet.Revenue != 1800 {
		t.Fatalf("суммы выкупа: %+v", kpi.Stages)
	}
}
