// Package ordercleanup выполняет адресную очистку старых заказов с резервной копией.
package ordercleanup

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/clever/clever-dashboard/internal/db"
)

type Result struct {
	OrderNumbers []string
	ItemsDeleted int64
	BackupPath   string
}

// PurgeBefore удаляет заказы до cutoff только при точном совпадении ожидаемого
// числа строк. SQLite-копия создаётся до транзакции удаления.
func PurgeBefore(d *db.DB, dsn string, cutoff time.Time, expected int) (*Result, error) {
	if d.IsPostgres() {
		return nil, fmt.Errorf("очистка поддерживается только для SQLite")
	}
	if cutoff.IsZero() || expected <= 0 {
		return nil, fmt.Errorf("нужны дата отсечения и ожидаемое число заказов")
	}
	cutoffDay := cutoff.Format("2006-01-02")
	before, err := candidates(d.DB, cutoffDay)
	if err != nil {
		return nil, err
	}
	if len(before) == 0 {
		return &Result{}, nil
	}
	if len(before) != expected {
		return nil, fmt.Errorf("найдено %d заказов до %s, ожидалось %d; ничего не удалено", len(before), cutoffDay, expected)
	}
	var dataVersion int64
	if err := d.QueryRow(`PRAGMA data_version`).Scan(&dataVersion); err != nil {
		return nil, err
	}

	backupPath := filepath.Join(filepath.Dir(dsn),
		fmt.Sprintf("clever-before-order-cleanup-%s-%s.db", cutoff.Format("20060102"), time.Now().UTC().Format("20060102T150405.000000000Z")))
	if _, err := d.Exec(`VACUUM INTO ?`, backupPath); err != nil {
		return nil, fmt.Errorf("создать резервную копию: %w", err)
	}
	if err := os.Chmod(backupPath, 0o600); err != nil {
		return nil, fmt.Errorf("права резервной копии %s: %w", backupPath, err)
	}
	backup, err := sql.Open("sqlite", backupPath)
	if err != nil {
		return nil, fmt.Errorf("открыть резервную копию: %w", err)
	}
	var integrity string
	checkErr := backup.QueryRow(`PRAGMA integrity_check`).Scan(&integrity)
	backupCandidates, listErr := candidates(backup, cutoffDay)
	_ = backup.Close()
	if checkErr != nil || integrity != "ok" || listErr != nil || !reflect.DeepEqual(before, backupCandidates) {
		return nil, fmt.Errorf("резервная копия %s не прошла проверку целостности или состава заказов", backupPath)
	}

	tx, err := d.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var currentVersion int64
	if err := tx.QueryRow(`PRAGMA data_version`).Scan(&currentVersion); err != nil {
		return nil, err
	}
	if currentVersion != dataVersion {
		return nil, fmt.Errorf("база изменилась во время резервного копирования; ничего не удалено")
	}
	current, err := candidates(tx, cutoffDay)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(before, current) {
		return nil, fmt.Errorf("список старых заказов изменился после резервного копирования; ничего не удалено")
	}
	itemsResult, err := tx.Exec(`DELETE FROM order_items WHERE order_number IN (
		SELECT order_number FROM orders WHERE created_at < ?)`, cutoffDay)
	if err != nil {
		return nil, err
	}
	itemsDeleted, err := itemsResult.RowsAffected()
	if err != nil {
		return nil, err
	}
	ordersResult, err := tx.Exec(`DELETE FROM orders WHERE created_at < ?`, cutoffDay)
	if err != nil {
		return nil, err
	}
	ordersDeleted, err := ordersResult.RowsAffected()
	if err != nil {
		return nil, err
	}
	if ordersDeleted != int64(expected) {
		return nil, fmt.Errorf("удалено %d заказов, ожидалось %d; транзакция отменена", ordersDeleted, expected)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	log.Printf("order cleanup: removed %d orders %v and %d items; backup %s", ordersDeleted, before, itemsDeleted, backupPath)
	return &Result{OrderNumbers: before, ItemsDeleted: itemsDeleted, BackupPath: backupPath}, nil
}

type queryer interface {
	Query(string, ...any) (*sql.Rows, error)
}

func candidates(q queryer, cutoff string) ([]string, error) {
	rows, err := q.Query(`SELECT order_number FROM orders WHERE created_at < ? ORDER BY created_at, order_number`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var number string
		if err := rows.Scan(&number); err != nil {
			return nil, err
		}
		out = append(out, number)
	}
	return out, rows.Err()
}
