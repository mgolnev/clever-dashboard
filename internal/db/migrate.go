package db

import (
	"fmt"
	"strings"
)

// Migrate создаёт схему БД. DDL написан в переносимом виде; различия диалектов
// (автоинкремент, типы времени) разрешаются через IsPostgres().
func (d *DB) Migrate() error {
	pkAuto := "INTEGER PRIMARY KEY AUTOINCREMENT"
	tsType := "TEXT"
	boolType := "INTEGER"
	boolDefault := "0"
	if d.IsPostgres() {
		pkAuto = "BIGSERIAL PRIMARY KEY"
		tsType = "TIMESTAMP"
		boolType = "BOOLEAN"
		boolDefault = "FALSE"
	}

	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS raw_import (
			id %s,
			filename TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'bitrix_file',
			rows_total INTEGER NOT NULL DEFAULT 0,
			orders_imported INTEGER NOT NULL DEFAULT 0,
			items_imported INTEGER NOT NULL DEFAULT 0,
			period_start %s,
			period_end %s,
			imported_at %s NOT NULL
		)`, pkAuto, tsType, tsType, tsType),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS orders (
			order_number TEXT PRIMARY KEY,
			created_at %s,
			updated_at %s,
			customer TEXT,
			email TEXT,
			phone TEXT,
			total_amount INTEGER NOT NULL DEFAULT 0,
			refund_amount INTEGER NOT NULL DEFAULT 0,
			delivery_cost INTEGER NOT NULL DEFAULT 0,
			status_raw TEXT,
			status_stage TEXT,
			is_paid %s NOT NULL DEFAULT %s,
			is_canceled %s NOT NULL DEFAULT %s,
			payment_system TEXT,
			delivery_service TEXT,
			channel TEXT,
			coupon TEXT,
			region TEXT,
			city TEXT,
			location_raw TEXT,
			has_problem %s NOT NULL DEFAULT %s,
			problem_desc TEXT,
			cancel_reason TEXT,
			items_count INTEGER NOT NULL DEFAULT 0,
			import_id INTEGER
		)`, tsType, tsType, boolType, boolDefault, boolType, boolDefault, boolType, boolDefault),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS order_items (
			id %s,
			order_number TEXT NOT NULL,
			offer_id TEXT,
			name TEXT,
			qty INTEGER NOT NULL DEFAULT 1,
			price INTEGER NOT NULL DEFAULT 0,
			line_sum INTEGER NOT NULL DEFAULT 0,
			brand TEXT,
			category TEXT,
			gender TEXT,
			size TEXT,
			import_id INTEGER
		)`, pkAuto),

		`CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_status_stage ON orders(status_stage)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_channel ON orders(channel)`,
		`CREATE INDEX IF NOT EXISTS idx_items_order_number ON order_items(order_number)`,
		`CREATE INDEX IF NOT EXISTS idx_items_category ON order_items(category)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS sales_plan (
			id %s,
			year INTEGER NOT NULL,
			month INTEGER NOT NULL,
			channel TEXT NOT NULL,
			net_target INTEGER NOT NULL DEFAULT 0,
			updated_at %s NOT NULL
		)`, pkAuto, tsType),

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_sales_plan_uniq ON sales_plan(year, month, channel)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS traffic (
			id %s,
			year INTEGER NOT NULL,
			month INTEGER NOT NULL,
			channel TEXT NOT NULL,
			visits INTEGER NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT 'manual',
			updated_at %s NOT NULL
		)`, pkAuto, tsType),

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_traffic_uniq ON traffic(year, month, channel, source)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS analytics_traffic_daily (
			id %s,
			day TEXT NOT NULL,
			channel TEXT NOT NULL,
			sessions INTEGER NOT NULL DEFAULT 0,
			users INTEGER NOT NULL DEFAULT 0,
			source TEXT NOT NULL,
			sampled %s NOT NULL DEFAULT %s,
			sample_share REAL NOT NULL DEFAULT 1,
			synced_at %s NOT NULL
		)`, pkAuto, boolType, boolDefault, tsType),

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_analytics_traffic_daily_uniq
			ON analytics_traffic_daily(day, channel, source)`,
		`CREATE INDEX IF NOT EXISTS idx_analytics_traffic_daily_day
			ON analytics_traffic_daily(day)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS analytics_sync_runs (
			id %s,
			source TEXT NOT NULL,
			date_from TEXT NOT NULL,
			date_to TEXT NOT NULL,
			status TEXT NOT NULL,
			rows_imported INTEGER NOT NULL DEFAULT 0,
			error_text TEXT,
			started_at %s NOT NULL,
			finished_at %s
		)`, pkAuto, tsType, tsType),

		`CREATE INDEX IF NOT EXISTS idx_analytics_sync_runs_source
			ON analytics_sync_runs(source, id)`,
	}

	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w\nstmt: %s", err, s)
		}
	}

	// Идемпотентные ALTER для БД, созданных до добавления полей. Ошибка
	// "duplicate column" игнорируется.
	for _, alter := range []string{
		fmt.Sprintf("ALTER TABLE orders ADD COLUMN has_problem %s NOT NULL DEFAULT %s", boolType, boolDefault),
		"ALTER TABLE orders ADD COLUMN problem_desc TEXT",
		"ALTER TABLE orders ADD COLUMN cancel_reason TEXT",
		"ALTER TABLE orders ADD COLUMN coupon TEXT",
		"ALTER TABLE orders ADD COLUMN refund_amount INTEGER NOT NULL DEFAULT 0",
	} {
		if _, err := d.Exec(alter); err != nil && !isDuplicateColumn(err) {
			return fmt.Errorf("migrate alter: %w\nstmt: %s", err, alter)
		}
	}

	// Индексы по колонкам, добавляемым через ALTER (создаём после ALTER, чтобы
	// колонка уже существовала на старых БД).
	if _, err := d.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_coupon ON orders(coupon)`); err != nil {
		return fmt.Errorf("migrate index: %w", err)
	}

	// До выделения самостоятельной стадии raw-статус «Оплачен» сохранялся как
	// processing. Исправляем существующую витрину при старте, чтобы не требовать
	// от пользователя повторного импорта.
	falseValue := "0"
	if d.IsPostgres() {
		falseValue = "FALSE"
	}
	if _, err := d.Exec(`UPDATE orders SET status_stage = 'paid'
		WHERE status_raw = 'Оплачен' AND is_canceled = ` + falseValue); err != nil {
		return fmt.Errorf("migrate paid status: %w", err)
	}
	// Старые импорты не содержали журнал оплат, поэтому корректно восстановить
	// можно только полные возвраты: их сумма всегда равна сумме заказа. Частичные
	// будут рассчитаны при следующем импорте исходной выгрузки.
	if _, err := d.Exec(`UPDATE orders SET refund_amount = total_amount
		WHERE status_stage = 'returned' AND status_raw IN ('Возврат заказа', 'Совершён возврат средств', 'Совершен возврат средств')
		AND refund_amount = 0`); err != nil {
		return fmt.Errorf("migrate full refund amount: %w", err)
	}
	return nil
}

func isDuplicateColumn(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}
