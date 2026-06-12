package plan

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
		want := i + 1
		if m.Month != want {
			t.Errorf("month[%d] = %d, want %d", i, m.Month, want)
		}
		if m.Targets.All != 0 || m.Targets.Site != 0 || m.Targets.App != 0 {
			t.Errorf("month %d: expected zero targets", m.Month)
		}
	}
}

func TestFebruary2024DaysAndPerDay(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	_, err := svc.Save(2024, []PlanItem{{Month: 2, Channel: "all", NetTarget: 2_900_000}})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	rep, err := svc.Get(2024)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	feb := rep.Months[1]
	if feb.DaysInMonth != 29 {
		t.Errorf("daysInMonth = %d, want 29", feb.DaysInMonth)
	}
	if feb.Targets.All != 2_900_000 {
		t.Errorf("target = %d, want 2900000", feb.Targets.All)
	}
	if feb.PerDay.All != 100_000 {
		t.Errorf("perDay = %d, want 100000", feb.PerDay.All)
	}
}

func TestUpsertAndRead(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	items := []PlanItem{
		{Month: 3, Channel: "site", NetTarget: 1_000_000},
		{Month: 3, Channel: "app", NetTarget: 500_000},
	}
	if _, err := svc.Save(2026, items); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// повторный upsert — обновление
	if _, err := svc.Save(2026, []PlanItem{{Month: 3, Channel: "site", NetTarget: 2_000_000}}); err != nil {
		t.Fatalf("Save update: %v", err)
	}
	rep, err := svc.Get(2026)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	mar := rep.Months[2]
	if mar.Targets.Site != 2_000_000 {
		t.Errorf("site target = %d, want 2000000", mar.Targets.Site)
	}
	if mar.Targets.App != 500_000 {
		t.Errorf("app target = %d, want 500000", mar.Targets.App)
	}
}

func TestValidation(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	if _, err := svc.Save(1999, nil); err == nil {
		t.Error("expected year validation error")
	}
	if _, err := svc.Save(2026, []PlanItem{{Month: 13, Channel: "all", NetTarget: 1}}); err == nil {
		t.Error("expected month validation error")
	}
	if _, err := svc.Save(2026, []PlanItem{{Month: 1, Channel: "web", NetTarget: 1}}); err == nil {
		t.Error("expected channel validation error")
	}
	if _, err := svc.Save(2026, []PlanItem{{Month: 1, Channel: "all", NetTarget: -1}}); err == nil {
		t.Error("expected netTarget validation error")
	}
}
