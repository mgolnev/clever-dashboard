package traffic

import (
	"time"

	"github.com/clever/clever-dashboard/internal/db"
)

const (
	tsLayout     = "2006-01-02 15:04:05"
	manualSource = "manual"
)

type row struct {
	Month   int
	Channel string
	Visits  int
}

// Repository — доступ к таблице traffic.
type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

func (r *Repository) loadYear(year int) ([]row, error) {
	rows, err := r.db.Query(r.db.Rebind(
		`SELECT month, channel, visits FROM traffic WHERE year = ? AND source = ? ORDER BY month, channel`),
		year, manualSource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []row
	for rows.Next() {
		var rw row
		if err := rows.Scan(&rw.Month, &rw.Channel, &rw.Visits); err != nil {
			return nil, err
		}
		out = append(out, rw)
	}
	return out, rows.Err()
}

// upsert сохраняет элементы трафика одной транзакцией (атомарно).
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
		if _, err := tx.Exec(upsert, year, it.Month, it.Channel, it.Visits, manualSource, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
