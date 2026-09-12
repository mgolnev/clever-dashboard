package orders

import (
	"database/sql"
	"fmt"
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

// Replacement инкапсулирует атомарную замену витрины. Транзакция остаётся
// открытой, пока сервис потоково разбирает файл и передаёт сюда небольшие батчи.
type Replacement struct {
	repo      *Repository
	tx        *sql.Tx
	orderStmt *sql.Stmt
	itemStmt  *sql.Stmt
	importID  int64
	cleared   int
	items     int
	closed    bool
}

func (r *Repository) beginReplacement(filename string) (_ *Replacement, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var cleared int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&cleared); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`DELETE FROM order_items`); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`DELETE FROM orders`); err != nil {
		return nil, err
	}

	importID, err := r.createImport(tx, filename)
	if err != nil {
		return nil, err
	}
	orderStmt, err := tx.Prepare(r.db.Rebind(`INSERT INTO orders (
		order_number, created_at, updated_at, customer, email, phone,
		total_amount, refund_amount, delivery_cost, status_raw, status_stage, is_paid, is_canceled,
		payment_system, delivery_service, channel, coupon, region, city, location_raw,
		has_problem, problem_desc, cancel_reason, items_count, import_id
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`))
	if err != nil {
		return nil, err
	}
	itemStmt, err := tx.Prepare(r.db.Rebind(`INSERT INTO order_items (
		order_number, offer_id, name, qty, price, line_sum, brand, category, gender, size, import_id
	) VALUES (?,?,?,?,?,?,?,?,?,?,?)`))
	if err != nil {
		_ = orderStmt.Close()
		return nil, err
	}

	return &Replacement{
		repo: r, tx: tx, orderStmt: orderStmt, itemStmt: itemStmt,
		importID: importID, cleared: cleared,
	}, nil
}

func (r *Repository) createImport(tx *sql.Tx, filename string) (int64, error) {
	now := time.Now().Format(tsLayout)
	query := r.db.Rebind(`INSERT INTO raw_import (filename, source, rows_total, period_start, period_end, imported_at)
		VALUES (?, 'bitrix_file', 0, NULL, NULL, ?)`)
	if r.db.IsPostgres() {
		var id int64
		err := tx.QueryRow(query+` RETURNING id`, filename, now).Scan(&id)
		return id, err
	}
	result, err := tx.Exec(query, filename, now)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// SaveBatch сохраняет ограниченный набор заказов подготовленными выражениями.
func (r *Replacement) SaveBatch(orders []model.Order) error {
	if r.closed {
		return fmt.Errorf("транзакция импорта уже закрыта")
	}
	for _, order := range orders {
		if _, err := r.orderStmt.Exec(
			order.OrderNumber, nullTime(order.CreatedAt), nullTime(order.UpdatedAt), order.Customer, order.Email, order.Phone,
			order.TotalAmount, order.RefundAmount, order.DeliveryCost, order.StatusRaw, order.StatusStage, order.IsPaid, order.IsCanceled,
			order.PaymentSystem, order.DeliveryService, order.Channel, order.Coupon, order.Region, order.City, order.LocationRaw,
			order.HasProblem, order.ProblemDesc, order.CancelReason, len(order.Items), r.importID,
		); err != nil {
			return err
		}
		for _, item := range order.Items {
			if _, err := r.itemStmt.Exec(
				order.OrderNumber, item.OfferID, item.Name, item.Qty, item.Price, item.LineSum,
				item.Brand, item.Category, item.Gender, item.Size, r.importID,
			); err != nil {
				return err
			}
			r.items++
		}
	}
	return nil
}

func (r *Replacement) Complete(rowsTotal, ordersTotal int, start, end *time.Time) error {
	if r.closed {
		return fmt.Errorf("транзакция импорта уже закрыта")
	}
	_, err := r.tx.Exec(r.repo.db.Rebind(`UPDATE raw_import
		SET rows_total = ?, orders_imported = ?, items_imported = ?, period_start = ?, period_end = ?
		WHERE id = ?`), rowsTotal, ordersTotal, r.items, ptrTime(start), ptrTime(end), r.importID)
	if err != nil {
		return err
	}
	r.closeStatements()
	if err := r.tx.Commit(); err != nil {
		return err
	}
	r.closed = true
	return nil
}

func (r *Replacement) Rollback() {
	if r.closed {
		return
	}
	r.closeStatements()
	_ = r.tx.Rollback()
	r.closed = true
}

func (r *Replacement) closeStatements() {
	if r.orderStmt != nil {
		_ = r.orderStmt.Close()
	}
	if r.itemStmt != nil {
		_ = r.itemStmt.Close()
	}
}

func (r *Replacement) ImportID() int64 { return r.importID }
func (r *Replacement) Cleared() int    { return r.cleared }
func (r *Replacement) Items() int      { return r.items }

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
