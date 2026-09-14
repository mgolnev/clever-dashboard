// Package customeranalytics строит независимый когортный анализ клиентов.
package customeranalytics

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

const dateLayout = "2006-01-02"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Report(start, end string, filters Filters, rawGranularity string) (*Report, error) {
	granularity, err := parseGranularity(rawGranularity)
	if err != nil {
		return nil, err
	}
	minDate, maxDate, err := s.repo.bounds()
	if err != nil {
		return nil, err
	}
	if minDate == "" {
		today := time.Now().Format(dateLayout)
		return emptyReport(today, today, granularity), nil
	}
	if start == "" {
		start = minDate
	}
	if end == "" {
		end = maxDate
	}
	st, err := time.Parse(dateLayout, start)
	if err != nil {
		return nil, fmt.Errorf("неверная дата начала: %w", err)
	}
	en, err := time.Parse(dateLayout, end)
	if err != nil {
		return nil, fmt.Errorf("неверная дата конца: %w", err)
	}
	if en.Before(st) {
		st, en = en, st
	}

	events, err := s.repo.events(en.Format(dateLayout), filters)
	if err != nil {
		return nil, err
	}
	report := &Report{
		Period:      Range{Start: st.Format(dateLayout), End: en.Format(dateLayout)},
		Granularity: granularity,
		Gross:       buildView("gross", events, st, en, granularity),
		Paid:        buildView("paid", events, st, en, granularity),
	}
	return report, nil
}

func emptyReport(start, end string, granularity Granularity) *Report {
	return &Report{
		Period:      Range{Start: start, End: end},
		Granularity: granularity,
		Gross:       View{Mode: "gross", Summary: Summary{Retention: []RetentionPoint{}}, Cohorts: []Cohort{}, Composition: []CompositionPeriod{}},
		Paid:        View{Mode: "paid", Summary: Summary{Retention: []RetentionPoint{}}, Cohorts: []Cohort{}, Composition: []CompositionPeriod{}},
	}
}

type customerActivity struct {
	first   time.Time
	periods map[string]bool
}

func buildView(mode string, events []orderEvent, start, end time.Time, granularity Granularity) View {
	activities := make(map[string]*customerActivity)
	for _, event := range events {
		if mode == "paid" && !event.Paid {
			continue
		}
		identity := customerIdentity(event)
		if identity == "" {
			continue
		}
		created, err := parseEventDate(event.CreatedAt)
		if err != nil || created.After(end) {
			continue
		}
		period := periodStart(created, granularity)
		activity, ok := activities[identity]
		if !ok {
			activity = &customerActivity{first: period, periods: make(map[string]bool)}
			activities[identity] = activity
		}
		if period.Before(activity.first) {
			activity.first = period
		}
		activity.periods[periodKey(period)] = true
	}

	startPeriod := periodStart(start, granularity)
	endPeriod := periodStart(end, granularity)
	members := make(map[string][]*customerActivity)
	for _, activity := range activities {
		if activity.first.Before(startPeriod) || activity.first.After(endPeriod) {
			continue
		}
		key := periodKey(activity.first)
		members[key] = append(members[key], activity)
	}

	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	maxOffset := periodsBetween(startPeriod, endPeriod, granularity)
	cohorts := make([]Cohort, 0, len(keys))
	retained := make([]int, maxOffset+1)
	eligible := make([]int, maxOffset+1)
	repeatCustomers := 0

	for _, key := range keys {
		cohortPeriod, _ := time.Parse(dateLayout, key)
		cohortComplete := !cohortPeriod.Equal(startPeriod) || start.Equal(startPeriod)
		people := members[key]
		cells := make([]RetentionCell, maxOffset+1)
		for _, person := range people {
			for period := range person.periods {
				activePeriod, _ := time.Parse(dateLayout, period)
				if activePeriod.After(cohortPeriod) {
					repeatCustomers++
					break
				}
			}
		}
		for offset := 0; offset <= maxOffset; offset++ {
			target := addPeriods(cohortPeriod, offset, granularity)
			available := !target.After(endPeriod)
			complete := available && cohortComplete && (target.Before(endPeriod) || isPeriodEnd(end, granularity))
			count := 0
			if available {
				for _, person := range people {
					if person.periods[periodKey(target)] {
						count++
					}
				}
			}
			cell := RetentionCell{
				Offset: offset, Month: periodKey(target), Customers: count,
				Rate: percentage(count, len(people)), Available: available, Complete: complete,
			}
			cells[offset] = cell
			if complete || (offset == 0 && cohortComplete) {
				retained[offset] += count
				eligible[offset] += len(people)
			}
		}
		cohorts = append(cohorts, Cohort{Month: key, Customers: len(people), Cells: cells})
	}

	retention := make([]RetentionPoint, maxOffset+1)
	for offset := 0; offset <= maxOffset; offset++ {
		retention[offset] = RetentionPoint{
			Offset: offset, Customers: retained[offset], EligibleCustomers: eligible[offset],
			Rate: percentage(retained[offset], eligible[offset]),
		}
	}
	total := 0
	for _, cohort := range cohorts {
		total += cohort.Customers
	}
	return View{
		Mode: mode,
		Summary: Summary{
			Customers: total, CohortCount: len(cohorts), RepeatCustomers: repeatCustomers,
			RepeatRate: percentage(repeatCustomers, total), Retention: retention,
		},
		Cohorts:     cohorts,
		Composition: buildComposition(activities, startPeriod, endPeriod, start, end, granularity),
	}
}

func buildComposition(activities map[string]*customerActivity, start, end, exactStart, exactEnd time.Time, granularity Granularity) []CompositionPeriod {
	count := periodsBetween(start, end, granularity) + 1
	out := make([]CompositionPeriod, 0, count)
	for i := 0; i < count; i++ {
		period := addPeriods(start, i, granularity)
		previous := addPeriods(period, -1, granularity)
		row := CompositionPeriod{
			Period:   periodKey(period),
			Complete: (!period.Equal(start) || exactStart.Equal(start)) && (period.Before(end) || isPeriodEnd(exactEnd, granularity)),
		}
		for _, activity := range activities {
			if !activity.periods[periodKey(period)] {
				continue
			}
			row.Customers++
			switch {
			case activity.first.Equal(period):
				row.New++
			case activity.periods[periodKey(previous)]:
				row.Active++
			default:
				row.Returned++
			}
		}
		row.NewRate = percentage(row.New, row.Customers)
		row.ActiveRate = percentage(row.Active, row.Customers)
		row.ReturnedRate = percentage(row.Returned, row.Customers)
		out = append(out, row)
	}
	return out
}

func customerIdentity(event orderEvent) string {
	if email := strings.ToLower(strings.TrimSpace(event.Email)); email != "" {
		return "email:" + email
	}
	var digits strings.Builder
	for _, r := range event.Phone {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	if digits.Len() > 0 {
		return "phone:" + digits.String()
	}
	if name := strings.ToLower(strings.TrimSpace(event.Customer)); name != "" {
		return "name:" + strings.Join(strings.Fields(name), " ")
	}
	return ""
}

func parseEventDate(raw string) (time.Time, error) {
	if len(raw) >= len(dateLayout) {
		raw = raw[:len(dateLayout)]
	}
	return time.Parse(dateLayout, raw)
}

func parseGranularity(raw string) (Granularity, error) {
	switch Granularity(raw) {
	case "", GranularityMonth:
		return GranularityMonth, nil
	case GranularityQuarter:
		return GranularityQuarter, nil
	default:
		return "", fmt.Errorf("неверная гранулярность %q: ожидается month или quarter", raw)
	}
}

func periodStart(value time.Time, granularity Granularity) time.Time {
	month := value.Month()
	if granularity == GranularityQuarter {
		month = time.Month((int(month)-1)/3*3 + 1)
	}
	return time.Date(value.Year(), month, 1, 0, 0, 0, 0, time.UTC)
}

func periodKey(value time.Time) string { return value.Format(dateLayout) }

func addPeriods(value time.Time, count int, granularity Granularity) time.Time {
	months := count
	if granularity == GranularityQuarter {
		months *= 3
	}
	return value.AddDate(0, months, 0)
}

func periodsBetween(start, end time.Time, granularity Granularity) int {
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month())
	if granularity == GranularityQuarter {
		return months / 3
	}
	return months
}

func isPeriodEnd(value time.Time, granularity Granularity) bool {
	period := periodStart(value, granularity)
	next := addPeriods(period, 1, granularity)
	last := next.AddDate(0, 0, -1)
	return value.Year() == last.Year() && value.Month() == last.Month() && value.Day() == last.Day()
}

func percentage(value, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(int((float64(value)/float64(total)*100)*10+0.5)) / 10
}
