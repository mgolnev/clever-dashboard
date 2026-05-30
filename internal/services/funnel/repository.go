package funnel

import (
	"fmt"
	"strings"

	"github.com/clever/clever-dashboard/internal/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

// geoCond — доп. условие WHERE по городу и/или области и его аргументы.
func geoCond(city, region string) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	if strings.TrimSpace(city) != "" {
		sb.WriteString(" AND city = ?")
		args = append(args, city)
	}
	if strings.TrimSpace(region) != "" {
		sb.WriteString(" AND region = ?")
		args = append(args, region)
	}
	return sb.String(), args
}

func boolTrue(d *db.DB) string {
	if d.IsPostgres() {
		return "TRUE"
	}
	return "1"
}

// reachCounts считает кумулятивные стадии и итоговые показатели за период.
type reachCounts struct {
	created, paid, processing, shipped, delivered, completed int
	canceled, returns, problems                              int
}

func (r *Repository) reach(start, end, city, region string) (reachCounts, error) {
	t := boolTrue(r.db)
	cc, cargs := geoCond(city, region)
	// Этап «оплачен» засчитывается, если заказ оплачен ИЛИ продвинулся дальше
	// оплаты (в сборку и далее) — иначе кумулятивная воронка ломается из-за
	// заказов в статусе processing+ с незаполненным флагом is_paid.
	q := r.db.Rebind(fmt.Sprintf(`SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN is_paid = %[1]s OR status_stage IN ('processing','shipped','in_pvz','completed','returned') THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage IN ('processing','shipped','in_pvz','completed','returned') THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage IN ('shipped','in_pvz','completed','returned') THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage IN ('in_pvz','completed','returned') THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage = 'completed' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled = %[1]s THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage = 'returned' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN has_problem = %[1]s THEN 1 ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc, t))
	var c reachCounts
	err := r.db.QueryRow(q, append([]interface{}{start, end}, cargs...)...).Scan(
		&c.created, &c.paid, &c.processing, &c.shipped, &c.delivered, &c.completed,
		&c.canceled, &c.returns, &c.problems)
	return c, err
}

// segment строит воронку в разрезе указанной колонки.
func (r *Repository) segment(col, start, end, city, region string, limit int) ([]SegmentRow, error) {
	t := boolTrue(r.db)
	cc, cargs := geoCond(city, region)
	expr := fmt.Sprintf("COALESCE(NULLIF(%s,''),'—')", col)
	q := r.db.Rebind(fmt.Sprintf(`SELECT %[1]s AS name,
		COUNT(*),
		COALESCE(SUM(CASE WHEN is_paid = %[2]s THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage = 'completed' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled = %[2]s THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN has_problem = %[2]s THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled <> %[2]s THEN total_amount ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc+`
		GROUP BY %[1]s ORDER BY COUNT(*) DESC LIMIT %[3]d`, expr, t, limit))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SegmentRow
	for rows.Next() {
		var s SegmentRow
		if err := rows.Scan(&s.Name, &s.Gross, &s.Paid, &s.Completed, &s.Canceled, &s.Problems, &s.Revenue); err != nil {
			return nil, err
		}
		if s.Gross > 0 {
			s.PaidRate = round1(float64(s.Paid) / float64(s.Gross) * 100)
			s.CompletedRate = round1(float64(s.Completed) / float64(s.Gross) * 100)
			s.CancelRate = round1(float64(s.Canceled) / float64(s.Gross) * 100)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// topLabeled возвращает топ непустых значений колонки (проблемы/причины отмены).
func (r *Repository) topLabeled(col, start, end, city, region string, limit int) ([]LabeledCount, error) {
	cc, cargs := geoCond(city, region)
	q := r.db.Rebind(fmt.Sprintf(`SELECT %[1]s AS name, COUNT(*)
		FROM orders WHERE created_at >= ? AND created_at <= ?
		AND %[1]s IS NOT NULL AND %[1]s <> ''`+cc+`
		GROUP BY %[1]s ORDER BY COUNT(*) DESC LIMIT %[2]d`, col, limit))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LabeledCount
	for rows.Next() {
		var l LabeledCount
		if err := rows.Scan(&l.Label, &l.Orders); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repository) dataBounds() (string, string, error) {
	var min, max *string
	row := r.db.QueryRow(`SELECT substr(MIN(created_at),1,10), substr(MAX(created_at),1,10) FROM orders WHERE created_at IS NOT NULL`)
	if err := row.Scan(&min, &max); err != nil {
		return "", "", err
	}
	if min == nil || max == nil {
		return "", "", nil
	}
	return *min, *max, nil
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}
