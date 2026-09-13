package goal

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

func TestReportBuildsCompactGoalReadModel(t *testing.T) {
	database := testDB(t)
	orders := [][]any{
		{"history-site", "2026-07-10 10:00:00", "Сайт", 600, 0, "completed", 0},
		{"history-app", "2026-07-20 10:00:00", "Приложение", 400, 0, "completed", 0},
		{"current-site", "2026-08-05 10:00:00", "Сайт", 1000, 0, "completed", 0},
		{"current-app-return", "2026-08-06 10:00:00", "Приложение", 2000, 500, "returned", 0},
		{"current-canceled", "2026-08-07 10:00:00", "Сайт", 3000, 0, "canceled", 1},
		{"current-other", "2026-08-11 10:00:00", "Маркетплейс", 400, 0, "new", 0},
	}
	for _, values := range orders {
		if _, err := database.Exec(`INSERT INTO orders
			(order_number, created_at, channel, total_amount, refund_amount, status_stage, is_canceled)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, values...); err != nil {
			t.Fatal(err)
		}
	}
	for _, values := range [][]any{
		{"2026-07-10", "site", 60, "metrika"},
		{"2026-07-20", "app", 40, "appmetrica"},
		{"2026-08-05", "site", 100, "metrika"},
		{"2026-08-06", "app", 50, "appmetrica"},
	} {
		if _, err := database.Exec(`INSERT INTO analytics_traffic_daily
			(day, channel, sessions, users, source, sampled, sample_share, synced_at)
			VALUES (?, ?, ?, 0, ?, 0, 1, '2026-08-12 05:00:00')`, values...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.Exec(`INSERT INTO sales_plan
		(year, month, channel, net_target, updated_at) VALUES (2026, 8, 'all', 3100, '2026-08-01 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO analytics_sync_runs
		(source, date_from, date_to, status, rows_imported, error_text, started_at, finished_at)
		VALUES ('metrika', '2026-08-01', '2026-08-10', 'success', 10, '',
		'2026-08-11 01:00:00', '2026-08-11 01:01:00')`); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewRepository(database), Options{
		AnalyticsEnabled: true,
		Sources: []SourceDefinition{
			{Source: "metrika", Channel: "site", Configured: true},
			{Source: "appmetrica", Channel: "app", Configured: true},
		},
	})
	report, err := service.Report(2026, 8)
	if err != nil {
		t.Fatal(err)
	}
	if report.Bounds.Min != "2026-07-10" || report.Bounds.Max != "2026-08-11" {
		t.Fatalf("unexpected bounds: %+v", report.Bounds)
	}
	if len(report.Plan.Months) != 12 || report.Plan.Months[7].Targets.All != 3100 {
		t.Fatalf("unexpected plan: %+v", report.Plan)
	}
	if report.Current.Start != "2026-08-01" || report.Current.End != "2026-08-31" {
		t.Fatalf("unexpected current range: %+v", report.Current)
	}
	all := report.Current.Channels[0]
	if all.Channel != "all" || all.Revenue != 3400 || all.NetRevenue != 2500 || all.AOV != 1133 || all.Sessions != 150 || all.NetCR != 1.33 {
		t.Fatalf("unexpected all summary: %+v", all)
	}
	site := report.Current.Channels[1]
	if site.Revenue != 1000 || site.NetRevenue != 1000 || site.Sessions != 100 || site.NetCR != 1 {
		t.Fatalf("unexpected site summary: %+v", site)
	}
	app := report.Current.Channels[2]
	if app.Revenue != 2000 || app.NetRevenue != 1500 || app.Sessions != 50 || app.NetCR != 2 {
		t.Fatalf("unexpected app summary: %+v", app)
	}
	if report.History == nil || report.History.Start != "2026-07-10" || report.History.End != "2026-07-31" {
		t.Fatalf("unexpected history range: %+v", report.History)
	}
	if report.History.Channels[1].Revenue != 600 || report.History.Channels[2].Revenue != 400 {
		t.Fatalf("unexpected history channels: %+v", report.History.Channels)
	}
	if !report.AnalyticsStatus.Enabled || len(report.AnalyticsStatus.Sources) != 2 {
		t.Fatalf("unexpected analytics status: %+v", report.AnalyticsStatus)
	}
	if source := report.AnalyticsStatus.Sources[0]; source.Status != "success" || source.LastDataDay != "2026-08-05" {
		t.Fatalf("unexpected metrika status: %+v", source)
	}
	if source := report.AnalyticsStatus.Sources[1]; source.Status != "never" || source.LastDataDay != "2026-08-06" {
		t.Fatalf("unexpected appmetrica status: %+v", source)
	}
}

func TestReportWithoutOrdersHasNoHistory(t *testing.T) {
	service := NewService(NewRepository(testDB(t)), Options{})
	report, err := service.Report(2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	if report.History != nil || len(report.Current.Channels) != 3 || report.Plan.Months[1].DaysInMonth != 28 {
		t.Fatalf("unexpected empty report: %+v", report)
	}
}

func TestReportValidatesPeriod(t *testing.T) {
	service := NewService(NewRepository(testDB(t)), Options{})
	if _, err := service.Report(1999, 1); err == nil {
		t.Fatal("expected year validation error")
	}
	if _, err := service.Report(2026, 13); err == nil {
		t.Fatal("expected month validation error")
	}
}
