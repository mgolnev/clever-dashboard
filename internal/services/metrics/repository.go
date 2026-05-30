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

// geoCond возвращает доп. условие WHERE по городу и/или области и его аргументы.
// prefix — префикс таблицы ("" или "o."). Пустые значения не фильтруют.
func geoCond(city, region, prefix string) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	if strings.TrimSpace(city) != "" {
		sb.WriteString(" AND " + prefix + "city = ?")
		args = append(args, city)
	}
	if strings.TrimSpace(region) != "" {
		sb.WriteString(" AND " + prefix + "region = ?")
		args = append(args, region)
	}
	return sb.String(), args
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

// geoValues возвращает значения колонки (city/region) по убыванию числа заказов.
func (r *Repository) geoValues(col string) ([]NamedCount, error) {
	rows, err := r.db.Query(fmt.Sprintf(`SELECT %[1]s, COUNT(*) AS orders FROM orders
		WHERE %[1]s IS NOT NULL AND %[1]s <> ''
		GROUP BY %[1]s ORDER BY orders DESC, %[1]s ASC`, col))
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

func (r *Repository) cities() ([]NamedCount, error)  { return r.geoValues("city") }
func (r *Repository) regions() ([]NamedCount, error) { return r.geoValues("region") }

func (r *Repository) kpi(start, end, city, region string) (KPI, error) {
	var k KPI
	cc, cargs := geoCond(city, region, "")
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
	oc, ocargs := geoCond(city, region, "o.")
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

	// Выручка по стадиям (оформлено / оплачено / транзит / выкуплено) и
	// служебные знаменатели terminal / paid+terminal для G2N/P2N.
	const transitCond = "status_stage IN ('new','processing','shipped','in_pvz')"
	const termCond = "status_stage IN ('completed','canceled','closed','returned')"
	tv := trueVal(r.db)
	var grossRev, paidRev, transitRev, compRev, termRev, paidTermRev int
	var paidTermOrders int
	rq := r.db.Rebind(`SELECT
		COALESCE(SUM(total_amount),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN ` + transitCond + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage = 'completed' THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN ` + termCond + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` AND ` + termCond + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` AND ` + termCond + ` THEN 1 ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?` + cc)
	if err := r.db.QueryRow(rq, append([]interface{}{start, end}, cargs...)...).Scan(
		&grossRev, &paidRev, &transitRev, &compRev, &termRev, &paidTermRev, &paidTermOrders); err != nil {
		return k, err
	}
	// Единицы товара по стадиям.
	var grossUnits, paidUnits, transitUnits, compUnits, termUnits, paidTermUnits int
	suq := r.db.Rebind(`SELECT
		COALESCE(SUM(oi.qty),0),
		COALESCE(SUM(CASE WHEN o.is_paid = ` + tv + ` THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN o.` + transitCond + ` THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN o.status_stage = 'completed' THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN o.` + termCond + ` THEN oi.qty ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN o.is_paid = ` + tv + ` AND o.` + termCond + ` THEN oi.qty ELSE 0 END),0)
		FROM order_items oi JOIN orders o ON o.order_number = oi.order_number
		WHERE o.created_at >= ? AND o.created_at <= ?` + oc)
	if err := r.db.QueryRow(suq, append([]interface{}{start, end}, ocargs...)...).Scan(
		&grossUnits, &paidUnits, &transitUnits, &compUnits, &termUnits, &paidTermUnits); err != nil {
		return k, err
	}
	k.Stages = KPIStages{
		Created:      makeStage(k.Orders, grossRev, grossUnits),
		Paid:         makeStage(k.PaidOrders, paidRev, paidUnits),
		InTransit:    makeStage(k.InTransit, transitRev, transitUnits),
		Completed:    makeStage(k.Completed, compRev, compUnits),
		Terminal:     makeStage(k.Terminal, termRev, termUnits),
		PaidTerminal: makeStage(paidTermOrders, paidTermRev, paidTermUnits),
	}
	return k, nil
}

// makeStage считает производные показатели стадии (AOV/ASP/UPT).
func makeStage(orders, revenue, units int) StageKPI {
	s := StageKPI{Orders: orders, Revenue: revenue, Units: units}
	if orders > 0 {
		s.AOV = revenue / orders
		s.UPT = round2(float64(units) / float64(orders))
	}
	if units > 0 {
		s.ASP = revenue / units
	}
	return s
}

func (r *Repository) funnel(start, end, city, region string) ([]FunnelStage, error) {
	cc, cargs := geoCond(city, region, "")
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
func (r *Repository) groupOrders(col, start, end, city, region string, limit int) ([]NamedCount, error) {
	cc, cargs := geoCond(city, region, "")
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
func (r *Repository) groupProducts(col, start, end, city, region string, limit int) ([]ProductRow, error) {
	oc, ocargs := geoCond(city, region, "o.")
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
