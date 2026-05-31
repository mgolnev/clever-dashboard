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
	if d.IsPostgres() {
		pkAuto = "BIGSERIAL PRIMARY KEY"
		tsType = "TIMESTAMP"
		boolType = "BOOLEAN"
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
			delivery_cost INTEGER NOT NULL DEFAULT 0,
			status_raw TEXT,
			status_stage TEXT,
			is_paid %s NOT NULL DEFAULT 0,
			is_canceled %s NOT NULL DEFAULT 0,
			payment_system TEXT,
			delivery_service TEXT,
			channel TEXT,
			coupon TEXT,
			region TEXT,
			city TEXT,
			location_raw TEXT,
			has_problem %s NOT NULL DEFAULT 0,
			problem_desc TEXT,
			cancel_reason TEXT,
			items_count INTEGER NOT NULL DEFAULT 0,
			import_id INTEGER
		)`, tsType, tsType, boolType, boolType, boolType),

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
	}

	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w\nstmt: %s", err, s)
		}
	}

	// Идемпотентные ALTER для БД, созданных до добавления полей. Ошибка
	// "duplicate column" игнорируется.
	for _, alter := range []string{
		fmt.Sprintf("ALTER TABLE orders ADD COLUMN has_problem %s NOT NULL DEFAULT 0", boolType),
		"ALTER TABLE orders ADD COLUMN problem_desc TEXT",
		"ALTER TABLE orders ADD COLUMN cancel_reason TEXT",
		"ALTER TABLE orders ADD COLUMN coupon TEXT",
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
	return nil
}

func isDuplicateColumn(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}
