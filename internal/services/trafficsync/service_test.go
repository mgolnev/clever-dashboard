package trafficsync

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/model"
)

type fakeSource struct {
	from time.Time
	to   time.Time
}

func (f *fakeSource) Name() string     { return "metrika" }
func (f *fakeSource) Channel() string  { return "site" }
func (f *fakeSource) Configured() bool { return true }
func (f *fakeSource) Fetch(_ context.Context, from, to time.Time) ([]model.DailyTraffic, error) {
	f.from, f.to = from, to
	return []model.DailyTraffic{{
		Day: to.Format("2006-01-02"), Channel: "site", Sessions: 123,
		Source: "metrika", SampleShare: 1,
	}}, nil
}

func syncTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(config.Config{DBDriver: "sqlite", DBDSN: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSyncBackfillThenLookback(t *testing.T) {
	d := syncTestDB(t)
	source := &fakeSource{}
	service := NewService(NewRepository(d), []Source{source}, true, 3, 10, "UTC")
	if err := service.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := int(source.to.Sub(source.from).Hours()/24) + 1; got != 10 {
		t.Fatalf("first sync range = %d days, want 10", got)
	}
	if err := service.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := int(source.to.Sub(source.from).Hours()/24) + 1; got != 3 {
		t.Fatalf("lookback range = %d days, want 3", got)
	}
	var sessions int
	if err := d.QueryRow(`SELECT MAX(sessions) FROM analytics_traffic_daily WHERE source='metrika'`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if sessions != 123 {
		t.Fatalf("sessions = %d, want 123", sessions)
	}
	status, err := service.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Sources) != 1 || status.Sources[0].Status != "success" || status.Sources[0].LastDataDay == "" {
		t.Fatalf("unexpected status: %+v", status)
	}
}
