package logistics

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
)

func logisticsTestDB(t *testing.T) *db.DB {
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

func TestDispatchMetricsMatchCumulativeFunnelAndAgeBacklog(t *testing.T) {
	database := logisticsTestDB(t)
	insert := `INSERT INTO orders (
		order_number, created_at, updated_at, status_raw, status_stage,
		is_paid, is_canceled, delivery_service, city
	) VALUES (?, ?, '2026-09-13 18:00:00', ?, ?, ?, ?, ?, ?)`
	rows := []struct {
		number, created, raw, stage, service, city string
		paid, canceled                             int
	}{
		{"a-new", "2026-09-12 10:00:00", "Собирается", "processing", "Служба A", "Пилот", 1, 0},
		{"a-aging", "2026-09-10 10:00:00", "Ожидает отправку", "processing", "Служба A", "Пилот", 1, 0},
		{"a-shipped", "2026-09-11 10:00:00", "Отправлен", "shipped", "Служба A", "Пилот", 1, 0},
		{"a-completed", "2026-09-09 10:00:00", "Выполнен", "completed", "Служба A", "Пилот", 1, 0},
		{"b-old", "2026-09-08 10:00:00", "Обработан", "processing", "Служба B", "Контроль", 1, 0},
		{"b-canceled", "2026-09-07 10:00:00", "Закрыт", "canceled", "Служба B", "Контроль", 0, 1},
	}
	for _, row := range rows {
		if _, err := database.Exec(insert, row.number, row.created, row.raw, row.stage, row.paid, row.canceled, row.service, row.city); err != nil {
			t.Fatalf("insert %s: %v", row.number, err)
		}
	}

	repo := NewRepository(database)
	summary, err := repo.summary("2026-09-07 00:00:00", "2026-09-13 23:59:59", "2026-09-13", Filters{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.ProcessedOrders != 5 || summary.ShippedOrders != 2 || summary.PendingShipment != 3 {
		t.Fatalf("dispatch counts: %+v", summary)
	}
	if summary.ShipmentRate != 40 {
		t.Fatalf("shipmentRate=%v, want 40", summary.ShipmentRate)
	}
	if summary.Pending0To1 != 1 || summary.Pending2To3 != 1 || summary.Pending4Plus != 1 {
		t.Fatalf("age buckets: %+v", summary)
	}
	if math.Abs(summary.AvgPendingAgeDays-3) > 0.001 {
		t.Fatalf("avgPendingAgeDays=%v, want 3", summary.AvgPendingAgeDays)
	}

	services, err := repo.byService("2026-09-07 00:00:00", "2026-09-13 23:59:59", "2026-09-13", Filters{}, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 || services[0].Name != "Служба A" {
		t.Fatalf("unexpected service order: %+v", services)
	}
	if services[0].ProcessedOrders != 4 || services[0].ShippedOrders != 2 || services[0].PendingShipment != 2 || services[0].ShipmentRate != 50 {
		t.Fatalf("service A dispatch: %+v", services[0])
	}

	pilot, err := repo.cohortSummary("2026-09-07 00:00:00", "2026-09-13 23:59:59", "2026-09-13", Filters{}, []string{"Пилот"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if pilot.ProcessedOrders != 4 || pilot.ShippedOrders != 2 || pilot.PendingShipment != 2 || pilot.Pending2To3 != 1 {
		t.Fatalf("pilot dispatch: %+v", pilot)
	}
}
