package trafficsync

import (
	"database/sql"
	"fmt"
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
	err := r.db.QueryRow(r.db.Rebind(`SELECT MAX(day) FROM analytics_traffic_daily WHERE source = ?`), source).Scan(&day)
	if err != nil {
		return "", err
	}
	return day.String, nil
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

func (r *Repository) upsert(items []model.DailyTraffic) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now().Format(timestampLayout)
	q := r.db.Rebind(`INSERT INTO analytics_traffic_daily
		(day, channel, sessions, users, source, sampled, sample_share, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(day, channel, source) DO UPDATE SET
			sessions = excluded.sessions,
			users = excluded.users,
			sampled = excluded.sampled,
			sample_share = excluded.sample_share,
			synced_at = excluded.synced_at`)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range items {
		if _, err := tx.Exec(q, item.Day, item.Channel, item.Sessions, item.Users, item.Source,
			item.Sampled, item.SampleShare, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) latestRuns() (map[string]SourceStatus, error) {
	rows, err := r.db.Query(`SELECT source, date_from, date_to, status, rows_imported,
		COALESCE(error_text,''), started_at, finished_at
		FROM analytics_sync_runs ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]SourceStatus)
	for rows.Next() {
		var st SourceStatus
		var started any
		var finished any
		if err := rows.Scan(&st.Source, &st.DateFrom, &st.DateTo, &st.Status, &st.RowsImported,
			&st.Error, &started, &finished); err != nil {
			return nil, err
		}
		if _, exists := out[st.Source]; exists {
			continue
		}
		st.StartedAt = dbTimeString(started)
		st.FinishedAt = dbTimeString(finished)
		out[st.Source] = st
	}
	return out, rows.Err()
}

func dbTimeString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case time.Time:
		return t.Format(time.RFC3339)
	case []byte:
		return string(t)
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
