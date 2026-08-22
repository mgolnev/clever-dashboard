package traffic

import (
	"fmt"
	"strconv"
	"time"

	"github.com/clever/clever-dashboard/internal/db"
)

const tsLayout = "2006-01-02 15:04:05"

type row struct {
	Month     int
	Channel   string
	Visits    int
	Source    string
	UpdatedAt string
}

// Repository — доступ к таблице traffic.
type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

// loadYear возвращает все источники за год; приоритет разрешается в сервисе.
func (r *Repository) loadYear(year int) ([]row, error) {
	rows, err := r.db.Query(r.db.Rebind(
		`SELECT month, channel, visits, source, updated_at FROM traffic WHERE year = ? ORDER BY month, channel`),
		year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []row
	for rows.Next() {
		var rw row
		if err := rows.Scan(&rw.Month, &rw.Channel, &rw.Visits, &rw.Source, &rw.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rw)
	}
	return out, rows.Err()
}

// loadDailyYear агрегирует сохранённые дневные данные внешней аналитики до
// помесячного контракта /api/traffic. Ручные значения остаются fallback.
func (r *Repository) loadDailyYear(year int) ([]row, error) {
	start := fmt.Sprintf("%04d-01-01", year)
	end := fmt.Sprintf("%04d-12-31", year)
	rows, err := r.db.Query(r.db.Rebind(`SELECT day, channel, sessions, source, synced_at
		FROM analytics_traffic_daily WHERE day >= ? AND day <= ? ORDER BY day`), start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type key struct {
		month, channel, source string
	}
	agg := make(map[key]row)
	for rows.Next() {
		var day, channel, source string
		var sessions int
		var updated any
		if err := rows.Scan(&day, &channel, &sessions, &source, &updated); err != nil {
			return nil, err
		}
		if len(day) < 7 {
			continue
		}
		month := day[5:7]
		k := key{month: month, channel: channel, source: source}
		cur := agg[k]
		cur.Month, _ = strconv.Atoi(month)
		cur.Channel = channel
		cur.Source = source
		cur.Visits += sessions
		stamp := fmt.Sprint(updated)
		if stamp > cur.UpdatedAt {
			cur.UpdatedAt = stamp
		}
		agg[k] = cur
	}
	out := make([]row, 0, len(agg))
	for _, rw := range agg {
		out = append(out, rw)
	}
	return out, rows.Err()
}

// upsert сохраняет элементы трафика одной транзакцией (атомарно), сохраняя источник.
func (r *Repository) upsert(year int, items []TrafficItem) error {
	now := time.Now().Format(tsLayout)
	upsert := r.db.Rebind(`INSERT INTO traffic (year, month, channel, visits, source, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(year, month, channel, source) DO UPDATE SET
			visits = excluded.visits,
			updated_at = excluded.updated_at`)

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, it := range items {
		if _, err := tx.Exec(upsert, year, it.Month, it.Channel, it.Visits, it.Source, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
