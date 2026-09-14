package customeranalytics

import (
	"testing"
	"time"
)

func TestBuildViewGrossAndPaidCohorts(t *testing.T) {
	events := []orderEvent{
		{CreatedAt: "2026-01-10 10:00:00", Email: "ONE@example.com", Paid: true},
		{CreatedAt: "2026-02-02 10:00:00", Email: "one@example.com", Paid: false},
		{CreatedAt: "2026-03-02 10:00:00", Email: "one@example.com", Paid: true},
		{CreatedAt: "2026-02-11 10:00:00", Phone: "+7 (900) 123-45-67", Paid: false},
		{CreatedAt: "2026-03-11 10:00:00", Phone: "79001234567", Paid: true},
	}
	start := mustDate(t, "2026-01-01")
	end := mustDate(t, "2026-03-31")

	gross := buildView("gross", events, start, end, GranularityMonth)
	if gross.Summary.Customers != 2 || gross.Summary.RepeatCustomers != 2 {
		t.Fatalf("gross summary = %+v", gross.Summary)
	}
	if got := gross.Cohorts[0].Cells[1].Rate; got != 100 {
		t.Fatalf("gross Jan M1 = %.1f, want 100", got)
	}

	paid := buildView("paid", events, start, end, GranularityMonth)
	if paid.Summary.Customers != 2 || len(paid.Cohorts) != 2 {
		t.Fatalf("paid view = %+v", paid)
	}
	if paid.Cohorts[0].Month != "2026-01-01" || paid.Cohorts[0].Cells[2].Rate != 100 {
		t.Fatalf("paid Jan cohort = %+v", paid.Cohorts[0])
	}
}

func TestBuildViewExcludesPartialMonthFromWeightedRetention(t *testing.T) {
	events := []orderEvent{
		{CreatedAt: "2026-01-05", Email: "a@example.com", Paid: true},
		{CreatedAt: "2026-02-05", Email: "a@example.com", Paid: true},
	}
	view := buildView("gross", events, mustDate(t, "2026-01-01"), mustDate(t, "2026-02-14"), GranularityMonth)
	if view.Cohorts[0].Cells[1].Complete {
		t.Fatal("текущий неполный месяц не должен считаться завершённым")
	}
	if view.Summary.Retention[1].EligibleCustomers != 0 {
		t.Fatalf("M1 denominator = %d, want 0", view.Summary.Retention[1].EligibleCustomers)
	}
}

func TestQuarterlyRetentionAndComposition(t *testing.T) {
	events := []orderEvent{
		{CreatedAt: "2026-01-05", Email: "active@example.com", Paid: true},
		{CreatedAt: "2026-04-05", Email: "active@example.com", Paid: true},
		{CreatedAt: "2026-01-10", Email: "returned@example.com", Paid: true},
		{CreatedAt: "2026-07-10", Email: "returned@example.com", Paid: true},
		{CreatedAt: "2026-07-12", Email: "new@example.com", Paid: true},
	}
	view := buildView(
		"gross",
		events,
		mustDate(t, "2026-01-01"),
		mustDate(t, "2026-09-30"),
		GranularityQuarter,
	)
	if len(view.Cohorts) != 2 || view.Cohorts[0].Month != "2026-01-01" {
		t.Fatalf("quarter cohorts = %+v", view.Cohorts)
	}
	if got := view.Cohorts[0].Cells[1].Rate; got != 50 {
		t.Fatalf("Q1 retention = %.1f, want 50", got)
	}
	if len(view.Composition) != 3 {
		t.Fatalf("composition periods = %d, want 3", len(view.Composition))
	}
	q3 := view.Composition[2]
	if q3.Customers != 2 || q3.New != 1 || q3.Active != 0 || q3.Returned != 1 {
		t.Fatalf("Q3 composition = %+v", q3)
	}
}

func TestParseGranularityRejectsUnknownValue(t *testing.T) {
	if _, err := parseGranularity("week"); err == nil {
		t.Fatal("ожидалась ошибка для неподдерживаемой гранулярности")
	}
}

func TestPartialFirstQuarterIsNotIncludedInWeightedRetention(t *testing.T) {
	events := []orderEvent{
		{CreatedAt: "2026-03-05", Email: "a@example.com", Paid: true},
		{CreatedAt: "2026-04-05", Email: "a@example.com", Paid: true},
	}
	view := buildView(
		"gross",
		events,
		mustDate(t, "2026-03-01"),
		mustDate(t, "2026-06-30"),
		GranularityQuarter,
	)
	if view.Cohorts[0].Cells[1].Complete {
		t.Fatal("когорта неполного первого квартала не должна входить в агрегированный retention")
	}
	if view.Summary.Retention[1].EligibleCustomers != 0 {
		t.Fatalf("Q1 denominator = %d, want 0", view.Summary.Retention[1].EligibleCustomers)
	}
	if view.Composition[0].Complete {
		t.Fatal("первый неполный квартал должен быть помечен как незавершённый")
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
