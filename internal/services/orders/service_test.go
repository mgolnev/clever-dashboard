package orders

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
)

type readerWithTerminalError struct {
	reader *strings.Reader
	failed bool
}

func (r *readerWithTerminalError) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	if err == io.EOF && !r.failed {
		r.failed = true
		return 0, errors.New("injected read failure")
	}
	return n, err
}

func testDB(t *testing.T) *db.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(config.Config{DBDriver: "sqlite", DBDSN: dsn})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func TestImportStreamsMoreThanOneBatch(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	var csv strings.Builder
	csv.WriteString("Номер заказа;Дата создания;Сумма;Статус;Оплачен;Отменен;Позиции;Цена товара\n")
	for index := 0; index < importBatchSize*2+17; index++ {
		fmt.Fprintf(&csv, "№%d;01.07.2026 10:00:00;1000 руб;Выполнен;Да;Нет;[%d] CLEVER Футболка Мужской M (1 шт);1000 руб\n", index, index)
	}
	result, err := svc.Import("large.csv", strings.NewReader(csv.String()))
	if err != nil {
		t.Fatal(err)
	}
	want := importBatchSize*2 + 17
	if result.RowsTotal != want || result.OrdersImported != want || result.OrdersAdded != want || result.ItemsImported != want {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestFailedStreamingImportKeepsPreviousOrders(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	valid := "Номер заказа;Дата создания;Сумма\n№A1;01.07.2026 10:00:00;1000 руб\n"
	if _, err := svc.Import("valid.csv", strings.NewReader(valid)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Import("empty.csv", strings.NewReader("Номер заказа;Дата создания\n")); err == nil {
		t.Fatal("expected empty import error")
	}
	var number string
	if err := svc.repo.db.QueryRow(`SELECT order_number FROM orders`).Scan(&number); err != nil {
		t.Fatal(err)
	}
	if number != "№A1" {
		t.Fatalf("order_number=%q, want №A1", number)
	}
	var imports int
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM raw_import`).Scan(&imports); err != nil {
		t.Fatal(err)
	}
	if imports != 1 {
		t.Fatalf("raw_import count=%d, want 1", imports)
	}
}

func TestReadFailureAfterSavedBatchesRollsBackEntireMerge(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	valid := "Номер заказа;Дата создания;Сумма\n№EXISTING;01.07.2026 10:00:00;1000 руб\n"
	if _, err := svc.Import("valid.csv", strings.NewReader(valid)); err != nil {
		t.Fatal(err)
	}

	var csv strings.Builder
	csv.WriteString("Номер заказа;Дата создания;Сумма;Покупатель\n")
	for index := 0; index < importBatchSize*3; index++ {
		fmt.Fprintf(&csv, "№NEW-%d;02.07.2026 10:00:00;2000 руб;%s\n", index, strings.Repeat("x", 200))
	}
	reader := &readerWithTerminalError{reader: strings.NewReader(csv.String())}
	if _, err := svc.Import("broken.csv", reader); err == nil {
		t.Fatal("expected terminal read error")
	}

	var ordersCount, importsCount int
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&ordersCount); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM raw_import`).Scan(&importsCount); err != nil {
		t.Fatal(err)
	}
	if ordersCount != 1 || importsCount != 1 {
		t.Fatalf("partial merge became visible: orders=%d imports=%d", ordersCount, importsCount)
	}
}

func TestImportMergesHistoryAndUpdatesMatchingOrder(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))

	header := "Номер заказа;Дата создания;Дата изменения;Сумма;Статус;Оплачен;Отменен;Позиции;Цена товара\n"
	csv1 := []byte(header +
		"№A1;01.07.2026 10:00:00;02.07.2026 10:00:00;1000 руб;Собирается;Да;Нет;[11] CLEVER Футболка Мужской M (1 шт);1000 руб\n" +
		"№A2;02.07.2026 10:00:00;03.07.2026 10:00:00;2000 руб;Выполнен;Да;Нет;[12] CLEVER Футболка Мужской L (1 шт);2000 руб\n")
	res1, err := svc.ImportFile("a.csv", csv1)
	if err != nil {
		t.Fatalf("import1: %v", err)
	}
	if res1.OrdersImported != 2 || res1.OrdersAdded != 2 || res1.OrdersUpdated != 0 || res1.OrdersCleared != 0 {
		t.Fatalf("import1 unexpected: %+v", res1)
	}

	csv2 := []byte(header +
		"№A1;01.07.2026 10:00:00;04.07.2026 10:00:00;1500 руб;Выполнен;Да;Нет;[21] CLEVER Брюки Мужской M (2 шт);750 руб\n" +
		"№B1;03.07.2026 10:00:00;04.07.2026 11:00:00;3000 руб;Выполнен;Да;Нет;[13] CLEVER Футболка Женский S (1 шт);3000 руб\n")
	res2, err := svc.ImportFile("b.csv", csv2)
	if err != nil {
		t.Fatalf("import2: %v", err)
	}
	if res2.OrdersImported != 2 || res2.OrdersAdded != 1 || res2.OrdersUpdated != 1 || res2.OrdersSkipped != 0 {
		t.Fatalf("import2 unexpected: %+v", res2)
	}

	var n int
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 3 {
		t.Fatalf("orders in db=%d, want 3", n)
	}
	var amount int
	var status string
	if err := svc.repo.db.QueryRow(`SELECT total_amount, status_stage FROM orders WHERE order_number = '№A1'`).Scan(&amount, &status); err != nil {
		t.Fatalf("select: %v", err)
	}
	if amount != 1500 || status != "completed" {
		t.Fatalf("updated order amount=%d status=%q", amount, status)
	}
	var offerID string
	var qty int
	if err := svc.repo.db.QueryRow(`SELECT offer_id, qty FROM order_items WHERE order_number = '№A1'`).Scan(&offerID, &qty); err != nil {
		t.Fatalf("select items: %v", err)
	}
	if offerID != "21" || qty != 2 {
		t.Fatalf("updated item offer=%q qty=%d", offerID, qty)
	}
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE order_number = '№A2'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("historical order was not retained: count=%d err=%v", n, err)
	}
}

func TestImportSkipsOlderOrderVersion(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	header := "Номер заказа;Дата создания;Дата изменения;Сумма;Статус;Оплачен;Отменен;Позиции;Цена товара\n"
	newer := []byte(header +
		"№A1;01.07.2026 10:00:00;10.07.2026 10:00:00;1000 руб;Выполнен;Да;Нет;[11] CLEVER Футболка Мужской M (1 шт);1000 руб\n")
	if _, err := svc.ImportFile("newer.csv", newer); err != nil {
		t.Fatal(err)
	}
	older := []byte(header +
		"№A1;01.07.2026 10:00:00;05.07.2026 10:00:00;500 руб;Закрыт;Нет;Нет;[99] CLEVER Брюки Мужской M (1 шт);500 руб\n")
	result, err := svc.ImportFile("older.csv", older)
	if err != nil {
		t.Fatal(err)
	}
	if result.OrdersImported != 0 || result.OrdersSkipped != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	var amount int
	var status, offerID string
	if err := svc.repo.db.QueryRow(`SELECT total_amount, status_stage FROM orders WHERE order_number = '№A1'`).Scan(&amount, &status); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.db.QueryRow(`SELECT offer_id FROM order_items WHERE order_number = '№A1'`).Scan(&offerID); err != nil {
		t.Fatal(err)
	}
	if amount != 1000 || status != "completed" || offerID != "11" {
		t.Fatalf("newer state was overwritten: amount=%d status=%q offer=%q", amount, status, offerID)
	}
	var added, updated, skipped int
	if err := svc.repo.db.QueryRow(`SELECT orders_added, orders_updated, orders_skipped
		FROM raw_import WHERE filename = 'older.csv'`).Scan(&added, &updated, &skipped); err != nil {
		t.Fatal(err)
	}
	if added != 0 || updated != 0 || skipped != 1 {
		t.Fatalf("unexpected audit counters: added=%d updated=%d skipped=%d", added, updated, skipped)
	}
}

func TestRepeatedImportReplacesItemsWithoutDuplicates(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))
	csv := []byte("Номер заказа;Дата создания;Дата изменения;Сумма;Позиции;Цена товара\n" +
		"№A1;01.07.2026 10:00:00;02.07.2026 10:00:00;1500 руб;[11] CLEVER Футболка Мужской M (1 шт) [12] CLEVER Брюки Мужской M (1 шт);500 руб 1000 руб\n")
	if _, err := svc.ImportFile("first.csv", csv); err != nil {
		t.Fatal(err)
	}
	result, err := svc.ImportFile("repeat.csv", csv)
	if err != nil {
		t.Fatal(err)
	}
	if result.OrdersAdded != 0 || result.OrdersUpdated != 1 || result.ItemsImported != 2 {
		t.Fatalf("unexpected repeated import result: %+v", result)
	}
	var ordersCount, itemsCount int
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&ordersCount); err != nil {
		t.Fatal(err)
	}
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM order_items`).Scan(&itemsCount); err != nil {
		t.Fatal(err)
	}
	if ordersCount != 1 || itemsCount != 2 {
		t.Fatalf("duplicates after repeated import: orders=%d items=%d", ordersCount, itemsCount)
	}
}
