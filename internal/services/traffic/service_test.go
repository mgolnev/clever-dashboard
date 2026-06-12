package traffic

import (
	"path/filepath"
	"testing"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(config.Config{DBDriver: "sqlite", DBDSN: dsn})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func TestGetReturns12Months(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	rep, err := svc.Get(2026)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(rep.Months) != 12 {
		t.Fatalf("months = %d, want 12", len(rep.Months))
	}
	for i, m := range rep.Months {
		if m.Month != i+1 {
			t.Errorf("month[%d] = %d", i, m.Month)
		}
		if m.Site != 0 || m.App != 0 {
			t.Errorf("month %d: expected zero visits", m.Month)
		}
	}
}

func TestUpsertAndRead(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	items := []TrafficItem{
		{Month: 5, Channel: "site", Visits: 50_000},
		{Month: 5, Channel: "app", Visits: 20_000},
	}
	if _, err := svc.Save(2026, items); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := svc.Save(2026, []TrafficItem{{Month: 5, Channel: "site", Visits: 60_000}}); err != nil {
		t.Fatalf("Save update: %v", err)
	}
	rep, err := svc.Get(2026)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	may := rep.Months[4]
	if may.Site != 60_000 {
		t.Errorf("site visits = %d, want 60000", may.Site)
	}
	if may.App != 20_000 {
		t.Errorf("app visits = %d, want 20000", may.App)
	}
}

func TestValidation(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	if _, err := svc.Save(2101, nil); err == nil {
		t.Error("expected year validation error")
	}
	if _, err := svc.Save(2026, []TrafficItem{{Month: 0, Channel: "site", Visits: 1}}); err == nil {
		t.Error("expected month validation error")
	}
	if _, err := svc.Save(2026, []TrafficItem{{Month: 1, Channel: "all", Visits: 1}}); err == nil {
		t.Error("expected channel validation error")
	}
	if _, err := svc.Save(2026, []TrafficItem{{Month: 1, Channel: "site", Visits: -1}}); err == nil {
		t.Error("expected visits validation error")
	}
}
