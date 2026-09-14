package customeranalytics

import (
	"fmt"
	"strings"

	"github.com/clever/clever-dashboard/internal/db"
)

type Repository struct {
	db *db.DB
}

type orderEvent struct {
	CreatedAt string
	Customer  string
	Email     string
	Phone     string
	Paid      bool
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

func (r *Repository) bounds() (string, string, error) {
	var min, max *string
	err := r.db.QueryRow(`SELECT substr(MIN(created_at),1,10), substr(MAX(created_at),1,10)
		FROM orders WHERE created_at IS NOT NULL`).Scan(&min, &max)
	if err != nil {
		return "", "", err
	}
	if min == nil || max == nil {
		return "", "", nil
	}
	return *min, *max, nil
}

func (r *Repository) events(end string, f Filters) ([]orderEvent, error) {
	cond, args := filterCond(f)
	q := r.db.Rebind(`SELECT CAST(created_at AS TEXT), COALESCE(customer,''),
		COALESCE(email,''), COALESCE(phone,''), is_paid
		FROM orders
		WHERE created_at IS NOT NULL AND created_at <= ?` + cond + `
		ORDER BY created_at ASC, order_number ASC`)
	rows, err := r.db.Query(q, append([]interface{}{end + " 23:59:59"}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]orderEvent, 0)
	for rows.Next() {
		var event orderEvent
		if err := rows.Scan(&event.CreatedAt, &event.Customer, &event.Email, &event.Phone, &event.Paid); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func filterCond(f Filters) (string, []interface{}) {
	var sb strings.Builder
	var args []interface{}
	add := func(column, raw string) {
		values := splitCSV(raw)
		if len(values) == 0 {
			return
		}
		placeholders := make([]string, len(values))
		for i, value := range values {
			placeholders[i] = "?"
			args = append(args, value)
		}
		sb.WriteString(fmt.Sprintf(" AND %s IN (%s)", column, strings.Join(placeholders, ", ")))
	}
	add("city", f.City)
	add("region", f.Region)
	add("channel", f.Channel)
	add("payment_system", f.Payment)
	add("delivery_service", f.Delivery)
	add("coupon", f.Coupon)
	return sb.String(), args
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}
