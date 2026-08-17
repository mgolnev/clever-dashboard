// Package orderstage — SQL-предикаты стадий заказа, общие для metrics и funnel.
package orderstage

import "fmt"

// PaidReach — кумулятивная стадия «оплачен»: is_paid ИЛИ заказ продвинулся дальше
// оплаты (иначе воронка теряет монотонность из-за processing+ с пустым is_paid).
const paidReach = "%[2]sis_paid = %[1]s OR %[2]sstatus_stage IN ('paid','processing','shipped','in_pvz','completed','returned')"

// PaidReachSQL возвращает предикат с подставленным литералом TRUE/1.
// prefix — необязательный префикс колонок (например "o." в JOIN).
func PaidReachSQL(boolTrue string, prefix ...string) string {
	p := ""
	if len(prefix) > 0 {
		p = prefix[0]
	}
	return fmt.Sprintf(paidReach, boolTrue, p)
}

const (
	// InTransit — физическая доставка: заказ уже отправлен и ещё не завершён.
	InTransit = "status_stage IN ('shipped','in_pvz')"
	Terminal  = "status_stage IN ('completed','canceled','closed','returned')"
	Completed = "status_stage = 'completed'"
)
