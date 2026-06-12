package plan

import (
	"time"

	"github.com/clever/clever-dashboard/internal/db"
)

const tsLayout = "2006-01-02 15:04:05"

type row struct {
	Month     int
	Channel   string
	NetTarget int
}

// Repository — доступ к таблице sales_plan.
type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

func (r *Repository) loadYear(year int) ([]row, error) {
	rows, err := r.db.Query(r.db.Rebind(
		`SELECT month, channel, net_target FROM sales_plan WHERE year = ? ORDER BY month, channel`),
		year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []row
	for rows.Next() {
		var rw row
		if err := rows.Scan(&rw.Month, &rw.Channel, &rw.NetTarget); err != nil {
			return nil, err
		}
		out = append(out, rw)
	}
	return out, rows.Err()
}

// upsert сохраняет элементы плана одной транзакцией (атомарно: либо все
// месяцы, либо ни одного).
func (r *Repository) upsert(year int, items []PlanItem) error {
	now := time.Now().Format(tsLayout)
	upsert := r.db.Rebind(`INSERT INTO sales_plan (year, month, channel, net_target, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(year, month, channel) DO UPDATE SET
			net_target = excluded.net_target,
			updated_at = excluded.updated_at`)

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, it := range items {
		if _, err := tx.Exec(upsert, year, it.Month, it.Channel, it.NetTarget, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
