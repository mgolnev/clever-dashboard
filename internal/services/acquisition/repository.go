package acquisition

import (
	"strings"

	"github.com/clever/clever-dashboard/internal/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

type channelTotals struct {
	Sessions   int
	Users      int
	Orders     int
	PaidOrders int
	NetOrders  int
	Sampled    bool
}

type dailyValues struct {
	Sessions int
	Orders   int
}

func (r *Repository) dataBounds() (string, string, error) {
	var min, max *string
	if err := r.db.QueryRow(`SELECT MIN(created_at), MAX(created_at) FROM orders`).Scan(&min, &max); err != nil {
		return "", "", err
	}
	trim := func(v *string) string {
		if v == nil || len(*v) < 10 {
			return ""
		}
		return (*v)[:10]
	}
	return trim(min), trim(max), nil
}

func (r *Repository) trafficTotals(start, end string) (map[string]channelTotals, error) {
	tv := "1"
	if r.db.IsPostgres() {
		tv = "TRUE"
	}
	q := r.db.Rebind(`SELECT channel, COALESCE(SUM(sessions),0), COALESCE(SUM(users),0),
		COALESCE(SUM(CASE WHEN sampled = ` + tv + ` THEN 1 ELSE 0 END),0)
		FROM analytics_traffic_daily WHERE day >= ? AND day <= ? GROUP BY channel`)
	rows, err := r.db.Query(q, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]channelTotals{"site": {}, "app": {}}
	for rows.Next() {
		var channel string
		var total channelTotals
		var sampledRows int
		if err := rows.Scan(&channel, &total.Sessions, &total.Users, &sampledRows); err != nil {
			return nil, err
		}
		total.Sampled = sampledRows > 0
		out[channel] = total
	}
	return out, rows.Err()
}

func (r *Repository) orderTotals(start, end string, totals map[string]channelTotals) error {
	tv, fv := "1", "0"
	if r.db.IsPostgres() {
		tv, fv = "TRUE", "FALSE"
	}
	q := r.db.Rebind(`SELECT COALESCE(channel,''), COUNT(*),
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled = ` + fv + ` THEN 1 ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ? GROUP BY channel`)
	rows, err := r.db.Query(q, start+" 00:00:00", end+" 23:59:59")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		var orders, paid, net int
		if err := rows.Scan(&raw, &orders, &paid, &net); err != nil {
			return err
		}
		channel := normalizeOrderChannel(raw)
		if channel == "" {
			continue
		}
		t := totals[channel]
		t.Orders += orders
		t.PaidOrders += paid
		t.NetOrders += net
		totals[channel] = t
	}
	return rows.Err()
}

func (r *Repository) dailyTraffic(start, end string) (map[string]map[string]dailyValues, error) {
	q := r.db.Rebind(`SELECT day, channel, COALESCE(SUM(sessions),0)
		FROM analytics_traffic_daily WHERE day >= ? AND day <= ? GROUP BY day, channel`)
	rows, err := r.db.Query(q, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]map[string]dailyValues)
	for rows.Next() {
		var day, channel string
		var sessions int
		if err := rows.Scan(&day, &channel, &sessions); err != nil {
			return nil, err
		}
		if out[day] == nil {
			out[day] = make(map[string]dailyValues)
		}
		v := out[day][channel]
		v.Sessions += sessions
		out[day][channel] = v
	}
	return out, rows.Err()
}

func (r *Repository) dailyOrders(start, end string, days map[string]map[string]dailyValues) error {
	dateExpr := "SUBSTR(created_at,1,10)"
	if r.db.IsPostgres() {
		dateExpr = "TO_CHAR(created_at, 'YYYY-MM-DD')"
	}
	q := r.db.Rebind(`SELECT ` + dateExpr + ` AS day, COALESCE(channel,''), COUNT(*)
		FROM orders WHERE created_at >= ? AND created_at <= ? GROUP BY ` + dateExpr + `, channel`)
	rows, err := r.db.Query(q, start+" 00:00:00", end+" 23:59:59")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var day, raw string
		var orders int
		if err := rows.Scan(&day, &raw, &orders); err != nil {
			return err
		}
		channel := normalizeOrderChannel(raw)
		if channel == "" {
			continue
		}
		if days[day] == nil {
			days[day] = make(map[string]dailyValues)
		}
		v := days[day][channel]
		v.Orders += orders
		days[day][channel] = v
	}
	return rows.Err()
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
