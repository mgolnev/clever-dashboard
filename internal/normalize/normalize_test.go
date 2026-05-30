package normalize

import "testing"

func TestMoney(t *testing.T) {
	cases := map[string]int{
		"3 313 руб": 3313,
		"719 руб":   719,
		"1\u00a0710 руб": 1710,
		"":          0,
		"0 руб":     0,
	}
	for in, want := range cases {
		if got := Money(in); got != want {
			t.Errorf("Money(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestLocation(t *testing.T) {
	reg, city := Location("Россия, Пермский край, Пермь")
	if reg != "Пермский край" || city != "Пермь" {
		t.Errorf("Location = %q/%q", reg, city)
	}
	reg, city = Location("Россия, Мурманская область, Мурманск")
	if city != "Мурманск" {
		t.Errorf("city = %q", city)
	}
}

func TestStatusStage(t *testing.T) {
	if got := StatusStage("Выполнен", false); got != StageCompleted {
		t.Errorf("got %q", got)
	}
	if got := StatusStage("Выполнен", true); got != StageCanceled {
		t.Errorf("canceled priority failed: %q", got)
	}
	if got := StatusStage("Прибыл в ПВЗ", false); got != StageInPVZ {
		t.Errorf("got %q", got)
	}
	// Полный возврат средств — это возврат, а не «прочее».
	if got := StatusStage("Совершён возврат средств", false); got != StageReturned {
		t.Errorf("полный возврат -> %q, ожидали returned", got)
	}
	// Статус «Оплачен» — заказ в обработке (в пути), а не «прочее».
	if got := StatusStage("Оплачен", false); got != StageProcessing {
		t.Errorf("оплачен -> %q, ожидали processing", got)
	}
}

func TestProduct(t *testing.T) {
	brand, cat, gender, size := Product("CLEVER Футболка мужская 562538/04кжн т.синий/белый 48")
	if brand != "CLEVER" || cat != "Футболка" || gender != "Мужской" || size != "48" {
		t.Errorf("Product = %q/%q/%q/%q", brand, cat, gender, size)
	}
}
