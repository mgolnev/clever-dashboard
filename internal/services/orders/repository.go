package orders

import (
	"database/sql"
	"time"

	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/model"
)

const tsLayout = "2006-01-02 15:04:05"

// Repository — доступ к данным заказов и позиций.
type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

func (r *Repository) createImport(filename string, rows int, start, end *time.Time) (int64, error) {
	now := time.Now().Format(tsLayout)
	if r.db.IsPostgres() {
		var id int64
		err := r.db.QueryRow(r.db.Rebind(`INSERT INTO raw_import (filename, source, rows_total, period_start, period_end, imported_at)
			VALUES (?, 'bitrix_file', ?, ?, ?, ?) RETURNING id`),
			filename, rows, ptrTime(start), ptrTime(end), now).Scan(&id)
		return id, err
	}
	res, err := r.db.Exec(r.db.Rebind(`INSERT INTO raw_import (filename, source, rows_total, period_start, period_end, imported_at)
		VALUES (?, 'bitrix_file', ?, ?, ?, ?)`), filename, rows, ptrTime(start), ptrTime(end), now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) updateImportStats(id int64, ordersN, itemsN int) error {
	_, err := r.db.Exec(r.db.Rebind(`UPDATE raw_import SET orders_imported = ?, items_imported = ? WHERE id = ?`),
		ordersN, itemsN, id)
	return err
}

// saveOrders выполняет идемпотентный upsert заказов и их позиций в транзакции.
func (r *Repository) saveOrders(orders []model.Order, importID int64) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	upsert := r.db.Rebind(`INSERT INTO orders (
		order_number, created_at, updated_at, customer, email, phone,
		total_amount, delivery_cost, status_raw, status_stage, is_paid, is_canceled,
		payment_system, delivery_service, channel, coupon, region, city, location_raw,
		has_problem, problem_desc, cancel_reason, items_count, import_id
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(order_number) DO UPDATE SET
		created_at=excluded.created_at, updated_at=excluded.updated_at,
		customer=excluded.customer, email=excluded.email, phone=excluded.phone,
		total_amount=excluded.total_amount, delivery_cost=excluded.delivery_cost,
		status_raw=excluded.status_raw, status_stage=excluded.status_stage,
		is_paid=excluded.is_paid, is_canceled=excluded.is_canceled,
		payment_system=excluded.payment_system, delivery_service=excluded.delivery_service,
		channel=excluded.channel, coupon=excluded.coupon, region=excluded.region, city=excluded.city,
		location_raw=excluded.location_raw, has_problem=excluded.has_problem,
		problem_desc=excluded.problem_desc, cancel_reason=excluded.cancel_reason,
		items_count=excluded.items_count, import_id=excluded.import_id`)

	delItems := r.db.Rebind(`DELETE FROM order_items WHERE order_number = ?`)
	insItem := r.db.Rebind(`INSERT INTO order_items (
		order_number, offer_id, name, qty, price, line_sum, brand, category, gender, size, import_id
	) VALUES (?,?,?,?,?,?,?,?,?,?,?)`)

	itemsTotal := 0
	for _, o := range orders {
		if _, err := tx.Exec(upsert,
			o.OrderNumber, nullTime(o.CreatedAt), nullTime(o.UpdatedAt), o.Customer, o.Email, o.Phone,
			o.TotalAmount, o.DeliveryCost, o.StatusRaw, o.StatusStage, o.IsPaid, o.IsCanceled,
			o.PaymentSystem, o.DeliveryService, o.Channel, o.Coupon, o.Region, o.City, o.LocationRaw,
			o.HasProblem, o.ProblemDesc, o.CancelReason, len(o.Items), importID,
		); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(delItems, o.OrderNumber); err != nil {
			return 0, err
		}
		for _, it := range o.Items {
			if _, err := tx.Exec(insItem,
				o.OrderNumber, it.OfferID, it.Name, it.Qty, it.Price, it.LineSum,
				it.Brand, it.Category, it.Gender, it.Size, importID,
			); err != nil {
				return 0, err
			}
			itemsTotal++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return itemsTotal, nil
}

func ptrTime(t *time.Time) interface{} {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.Format(tsLayout)
}

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.Format(tsLayout)
}

var _ = sql.ErrNoRows
