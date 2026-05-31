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

// Filters — сквозные фильтры запроса (мультивыбор через запятую по каждому полю).
type Filters struct {
	City     string
	Region   string
	Channel  string
	Payment  string
	Delivery string
	Coupon   string
}

// geoCond — доп. условие WHERE по фильтрам (мультивыбор через запятую). Внутри
// одного поля — ИЛИ (IN), между разными полями — И.
func geoCond(f Filters) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	addIn := func(col, raw string) {
		var vals []string
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				vals = append(vals, p)
			}
		}
		if len(vals) == 0 {
			return
		}
		ph := make([]string, len(vals))
		for i, v := range vals {
			ph[i] = "?"
			args = append(args, v)
		}
		sb.WriteString(" AND " + col + " IN (" + strings.Join(ph, ", ") + ")")
	}
	addIn("city", f.City)
	addIn("region", f.Region)
	addIn("channel", f.Channel)
	addIn("payment_system", f.Payment)
	addIn("delivery_service", f.Delivery)
	addIn("coupon", f.Coupon)
	return sb.String(), args
}

func boolTrue(d *db.DB) string {
	if d.IsPostgres() {
		return "TRUE"
	}
	return "1"
}

// reachCounts считает кумулятивные стадии и итоговые показатели за период.
// orders/revenue/units — карты «ключ стадии → значение метрики».
type reachCounts struct {
	canceled, returns, problems int
	orders                      map[string]int
	revenue                     map[string]int
	units                       map[string]int
}

// предикаты накопительных стадий (без алиаса таблицы); %[1]s — boolTrue.
const (
	condPaid     = "is_paid = %[1]s OR status_stage IN ('processing','shipped','in_pvz','completed','returned')"
	condProc     = "status_stage IN ('processing','shipped','in_pvz','completed','returned')"
	condShip     = "status_stage IN ('shipped','in_pvz','completed','returned')"
	condDelivers = "status_stage IN ('in_pvz','completed','returned')"
	condComp     = "status_stage = 'completed'"
)

func (r *Repository) reach(start, end string, f Filters) (reachCounts, error) {
	t := boolTrue(r.db)
	cc, cargs := geoCond(f)
	c := reachCounts{orders: map[string]int{}, revenue: map[string]int{}, units: map[string]int{}}

	// Этап «оплачен» засчитывается, если заказ оплачен ИЛИ продвинулся дальше
	// оплаты (в сборку и далее) — иначе кумулятивная воронка ломается из-за
	// заказов в статусе processing+ с незаполненным флагом is_paid.
	// Одним запросом считаем по каждой стадии число заказов и сумму выручки.
	q := r.db.Rebind(fmt.Sprintf(`SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN `+condPaid+` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condProc+` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condShip+` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condDelivers+` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condComp+` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(total_amount),0),
		COALESCE(SUM(CASE WHEN `+condPaid+` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condProc+` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condShip+` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condDelivers+` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condComp+` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled = %[1]s THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage = 'returned' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN has_problem = %[1]s THEN 1 ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc, t))
	var oc, op, opr, osh, od, ocm int
	var rc, rp, rpr, rsh, rd, rcm int
	if err := r.db.QueryRow(q, append([]interface{}{start, end}, cargs...)...).Scan(
		&oc, &op, &opr, &osh, &od, &ocm,
		&rc, &rp, &rpr, &rsh, &rd, &rcm,
		&c.canceled, &c.returns, &c.problems); err != nil {
		return c, err
	}
	c.orders["created"], c.orders["paid"], c.orders["processing"] = oc, op, opr
	c.orders["shipped"], c.orders["delivered"], c.orders["completed"] = osh, od, ocm
	c.revenue["created"], c.revenue["paid"], c.revenue["processing"] = rc, rp, rpr
	c.revenue["shipped"], c.revenue["delivered"], c.revenue["completed"] = rsh, rd, rcm

	// Товары (qty) по тем же стадиям — отдельным запросом с join на позиции.
	// Имена колонок города/региона/канала уникальны для orders, поэтому условие
	// geoCond без алиаса корректно работает и в join.
	uq := r.db.Rebind(fmt.Sprintf(`SELECT
		COALESCE(SUM(oi.qty),0),
		COALESCE(SUM(CASE WHEN `+condPaid+` THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condProc+` THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condShip+` THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condDelivers+` THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN `+condComp+` THEN oi.qty ELSE 0 END),0)
		FROM order_items oi JOIN orders o ON o.order_number = oi.order_number
		WHERE o.created_at >= ? AND o.created_at <= ?`+cc, t))
	var uc, up, upr, ush, ud, ucm int
	if err := r.db.QueryRow(uq, append([]interface{}{start, end}, cargs...)...).Scan(
		&uc, &up, &upr, &ush, &ud, &ucm); err != nil {
		return c, err
	}
	c.units["created"], c.units["paid"], c.units["processing"] = uc, up, upr
	c.units["shipped"], c.units["delivered"], c.units["completed"] = ush, ud, ucm
	return c, nil
}

// segment строит воронку в разрезе указанной колонки.
func (r *Repository) segment(col, start, end string, f Filters, limit int) ([]SegmentRow, error) {
	t := boolTrue(r.db)
	cc, cargs := geoCond(f)
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
func (r *Repository) topLabeled(col, start, end string, f Filters, limit int) ([]LabeledCount, error) {
	cc, cargs := geoCond(f)
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
