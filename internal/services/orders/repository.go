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

// Merge инкапсулирует атомарное накопительное обновление витрины. Отсутствующие
// в файле заказы не затрагиваются, а совпавшие обновляются вместе с позициями.
type Merge struct {
	repo            *Repository
	tx              *sql.Tx
	insertOrderStmt *sql.Stmt
	updateOrderStmt *sql.Stmt
	deleteItemsStmt *sql.Stmt
	insertItemStmt  *sql.Stmt
	importID        int64
	added           int
	updated         int
	skipped         int
	items           int
	closed          bool
}

func (r *Repository) beginMerge(filename string) (_ *Merge, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	importID, err := r.createImport(tx, filename)
	if err != nil {
		return nil, err
	}
	insertOrderStmt, err := tx.Prepare(r.db.Rebind(`INSERT INTO orders (
		order_number, created_at, updated_at, customer, email, phone,
		total_amount, refund_amount, delivery_cost, status_raw, status_stage, is_paid, is_canceled,
		payment_system, delivery_service, channel, coupon, region, city, location_raw,
		has_problem, problem_desc, cancel_reason, items_count, import_id
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(order_number) DO NOTHING`))
	if err != nil {
		return nil, err
	}
	updateOrderStmt, err := tx.Prepare(r.db.Rebind(`UPDATE orders SET
		created_at = COALESCE(?, created_at), updated_at = COALESCE(?, updated_at),
		customer = ?, email = ?, phone = ?, total_amount = ?, refund_amount = ?, delivery_cost = ?,
		status_raw = ?, status_stage = ?, is_paid = ?, is_canceled = ?, payment_system = ?,
		delivery_service = ?, channel = ?, coupon = ?, region = ?, city = ?, location_raw = ?,
		has_problem = ?, problem_desc = ?, cancel_reason = ?, items_count = ?, import_id = ?
	WHERE order_number = ? AND (updated_at IS NULL OR COALESCE(?, updated_at) >= updated_at)`))
	if err != nil {
		_ = insertOrderStmt.Close()
		return nil, err
	}
	deleteItemsStmt, err := tx.Prepare(r.db.Rebind(`DELETE FROM order_items WHERE order_number = ?`))
	if err != nil {
		_ = insertOrderStmt.Close()
		_ = updateOrderStmt.Close()
		return nil, err
	}
	insertItemStmt, err := tx.Prepare(r.db.Rebind(`INSERT INTO order_items (
		order_number, offer_id, name, qty, price, line_sum, brand, category, gender, size, import_id
	) VALUES (?,?,?,?,?,?,?,?,?,?,?)`))
	if err != nil {
		_ = insertOrderStmt.Close()
		_ = updateOrderStmt.Close()
		_ = deleteItemsStmt.Close()
		return nil, err
	}

	return &Merge{
		repo: r, tx: tx, insertOrderStmt: insertOrderStmt, updateOrderStmt: updateOrderStmt,
		deleteItemsStmt: deleteItemsStmt, insertItemStmt: insertItemStmt, importID: importID,
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

// SaveBatch добавляет новые заказы и обновляет существующие, если версия из
// файла не старше сохранённой. Позиции принятого заказа заменяются целиком.
func (m *Merge) SaveBatch(orders []model.Order) error {
	if m.closed {
		return fmt.Errorf("транзакция импорта уже закрыта")
	}
	for _, order := range orders {
		createdAt := nullTime(order.CreatedAt)
		updatedAt := nullTime(order.UpdatedAt)
		values := []any{
			createdAt, updatedAt, order.Customer, order.Email, order.Phone,
			order.TotalAmount, order.RefundAmount, order.DeliveryCost, order.StatusRaw, order.StatusStage,
			order.IsPaid, order.IsCanceled, order.PaymentSystem, order.DeliveryService, order.Channel,
			order.Coupon, order.Region, order.City, order.LocationRaw, order.HasProblem,
			order.ProblemDesc, order.CancelReason, len(order.Items), m.importID,
		}

		insertResult, err := m.insertOrderStmt.Exec(append([]any{order.OrderNumber}, values...)...)
		if err != nil {
			return err
		}
		inserted, err := insertResult.RowsAffected()
		if err != nil {
			return err
		}
		if inserted > 0 {
			m.added++
		} else {
			updateArgs := append(values, order.OrderNumber, updatedAt)
			updateResult, updateErr := m.updateOrderStmt.Exec(updateArgs...)
			if updateErr != nil {
				return updateErr
			}
			updated, rowsErr := updateResult.RowsAffected()
			if rowsErr != nil {
				return rowsErr
			}
			if updated == 0 {
				m.skipped++
				continue
			}
			m.updated++
		}

		if _, err := m.deleteItemsStmt.Exec(order.OrderNumber); err != nil {
			return err
		}
		for _, item := range order.Items {
			if _, err := m.insertItemStmt.Exec(
				order.OrderNumber, item.OfferID, item.Name, item.Qty, item.Price, item.LineSum,
				item.Brand, item.Category, item.Gender, item.Size, m.importID,
			); err != nil {
				return err
			}
			m.items++
		}
	}
	return nil
}

func (m *Merge) Complete(rowsTotal int, start, end *time.Time) error {
	if m.closed {
		return fmt.Errorf("транзакция импорта уже закрыта")
	}
	_, err := m.tx.Exec(m.repo.db.Rebind(`UPDATE raw_import SET
		rows_total = ?, orders_imported = ?, orders_added = ?, orders_updated = ?,
		orders_skipped = ?, items_imported = ?, period_start = ?, period_end = ?
		WHERE id = ?`), rowsTotal, m.added+m.updated, m.added, m.updated, m.skipped,
		m.items, ptrTime(start), ptrTime(end), m.importID)
	if err != nil {
		return err
	}
	m.closeStatements()
	if err := m.tx.Commit(); err != nil {
		return err
	}
	m.closed = true
	return nil
}

func (m *Merge) Rollback() {
	if m.closed {
		return
	}
	m.closeStatements()
	_ = m.tx.Rollback()
	m.closed = true
}

func (m *Merge) closeStatements() {
	for _, statement := range []*sql.Stmt{m.insertOrderStmt, m.updateOrderStmt, m.deleteItemsStmt, m.insertItemStmt} {
		if statement != nil {
			_ = statement.Close()
		}
	}
}

func (m *Merge) ImportID() int64 { return m.importID }
func (m *Merge) Added() int      { return m.added }
func (m *Merge) Updated() int    { return m.updated }
func (m *Merge) Skipped() int    { return m.skipped }
func (m *Merge) Items() int      { return m.items }

func ptrTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.Format(tsLayout)
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(tsLayout)
}
