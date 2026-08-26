package ecomsync

import (
	"database/sql"
	"time"

	"github.com/clever/clever-dashboard/internal/db"
	"github.com/clever/clever-dashboard/internal/model"
)

const timestampLayout = "2006-01-02 15:04:05"

type Repository struct {
	db *db.DB
}

func NewRepository(d *db.DB) *Repository { return &Repository{db: d} }

func (r *Repository) latestDataDay(source string) (string, error) {
	var day sql.NullString
	err := r.db.QueryRow(r.db.Rebind(`SELECT MAX(day) FROM analytics_ecommerce_daily WHERE source = ?`), source).Scan(&day)
	return day.String, err
}

func (r *Repository) sourceRevision(source string) (string, error) {
	var revision string
	err := r.db.QueryRow(r.db.Rebind(`SELECT revision FROM analytics_source_revisions WHERE source = ?`), source).Scan(&revision)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return revision, err
}

func (r *Repository) setSourceRevision(source, revision string) error {
	_, err := r.db.Exec(r.db.Rebind(`INSERT INTO analytics_source_revisions (source, revision, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(source) DO UPDATE SET revision = excluded.revision, updated_at = excluded.updated_at`),
		source, revision, time.Now().Format(timestampLayout))
	return err
}

func (r *Repository) startRun(source, from, to string) (int64, error) {
	now := time.Now().Format(timestampLayout)
	if r.db.IsPostgres() {
		var id int64
		err := r.db.QueryRow(r.db.Rebind(`INSERT INTO analytics_sync_runs
			(source, date_from, date_to, status, started_at) VALUES (?, ?, ?, 'running', ?) RETURNING id`),
			source, from, to, now).Scan(&id)
		return id, err
	}
	res, err := r.db.Exec(`INSERT INTO analytics_sync_runs
		(source, date_from, date_to, status, started_at) VALUES (?, ?, ?, 'running', ?)`,
		source, from, to, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) finishRun(id int64, status string, rows int, syncErr error) error {
	errText := ""
	if syncErr != nil {
		errText = syncErr.Error()
	}
	_, err := r.db.Exec(r.db.Rebind(`UPDATE analytics_sync_runs
		SET status = ?, rows_imported = ?, error_text = ?, finished_at = ? WHERE id = ?`),
		status, rows, errText, time.Now().Format(timestampLayout), id)
	return err
}

func (r *Repository) upsert(items []model.DailyEcommerce) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now().Format(timestampLayout)
	q := r.db.Rebind(`INSERT INTO analytics_ecommerce_daily
		(day, channel, product_view_users, add_to_cart_users, begin_checkout_users,
		 tracked_purchase_users, source, sampled, sample_share, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(day, channel, source) DO UPDATE SET
			product_view_users = excluded.product_view_users,
			add_to_cart_users = excluded.add_to_cart_users,
			begin_checkout_users = excluded.begin_checkout_users,
			tracked_purchase_users = excluded.tracked_purchase_users,
			sampled = excluded.sampled,
			sample_share = excluded.sample_share,
			synced_at = excluded.synced_at`)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		if _, err := tx.Exec(q, item.Day, item.Channel, item.ProductViewUsers, item.AddToCartUsers,
			item.BeginCheckoutUsers, item.TrackedPurchaseUsers, item.Source, item.Sampled,
			item.SampleShare, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
