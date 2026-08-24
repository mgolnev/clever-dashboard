package acquisition

import (
	"path/filepath"
	"testing"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
)

func testDB(t *testing.T) *db.DB {
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

func TestReportAggregatesTrafficAndOrders(t *testing.T) {
	d := testDB(t)
	for _, args := range [][]any{
		{"2026-08-10", "site", 100, 80, "metrika"},
		{"2026-08-11", "site", 200, 150, "metrika"},
		{"2026-08-10", "app", 50, 30, "appmetrica"},
		{"2026-08-11", "app", 50, 25, "appmetrica"},
	} {
		if _, err := d.Exec(`INSERT INTO analytics_traffic_daily
			(day, channel, sessions, users, source, sampled, sample_share, synced_at)
			VALUES (?, ?, ?, ?, ?, 0, 1, '2026-08-12 05:00:00')`, args...); err != nil {
			t.Fatal(err)
		}
	}
	orders := [][]any{
		{"site-1", "2026-08-10 10:00:00", "Сайт", 1, 0},
		{"site-2", "2026-08-11 10:00:00", "Сайт", 0, 1},
		{"app-1", "2026-08-11 11:00:00", "Приложение", 1, 0},
	}
	for _, args := range orders {
		if _, err := d.Exec(`INSERT INTO orders
			(order_number, created_at, channel, is_paid, is_canceled)
			VALUES (?, ?, ?, ?, ?)`, args...); err != nil {
			t.Fatal(err)
		}
	}

	report, err := NewService(NewRepository(d)).Report("2026-08-10", "2026-08-11", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Current.HasTraffic || len(report.Current.Channels) != 3 || len(report.Current.Daily) != 2 {
		t.Fatalf("unexpected report shape: %+v", report.Current)
	}
	all := report.Current.Channels[0]
	if all.Sessions != 400 || all.Users != 285 || all.Orders != 3 || all.PaidOrders != 2 || all.NetOrders != 2 {
		t.Fatalf("unexpected totals: %+v", all)
	}
	if all.OrderCR != 0.75 || all.PaidCR != 0.5 || all.NetCR != 0.5 {
		t.Fatalf("unexpected conversion: %+v", all)
	}
	site := report.Current.Channels[1]
	if site.Sessions != 300 || site.Users != 230 || site.Orders != 2 || site.NetOrders != 1 {
		t.Fatalf("unexpected site totals: %+v", site)
	}
	if report.Current.Daily[1].AppOrders != 1 || report.Current.Daily[1].AppPaidOrders != 1 ||
		report.Current.Daily[1].SitePaidOrders != 0 || report.Current.Daily[1].SiteSessions != 200 ||
		report.Current.Daily[1].SiteUsers != 150 || report.Current.Daily[1].AppUsers != 25 {
		t.Fatalf("unexpected daily point: %+v", report.Current.Daily[1])
	}
}
