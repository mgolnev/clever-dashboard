package orders

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clever/clever-dashboard/internal/config"
	"github.com/clever/clever-dashboard/internal/db"
)

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
	if result.RowsTotal != want || result.OrdersImported != want || result.ItemsImported != want {
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

func TestImportAlwaysReplacesOrders(t *testing.T) {
	svc := NewService(NewRepository(testDB(t)))

	csv1 := []byte("Номер заказа;Дата создания;Сумма;Статус;Оплачен;Отменен;Позиции;Цена товара\n" +
		"№A1;01.07.2026 10:00:00;1000 руб;Выполнен;Да;Нет;[11] CLEVER Футболка Мужской M (1 шт);1000 руб\n" +
		"№A2;02.07.2026 10:00:00;2000 руб;Выполнен;Да;Нет;[12] CLEVER Футболка Мужской L (1 шт);2000 руб\n")
	res1, err := svc.ImportFile("a.csv", csv1)
	if err != nil {
		t.Fatalf("import1: %v", err)
	}
	if res1.OrdersImported != 2 || res1.OrdersCleared != 0 {
		t.Fatalf("import1 unexpected: %+v", res1)
	}

	csv2 := []byte("Номер заказа;Дата создания;Сумма;Статус;Оплачен;Отменен;Позиции;Цена товара\n" +
		"№B1;03.07.2026 10:00:00;3000 руб;Выполнен;Да;Нет;[13] CLEVER Футболка Женский S (1 шт);3000 руб\n")
	res2, err := svc.ImportFile("b.csv", csv2)
	if err != nil {
		t.Fatalf("import2: %v", err)
	}
	if res2.OrdersCleared != 2 {
		t.Fatalf("ordersCleared=%d, want 2", res2.OrdersCleared)
	}
	if res2.OrdersImported != 1 {
		t.Fatalf("ordersImported=%d, want 1", res2.OrdersImported)
	}

	var n int
	if err := svc.repo.db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("orders in db=%d, want 1", n)
	}
	var num string
	if err := svc.repo.db.QueryRow(`SELECT order_number FROM orders`).Scan(&num); err != nil {
		t.Fatalf("select: %v", err)
	}
	if num != "№B1" {
		t.Fatalf("order_number=%q, want №B1", num)
	}
}
