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

// Filters — сквозные фильтры запроса (город/область/витрина/способ оплаты/способ
// доставки). Каждое поле — список значений через запятую (мультивыбор).
type Filters struct {
	City     string
	Region   string
	Channel  string
	Payment  string
	Delivery string
	Coupon   string
}

// geoCond возвращает доп. условие WHERE по фильтрам и его аргументы. Внутри
// одного поля — логика ИЛИ (IN), между разными полями — И. prefix — префикс
// таблицы ("" или "o."). Пустые значения не фильтруют.
func geoCond(f Filters, prefix string) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	addIn := func(col, raw string) {
		vals := splitCSV(raw)
		if len(vals) == 0 {
			return
		}
		ph := make([]string, len(vals))
		for i, v := range vals {
			ph[i] = "?"
			args = append(args, v)
		}
		sb.WriteString(" AND " + prefix + col + " IN (" + strings.Join(ph, ", ") + ")")
	}
	addIn("city", f.City)
	addIn("region", f.Region)
	addIn("channel", f.Channel)
	addIn("payment_system", f.Payment)
	addIn("delivery_service", f.Delivery)
	addIn("coupon", f.Coupon)
	return sb.String(), args
}

// splitCSV разбивает строку значений через запятую: тримит, отбрасывает пустые.
func splitCSV(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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
	out := make([]NamedCount, 0)
	for rows.Next() {
		var n NamedCount
		if err := rows.Scan(&n.Name, &n.Orders); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) cities() ([]NamedCount, error)     { return r.geoValues("city") }
func (r *Repository) regions() ([]NamedCount, error)    { return r.geoValues("region") }
func (r *Repository) channels() ([]NamedCount, error)   { return r.geoValues("channel") }
func (r *Repository) payments() ([]NamedCount, error)   { return r.geoValues("payment_system") }
func (r *Repository) deliveries() ([]NamedCount, error) { return r.geoValues("delivery_service") }
func (r *Repository) coupons() ([]NamedCount, error)    { return r.geoValues("coupon") }

func (r *Repository) kpi(start, end string, f Filters) (KPI, error) {
	var k KPI
	cc, cargs := geoCond(f, "")
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
	oc, ocargs := geoCond(f, "o.")
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
	const custFilter = "customer IS NOT NULL AND customer <> ''"
	tv := trueVal(r.db)
	var grossRev, paidRev, transitRev, compRev, termRev, paidTermRev int
	var paidTermOrders int
	var grossCust, paidCust, transitCust, compCust, termCust, paidTermCust int
	rq := r.db.Rebind(`SELECT
		COALESCE(SUM(total_amount),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN ` + transitCond + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN status_stage = 'completed' THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN ` + termCond + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` AND ` + termCond + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` AND ` + termCond + ` THEN 1 ELSE 0 END),0),
		COUNT(DISTINCT CASE WHEN ` + custFilter + ` THEN customer END),
		COUNT(DISTINCT CASE WHEN is_paid = ` + tv + ` AND ` + custFilter + ` THEN customer END),
		COUNT(DISTINCT CASE WHEN ` + transitCond + ` AND ` + custFilter + ` THEN customer END),
		COUNT(DISTINCT CASE WHEN status_stage = 'completed' AND ` + custFilter + ` THEN customer END),
		COUNT(DISTINCT CASE WHEN ` + termCond + ` AND ` + custFilter + ` THEN customer END),
		COUNT(DISTINCT CASE WHEN is_paid = ` + tv + ` AND ` + termCond + ` AND ` + custFilter + ` THEN customer END)
		FROM orders WHERE created_at >= ? AND created_at <= ?` + cc)
	if err := r.db.QueryRow(rq, append([]interface{}{start, end}, cargs...)...).Scan(
		&grossRev, &paidRev, &transitRev, &compRev, &termRev, &paidTermRev, &paidTermOrders,
		&grossCust, &paidCust, &transitCust, &compCust, &termCust, &paidTermCust); err != nil {
		return k, err
	}

	// Повторные и отменившие покупатели за период.
	rptq := r.db.Rebind(`SELECT COUNT(*) FROM (
		SELECT customer FROM orders
		WHERE created_at >= ? AND created_at <= ? AND ` + custFilter + cc + `
		GROUP BY customer HAVING COUNT(*) > 1
	) t`)
	if err := r.db.QueryRow(rptq, append([]interface{}{start, end}, cargs...)...).Scan(&k.RepeatCustomers); err != nil {
		return k, err
	}
	cancq := r.db.Rebind(`SELECT COUNT(DISTINCT customer) FROM orders
		WHERE created_at >= ? AND created_at <= ? AND is_canceled = ` + tv + ` AND ` + custFilter + cc)
	if err := r.db.QueryRow(cancq, append([]interface{}{start, end}, cargs...)...).Scan(&k.CanceledCustomers); err != nil {
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
		Created:      makeStage(k.Orders, grossRev, grossUnits, grossCust),
		Paid:         makeStage(k.PaidOrders, paidRev, paidUnits, paidCust),
		InTransit:    makeStage(k.InTransit, transitRev, transitUnits, transitCust),
		Completed:    makeStage(k.Completed, compRev, compUnits, compCust),
		Terminal:     makeStage(k.Terminal, termRev, termUnits, termCust),
		PaidTerminal: makeStage(paidTermOrders, paidTermRev, paidTermUnits, paidTermCust),
	}
	return k, nil
}

// makeStage считает производные показатели стадии (AOV/ASP/UPT).
func makeStage(orders, revenue, units, customers int) StageKPI {
	s := StageKPI{Orders: orders, Revenue: revenue, Units: units, Customers: customers}
	if orders > 0 {
		s.AOV = revenue / orders
		s.UPT = round2(float64(units) / float64(orders))
	}
	if units > 0 {
		s.ASP = revenue / units
	}
	return s
}

func (r *Repository) funnel(start, end string, f Filters) ([]FunnelStage, error) {
	cc, cargs := geoCond(f, "")
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
func (r *Repository) groupOrders(col, start, end string, f Filters, limit int) ([]NamedCount, error) {
	cc, cargs := geoCond(f, "")
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
func (r *Repository) groupProducts(col, start, end string, f Filters, limit int) ([]ProductRow, error) {
	oc, ocargs := geoCond(f, "o.")
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

// topCustomers — топ покупателей: объединяет лидеров по выручке и по числу заказов.
func (r *Repository) topCustomers(start, end string, f Filters, limit int) ([]CustomerRow, error) {
	byRev, err := r.topCustomersQuery(start, end, f, limit, "revenue")
	if err != nil {
		return nil, err
	}
	byOrd, err := r.topCustomersQuery(start, end, f, limit/2+5, "orders")
	if err != nil {
		return nil, err
	}
	return mergeCustomerRows(byRev, byOrd, limit), nil
}

func (r *Repository) topCustomersQuery(start, end string, f Filters, limit int, orderBy string) ([]CustomerRow, error) {
	cc, cargs := geoCond(f, "")
	order := "revenue DESC, orders DESC"
	if orderBy == "orders" {
		order = "orders DESC, revenue DESC"
	}
	tv := trueVal(r.db)
	fv := falseVal(r.db)
	const transitCond = "status_stage IN ('new','processing','shipped','in_pvz')"
	q := r.db.Rebind(`SELECT customer,
		COUNT(*) AS orders,
		COALESCE(SUM(CASE WHEN is_canceled = ` + fv + ` THEN total_amount ELSE 0 END),0) AS revenue,
		COALESCE(SUM(CASE WHEN is_paid = ` + tv + ` THEN 1 ELSE 0 END),0) AS paid_orders,
		COALESCE(SUM(CASE WHEN ` + transitCond + ` THEN 1 ELSE 0 END),0) AS in_transit_orders,
		COALESCE(SUM(CASE WHEN status_stage = 'completed' THEN 1 ELSE 0 END),0) AS completed_orders,
		COALESCE(SUM(CASE WHEN is_canceled = ` + tv + ` THEN 1 ELSE 0 END),0) AS canceled_orders
		FROM orders
		WHERE created_at >= ? AND created_at <= ?
		AND customer IS NOT NULL AND customer <> ''` + cc + `
		GROUP BY customer ORDER BY ` + order + ` LIMIT ` + fmt.Sprintf("%d", limit))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomerRow
	for rows.Next() {
		var n CustomerRow
		if err := rows.Scan(&n.Name, &n.Orders, &n.Revenue, &n.PaidOrders, &n.InTransitOrders, &n.CompletedOrders, &n.CanceledOrders); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func mergeCustomerRows(primary, extra []CustomerRow, limit int) []CustomerRow {
	seen := make(map[string]bool, len(primary)+len(extra))
	out := make([]CustomerRow, 0, limit)
	add := func(rows []CustomerRow) {
		for _, row := range rows {
			if seen[row.Name] {
				continue
			}
			seen[row.Name] = true
			out = append(out, row)
			if len(out) >= limit {
				return
			}
		}
	}
	add(primary)
	if len(out) < limit {
		add(extra)
	}
	return out
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
