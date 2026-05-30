package metrics

import (
	"fmt"
	"strings"

	"github.com/clever/clever-dashboard/internal/db"
)

type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

// cityCond возвращает доп. условие WHERE по городу и его аргументы.
// col — имя колонки (например "city" или "o.city"). Пустой город — без фильтра.
func cityCond(city, col string) (string, []interface{}) {
	if strings.TrimSpace(city) == "" {
		return "", nil
	}
	return " AND " + col + " = ?", []interface{}{city}
}

// dataBounds возвращает минимальную и максимальную дату заказов (YYYY-MM-DD).
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

// cities возвращает города, отсортированные по числу заказов (для фильтра).
func (r *Repository) cities() ([]NamedCount, error) {
	rows, err := r.db.Query(`SELECT city, COUNT(*) AS orders FROM orders
		WHERE city IS NOT NULL AND city <> ''
		GROUP BY city ORDER BY orders DESC, city ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NamedCount
	for rows.Next() {
		var n NamedCount
		if err := rows.Scan(&n.Name, &n.Orders); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) kpi(start, end, city string) (KPI, error) {
	var k KPI
	cc, cargs := cityCond(city, "city")
	q := r.db.Rebind(`SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN is_canceled = ` + falseVal(r.db) + ` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled = ` + falseVal(r.db) + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + trueVal(r.db) + ` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_canceled = ` + trueVal(r.db) + ` THEN 1 ELSE 0 END),0),
		COUNT(DISTINCT customer),
		COALESCE(SUM(CASE WHEN status_stage = 'completed' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage IN ('completed','canceled','closed','returned') THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage IN ('new','processing','shipped','in_pvz') THEN 1 ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?` + cc)
	err := r.db.QueryRow(q, append([]interface{}{start, end}, cargs...)...).Scan(
		&k.Orders, &k.NetOrders, &k.Revenue, &k.PaidOrders, &k.CanceledOrders, &k.Customers,
		&k.Completed, &k.Terminal, &k.InTransit)
	if err != nil {
		return k, err
	}
	// Units и выручка позиций (только не отменённые заказы) — для ASP.
	var itemRevenue int
	oc, ocargs := cityCond(city, "o.city")
	uq := r.db.Rebind(`SELECT COALESCE(SUM(oi.qty),0), COALESCE(SUM(oi.line_sum),0)
		FROM order_items oi JOIN orders o ON o.order_number = oi.order_number
		WHERE o.created_at >= ? AND o.created_at <= ? AND o.is_canceled = ` + falseVal(r.db) + oc)
	if err := r.db.QueryRow(uq, append([]interface{}{start, end}, ocargs...)...).Scan(&k.Units, &itemRevenue); err != nil {
		return k, err
	}
	if k.NetOrders > 0 {
		k.AOV = k.Revenue / k.NetOrders
	}
	if k.Units > 0 {
		k.ASP = itemRevenue / k.Units
	}
	if k.Orders > 0 {
		k.PaidRate = round2(float64(k.PaidOrders) / float64(k.Orders) * 100)
		k.CanceledRate = round2(float64(k.CanceledOrders) / float64(k.Orders) * 100)
		k.G2N = round2(float64(k.Completed) / float64(k.Orders) * 100)
	}
	if k.Terminal > 0 {
		k.RedemptionRate = round2(float64(k.Completed) / float64(k.Terminal) * 100)
	}
	return k, nil
}

func (r *Repository) funnel(start, end, city string) ([]FunnelStage, error) {
	cc, cargs := cityCond(city, "city")
	q := r.db.Rebind(`SELECT status_stage, COUNT(*) FROM orders
		WHERE created_at >= ? AND created_at <= ?` + cc + ` GROUP BY status_stage`)
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var stage *string
		var c int
		if err := rows.Scan(&stage, &c); err != nil {
			return nil, err
		}
		key := ""
		if stage != nil {
			key = *stage
		}
		counts[key] += c
	}
	out := make([]FunnelStage, 0, len(funnelOrder))
	for _, st := range funnelOrder {
		if c, ok := counts[st]; ok {
			out = append(out, FunnelStage{Stage: st, Label: stageLabels[st], Orders: c})
		}
	}
	return out, rows.Err()
}

// groupOrders — срез по колонке заказов (channel/payment_system/...).
func (r *Repository) groupOrders(col, start, end, city string, limit int) ([]NamedCount, error) {
	cc, cargs := cityCond(city, "city")
	q := r.db.Rebind(fmt.Sprintf(`SELECT COALESCE(NULLIF(%[1]s,''),'—') AS label, COUNT(*) AS orders,
		COALESCE(SUM(CASE WHEN is_canceled = `+falseVal(r.db)+` THEN total_amount ELSE 0 END),0) AS revenue
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc+`
		GROUP BY COALESCE(NULLIF(%[1]s,''),'—') ORDER BY orders DESC LIMIT %[2]d`, col, limit))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NamedCount
	for rows.Next() {
		var n NamedCount
		if err := rows.Scan(&n.Name, &n.Orders, &n.Revenue); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// groupProducts — товарная агрегация по колонке позиций (name/category/gender/brand).
func (r *Repository) groupProducts(col, start, end, city string, limit int) ([]ProductRow, error) {
	oc, ocargs := cityCond(city, "o.city")
	q := r.db.Rebind(fmt.Sprintf(`SELECT COALESCE(NULLIF(oi.%[1]s,''),'—') AS label,
		COALESCE(SUM(oi.qty),0) AS units,
		COUNT(DISTINCT oi.order_number) AS orders,
		COALESCE(SUM(oi.line_sum),0) AS revenue
		FROM order_items oi JOIN orders o ON o.order_number = oi.order_number
		WHERE o.created_at >= ? AND o.created_at <= ? AND o.is_canceled = `+falseVal(r.db)+oc+`
		GROUP BY COALESCE(NULLIF(oi.%[1]s,''),'—') ORDER BY revenue DESC LIMIT %[2]d`, col, limit))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, ocargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductRow
	for rows.Next() {
		var p ProductRow
		if err := rows.Scan(&p.Name, &p.Units, &p.Orders, &p.Revenue); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// falseVal/trueVal — литералы булевых значений для разных диалектов.
func falseVal(d *db.DB) string {
	if d.IsPostgres() {
		return "FALSE"
	}
	return "0"
}

func trueVal(d *db.DB) string {
	if d.IsPostgres() {
		return "TRUE"
	}
	return "1"
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
