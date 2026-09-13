package goal

import (
	"fmt"
	"strings"
	"time"

	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/orderstage"
)

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository { return &Repository{db: database} }

type planRow struct {
	Month     int
	Channel   string
	NetTarget int
}

type orderAggregate struct {
	NetOrders            int
	Revenue              int
	RedeemedGrossRevenue int
	RefundAmount         int
}

type orderPeriod struct {
	All      orderAggregate
	Channels map[string]orderAggregate
}

type syncRun struct {
	Source       string
	Status       string
	DateFrom     string
	DateTo       string
	RowsImported int
	Error        string
	StartedAt    string
	FinishedAt   string
}

func (r *Repository) bounds() (Bounds, error) {
	query := `SELECT substr(MIN(created_at),1,10), substr(MAX(created_at),1,10)
		FROM orders WHERE created_at IS NOT NULL`
	if r.db.IsPostgres() {
		query = `SELECT TO_CHAR(MIN(created_at), 'YYYY-MM-DD'), TO_CHAR(MAX(created_at), 'YYYY-MM-DD')
			FROM orders WHERE created_at IS NOT NULL`
	}
	var min, max *string
	if err := r.db.QueryRow(query).Scan(&min, &max); err != nil {
		return Bounds{}, err
	}
	if min == nil || max == nil {
		return Bounds{}, nil
	}
	return Bounds{Min: *min, Max: *max}, nil
}

func (r *Repository) plan(year int) ([]planRow, error) {
	rows, err := r.db.Query(r.db.Rebind(
		`SELECT month, channel, net_target FROM sales_plan WHERE year = ? ORDER BY month, channel`), year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []planRow
	for rows.Next() {
		var row planRow
		if err := rows.Scan(&row.Month, &row.Channel, &row.NetTarget); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) orders(start, end string) (orderPeriod, error) {
	falseValue := "0"
	if r.db.IsPostgres() {
		falseValue = "FALSE"
	}
	query := r.db.Rebind(`SELECT COALESCE(channel,''),
		COALESCE(SUM(CASE WHEN is_canceled = ` + falseValue + ` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled = ` + falseValue + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN ` + orderstage.RedeemedGross + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN ` + orderstage.Returned + ` THEN refund_amount ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?
		GROUP BY COALESCE(channel,'')`)
	rows, err := r.db.Query(query, start+" 00:00:00", end+" 23:59:59")
	if err != nil {
		return orderPeriod{}, err
	}
	defer rows.Close()
	result := orderPeriod{Channels: map[string]orderAggregate{"site": {}, "app": {}}}
	for rows.Next() {
		var raw string
		var aggregate orderAggregate
		if err := rows.Scan(&raw, &aggregate.NetOrders, &aggregate.Revenue,
			&aggregate.RedeemedGrossRevenue, &aggregate.RefundAmount); err != nil {
			return orderPeriod{}, err
		}
		result.All = addOrders(result.All, aggregate)
		if channel := normalizeOrderChannel(raw); channel != "" {
			result.Channels[channel] = addOrders(result.Channels[channel], aggregate)
		}
	}
	return result, rows.Err()
}

func addOrders(left, right orderAggregate) orderAggregate {
	left.NetOrders += right.NetOrders
	left.Revenue += right.Revenue
	left.RedeemedGrossRevenue += right.RedeemedGrossRevenue
	left.RefundAmount += right.RefundAmount
	return left
}

func normalizeOrderChannel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "сайт", "site", "web":
		return "site"
	case "приложение", "app", "application":
		return "app"
	default:
		return ""
	}
}

func (r *Repository) sessions(start, end string) (map[string]int, error) {
	rows, err := r.db.Query(r.db.Rebind(`SELECT channel, COALESCE(SUM(sessions),0)
		FROM analytics_traffic_daily WHERE day >= ? AND day <= ? GROUP BY channel`), start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{"site": 0, "app": 0}
	for rows.Next() {
		var channel string
		var value int
		if err := rows.Scan(&channel, &value); err != nil {
			return nil, err
		}
		if channel == "site" || channel == "app" {
			out[channel] += value
		}
	}
	return out, rows.Err()
}

func (r *Repository) latestRuns() (map[string]syncRun, error) {
	rows, err := r.db.Query(`SELECT r.source, r.status, r.date_from, r.date_to,
		r.rows_imported, COALESCE(r.error_text,''), r.started_at, r.finished_at
		FROM analytics_sync_runs r
		JOIN (SELECT source, MAX(id) AS id FROM analytics_sync_runs GROUP BY source) latest
			ON latest.id = r.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]syncRun)
	for rows.Next() {
		var run syncRun
		var started, finished any
		if err := rows.Scan(&run.Source, &run.Status, &run.DateFrom, &run.DateTo,
			&run.RowsImported, &run.Error, &started, &finished); err != nil {
			return nil, err
		}
		run.StartedAt = dbTimeString(started)
		run.FinishedAt = dbTimeString(finished)
		out[run.Source] = run
	}
	return out, rows.Err()
}

func (r *Repository) latestDataDays() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT source, MAX(day) FROM analytics_traffic_daily GROUP BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var source, day string
		if err := rows.Scan(&source, &day); err != nil {
			return nil, err
		}
		out[source] = day
	}
	return out, rows.Err()
}

func dbTimeString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case time.Time:
		return typed.Format(time.RFC3339)
	case []byte:
		return string(typed)
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}
