package ingestion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseItems(t *testing.T) {
	positions := "[2304582] CLEVER Футболка мужская 562538/04кжн т.синий/белый 48 (1 шт) [2281972] CLEVER Футболка мужская 452589/02кд_п молочный 48 (2 шт)"
	prices := "719 руб 503 руб"
	items := parseItems(positions, prices)
	if len(items) != 2 {
		t.Fatalf("ожидалось 2 позиции, получено %d", len(items))
	}
	if items[0].OfferID != "2304582" || items[0].Price != 719 || items[0].Qty != 1 {
		t.Errorf("позиция 0: %+v", items[0])
	}
	if items[1].Qty != 2 || items[1].LineSum != 1006 {
		t.Errorf("позиция 1 line_sum: %+v", items[1])
	}
	if items[0].Category != "Футболка" || items[0].Gender != "Мужской" {
		t.Errorf("атрибуты товара: %+v", items[0])
	}
}

func TestMapOrdersHTML(t *testing.T) {
	html := `<table>
	<tr><td>Дата создания</td><td>Номер заказа</td><td>Отменен</td><td>Сумма</td><td>Статус</td><td>Оплачен</td><td>Заказ из приложения</td><td>Позиции</td><td>Цена товара</td><td>Выберите свой населенный пункт</td></tr>
	<tr><td>28.05.2026 18:34:24</td><td>№13019_75</td><td>Нет</td><td>1 222 руб</td><td>Выполнен</td><td>Да</td><td>Да</td><td>[100] CLEVER Носки женские Б700 белый 25 (3 шт)</td><td>149 руб</td><td>Россия, Пермский край, Пермь</td></tr>
	</table>`
	recs, err := ParseFile([]byte(html))
	if err != nil {
		t.Fatal(err)
	}
	orders := MapOrders(recs)
	if len(orders) != 1 {
		t.Fatalf("ожидался 1 заказ, получено %d", len(orders))
	}
	o := orders[0]
	if o.OrderNumber != "№13019_75" || o.TotalAmount != 1222 || o.Channel != "Приложение" {
		t.Errorf("заказ: %+v", o)
	}
	if o.City != "Пермь" || o.Region != "Пермский край" {
		t.Errorf("гео: %q/%q", o.Region, o.City)
	}
	if len(o.Items) != 1 || o.Items[0].Qty != 3 || o.Items[0].Price != 149 {
		t.Errorf("позиции: %+v", o.Items)
	}
}

// TestRealFile прогоняет реальную выгрузку, если она лежит в корне репозитория.
func TestRealFile(t *testing.T) {
	path := filepath.Join("..", "..", "sale_order.xls")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("реальный файл не найден: %v", err)
	}
	recs, err := ParseFile(data)
	if err != nil {
		t.Fatal(err)
	}
	orders := MapOrders(recs)
	items := 0
	for _, o := range orders {
		items += len(o.Items)
	}
	t.Logf("реальный файл: заказов=%d, позиций=%d", len(orders), items)
	if len(orders) < 1000 {
		t.Errorf("ожидалось >1000 заказов, получено %d", len(orders))
	}
}
