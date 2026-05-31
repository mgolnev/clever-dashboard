package ingestion

import (
	"regexp"
	"strings"

	"github.com/clever/clever-dashboard/internal/model"
	"github.com/clever/clever-dashboard/internal/normalize"
)

// Заголовки выгрузки «по заказам».
const (
	hCreatedAt = "Дата создания"
	hUpdatedAt = "Дата изменения"
	hNumber    = "Номер заказа"
	hCanceled  = "Отменен"
	hCustomer  = "Покупатель"
	hTotal     = "Сумма"
	hStatus    = "Статус"
	hPaid      = "Оплачен"
	hPayment   = "Платежная система"
	hDelivery  = "Служба доставки"
	hLocation  = "Выберите свой населенный пункт"
	hEmail     = "Email покупателя"
	hPhone     = "Номер телефона"
	hDeliveryC = "Стоимость доставки"
	hFromApp   = "Заказ из приложения"
	hPositions = "Позиции"
	hPrices    = "Цена товара"
	hProblem   = "Проблема с заказом"
	hProblemD  = "Описание проблемы с заказом"
	hCancelR   = "Причина отмены"
	hCoupon    = "Купоны заказа"
)

// positionRe вытаскивает позиции из столбца «Позиции»: [offer_id] Название (N шт).
var positionRe = regexp.MustCompile(`\[(\d+)\]\s*(.*?)\s*\((\d+)\s*шт\)`)

// MapOrders преобразует строки выгрузки в доменные заказы.
func MapOrders(records []Record) []model.Order {
	orders := make([]model.Order, 0, len(records))
	for _, rec := range records {
		num := rec.get(hNumber)
		if num == "" {
			continue
		}
		canceled := normalize.Bool(rec.get(hCanceled))
		statusRaw := rec.get(hStatus)
		region, city := normalize.Location(rec.get(hLocation))

		o := model.Order{
			OrderNumber:     num,
			CreatedAt:       normalize.Time(rec.get(hCreatedAt)),
			UpdatedAt:       normalize.Time(rec.get(hUpdatedAt)),
			Customer:        rec.get(hCustomer),
			Email:           rec.get(hEmail),
			Phone:           rec.get(hPhone),
			TotalAmount:     normalize.Money(rec.get(hTotal)),
			DeliveryCost:    normalize.Money(rec.get(hDeliveryC)),
			StatusRaw:       statusRaw,
			StatusStage:     normalize.StatusStage(statusRaw, canceled),
			IsPaid:          normalize.Bool(rec.get(hPaid)),
			IsCanceled:      canceled,
			PaymentSystem:   cleanDict(rec.get(hPayment)),
			DeliveryService: cleanDict(rec.get(hDelivery)),
			Channel:         channel(rec.get(hFromApp)),
			Coupon:          cleanDict(rec.get(hCoupon)),
			Region:          region,
			City:            city,
			LocationRaw:     rec.get(hLocation),
			HasProblem:      normalize.Bool(rec.get(hProblem)),
			ProblemDesc:     rec.get(hProblemD),
			CancelReason:    rec.get(hCancelR),
		}
		o.Items = parseItems(rec.get(hPositions), rec.get(hPrices))
		orders = append(orders, o)
	}
	return orders
}

// parseItems собирает позиции из столбца «Позиции» и сопоставляет цены из
// столбца «Цена товара» по порядку (валидировано: 1:1 на всех заказах).
func parseItems(positions, prices string) []model.OrderItem {
	matches := positionRe.FindAllStringSubmatch(positions, -1)
	if len(matches) == 0 {
		return nil
	}
	priceList := splitPrices(prices)
	items := make([]model.OrderItem, 0, len(matches))
	for i, m := range matches {
		offer := m[1]
		name := strings.TrimSpace(m[2])
		qty := atoi(m[3])
		price := 0
		if i < len(priceList) {
			price = priceList[i]
		}
		brand, category, gender, size := normalize.Product(name)
		items = append(items, model.OrderItem{
			OfferID:  offer,
			Name:     name,
			Qty:      qty,
			Price:    price,
			LineSum:  price * qty,
			Brand:    brand,
			Category: category,
			Gender:   gender,
			Size:     size,
		})
	}
	return items
}

// splitPrices разбивает "719 руб 719 руб 503 руб" -> [719, 719, 503].
func splitPrices(s string) []int {
	parts := strings.Split(s, "руб")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		out = append(out, normalize.Money(p))
	}
	return out
}

var bracketCodeRe = regexp.MustCompile(`\[[^\]]*\]`)
var multiSpaceRe = regexp.MustCompile(`\s{2,}`)

// cleanDict убирает технические коды Битрикса вида "[s1]" / "[128136]" из
// значений справочников (оплата/доставка) и схлопывает дубли.
func cleanDict(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = bracketCodeRe.ReplaceAllString(s, " ")
	s = multiSpaceRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, " ,", ",")
	s = strings.ReplaceAll(s, ",,", ",")
	s = strings.Trim(s, " ,")
	// Дедуп вида "Забрать из магазина Забрать из магазина" -> один экземпляр.
	if fields := strings.Fields(s); len(fields) >= 2 && len(fields)%2 == 0 {
		half := len(fields) / 2
		if strings.Join(fields[:half], " ") == strings.Join(fields[half:], " ") {
			s = strings.Join(fields[:half], " ")
		}
	}
	return s
}

func channel(fromApp string) string {
	if normalize.Bool(fromApp) {
		return "Приложение"
	}
	return "Сайт"
}

func atoi(s string) int {
	n := 0
	for _, ch := range strings.TrimSpace(s) {
		if ch < '0' || ch > '9' {
			continue
		}
		n = n*10 + int(ch-'0')
	}
	if n == 0 {
		return 1
	}
	return n
}
