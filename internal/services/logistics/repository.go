package logistics

import (
	"fmt"
	"sort"
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

func splitCSV(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

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

// weekExpr — понедельник ISO-недели, содержащей дату (начало недели).
// Postgres: DATE_TRUNC('week') уже даёт понедельник. SQLite: 'weekday 0'
// сдвигает к ближайшему воскресенью (включительно), '-6 days' — к понедельнику
// той же недели; это корректно и для самого понедельника (в отличие от
// 'weekday 1','-7 days', который уводил понедельник в предыдущую неделю).
func weekExpr(d *db.DB, col string) string {
	if d.IsPostgres() {
		return fmt.Sprintf(`TO_CHAR(DATE_TRUNC('week', %s::timestamp), 'YYYY-MM-DD')`, col)
	}
	return fmt.Sprintf(`strftime('%%Y-%%m-%%d', %s, 'weekday 0', '-6 days')`, col)
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

func (r *Repository) summary(start, end string, f Filters) (Summary, error) {
	var s Summary
	cc, cargs := geoCond(f, "")
	q := r.db.Rebind(`SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN is_canceled = ` + falseVal(r.db) + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + trueVal(r.db) + ` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(delivery_cost),0),
		COALESCE(SUM(CASE WHEN delivery_cost = 0 THEN 1 ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?` + cc)
	err := r.db.QueryRow(q, append([]interface{}{start, end}, cargs...)...).Scan(
		&s.Orders, &s.Revenue, &s.PaidOrders, &s.DeliveryTotal, &s.FreeOrders)
	if err != nil {
		return s, err
	}
	if s.Orders > 0 {
		s.PaidRate = round2(float64(s.PaidOrders) / float64(s.Orders) * 100)
		s.AvgDelivery = s.DeliveryTotal / s.Orders
		s.FreeDeliveryRate = round2(float64(s.FreeOrders) / float64(s.Orders) * 100)
	}
	return s, nil
}

func (r *Repository) cohortSummary(start, end string, f Filters, pilotCities []string, isPilot bool) (Summary, error) {
	if len(pilotCities) == 0 {
		return Summary{}, nil
	}
	ph := make([]string, len(pilotCities))
	args := []interface{}{start, end}
	for i, c := range pilotCities {
		ph[i] = "?"
		args = append(args, c)
	}
	inClause := "city IN (" + strings.Join(ph, ", ") + ")"
	if !isPilot {
		inClause = "city NOT IN (" + strings.Join(ph, ", ") + ")"
	}
	// Когорты сами разбивают города (пилот / контроль), поэтому фильтр по
	// конкретному городу здесь игнорируем; остальные фильтры применяем.
	cohortFilters := f
	cohortFilters.City = ""
	var regionCond string
	if rc, rargs := geoCond(cohortFilters, ""); rc != "" {
		regionCond = rc
		args = append(args, rargs...)
	}
	q := r.db.Rebind(`SELECT
		COUNT(*),
		COALESCE(SUM(CASE WHEN is_canceled = ` + falseVal(r.db) + ` THEN total_amount ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN is_paid = ` + trueVal(r.db) + ` THEN 1 ELSE 0 END),0),
		COALESCE(SUM(delivery_cost),0),
		COALESCE(SUM(CASE WHEN delivery_cost = 0 THEN 1 ELSE 0 END),0)
		FROM orders WHERE created_at >= ? AND created_at <= ?
		AND ` + inClause + ` AND city IS NOT NULL AND city <> ''` + regionCond)
	var s Summary
	err := r.db.QueryRow(q, args...).Scan(
		&s.Orders, &s.Revenue, &s.PaidOrders, &s.DeliveryTotal, &s.FreeOrders)
	if err != nil {
		return s, err
	}
	if s.Orders > 0 {
		s.PaidRate = round2(float64(s.PaidOrders) / float64(s.Orders) * 100)
		s.AvgDelivery = s.DeliveryTotal / s.Orders
		s.FreeDeliveryRate = round2(float64(s.FreeOrders) / float64(s.Orders) * 100)
	}
	return s, nil
}

func (r *Repository) byService(start, end string, f Filters, limit int) ([]ServiceRow, error) {
	cc, cargs := geoCond(f, "")
	q := r.db.Rebind(fmt.Sprintf(`SELECT COALESCE(NULLIF(delivery_service,''),'—') AS label,
		COUNT(*) AS orders,
		COALESCE(SUM(CASE WHEN is_canceled = `+falseVal(r.db)+` THEN total_amount ELSE 0 END),0) AS revenue,
		COALESCE(SUM(CASE WHEN is_paid = `+trueVal(r.db)+` THEN 1 ELSE 0 END),0) AS paid,
		COALESCE(SUM(delivery_cost),0) AS delivery_total,
		COALESCE(SUM(CASE WHEN delivery_cost = 0 THEN 1 ELSE 0 END),0) AS free_orders
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc+`
		GROUP BY COALESCE(NULLIF(delivery_service,''),'—')
		ORDER BY orders DESC LIMIT %d`, limit))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ServiceRow
	paidByRow := make([]int, 0)
	var totalOrders int
	for rows.Next() {
		var row ServiceRow
		var paid int
		if err := rows.Scan(&row.Name, &row.Orders, &row.Revenue, &paid, &row.DeliveryTotal, &row.FreeOrders); err != nil {
			return nil, err
		}
		totalOrders += row.Orders
		paidByRow = append(paidByRow, paid)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if totalOrders > 0 {
			out[i].Share = round2(float64(out[i].Orders) / float64(totalOrders) * 100)
		}
		out[i].PaidOrders = paidByRow[i]
		if out[i].Orders > 0 {
			out[i].PaidRate = round2(float64(paidByRow[i]) / float64(out[i].Orders) * 100)
			out[i].AvgDelivery = out[i].DeliveryTotal / out[i].Orders
			out[i].FreeDeliveryRate = round2(float64(out[i].FreeOrders) / float64(out[i].Orders) * 100)
		}
	}
	return out, nil
}

func (r *Repository) byCity(start, end string, f Filters, pilotSet map[string]bool, limit int) ([]CityRow, error) {
	cc, cargs := geoCond(f, "")
	q := r.db.Rebind(fmt.Sprintf(`SELECT COALESCE(NULLIF(city,''),'—') AS label,
		COUNT(*) AS orders,
		COALESCE(SUM(CASE WHEN is_canceled = `+falseVal(r.db)+` THEN total_amount ELSE 0 END),0) AS revenue,
		COALESCE(SUM(CASE WHEN is_paid = `+trueVal(r.db)+` THEN 1 ELSE 0 END),0) AS paid,
		COALESCE(SUM(delivery_cost),0) AS delivery_total,
		COALESCE(SUM(CASE WHEN delivery_cost = 0 THEN 1 ELSE 0 END),0) AS free_orders
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc+`
		GROUP BY COALESCE(NULLIF(city,''),'—')
		ORDER BY orders DESC LIMIT %d`, limit))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CityRow
	var totalOrders int
	for rows.Next() {
		var row CityRow
		var paid, freeOrders int
		if err := rows.Scan(&row.Name, &row.Orders, &row.Revenue, &paid, &row.DeliveryTotal, &freeOrders); err != nil {
			return nil, err
		}
		row.PaidOrders = paid
		row.FreeOrders = freeOrders
		if row.Orders > 0 {
			row.PaidRate = round2(float64(paid) / float64(row.Orders) * 100)
			row.AvgDelivery = row.DeliveryTotal / row.Orders
			row.FreeDeliveryRate = round2(float64(freeOrders) / float64(row.Orders) * 100)
		}
		row.IsPilot = pilotSet[row.Name]
		totalOrders += row.Orders
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if totalOrders > 0 {
			out[i].Share = round2(float64(out[i].Orders) / float64(totalOrders) * 100)
		}
	}
	return out, nil
}

// seriesBreakdown — недельная динамика, сгруппированная по значению измерения
// groupCol (city/delivery_service/payment_system/channel). Возвращает топ-limit
// значений по числу заказов; Points каждого значения выровнены по общему
// отсортированному списку недель (пропуски заполняются нулями).
func (r *Repository) seriesBreakdown(start, end string, f Filters, groupCol string, limit int) ([]SeriesGroup, []string, error) {
	cc, cargs := geoCond(f, "")
	wk := weekExpr(r.db, "created_at")
	gexpr := fmt.Sprintf("COALESCE(NULLIF(%s,''),'—')", groupCol)
	q := r.db.Rebind(fmt.Sprintf(`SELECT %[1]s AS week, %[2]s AS grp,
		COUNT(*) AS orders,
		COALESCE(SUM(CASE WHEN is_canceled = `+falseVal(r.db)+` THEN 1 ELSE 0 END),0) AS net_orders,
		COALESCE(SUM(CASE WHEN is_paid = `+trueVal(r.db)+` THEN 1 ELSE 0 END),0) AS paid,
		COALESCE(SUM(CASE WHEN is_canceled = `+falseVal(r.db)+` THEN total_amount ELSE 0 END),0) AS revenue,
		COALESCE(SUM(delivery_cost),0) AS delivery_total,
		COALESCE(SUM(CASE WHEN delivery_cost = 0 THEN 1 ELSE 0 END),0) AS free_orders
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc+`
		GROUP BY week, grp ORDER BY week ASC`, wk, gexpr))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	data := map[string]map[string]*WeekPoint{} // grp -> week -> point
	totals := map[string]int{}
	weekSet := map[string]bool{}
	for rows.Next() {
		var week, grp string
		var p WeekPoint
		var freeOrders int
		if err := rows.Scan(&week, &grp, &p.Orders, &p.NetOrders, &p.PaidOrders, &p.Revenue, &p.DeliveryTotal, &freeOrders); err != nil {
			return nil, nil, err
		}
		p.Week = week
		if p.Orders > 0 {
			p.PaidRate = round2(float64(p.PaidOrders) / float64(p.Orders) * 100)
			p.AvgDelivery = p.DeliveryTotal / p.Orders
			p.FreeDeliveryRate = round2(float64(freeOrders) / float64(p.Orders) * 100)
		}
		if p.NetOrders > 0 {
			p.AOV = p.Revenue / p.NetOrders
		}
		if data[grp] == nil {
			data[grp] = map[string]*WeekPoint{}
		}
		pc := p
		data[grp][week] = &pc
		totals[grp] += p.Orders
		weekSet[week] = true
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Товары и выручка позиций (по не отменённым) — для ASP/UPT, в том же разрезе.
	oc, ocargs := geoCond(f, "o.")
	wkItems := weekExpr(r.db, "o.created_at")
	gexprO := fmt.Sprintf("COALESCE(NULLIF(o.%s,''),'—')", groupCol)
	iq := r.db.Rebind(fmt.Sprintf(`SELECT %[1]s AS week, %[2]s AS grp,
		COALESCE(SUM(oi.qty),0) AS units,
		COALESCE(SUM(oi.line_sum),0) AS item_revenue
		FROM order_items oi JOIN orders o ON o.order_number = oi.order_number
		WHERE o.created_at >= ? AND o.created_at <= ? AND o.is_canceled = `+falseVal(r.db)+oc+`
		GROUP BY week, grp`, wkItems, gexprO))
	irows, err := r.db.Query(iq, append([]interface{}{start, end}, ocargs...)...)
	if err != nil {
		return nil, nil, err
	}
	defer irows.Close()
	for irows.Next() {
		var week, grp string
		var units, itemRevenue int
		if err := irows.Scan(&week, &grp, &units, &itemRevenue); err != nil {
			return nil, nil, err
		}
		wk, ok := data[grp]
		if !ok {
			continue
		}
		p, ok := wk[week]
		if !ok {
			continue
		}
		p.Units = units
		if units > 0 {
			p.ASP = itemRevenue / units
		}
		if p.NetOrders > 0 {
			p.UPT = round2(float64(units) / float64(p.NetOrders))
		}
	}
	if err := irows.Err(); err != nil {
		return nil, nil, err
	}

	// Топ значений измерения по числу заказов.
	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if totals[names[i]] != totals[names[j]] {
			return totals[names[i]] > totals[names[j]]
		}
		return names[i] < names[j]
	})
	if limit > 0 && len(names) > limit {
		names = names[:limit]
	}

	weeks := make([]string, 0, len(weekSet))
	for w := range weekSet {
		weeks = append(weeks, w)
	}
	sort.Strings(weeks)

	groups := make([]SeriesGroup, 0, len(names))
	for _, name := range names {
		pts := make([]WeekPoint, len(weeks))
		for i, w := range weeks {
			if p, ok := data[name][w]; ok {
				pts[i] = *p
			} else {
				pts[i] = WeekPoint{Week: w}
			}
		}
		groups = append(groups, SeriesGroup{Name: name, Points: pts})
	}
	return groups, weeks, nil
}

func (r *Repository) series(start, end string, f Filters) ([]WeekPoint, error) {
	cc, cargs := geoCond(f, "")
	wk := weekExpr(r.db, "created_at")
	q := r.db.Rebind(fmt.Sprintf(`SELECT %s AS week,
		COUNT(*) AS orders,
		COALESCE(SUM(CASE WHEN is_canceled = `+falseVal(r.db)+` THEN 1 ELSE 0 END),0) AS net_orders,
		COALESCE(SUM(CASE WHEN is_paid = `+trueVal(r.db)+` THEN 1 ELSE 0 END),0) AS paid,
		COALESCE(SUM(CASE WHEN is_canceled = `+falseVal(r.db)+` THEN total_amount ELSE 0 END),0) AS revenue,
		COALESCE(SUM(delivery_cost),0) AS delivery_total,
		COALESCE(SUM(CASE WHEN delivery_cost = 0 THEN 1 ELSE 0 END),0) AS free_orders
		FROM orders WHERE created_at >= ? AND created_at <= ?`+cc+`
		GROUP BY week ORDER BY week ASC`, wk))
	rows, err := r.db.Query(q, append([]interface{}{start, end}, cargs...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WeekPoint
	idx := map[string]int{}
	for rows.Next() {
		var p WeekPoint
		var freeOrders int
		if err := rows.Scan(&p.Week, &p.Orders, &p.NetOrders, &p.PaidOrders, &p.Revenue, &p.DeliveryTotal, &freeOrders); err != nil {
			return nil, err
		}
		if p.Orders > 0 {
			p.PaidRate = round2(float64(p.PaidOrders) / float64(p.Orders) * 100)
			p.AvgDelivery = p.DeliveryTotal / p.Orders
			p.FreeDeliveryRate = round2(float64(freeOrders) / float64(p.Orders) * 100)
		}
		if p.NetOrders > 0 {
			p.AOV = p.Revenue / p.NetOrders
		}
		idx[p.Week] = len(out)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Товары и выручка позиций (по не отменённым) — для ASP/UPT.
	oc, ocargs := geoCond(f, "o.")
	wkItems := weekExpr(r.db, "o.created_at")
	iq := r.db.Rebind(fmt.Sprintf(`SELECT %s AS week,
		COALESCE(SUM(oi.qty),0) AS units,
		COALESCE(SUM(oi.line_sum),0) AS item_revenue
		FROM order_items oi JOIN orders o ON o.order_number = oi.order_number
		WHERE o.created_at >= ? AND o.created_at <= ? AND o.is_canceled = `+falseVal(r.db)+oc+`
		GROUP BY week`, wkItems))
	irows, err := r.db.Query(iq, append([]interface{}{start, end}, ocargs...)...)
	if err != nil {
		return nil, err
	}
	defer irows.Close()
	for irows.Next() {
		var week string
		var units, itemRevenue int
		if err := irows.Scan(&week, &units, &itemRevenue); err != nil {
			return nil, err
		}
		i, ok := idx[week]
		if !ok {
			continue
		}
		out[i].Units = units
		if units > 0 {
			out[i].ASP = itemRevenue / units
		}
		if out[i].NetOrders > 0 {
			out[i].UPT = round2(float64(units) / float64(out[i].NetOrders))
		}
	}
	return out, irows.Err()
}
