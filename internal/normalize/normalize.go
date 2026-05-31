// Package normalize приводит «сырые» строковые значения выгрузки Битрикса к
// доменным типам: деньги, география, стадии воронки, атрибуты товара.
package normalize

import (
	"regexp"
	"strings"
	"time"
)

var digitsOnly = regexp.MustCompile(`\d+`)

// Money парсит строку вида "3 313 руб" (с обычными и неразрывными пробелами)
// в целое число рублей.
func Money(s string) int {
	var n int
	for _, m := range digitsOnly.FindAllString(s, -1) {
		for _, ch := range m {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}

// Bool парсит "Да"/"Нет" (регистр и пробелы игнорируются).
func Bool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "да", "yes", "y", "true", "1":
		return true
	default:
		return false
	}
}

// Location разбирает "Россия, Пермский край, Пермь" -> регион, город.
// Регион — предпоследний значимый элемент, город — последний.
func Location(s string) (region, city string) {
	parts := splitTrim(s, ",")
	if len(parts) == 0 {
		return "", ""
	}
	// Отбрасываем ведущую "Россия".
	if strings.EqualFold(parts[0], "Россия") && len(parts) > 1 {
		parts = parts[1:]
	}
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], parts[0]
	default:
		city = parts[len(parts)-1]
		region = parts[0]
		return region, city
	}
}

// Stage code-стадии воронки.
const (
	StageNew        = "new"
	StageProcessing = "processing"
	StageShipped    = "shipped"
	StageInPVZ      = "in_pvz"
	StageCompleted  = "completed"
	StageClosed     = "closed"
	StageReturned   = "returned"
	StageCanceled   = "canceled"
	StageOther      = "other"
)

var statusMap = map[string]string{
	"не подтвержден":   StageNew,
	"оплачен":          StageProcessing,
	"принят":           StageProcessing,
	"обработан":        StageProcessing,
	"ожидает отправку": StageProcessing,
	"собирается":       StageProcessing,
	"отправлен":        StageShipped,
	"прибыл в пвз":     StageInPVZ,
	"выполнен":         StageCompleted,
	"закрыт":           StageClosed,
	"возврат заказа":   StageReturned,
	"совершён частичный возврат средств": StageReturned,
	"совершен частичный возврат средств": StageReturned,
	"совершён возврат средств":           StageReturned,
	"совершен возврат средств":           StageReturned,
}

// StatusStage маппит сырой статус Битрикса в каноническую стадию воронки.
// Флаг isCanceled имеет приоритет.
func StatusStage(statusRaw string, isCanceled bool) string {
	if isCanceled {
		return StageCanceled
	}
	key := strings.ToLower(strings.TrimSpace(statusRaw))
	if st, ok := statusMap[key]; ok {
		return st
	}
	return StageOther
}

// genderPrefixes нормализует "мужская/мужской/мужские" -> "Мужской" и т.д.
var genderPrefixes = []struct{ prefix, label string }{
	{"мужск", "Мужской"},
	{"женск", "Женский"},
	{"детск", "Детский"},
	{"унисекс", "Унисекс"},
}

// Product извлекает атрибуты из названия вида
// "CLEVER Футболка мужская 562538/04кжн т.синий/белый 48".
func Product(name string) (brand, category, gender, size string) {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "", "", "", ""
	}
	brand = fields[0]
	if len(fields) > 1 {
		category = fields[1]
	}
	if len(fields) > 2 {
		gender = normalizeGender(fields[2])
	}
	// Размер — последний токен, если он короткий (число или 1-3 символа).
	last := fields[len(fields)-1]
	if len(fields) > 2 && len([]rune(last)) <= 4 {
		size = last
	}
	return brand, category, gender, size
}

func normalizeGender(token string) string {
	low := strings.ToLower(token)
	for _, g := range genderPrefixes {
		if strings.HasPrefix(low, g.prefix) {
			return g.label
		}
	}
	return ""
}

// Time парсит "28.05.2026 18:34:24"; при ошибке возвращает нулевое время.
func Time(s string) time.Time {
	s = strings.TrimSpace(s)
	layouts := []string{"02.01.2006 15:04:05", "02.01.2006 15:04", "02.01.2006"}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func splitTrim(s, sep string) []string {
	raw := strings.Split(s, sep)
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
